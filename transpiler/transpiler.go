package main

import (
	"erigon-transpiler-risc-v/prover"

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

type Transpiler struct {
	instructions []prover.Instruction
}

func NewTranspiler() *Transpiler {
	return &Transpiler{
		instructions: make([]prover.Instruction, 0),
	}
}

func (tr *Transpiler) AddInstruction(op *evmInstructionMetadata) {
	switch op.opcode {
	case vm.ADD:
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "lw",
			Operands: []string{"t0", "0(sp)"},
		})
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "addi",
			Operands: []string{"sp", "sp", "8"},
		})
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "lw",
			Operands: []string{"t1", "0(sp)"},
		})
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "add",
			Operands: []string{"t2", "t0", "t1"},
		})
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "sw",
			Operands: []string{"t2", "0(sp)"},
		})
	case vm.PUSH1:
		// Move the stack pointer
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "addi",
			Operands: []string{"sp", "sp", "-8"},
		})
		constant := uint64(op.arguments[0])
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "li",
			Operands: []string{"t0", strconv.FormatUint(uint64(constant), 10)},
		})
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "sw",
			Operands: []string{"t0", "0(sp)"},
		})
	case vm.STOP:
		// no operation opcode
		return
	default:
		panic(fmt.Errorf("unimplemented opcode %d", uint64(op.opcode)))
	}
	// TODO: only add this for testing, not production.
	tr.instructions = append(tr.instructions, prover.Instruction{
		Name:     "EBREAK",
		Operands: []string{},
	})
}

func (tr *Transpiler) toAssembly() *prover.AssemblyFile {
	return &prover.AssemblyFile{
		Instructions: tr.instructions,
	}
}
