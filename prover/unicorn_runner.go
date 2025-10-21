package prover

import (
	"debug/elf"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/holiman/uint256"
	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

const EbreakInstr = 0x00100073

type VmRunner struct{}

type ExecutionResult struct {
	StackSnapshots *[][]uint256.Int
}

func NewUnicornRunner() (*VmRunner, error) {
	return &VmRunner{}, nil
}

func (vm *VmRunner) Execute(bytecode []byte) (*ExecutionResult, error) {
	mu, err := uc.NewUnicorn(uc.ARCH_RISCV, uc.MODE_RISCV64)
	if err != nil {
		return nil, err
	}

	codeAddr := uint64(0x10000)
	stackAddr := uint64(0x7fff0000)
	memSize := uint64(0x10000)

	err = mu.MemMap(0, memSize)
	if err != nil {
		return nil, NewPreRuntimeError(err)
	}

	err = mu.MemMap(codeAddr, memSize)
	if err != nil {
		return nil, NewPreRuntimeError(err)
	}
	err = mu.MemMap(stackAddr, memSize)
	if err != nil {
		return nil, NewPreRuntimeError(err)
	}

	stackTop := stackAddr + memSize - 16
	err = mu.RegWrite(uc.RISCV_REG_SP, stackTop)
	if err != nil {
		return nil, NewPreRuntimeError(err)
	}
	
	err = mu.RegWrite(uc.RISCV_REG_S3, stackTop)
	if err != nil {
		return nil, NewPreRuntimeError(err)
	}
	
	contextStackTop := uint64(0x6fff0000)
	err = mu.RegWrite(uc.RISCV_REG_S1, contextStackTop)
	if err != nil {
		return nil, NewPreRuntimeError(err)
	}
	
	err = mu.MemMap(contextStackTop-0x10000, 0x10000)
	if err != nil {
		return nil, NewPreRuntimeError(err)
	}

	allStackSnapshots := make([][]uint256.Int, 0)
	executionResults := &ExecutionResult{
		StackSnapshots: &allStackSnapshots,
	}

	instructionCounter := 0

	hook, err := mu.HookAdd(uc.HOOK_CODE, func(mu uc.Unicorn, addr uint64, size uint32) {
		instructionCounter++
		//	fmt.Printf("Executing instruction at address: 0x%x\n", addr)
		// Read the instruction at this address
		instrBytes, err := mu.MemRead(addr, uint64(size))
		if err != nil {
			panic(NewRuntimeError(err))
		}
		instr := binary.LittleEndian.Uint32(instrBytes)

		/*
			fmt.Printf("Executed %d instructions\n", instructionCounter)
			relativeAddress := addr - codeAddr - 0x00001000
			fmt.Printf("Current instruction address: 0x%x\n", relativeAddress)
			fmt.Printf("Instruction bytes: %x\n", instrBytes)

				if instructionCounter%1 == 0 && instructionCounter <= 100 {
					fmt.Printf("Executed %d instructions\n", instructionCounter)
					fmt.Printf("Current instruction address: 0x%x\n", addr)
					fmt.Printf("Instruction bytes: %x\n", instrBytes)
					fmt.Printf("start address: 0x%x\n", codeAddr+0x34)
					fmt.Printf("Instruction: 0x%x\n", instr)
				}
		*/

		if instr == uint32(EbreakInstr) {
			snapshot, err := printStackState(mu, stackAddr, memSize)
			if err != nil {
				panic(NewRuntimeError(err))
			}
			allStackSnapshots = append(allStackSnapshots, snapshot)
			pc, err := mu.RegRead(uc.RISCV_REG_PC)
			if err != nil {
				panic(NewRuntimeError(err))
			}
			// Skip 4 bytes (EBREAK instruction size)
			err = mu.RegWrite(uc.RISCV_REG_PC, pc+4)
			if err != nil {
				panic(NewRuntimeError(err))
			}
		}
	}, 1, 0)

	if err != nil {
		return nil, NewPreRuntimeError(err)
	}

	entryPoint, err := loadElfSections(mu, bytecode)
	if err != nil {
		return nil, NewPreRuntimeError(err)
	}

	err = mu.Start(entryPoint, 0)
	if err != nil {
		return nil, NewRuntimeError(err)
	}

	err = mu.HookDel(hook)
	if err != nil {
		return nil, NewPostRuntimeError(err)
	}
	err = mu.Close()

	if err != nil {
		return nil, NewPostRuntimeError(err)
	}
	return executionResults, nil
}

/*
func debugStackMemory(mu uc.Unicorn, stackBase, memSize uint64) {
	sp, _ := mu.RegRead(uc.RISCV_REG_SP)
	stackTop := stackBase + memSize - 16

	fmt.Printf("=== STACK DEBUG ===\n")
	fmt.Printf("Stack base: 0x%x\n", stackBase)
	fmt.Printf("Stack top: 0x%x\n", stackTop)
	fmt.Printf("SP: 0x%x\n", sp)
	fmt.Printf("Stack size used: %d bytes\n", stackTop-sp)

	// Print raw memory in 4-byte chunks from SP to stackTop
	fmt.Printf("Raw stack memory (from SP upward):\n")
	for addr := sp; addr < stackTop; addr += 4 {
		data, err := mu.MemRead(addr, 4)
		if err != nil {
			fmt.Printf("  0x%x: ERROR reading memory\n", addr)
			continue
		}
		word := binary.LittleEndian.Uint32(data)
		fmt.Printf("  0x%x: 0x%08x (%d)\n", addr, word, word)
	}
	fmt.Printf("===================\n")
}

	func printStackState(mu uc.Unicorn, stackBase, memSize uint64) ([]uint256.Int, error) {
		sp, _ := mu.RegRead(uc.RISCV_REG_SP)
		stackTop := stackBase + memSize - 16
		if sp > stackTop {
			return nil, fmt.Errorf("stack pointer (%d) exceeds stack top (%d)", sp, stackTop)
		}
		numWords := (stackTop - sp) / 4
		// 8 words per 256-bit entry (32-bit words)
		numEntries := numWords / 8
		stack := make([]uint256.Int, numEntries)
		fmt.Println("numEntries:", numEntries)
		for i := range numEntries {
			addr := sp + (i * 8 * 4)
			// Since our loadFromDataSection puts the significant word at sp+0,
			// we just need to read the first 4 bytes as a uint32
			data, err := mu.MemRead(addr, 4)
			if err != nil {
				return nil, err
			}
			word := binary.LittleEndian.Uint32(data)

			fmt.Printf("Entry %d: addr=0x%x, word=0x%x\n", i, addr, word)
			stack[uint64(len(stack)-1)-i] = *uint256.NewInt(uint64(word))
		}
		return stack, nil
	}
*/

func printStackState(mu uc.Unicorn, stackBase, memSize uint64) ([]uint256.Int, error) {
	sp, _ := mu.RegRead(uc.RISCV_REG_SP)
	
	logicalStackTop, err := mu.RegRead(uc.RISCV_REG_S3)
	var stackTop uint64
	if err != nil || logicalStackTop == 0 {
		stackTop = stackBase + memSize - 16
	} else {
		stackTop = logicalStackTop
	}

	if sp > stackTop {
		return nil, fmt.Errorf("stack pointer (%d) exceeds stack top (%d)", sp, stackTop)
	}
	numWords := (stackTop - sp) / 4
	// 8 words per 256-bit entry (32-bit words)
	numEntries := numWords / 8
	stack := make([]uint256.Int, numEntries)
	for i := range numEntries {
		// Read 8 consecutive 4-byte words for each 256-bit entry
		result := make([]byte, 32)
		for wordIdx := 0; wordIdx < 8; wordIdx++ {
			addr := sp + ((i*8 + uint64(wordIdx)) * 4)
			data, err := mu.MemRead(addr, 4)
			if err != nil {
				return nil, err
			}
			word := binary.LittleEndian.Uint32(data)
			// Place word in big-endian position (reverse word order)
			resultStart := (7 - wordIdx) * 4
			binary.BigEndian.PutUint32(result[resultStart:resultStart+4], word)
		}
		stack[uint64(len(stack)-1)-i].SetBytes(result)
	}
	return stack, nil
}

func loadElfSections(mu uc.Unicorn, bytecode []byte) (uint64, error) {
	tmpFile, err := os.CreateTemp("", "*.elf")
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(bytecode); err != nil {
		return 0, err
	}
	tmpFile.Close()

	elfFile, err := elf.Open(tmpFile.Name())
	if err != nil {
		return 0, fmt.Errorf("failed to parse ELF file: %v", err)
	}
	defer elfFile.Close()

	for _, section := range elfFile.Sections {
		if section.Type != elf.SHT_PROGBITS || section.Size == 0 || section.Addr == 0 {
			continue
		}

		data, err := section.Data()
		if err != nil {
			continue
		}

		mu.MemWrite(section.Addr, data)
	}

	return elfFile.Entry, nil
}

type RuntimeError struct {
	Err   error
	Stage string // "pre-runtime", "runtime" and "post-runtime"
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("%s error: %v", e.Stage, e.Err)
}

func (e RuntimeError) Unwrap() error {
	return e.Err
}

func NewRuntimeError(err error) error {
	return RuntimeError{Err: err, Stage: "runtime"}
}

func NewPreRuntimeError(err error) error {
	return RuntimeError{Err: err, Stage: "pre-runtime"}
}

func NewPostRuntimeError(err error) error {
	return RuntimeError{Err: err, Stage: "post-runtime"}
}
