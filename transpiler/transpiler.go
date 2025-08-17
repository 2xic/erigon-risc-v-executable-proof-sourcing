package main

import (
	"encoding/binary"
	"erigon-transpiler-risc-v/prover"
	"erigon-transpiler-risc-v/tracer"

	"fmt"
	"strconv"

	"github.com/erigontech/erigon/core/vm"
	"github.com/holiman/uint256"
)

type DataSection struct {
	values []*uint256.Int
}

func NewDataSection() *DataSection {
	return &DataSection{
		values: make([]*uint256.Int, 0),
	}
}

func (ds *DataSection) Add(value *uint256.Int) string {
	ds.values = append(ds.values, value)
	index := len(ds.values) - 1
	return fmt.Sprintf("data_var_%d", index)
}

func (ds *DataSection) Iter() []struct {
	Name  string
	Value *uint256.Int
} {
	result := make([]struct {
		Name  string
		Value *uint256.Int
	}, len(ds.values))

	for i, value := range ds.values {
		result[i] = struct {
			Name  string
			Value *uint256.Int
		}{
			Name:  fmt.Sprintf("data_var_%d", i),
			Value: value,
		}
	}
	return result
}

type transpiler struct {
	instructions []prover.Instruction
	dataSection  *DataSection
}

func NewTranspiler() *transpiler {
	return &transpiler{
		instructions: make([]prover.Instruction, 0),
		dataSection:  NewDataSection(),
	}
}

func (tr *transpiler) AddInstruction(op *tracer.EvmInstructionMetadata) {
	switch op.Opcode {
	case vm.ADD:
		tr.instructions = append(tr.instructions, tr.add256Call()...)
	case vm.EQ:
		tr.instructions = append(tr.instructions, tr.eq256Call()...)
	case vm.SLT:
		tr.instructions = append(tr.instructions, tr.slt256Call()...)
	case vm.SHR:
		tr.instructions = append(tr.instructions, tr.shr256Call()...)
	case vm.GT:
		tr.instructions = append(tr.instructions, tr.gt256Call()...)
	case vm.LT:
		tr.instructions = append(tr.instructions, tr.lt256Call()...)
	case vm.NOT:
		tr.instructions = append(tr.instructions, tr.not256Call()...)
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
		tr.instructions = append(tr.instructions, tr.mstore256Call()...)
	case vm.MLOAD:
		tr.instructions = append(tr.instructions, tr.mload256Call()...)
	case vm.JUMPDEST:
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name: "NOP",
		})
	case vm.ISZERO:
		// TODO: optimize?
		tr.instructions = append(tr.instructions, tr.pushOpcode(0)...)
		tr.instructions = append(tr.instructions, tr.eq256Call()...)
	case vm.CALLVALUE:
		varName := tr.addArgumentToDataSection(op)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.CALLDATASIZE:
		varName := tr.addArgumentToDataSection(op)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
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

func (tr *transpiler) add256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "addi", Operands: []string{"a1", "sp", "32"}},
		{Name: "addi", Operands: []string{"a2", "sp", "32"}},
		{Name: "call", Operands: []string{"add256_stack_scratch"}},
	}
}

func (tr *transpiler) not256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "call", Operands: []string{"not256_stack_scratch"}},
	}
}

func (tr *transpiler) shr256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "32"}},
		{Name: "addi", Operands: []string{"a1", "sp", "0"}},
		{Name: "call", Operands: []string{"shr256_stack_scratch"}},
	}
}

func (tr *transpiler) slt256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "addi", Operands: []string{"a1", "sp", "32"}},
		{Name: "call", Operands: []string{"slt256_stack_scratch"}},
	}
}

func (tr *transpiler) eq256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "addi", Operands: []string{"a1", "sp", "32"}},
		{Name: "call", Operands: []string{"eq256_stack_scratch"}},
	}
}

func (tr *transpiler) gt256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "addi", Operands: []string{"a1", "sp", "32"}},
		{Name: "call", Operands: []string{"gt256_stack_scratch"}},
	}
}

func (tr *transpiler) lt256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "addi", Operands: []string{"a1", "sp", "32"}},
		{Name: "call", Operands: []string{"lt256_stack_scratch"}},
	}
}

func (tr *transpiler) mstore256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "addi", Operands: []string{"a1", "sp", "32"}},
		{Name: "call", Operands: []string{"mstore256_stack_scratch"}},
	}
}

func (tr *transpiler) mload256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "call", Operands: []string{"mload256_stack_scratch"}},
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

// TODO: make all arguments be uint256 instead?
func (tr *transpiler) addArgumentToDataSection(op *tracer.EvmInstructionMetadata) string {
	value := new(uint256.Int)
	if len(op.Arguments) > 0 {
		value.SetBytes(op.Arguments)
	}
	return tr.dataSection.Add(value)
}

func (tr *transpiler) loadFromDataSection(varName string) []prover.Instruction {
	instructions := []prover.Instruction{
		{
			Name:     "addi",
			Operands: []string{"sp", "sp", "-32"},
		},
		{
			Name:     "la",
			Operands: []string{"t0", varName},
		},
	}

	// Load all 8 32-bit words from data section to stack
	for i := 0; i < 8; i++ {
		offset := i * 4
		instructions = append(instructions, []prover.Instruction{
			{
				Name:     "lw",
				Operands: []string{"t1", fmt.Sprintf("%d(t0)", offset)},
			},
			{
				Name:     "sw",
				Operands: []string{"t1", fmt.Sprintf("%d(sp)", offset)},
			},
		}...)
	}

	return instructions
}

func (tr *transpiler) toAssembly() *prover.AssemblyFile {
	// Convert data section to prover format
	dataSection := make([]prover.DataVariable, 0)
	for _, dataVar := range tr.dataSection.Iter() {
		dataSection = append(dataSection, prover.DataVariable{
			Name:  dataVar.Name,
			Value: dataVar.Value,
		})
	}

	return &prover.AssemblyFile{
		Instructions: tr.instructions,
		DataSection:  dataSection,
	}
}
