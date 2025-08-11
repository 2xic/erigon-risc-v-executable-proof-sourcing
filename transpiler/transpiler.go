package main

import (
	"encoding/binary"
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
		tr.instructions = append(tr.instructions, tr.add256Inline()...)
	case vm.EQ:
		tr.instructions = append(tr.instructions, tr.primitiveStackOperator(op.Opcode)...)
	case vm.SLT:
		tr.instructions = append(tr.instructions, tr.primitiveStackOperator(op.Opcode)...)
	case vm.SHR:
		tr.instructions = append(tr.instructions, tr.primitiveStackOperator(op.Opcode)...)
	case vm.GT:
		tr.instructions = append(tr.instructions, tr.primitiveStackOperator(op.Opcode)...)
	case vm.LT:
		tr.instructions = append(tr.instructions, tr.primitiveStackOperator(op.Opcode)...)
	case vm.NOT:
		tr.instructions = append(tr.instructions, tr.primitiveStackOperator(op.Opcode)...)
	case vm.PUSH0:
		tr.instructions = append(tr.instructions, tr.pushOpcode(uint64(0))...)
	case vm.PUSH1:
		tr.instructions = append(tr.instructions, tr.pushOpcode(uint64(op.Arguments[0]))...)
	case vm.PUSH4:
		value := binary.BigEndian.Uint32(op.Arguments)
		tr.instructions = append(tr.instructions, tr.pushOpcode(uint64(value))...)
	case vm.JUMPI:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.DUP1:
		tr.instructions = append(tr.instructions, tr.DupOpcode(1)...)
	case vm.DUP2:
		tr.instructions = append(tr.instructions, tr.DupOpcode(2)...)
	case vm.DUP3:
		tr.instructions = append(tr.instructions, tr.DupOpcode(3)...)
	case vm.SWAP1:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(1)...)
	case vm.SWAP2:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(2)...)
	case vm.POP:
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.MSTORE:
		tr.instructions = append(tr.instructions, []prover.Instruction{
			{
				Name:     "lw",
				Operands: []string{"t0", "0(sp)"},
			},
			{
				Name:     "lw",
				Operands: []string{"t1", "32(sp)"},
			},
			{
				Name:     "sw",
				Operands: []string{"t1", "0(t0)"},
			},
			{
				Name:     "addi",
				Operands: []string{"sp", "sp", "64"},
			},
		}...)
	case vm.MLOAD:
		tr.instructions = append(tr.instructions, []prover.Instruction{
			{
				Name:     "lw",
				Operands: []string{"t0", "0(sp)"},
			},
			{
				Name:     "lw",
				Operands: []string{"t1", "0(t0)"},
			},
			{
				Name:     "sw",
				Operands: []string{"t1", "0(sp)"},
			},
		}...)
	case vm.JUMPDEST:
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name: "NOP",
		})
	case vm.ISZERO:
		// TODO: optimize?
		tr.instructions = append(tr.instructions, tr.pushOpcode(0)...)
		tr.instructions = append(tr.instructions, tr.primitiveStackOperator(vm.EQ)...)
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
		riscOpcode = "srl"
		return []prover.Instruction{
			{
				Name:     "lw",
				Operands: []string{"t0", "0(sp)"},
			},
			{
				Name:     "addi",
				Operands: []string{"sp", "sp", "32"},
			},
			{
				Name:     "lw",
				Operands: []string{"t1", "0(sp)"},
			},
			{
				Name:     riscOpcode,
				Operands: []string{"t2", "t1", "t0"},
			},
			{
				Name:     "sw",
				Operands: []string{"t2", "0(sp)"},
			},
		}
	case vm.GT:
		riscOpcode = "SGT"
	case vm.LT:
		riscOpcode = "SLT"
	case vm.NOT:
		// Implement 256-bit NOT by XORing each of the 8 u32 words with -1
		instructions := make([]prover.Instruction, 0)
		for i := 0; i < 8; i++ {
			offset := i * 4
			instructions = append(instructions, []prover.Instruction{
				{
					Name:     "lw",
					Operands: []string{"t0", fmt.Sprintf("%d(sp)", offset)},
				},
				{
					Name:     "li",
					Operands: []string{"t1", "-1"},
				},
				{
					Name:     "xor",
					Operands: []string{"t2", "t0", "t1"},
				},
				{
					Name:     "sw",
					Operands: []string{"t2", fmt.Sprintf("%d(sp)", offset)},
				},
			}...)
		}
		return instructions
	case vm.EQ:
		return []prover.Instruction{
			{
				Name:     "lw",
				Operands: []string{"t0", "0(sp)"},
			},
			{
				Name:     "addi",
				Operands: []string{"sp", "sp", "32"},
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

	// For SLT, we need to swap operands because EVM compares top < second,
	// but RISC-V SLT compares rs1 < rs2
	var operand1, operand2 string
	if opcode == vm.SLT {
		operand1, operand2 = "t0", "t1" // Compare top < second
	} else {
		operand1, operand2 = "t1", "t0" // Compare second < top (default)
	}

	instructions := []prover.Instruction{
		{
			Name:     "lw",
			Operands: []string{"t0", "0(sp)"},
		},
		{
			Name:     "addi",
			Operands: []string{"sp", "sp", "32"},
		},
		{
			Name:     "lw",
			Operands: []string{"t1", "0(sp)"},
		},
		{
			Name:     riscOpcode,
			Operands: []string{"t2", operand1, operand2},
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
			Operands: []string{"sp", "sp", "-32"},
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

func (tr *transpiler) DupOpcode(index uint64) []prover.Instruction {
	spIndex := (32 * (index - 1))
	return []prover.Instruction{
		{
			Name:     "lw",
			Operands: []string{"t0", fmt.Sprintf("%d(sp)", spIndex)},
		},
		{
			Name:     "addi",
			Operands: []string{"sp", "sp", "-32"},
		},
		{
			Name:     "sw",
			Operands: []string{"t0", "0(sp)"},
		},
	}
}

func (tr *transpiler) SwapOpcode(index uint64) []prover.Instruction {
	spIndex := (32 * (index))
	return []prover.Instruction{
		{
			Name:     "lw",
			Operands: []string{"t0", "0(sp)"},
		},
		{
			Name:     "lw",
			Operands: []string{"t1", fmt.Sprintf("%d(sp)", spIndex)},
		},
		{
			Name:     "sw",
			Operands: []string{"t1", "0(sp)"},
		},
		{
			Name:     "sw",
			Operands: []string{"t0", fmt.Sprintf("%d(sp)", spIndex)},
		},
	}
}

func (tr *transpiler) add256Inline() []prover.Instruction {
	return []prover.Instruction{
		{Name: "li", Operands: []string{"t6", "0"}},
		{Name: "lw", Operands: []string{"t0", "0(sp)"}},
		{Name: "lw", Operands: []string{"t1", "32(sp)"}},
		{Name: "add", Operands: []string{"t2", "t0", "t1"}},
		{Name: "add", Operands: []string{"t3", "t2", "t6"}},
		{Name: "sltu", Operands: []string{"t4", "t2", "t0"}},
		{Name: "sltu", Operands: []string{"t5", "t3", "t2"}},
		{Name: "or", Operands: []string{"t6", "t4", "t5"}},
		{Name: "sw", Operands: []string{"t3", "32(sp)"}},
		{Name: "lw", Operands: []string{"t0", "4(sp)"}},
		{Name: "lw", Operands: []string{"t1", "36(sp)"}},
		{Name: "add", Operands: []string{"t2", "t0", "t1"}},
		{Name: "add", Operands: []string{"t3", "t2", "t6"}},
		{Name: "sltu", Operands: []string{"t4", "t2", "t0"}},
		{Name: "sltu", Operands: []string{"t5", "t3", "t2"}},
		{Name: "or", Operands: []string{"t6", "t4", "t5"}},
		{Name: "sw", Operands: []string{"t3", "36(sp)"}},
		{Name: "lw", Operands: []string{"t0", "8(sp)"}},
		{Name: "lw", Operands: []string{"t1", "40(sp)"}},
		{Name: "add", Operands: []string{"t2", "t0", "t1"}},
		{Name: "add", Operands: []string{"t3", "t2", "t6"}},
		{Name: "sltu", Operands: []string{"t4", "t2", "t0"}},
		{Name: "sltu", Operands: []string{"t5", "t3", "t2"}},
		{Name: "or", Operands: []string{"t6", "t4", "t5"}},
		{Name: "sw", Operands: []string{"t3", "40(sp)"}},
		{Name: "lw", Operands: []string{"t0", "12(sp)"}},
		{Name: "lw", Operands: []string{"t1", "44(sp)"}},
		{Name: "add", Operands: []string{"t2", "t0", "t1"}},
		{Name: "add", Operands: []string{"t3", "t2", "t6"}},
		{Name: "sltu", Operands: []string{"t4", "t2", "t0"}},
		{Name: "sltu", Operands: []string{"t5", "t3", "t2"}},
		{Name: "or", Operands: []string{"t6", "t4", "t5"}},
		{Name: "sw", Operands: []string{"t3", "44(sp)"}},
		{Name: "lw", Operands: []string{"t0", "16(sp)"}},
		{Name: "lw", Operands: []string{"t1", "48(sp)"}},
		{Name: "add", Operands: []string{"t2", "t0", "t1"}},
		{Name: "add", Operands: []string{"t3", "t2", "t6"}},
		{Name: "sltu", Operands: []string{"t4", "t2", "t0"}},
		{Name: "sltu", Operands: []string{"t5", "t3", "t2"}},
		{Name: "or", Operands: []string{"t6", "t4", "t5"}},
		{Name: "sw", Operands: []string{"t3", "48(sp)"}},
		{Name: "lw", Operands: []string{"t0", "20(sp)"}},
		{Name: "lw", Operands: []string{"t1", "52(sp)"}},
		{Name: "add", Operands: []string{"t2", "t0", "t1"}},
		{Name: "add", Operands: []string{"t3", "t2", "t6"}},
		{Name: "sltu", Operands: []string{"t4", "t2", "t0"}},
		{Name: "sltu", Operands: []string{"t5", "t3", "t2"}},
		{Name: "or", Operands: []string{"t6", "t4", "t5"}},
		{Name: "sw", Operands: []string{"t3", "52(sp)"}},
		{Name: "lw", Operands: []string{"t0", "24(sp)"}},
		{Name: "lw", Operands: []string{"t1", "56(sp)"}},
		{Name: "add", Operands: []string{"t2", "t0", "t1"}},
		{Name: "add", Operands: []string{"t3", "t2", "t6"}},
		{Name: "sltu", Operands: []string{"t4", "t2", "t0"}},
		{Name: "sltu", Operands: []string{"t5", "t3", "t2"}},
		{Name: "or", Operands: []string{"t6", "t4", "t5"}},
		{Name: "sw", Operands: []string{"t3", "56(sp)"}},
		{Name: "lw", Operands: []string{"t0", "28(sp)"}},
		{Name: "lw", Operands: []string{"t1", "60(sp)"}},
		{Name: "add", Operands: []string{"t2", "t0", "t1"}},
		{Name: "add", Operands: []string{"t3", "t2", "t6"}},
		{Name: "sltu", Operands: []string{"t4", "t2", "t0"}},
		{Name: "sltu", Operands: []string{"t5", "t3", "t2"}},
		{Name: "or", Operands: []string{"t6", "t4", "t5"}},
		{Name: "sw", Operands: []string{"t3", "60(sp)"}},
		{Name: "addi", Operands: []string{"sp", "sp", "32"}},
	}
}

func (tr *transpiler) popStack() []prover.Instruction {
	return []prover.Instruction{
		{
			Name:     "addi",
			Operands: []string{"sp", "sp", "32"},
		},
	}
}

func (tr *transpiler) toAssembly() *prover.AssemblyFile {
	return &prover.AssemblyFile{
		Instructions: tr.instructions,
	}
}
