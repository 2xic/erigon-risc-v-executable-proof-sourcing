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

type transpiler struct {
	instructions   []prover.Instruction
	dataSection    *DataSection
	storageSection *StorageSection
}

func NewTranspiler() *transpiler {
	return &transpiler{
		instructions:   make([]prover.Instruction, 0),
		dataSection:    NewDataSection(),
		storageSection: NewStorageSection(),
	}
}

func (tr *transpiler) AddInstruction(op *tracer.EvmInstructionMetadata, state *tracer.EvmExecutionState) {
	switch op.Opcode {
	case vm.ADD:
		tr.instructions = append(tr.instructions, tr.add256Call()...)
	case vm.EQ:
		tr.instructions = append(tr.instructions, tr.eq256Call()...)
	case vm.SLT:
		tr.instructions = append(tr.instructions, tr.slt256Call()...)
	case vm.SHR:
		tr.instructions = append(tr.instructions, tr.shr256Call()...)
	case vm.SHL:
		tr.instructions = append(tr.instructions, tr.shl256Call()...)
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
		varName := tr.dataSection.Add(state.CallValue)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.CALLDATASIZE:
		size := uint256.NewInt(uint64(len(state.CallData)))
		varName := tr.dataSection.Add(size)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.CALLDATALOAD:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		offset := op.StackSnapshot[0].Uint64()
		tr.instructions = append(tr.instructions, tr.calldataloadCall(offset, state.CallData)...)
	case vm.CODECOPY:
		// Pop arguments and get parameters
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)

		destOffset := op.StackSnapshot[2].Uint64()
		codeOffset := op.StackSnapshot[1].Uint64()
		length := op.StackSnapshot[0].Uint64()

		tr.instructions = append(tr.instructions, tr.codecopyCall(destOffset, codeOffset, length, state.CodeData)...)
	case vm.SSTORE:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		key := tr.getStorageKey(op.StackSnapshot[1])
		value := tr.getStorageValue(op.StackSnapshot[0])
		tr.storageSection.Store(tr.dataSection, key, value)
	case vm.SLOAD:
		key := tr.getStorageKey(op.StackSnapshot[0])
		tr.instructions = append(tr.instructions, tr.popStack()...)
		varName := tr.storageSection.Load(tr.dataSection, key)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.STOP:
		// no operation opcode
		return
	case vm.RETURN:
		// Pop offset and size from stack, return normally
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		// TODO: set a return code?
		return
	case vm.REVERT:
		// Pop offset and size from stack, return with revert status
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		return
		// TODO: set a return code?
	case vm.INVALID:
		// TODO: set a return code?
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

func (tr *transpiler) shl256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "32"}},
		{Name: "addi", Operands: []string{"a1", "sp", "0"}},
		{Name: "call", Operands: []string{"shl256_stack_scratch"}},
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

func (tr *transpiler) calldataloadCall(offset uint64, callData []byte) []prover.Instruction {
	data := make([]byte, 32)
	if offset < uint64(len(callData)) {
		end := offset + 32
		if end > uint64(len(callData)) {
			end = uint64(len(callData))
		}
		copy(data, callData[offset:end])
	}

	value := new(uint256.Int)
	value.SetBytes(data)
	varName := tr.dataSection.Add(value)
	return tr.loadFromDataSection(varName)
}

func (tr *transpiler) codecopyCall(destOffset, codeOffset, length uint64, codeData []byte) []prover.Instruction {
	var instructions []prover.Instruction

	codeToCopy := make([]byte, length)
	if codeOffset < uint64(len(codeData)) {
		end := codeOffset + length
		if end > uint64(len(codeData)) {
			end = uint64(len(codeData))
		}
		copy(codeToCopy, codeData[codeOffset:end])
	}

	for i := uint64(0); i < length; i += 32 {
		chunk := make([]byte, 32)
		if i < uint64(len(codeToCopy)) {
			copy(chunk, codeToCopy[i:])
		}

		value := new(uint256.Int)
		value.SetBytes(chunk)
		varName := tr.dataSection.Add(value)

		instructions = append(instructions, []prover.Instruction{
			{Name: "li", Operands: []string{"t0", fmt.Sprintf("%d", destOffset+i)}},
			{Name: "la", Operands: []string{"t1", varName}},
		}...)

		for wordOffset := 0; wordOffset < 32; wordOffset += 4 {
			instructions = append(instructions, []prover.Instruction{
				{Name: "lw", Operands: []string{"t2", fmt.Sprintf("%d(t1)", wordOffset)}},
				{Name: "sw", Operands: []string{"t2", fmt.Sprintf("%d(t0)", wordOffset)}},
			}...)
		}
	}

	return instructions
}

func (tr *transpiler) popStack() []prover.Instruction {
	return []prover.Instruction{
		{
			Name:     "addi",
			Operands: []string{"sp", "sp", "32"},
		},
	}
}

func (tr *transpiler) getStorageKey(arguments uint256.Int) string {
	value := arguments.Hex()
	return value
}

func (tr *transpiler) getStorageValue(arguments uint256.Int) *uint256.Int {
	return &arguments
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
		dataOffset := i * 4
		stackOffset := i * 4
		instructions = append(instructions, []prover.Instruction{
			{
				Name:     "lw",
				Operands: []string{"t1", fmt.Sprintf("%d(t0)", dataOffset)},
			},
			{
				Name:     "sw",
				Operands: []string{"t1", fmt.Sprintf("%d(sp)", stackOffset)},
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

type DataSection struct {
	values []*uint256.Int
}

type StorageSection struct {
	keyToVar map[string]string
}

func NewDataSection() *DataSection {
	return &DataSection{
		values: make([]*uint256.Int, 0),
	}
}

func NewStorageSection() *StorageSection {
	return &StorageSection{keyToVar: make(map[string]string)}
}

func (ss *StorageSection) Store(dataSection *DataSection, key string, value *uint256.Int) string {
	if varName, exists := ss.keyToVar[key]; exists {
		return varName
	}
	varName := dataSection.Add(value)
	ss.keyToVar[key] = varName
	return varName
}

func (ss *StorageSection) Load(dataSection *DataSection, key string) string {
	if varName, exists := ss.keyToVar[key]; exists {
		return varName
	}
	varName := dataSection.Add(uint256.NewInt(0))
	ss.keyToVar[key] = varName
	return varName
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
