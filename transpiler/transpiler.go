package transpiler

import (
	"encoding/binary"
	"erigon-transpiler-risc-v/prover"
	"erigon-transpiler-risc-v/tracer"

	libcommon "github.com/erigontech/erigon-lib/common"

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

func (tr *transpiler) ProcessExecution(instructions []*tracer.EvmInstructionMetadata, executionState *tracer.EvmExecutionState) (EvmStackSnapshot, error) {
	snapshot := EvmStackSnapshot{
		Snapshots: make([][]uint256.Int, 0),
	}

	for i := range instructions {
		var resultStack []uint256.Int
		if i+1 < len(instructions) {
			resultStack = instructions[i+1].StackSnapshot
		}

		err := tr.AddInstructionWithResult(instructions[i], executionState, resultStack)
		if err != nil {
			return snapshot, err
		}
		if i > 0 {
			snapshot.Snapshots = append(snapshot.Snapshots, instructions[i].StackSnapshot)
		}
	}
	return snapshot, nil
}

func (tr *transpiler) AddInstruction(op *tracer.EvmInstructionMetadata, state *tracer.EvmExecutionState) error {
	return tr.AddInstructionWithResult(op, state, nil)
}

func (tr *transpiler) AddInstructionWithResult(op *tracer.EvmInstructionMetadata, state *tracer.EvmExecutionState, resultStack []uint256.Int) error {
	switch op.Opcode {
	case vm.ADD:
		tr.instructions = append(tr.instructions, tr.add256Call()...)
	case vm.MUL:
		tr.instructions = append(tr.instructions, tr.mul256Call()...)
	case vm.SUB:
		tr.instructions = append(tr.instructions, tr.sub256Call()...)
	case vm.DIV:
		tr.instructions = append(tr.instructions, tr.div256Call()...)
	case vm.AND:
		tr.instructions = append(tr.instructions, tr.and256Call()...)
	case vm.OR:
		tr.instructions = append(tr.instructions, tr.or256Call()...)
	case vm.XOR:
		tr.instructions = append(tr.instructions, tr.xor256Call()...)
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
	case vm.PUSH2:
		value := binary.BigEndian.Uint16(op.Arguments)
		tr.instructions = append(tr.instructions, tr.pushOpcode(uint64(value))...)
	case vm.PUSH3:
		value := uint64(op.Arguments[0])<<16 | uint64(op.Arguments[1])<<8 | uint64(op.Arguments[2])
		tr.instructions = append(tr.instructions, tr.pushOpcode(value)...)
	case vm.PUSH4:
		value := binary.BigEndian.Uint32(op.Arguments)
		tr.instructions = append(tr.instructions, tr.pushOpcode(uint64(value))...)
	case vm.PUSH5, vm.PUSH6, vm.PUSH7:
		value := new(uint256.Int)
		value.SetBytes(op.Arguments)
		varName := tr.dataSection.Add(value)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.PUSH8:
		value := new(uint256.Int)
		value.SetBytes(op.Arguments)
		varName := tr.dataSection.Add(value)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.PUSH9, vm.PUSH10, vm.PUSH11, vm.PUSH12, vm.PUSH13, vm.PUSH14, vm.PUSH15, vm.PUSH16,
		vm.PUSH17, vm.PUSH18, vm.PUSH19, vm.PUSH20, vm.PUSH21, vm.PUSH22, vm.PUSH23, vm.PUSH24,
		vm.PUSH25, vm.PUSH26, vm.PUSH27, vm.PUSH28, vm.PUSH29, vm.PUSH30, vm.PUSH31, vm.PUSH32:
		value := new(uint256.Int)
		value.SetBytes(op.Arguments)
		varName := tr.dataSection.Add(value)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.JUMP:
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.JUMPI:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.DUP1:
		tr.instructions = append(tr.instructions, tr.DupOpcode(1)...)
	case vm.DUP2:
		tr.instructions = append(tr.instructions, tr.DupOpcode(2)...)
	case vm.DUP3:
		tr.instructions = append(tr.instructions, tr.DupOpcode(3)...)
	case vm.DUP4:
		tr.instructions = append(tr.instructions, tr.DupOpcode(4)...)
	case vm.DUP5:
		tr.instructions = append(tr.instructions, tr.DupOpcode(5)...)
	case vm.DUP6:
		tr.instructions = append(tr.instructions, tr.DupOpcode(6)...)
	case vm.DUP7:
		tr.instructions = append(tr.instructions, tr.DupOpcode(7)...)
	case vm.DUP8:
		tr.instructions = append(tr.instructions, tr.DupOpcode(8)...)
	case vm.DUP9:
		tr.instructions = append(tr.instructions, tr.DupOpcode(9)...)
	case vm.DUP10:
		tr.instructions = append(tr.instructions, tr.DupOpcode(10)...)
	case vm.DUP11:
		tr.instructions = append(tr.instructions, tr.DupOpcode(11)...)
	case vm.DUP12:
		tr.instructions = append(tr.instructions, tr.DupOpcode(12)...)
	case vm.DUP13:
		tr.instructions = append(tr.instructions, tr.DupOpcode(13)...)
	case vm.DUP14:
		tr.instructions = append(tr.instructions, tr.DupOpcode(14)...)
	case vm.DUP15:
		tr.instructions = append(tr.instructions, tr.DupOpcode(15)...)
	case vm.DUP16:
		tr.instructions = append(tr.instructions, tr.DupOpcode(16)...)
	case vm.SWAP1:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(1)...)
	case vm.SWAP2:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(2)...)
	case vm.SWAP3:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(3)...)
	case vm.SWAP4:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(4)...)
	case vm.SWAP5:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(5)...)
	case vm.SWAP6:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(6)...)
	case vm.SWAP7:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(7)...)
	case vm.SWAP8:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(8)...)
	case vm.SWAP9:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(9)...)
	case vm.SWAP10:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(10)...)
	case vm.SWAP11:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(11)...)
	case vm.SWAP12:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(12)...)
	case vm.SWAP13:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(13)...)
	case vm.SWAP14:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(14)...)
	case vm.SWAP15:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(15)...)
	case vm.SWAP16:
		tr.instructions = append(tr.instructions, tr.SwapOpcode(16)...)
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
	case vm.GAS:
		varName := tr.dataSection.Add(state.Gas)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.ADDRESS:
		addressUint256 := new(uint256.Int)
		addressUint256.SetBytes(state.Address.Bytes())
		varName := tr.dataSection.Add(addressUint256)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.TIMESTAMP:
		varName := tr.dataSection.Add(state.Timestamp)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.CHAINID:
		varName := tr.dataSection.Add(state.ChainId)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.EXTCODESIZE:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		// Get the address from the stack snapshot and look up its code size
		addressToCheck := op.StackSnapshot[0]
		var addr libcommon.Address
		addressToCheck.WriteToSlice(addr[:])

		codeSize := uint64(0)
		if size, exists := state.CodeSizes[addr]; exists {
			codeSize = size
		}

		codeSizeUint256 := uint256.NewInt(codeSize)
		varName := tr.dataSection.Add(codeSizeUint256)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.LOG1:
		// LOG1 pops 3 items: offset, size, topic1
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.LOG2:
		// LOG2 pops 4 items: offset, size, topic1, topic2
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.LOG3:
		// LOG3 pops 5 items: offset, size, topic1, topic2, topic3
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
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
		return nil
	case vm.RETURN:
		// Pop offset and size from stack, return normally
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		// TODO: set a return code?
		return nil
	case vm.REVERT:
		// Pop offset and size from stack, return with revert status
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		return nil
		// TODO: set a return code?
	case vm.INVALID:
		// TODO: set a return code?
		return nil
	case vm.CALLER:
		callerBytes := state.Caller.Bytes()
		callerValue := new(uint256.Int).SetBytes(callerBytes)
		varName := tr.dataSection.Add(callerValue)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.KECCAK256:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)

		if resultStack == nil || len(resultStack) == 0 {
			return fmt.Errorf("KECCAK256 requires result stack but none provided")
		}
		hashResult := resultStack[len(resultStack)-1]
		varName := tr.dataSection.Add(&hashResult)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.MCOPY:
		// Pop arguments and get parameters
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)

		destOffset := op.StackSnapshot[2].Uint64()
		srcOffset := op.StackSnapshot[1].Uint64()
		length := op.StackSnapshot[0].Uint64()

		tr.instructions = append(tr.instructions, tr.mcopyCall(destOffset, srcOffset, length)...)
	default:
		return fmt.Errorf("unimplemented opcode: 0x%02x", uint64(op.Opcode))
	}
	// TODO: only add this for testing, not production.
	tr.instructions = append(tr.instructions, prover.Instruction{
		Name:     "EBREAK",
		Operands: []string{},
	})

	return nil
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

func (tr *transpiler) mul256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "addi", Operands: []string{"a1", "sp", "32"}},
		{Name: "call", Operands: []string{"mul256_stack_scratch"}},
	}
}

func (tr *transpiler) sub256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "addi", Operands: []string{"a1", "sp", "32"}},
		{Name: "call", Operands: []string{"sub256_stack_scratch"}},
	}
}

func (tr *transpiler) div256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "addi", Operands: []string{"a1", "sp", "32"}},
		{Name: "call", Operands: []string{"div256_stack_scratch"}},
	}
}

func (tr *transpiler) and256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "addi", Operands: []string{"a1", "sp", "32"}},
		{Name: "call", Operands: []string{"and256_stack_scratch"}},
	}
}

func (tr *transpiler) or256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "addi", Operands: []string{"a1", "sp", "32"}},
		{Name: "call", Operands: []string{"or256_stack_scratch"}},
	}
}

func (tr *transpiler) xor256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "addi", Operands: []string{"a1", "sp", "32"}},
		{Name: "call", Operands: []string{"xor256_stack_scratch"}},
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

func (tr *transpiler) mcopyCall(destOffset, srcOffset, length uint64) []prover.Instruction {
	var instructions []prover.Instruction

	if length == 0 {
		return instructions
	}

	for i := uint64(0); i < length; i++ {
		instructions = append(instructions, []prover.Instruction{
			{Name: "li", Operands: []string{"t0", fmt.Sprintf("%d", srcOffset+i)}},
			{Name: "lb", Operands: []string{"t1", "0(t0)"}},
			{Name: "li", Operands: []string{"t2", fmt.Sprintf("%d", destOffset+i)}},
			{Name: "sb", Operands: []string{"t1", "0(t2)"}},
		}...)
	}

	return instructions
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

func (tr *transpiler) ToAssembly() *prover.AssemblyFile {
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
