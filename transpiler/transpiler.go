package main

import (
	"erigon-transpiler-risc-v/prover"
	"erigon-transpiler-risc-v/tracer"

	"fmt"
	"strconv"

	"github.com/erigontech/erigon/core/vm"
)

type transpiler struct {
	instructions []prover.Instruction
}

func NewTranspiler() *transpiler {
	return &transpiler{
		instructions: make([]prover.Instruction, 0),
	}
}

func (tr *transpiler) AddInstruction(op *tracer.EvmInstructionMetadata) {
	switch op.Opcode {
	case vm.ADD:
		tr.instructions = append(tr.instructions, tr.primitiveStackOperator(op.Opcode)...)
	case vm.EQ:
		tr.instructions = append(tr.instructions, tr.primitiveStackOperator(op.Opcode)...)
	case vm.SLT:
		tr.instructions = append(tr.instructions, tr.primitiveStackOperator(op.Opcode)...)
	case vm.SHR:
		tr.instructions = append(tr.instructions, tr.primitiveStackOperator(op.Opcode)...)
	case vm.PUSH0:
		tr.instructions = append(tr.instructions, tr.pushOpcode(uint64(0))...)
	case vm.PUSH1:
		tr.instructions = append(tr.instructions, tr.pushOpcode(uint64(op.Arguments[0]))...)
	case vm.STOP:
		// no operation opcode
		return
	default:
		panic(fmt.Errorf("unimplemented opcode: 0x%02x", uint64(op.Opcode)))
	}
	// TODO: only add this for testing, not production.
	tr.instructions = append(tr.instructions, prover.Instruction{
		Name:     "EBREAK",
		Operands: []string{},
	})
}

// Takes two arguments of the stack does an operator on it and put the results back on the stack
func (tr *transpiler) primitiveStackOperator(opcode vm.OpCode) []prover.Instruction {
	var riscOpcode string
	switch opcode {
	case vm.ADD:
		riscOpcode = "ADD"
	case vm.SLT:
		riscOpcode = "SLT"
	case vm.SHR:
		riscOpcode = "SHR"
	case vm.GT:
		riscOpcode = "GT"
	case vm.LT:
		riscOpcode = "LT"
	case vm.EQ:
		return []prover.Instruction{
			{
				Name:     "lw",
				Operands: []string{"t0", "0(sp)"},
			},
			{
				Name:     "addi",
				Operands: []string{"sp", "sp", "8"},
			},
			{
				Name:     "lw",
				Operands: []string{"t1", "0(sp)"},
			},
			{
				Name:     "xor",
				Operands: []string{"t0", "t0", "t1"},
			},
			{
				Name:     "sltiu",
				Operands: []string{"t2", "t0", "1"},
			},
			{
				Name:     "sw",
				Operands: []string{"t2", "0(sp)"},
			},
		}
	default:
		panic("Bad opcode")
	}

	instructions := []prover.Instruction{
		{
			Name:     "lw",
			Operands: []string{"t0", "0(sp)"},
		},
		{
			Name:     "addi",
			Operands: []string{"sp", "sp", "8"},
		},
		{
			Name:     "lw",
			Operands: []string{"t1", "0(sp)"},
		},
		{
			Name:     riscOpcode,
			Operands: []string{"t2", "t0", "t1"},
		},
		{
			Name:     "sw",
			Operands: []string{"t2", "0(sp)"},
		},
	}
	return instructions
}

func (tr *transpiler) pushOpcode(value uint64) []prover.Instruction {
	return []prover.Instruction{
		{
			Name:     "addi",
			Operands: []string{"sp", "sp", "-8"},
		},
		{
			Name:     "li",
			Operands: []string{"t0", strconv.FormatUint(value, 10)},
		},
		{
			Name:     "sw",
			Operands: []string{"t0", "0(sp)"},
		},
	}
}

func (tr *transpiler) toAssembly() *prover.AssemblyFile {
	return &prover.AssemblyFile{
		Instructions: tr.instructions,
	}
}
