package main

import (
	"fmt"
	"strconv"

	"github.com/erigontech/erigon/core/vm"
)

type evmInstructionMetadata struct {
	opcode    vm.OpCode
	arguments []byte
	numPush   uint64
	numPop    uint64
}

type Instruction struct {
	name     string
	operands []string
}

type Transpiler struct {
	instructions []Instruction
}

func NewTranspiler() *Transpiler {
	return &Transpiler{
		instructions: make([]Instruction, 0),
	}
}

func (tr *Transpiler) AddInstruction(op *evmInstructionMetadata) {
	switch op.opcode {
	case vm.ADD:
		tr.instructions = append(tr.instructions, Instruction{
			name:     "lw",
			operands: []string{"t0", "0(sp)"},
		})
		tr.instructions = append(tr.instructions, Instruction{
			name:     "addi",
			operands: []string{"sp", "sp", "8"},
		})
		tr.instructions = append(tr.instructions, Instruction{
			name:     "lw",
			operands: []string{"t1", "0(sp)"},
		})
		tr.instructions = append(tr.instructions, Instruction{
			name:     "add",
			operands: []string{"t2", "t0", "t1"},
		})
		tr.instructions = append(tr.instructions, Instruction{
			name:     "sw",
			operands: []string{"t2", "0(sp)"},
		})
	case vm.PUSH1:
		// Move the stack pointer
		tr.instructions = append(tr.instructions, Instruction{
			name:     "addi",
			operands: []string{"sp", "sp", "-8"},
		})
		constant := uint64(op.arguments[0])
		tr.instructions = append(tr.instructions, Instruction{
			name:     "li",
			operands: []string{"t0", strconv.FormatUint(uint64(constant), 10)},
		})
		tr.instructions = append(tr.instructions, Instruction{
			name:     "sw",
			operands: []string{"t0", "0(sp)"},
		})
	case vm.STOP:
		// no operation opcode
		return
	default:
		panic(fmt.Errorf("unimplemented opcode %d", uint64(op.opcode)))
	}
	// TODO: only add this for testing, not production.
	tr.instructions = append(tr.instructions, Instruction{
		name:     "EBREAK",
		operands: []string{},
	})
}

func (tr *Transpiler) toAssembly() *AssemblyFile {
	return &AssemblyFile{
		instructions: tr.instructions,
	}
}
