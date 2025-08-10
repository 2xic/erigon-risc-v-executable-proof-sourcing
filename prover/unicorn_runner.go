package prover

import (
	"encoding/binary"
	"fmt"

	"github.com/holiman/uint256"
	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

const EbreakInstr = 0x00100073

type VmRunner struct{}

type ExecutionResult struct {
	StackSnapshots *[][]uint256.Int
}

func NewVmRunner() (*VmRunner, error) {
	return &VmRunner{}, nil
}

type RuntimeError struct {
	Err   error
	Stage string // "runtime" or "post-runtime"
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

	allStackSnapshots := make([][]uint256.Int, 0)
	executionResults := &ExecutionResult{
		StackSnapshots: &allStackSnapshots,
	}

	instructionCounter := 0

	hook, err := mu.HookAdd(uc.HOOK_CODE, func(mu uc.Unicorn, addr uint64, size uint32) {
		instructionCounter++
		//	fmt.Printf("Executing instruction at address: 0x%x\n", addr)
		// Read the instruction at this address
		instrBytes, err := mu.MemRead(addr, 4)
		if err != nil {
			panic(NewRuntimeError(err))
		}

		instr := binary.LittleEndian.Uint32(instrBytes)

		/*
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

	err = mu.MemWrite(codeAddr, bytecode)
	if err != nil {
		return nil, NewPreRuntimeError(err)
	}

	err = mu.Start(codeAddr+0x001000, 0)
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

func printStackState(mu uc.Unicorn, stackBase, memSize uint64) ([]uint256.Int, error) {
	sp, _ := mu.RegRead(uc.RISCV_REG_SP)
	stackTop := stackBase + memSize - 16

	if sp > stackTop {
		return nil, fmt.Errorf("stack pointer (%d) exceeds stack top (%d)", sp, stackTop)
	}
	numWords := (stackTop - sp) / 4
	numEntries := numWords / 8 // 8 words per 256-bit entry (32-bit words)
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
