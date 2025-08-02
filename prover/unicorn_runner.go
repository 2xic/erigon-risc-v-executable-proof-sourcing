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
		return nil, err
	}

	err = mu.MemMap(codeAddr, memSize)
	if err != nil {
		return nil, err
	}
	err = mu.MemMap(stackAddr, memSize)
	if err != nil {
		return nil, err
	}

	stackTop := stackAddr + memSize - 16
	err = mu.RegWrite(uc.RISCV_REG_SP, stackTop)
	if err != nil {
		return nil, err
	}

	allStackSnapshots := make([][]uint256.Int, 0)
	executionResults := &ExecutionResult{
		StackSnapshots: &allStackSnapshots,
	}

	hook, err := mu.HookAdd(uc.HOOK_CODE, func(mu uc.Unicorn, addr uint64, size uint32) {
		// Read the instruction at this address
		instrBytes, err := mu.MemRead(addr, 4)
		if err != nil {
			panic(err)
		}
		instr := binary.LittleEndian.Uint32(instrBytes)

		if instr == uint32(EbreakInstr) {
			snapshot, err := printStackState(mu, stackAddr, memSize)
			if err != nil {
				panic(err)
			}

			allStackSnapshots = append(allStackSnapshots, snapshot)
			pc, err := mu.RegRead(uc.RISCV_REG_PC)
			if err != nil {
				panic(err)
			}
			// Skip 4 bytes (EBREAK instruction size)
			err = mu.RegWrite(uc.RISCV_REG_PC, pc+4)
			if err != nil {
				panic(err)
			}
		}
	}, 1, 0)

	if err != nil {
		return nil, err
	}

	err = mu.MemWrite(codeAddr, bytecode)
	if err != nil {
		return nil, err
	}
	err = mu.Start(codeAddr, codeAddr+uint64(len(bytecode)))
	if err != nil {
		return nil, err
	}

	err = mu.HookDel(hook)
	if err != nil {
		return nil, err
	}
	err = mu.Close()

	if err != nil {
		return nil, err
	}
	return executionResults, nil
}

func printStackState(mu uc.Unicorn, stackBase, memSize uint64) ([]uint256.Int, error) {
	sp, _ := mu.RegRead(uc.RISCV_REG_SP)
	stackTop := stackBase + memSize - 16

	if sp > stackTop {
		return nil, fmt.Errorf("stack pointer (%d) exceeds stack top (%d)", sp, stackTop)
	}
	numEntries := (stackTop - sp) / 8
	stack := make([]uint256.Int, numEntries)
	for i := range numEntries {
		addr := sp + (i * 8)
		data, err := mu.MemRead(addr, 8)
		if err != nil {
			return nil, err
		}
		value := binary.LittleEndian.Uint64(data)
		w := uint64(len(stack)) - 1 - i
		stack[w] = *uint256.NewInt(value)
	}
	return stack, nil
}
