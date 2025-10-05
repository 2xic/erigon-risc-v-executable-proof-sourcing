package transpiler

import (
	"encoding/binary"
	"encoding/json"
	"erigon-transpiler-risc-v/prover"
	"erigon-transpiler-risc-v/tracer"
	"os"

	"fmt"
	"strconv"

	"github.com/erigontech/erigon/core/vm"
	"github.com/holiman/uint256"
)

type transpiler struct {
	instructions    []prover.Instruction
	dataSection     *DataSection
	storageSection  *StorageSection
	enableSnapshots bool
	debugMappings   []EvmToRiscVMapping
	currentDepth    int
}

type EvmToRiscVMapping struct {
	EvmOpcode         string                `json:"evm_opcode"`
	RiscVInstructions []prover.Instruction  `json:"risc_v_instructions"`
	DataVariables     []prover.DataVariable `json:"data_variables"`
	CallDepth         int                   `json:"call_depth"`
}

func NewTranspiler() *transpiler {
	return &transpiler{
		instructions:    make([]prover.Instruction, 0),
		dataSection:     NewDataSection(),
		storageSection:  NewStorageSection(),
		enableSnapshots: false,
		debugMappings:   make([]EvmToRiscVMapping, 0),
		currentDepth:    0,
	}
}

func (tr *transpiler) EnableSnapshots() {
	tr.enableSnapshots = true
}

func (tr *transpiler) ProcessExecution(instructions []*tracer.EvmInstructionMetadata, executionState *tracer.EvmExecutionState) (EvmStackSnapshot, error) {
	snapshot := EvmStackSnapshot{
		Snapshots: make([][]uint256.Int, 0),
	}

	for i := range instructions {
		var resultStack *[]uint256.Int
		if i+1 < len(instructions) {
			resultStack = &instructions[i+1].StackSnapshot
		}

		err := tr.AddInstructionWithResult(instructions[i], executionState, resultStack)
		if err != nil {
			return snapshot, err
		}
		if i > 0 && tr.enableSnapshots {
			snapshot.Snapshots = append(snapshot.Snapshots, instructions[i].StackSnapshot)
		}
	}
	return snapshot, nil
}

func (tr *transpiler) AddInstruction(op *tracer.EvmInstructionMetadata, state *tracer.EvmExecutionState) error {
	return tr.AddInstructionWithResult(op, state, nil)
}

func (tr *transpiler) AddInstructionWithResult(op *tracer.EvmInstructionMetadata, state *tracer.EvmExecutionState, resultStack *[]uint256.Int) error {
	startInstructionCount := len(tr.instructions)

	if op.IsStackRestore {
		// Decrement call depth when returning from a call
		if tr.currentDepth > 0 {
			tr.currentDepth--
		}

		tr.instructions = append(tr.instructions, tr.restoreStackContext()...)
		if op.Result != nil {
			tr.instructions = append(tr.instructions, tr.pushOpcode(int32(op.Result.Uint64()))...)
		}
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "EBREAK",
			Operands: []string{},
		})

		// Record the mapping
		generatedInstructions := tr.instructions[startInstructionCount:]
		dataVars := tr.getDataSectionSnapshot()
		tr.debugMappings = append(tr.debugMappings, EvmToRiscVMapping{
			EvmOpcode:         "STACK_RESTORE",
			RiscVInstructions: make([]prover.Instruction, len(generatedInstructions)),
			DataVariables:     dataVars,
			CallDepth:         tr.currentDepth,
		})
		copy(tr.debugMappings[len(tr.debugMappings)-1].RiscVInstructions, generatedInstructions)

		return nil
	}

	switch op.Opcode {
	case vm.ADD:
		tr.instructions = append(tr.instructions, tr.add256Call()...)
	case vm.MUL:
		tr.instructions = append(tr.instructions, tr.mul256Call()...)
	case vm.SUB:
		tr.instructions = append(tr.instructions, tr.sub256Call()...)
	case vm.DIV:
		tr.instructions = append(tr.instructions, tr.div256Call()...)
	case vm.SDIV:
		sdivInstructions, err := tr.resultFromTraceCall(resultStack, 2, "SDIV")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, sdivInstructions...)
	case vm.MOD:
		modInstructions, err := tr.resultFromTraceCall(resultStack, 2, "MOD")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, modInstructions...)
	case vm.SMOD:
		smodInstructions, err := tr.resultFromTraceCall(resultStack, 2, "SMOD")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, smodInstructions...)
	case vm.ADDMOD:
		addmodInstructions, err := tr.resultFromTraceCall(resultStack, 3, "ADDMOD")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, addmodInstructions...)
	case vm.MULMOD:
		mulmodInstructions, err := tr.resultFromTraceCall(resultStack, 3, "MULMOD")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, mulmodInstructions...)
	case vm.EXP:
		expInstructions, err := tr.resultFromTraceCall(resultStack, 2, "EXP")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, expInstructions...)
	case vm.SIGNEXTEND:
		signextendInstructions, err := tr.resultFromTraceCall(resultStack, 2, "SIGNEXTEND")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, signextendInstructions...)
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
	case vm.SGT:
		sgtInstructions, err := tr.resultFromTraceCall(resultStack, 2, "SGT")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, sgtInstructions...)
	case vm.LT:
		tr.instructions = append(tr.instructions, tr.lt256Call()...)
	case vm.NOT:
		tr.instructions = append(tr.instructions, tr.not256Call()...)
	case vm.BYTE:
		byteInstructions, err := tr.resultFromTraceCall(resultStack, 2, "BYTE")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, byteInstructions...)
	case vm.SAR:
		sarInstructions, err := tr.resultFromTraceCall(resultStack, 2, "SAR")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, sarInstructions...)
	case vm.PUSH0:
		tr.instructions = append(tr.instructions, tr.pushOpcode(0)...)
	case vm.PUSH1:
		tr.instructions = append(tr.instructions, tr.pushOpcode(int32(op.Arguments[0]))...)
	case vm.PUSH2:
		value := binary.BigEndian.Uint16(op.Arguments)
		tr.instructions = append(tr.instructions, tr.pushOpcode(int32(value))...)
	case vm.PUSH3:
		value := uint64(op.Arguments[0])<<16 | uint64(op.Arguments[1])<<8 | uint64(op.Arguments[2])
		tr.instructions = append(tr.instructions, tr.pushOpcode(int32(value))...)
	case vm.PUSH4:
		value := binary.BigEndian.Uint32(op.Arguments)
		tr.instructions = append(tr.instructions, tr.pushOpcode(int32(value))...)
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
		// TODO: implement proper mstore operation that stores to memory
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.MLOAD:
		instructions, err := tr.resultFromTraceCall(resultStack, 1, "MLOAD")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, instructions...)
	case vm.JUMPDEST:
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name: "NOP",
		})
	case vm.ISZERO:
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
	case vm.BALANCE:
		instructions, err := tr.resultFromTraceCall(resultStack, 1, "BALANCE")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, instructions...)
	case vm.ORIGIN:
		originUint256 := new(uint256.Int)
		originUint256.SetBytes(state.Origin.Bytes())
		varName := tr.dataSection.Add(originUint256)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.TIMESTAMP:
		varName := tr.dataSection.Add(state.Timestamp)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.CHAINID:
		varName := tr.dataSection.Add(state.ChainId)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.COINBASE:
		coinbaseUint256 := new(uint256.Int)
		coinbaseUint256.SetBytes(state.Coinbase.Bytes())
		varName := tr.dataSection.Add(coinbaseUint256)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.BLOCKHASH:
		instructions, err := tr.resultFromTraceCall(resultStack, 1, "BLOCKHASH")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, instructions...)
	case vm.NUMBER:
		varName := tr.dataSection.Add(state.BlockNumber)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.DIFFICULTY:
		varName := tr.dataSection.Add(state.Difficulty)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.GASLIMIT:
		varName := tr.dataSection.Add(state.GasLimit)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.SELFBALANCE:
		instructions, err := tr.resultFromTraceCall(resultStack, 0, "SELFBALANCE")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, instructions...)
	case vm.BASEFEE:
		varName := tr.dataSection.Add(state.BaseFee)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.BLOBHASH:
		instructions, err := tr.resultFromTraceCall(resultStack, 1, "BLOBHASH")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, instructions...)
	case vm.BLOBBASEFEE:
		instructions, err := tr.resultFromTraceCall(resultStack, 0, "BLOBBASEFEE")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, instructions...)
	case vm.GASPRICE:
		instructions, err := tr.resultFromTraceCall(resultStack, 0, "GASPRICE")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, instructions...)
	case vm.CODESIZE:
		// Return size of current contract code
		codeSize := uint256.NewInt(uint64(len(state.CodeData)))
		varName := tr.dataSection.Add(codeSize)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.PC:
		instructions, err := tr.resultFromTraceCall(resultStack, 0, "PC")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, instructions...)
	case vm.MSIZE:
		instructions, err := tr.resultFromTraceCall(resultStack, 0, "MSIZE")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, instructions...)
	case vm.MSTORE8:
		// Pop offset and value, store byte to memory (dummy implementation)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.EXTCODECOPY:
		// Pop address, dest offset, code offset, size (dummy implementation)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.EXTCODEHASH:
		instructions, err := tr.resultFromTraceCall(resultStack, 1, "EXTCODEHASH")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, instructions...)
	case vm.CREATE:
		createInstructions, err := tr.resultFromTraceCall(resultStack, 3, "CREATE")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, createInstructions...)
	case vm.CREATE2:
		create2Instructions, err := tr.resultFromTraceCall(resultStack, 4, "CREATE2")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, create2Instructions...)
	case vm.SELFDESTRUCT:
		// Pop recipient address (dummy implementation)
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.EXTCODESIZE:
		instructions, err := tr.resultFromTraceCall(resultStack, 1, "EXTCODESIZE")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, instructions...)
	case vm.LOG0:
		// LOG0 pops 2 items: offset, size
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
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
	case vm.LOG4:
		// LOG4 pops 6 items: offset, size, topic1, topic2, topic3, topic4
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.CALLDATASIZE:
		size := uint256.NewInt(uint64(len(state.CallData)))
		varName := tr.dataSection.Add(size)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.RETURNDATASIZE:
		instructions, err := tr.resultFromTraceCall(resultStack, 0, "RETURNDATASIZE")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, instructions...)
	case vm.CALLDATALOAD:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		offset := op.StackSnapshot[0].Uint64()
		tr.instructions = append(tr.instructions, tr.calldataloadCall(offset, state.CallData)...)
	case vm.CALLDATACOPY:
		// TODO: implement calldata copy operation
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.CODECOPY:
		// TODO: implement proper codecopy operation
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.RETURNDATACOPY:
		// TODO: implement proper returndatacopy operation
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
	case vm.SSTORE:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		key := tr.getStorageKey(op.StackSnapshot[1])
		value := tr.getStorageValue(op.StackSnapshot[0])
		tr.storageSection.Store(tr.dataSection, key, value)
	case vm.TSTORE:
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
	case vm.TLOAD:
		key := tr.getStorageKey(op.StackSnapshot[0])
		tr.instructions = append(tr.instructions, tr.popStack()...)
		varName := tr.storageSection.Load(tr.dataSection, key)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.STOP:
		return nil
	case vm.RETURN:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "addi",
			Operands: []string{"sp", "s3", "0"},
		})
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "EBREAK",
			Operands: []string{},
		})
		return nil
	case vm.REVERT:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "addi",
			Operands: []string{"sp", "s3", "0"},
		})
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "EBREAK",
			Operands: []string{},
		})
		return nil
	case vm.INVALID:
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "addi",
			Operands: []string{"sp", "s3", "0"},
		})
		tr.instructions = append(tr.instructions, prover.Instruction{
			Name:     "EBREAK",
			Operands: []string{},
		})
		return nil
	case vm.CALLER:
		callerBytes := state.Caller.Bytes()
		callerValue := new(uint256.Int).SetBytes(callerBytes)
		varName := tr.dataSection.Add(callerValue)
		tr.instructions = append(tr.instructions, tr.loadFromDataSection(varName)...)
	case vm.KECCAK256:
		instructions, err := tr.resultFromTraceCall(resultStack, 2, "KECCAK256")
		if err != nil {
			return err
		}
		tr.instructions = append(tr.instructions, instructions...)
	case vm.MCOPY:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		// TODO: implement proper mcopy operation
		// destOffset := op.StackSnapshot[2].Uint64()
		// srcOffset := op.StackSnapshot[1].Uint64()
		// length := op.StackSnapshot[0].Uint64()
		// tr.instructions = append(tr.instructions, tr.mcopyCall(destOffset, srcOffset, length)...)
	case vm.CALL:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)

		tr.instructions = append(tr.instructions, tr.saveStackContext()...)
		tr.instructions = append(tr.instructions, tr.createNewStackFrame()...)
		tr.currentDepth++
	case vm.DELEGATECALL:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)

		tr.instructions = append(tr.instructions, tr.saveStackContext()...)
		tr.instructions = append(tr.instructions, tr.createNewStackFrame()...)
		tr.currentDepth++
	case vm.STATICCALL:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)

		tr.instructions = append(tr.instructions, tr.saveStackContext()...)
		tr.instructions = append(tr.instructions, tr.createNewStackFrame()...)
		tr.currentDepth++
	case vm.CALLCODE:
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)
		tr.instructions = append(tr.instructions, tr.popStack()...)

		tr.instructions = append(tr.instructions, tr.saveStackContext()...)
		tr.instructions = append(tr.instructions, tr.createNewStackFrame()...)
		tr.currentDepth++
	default:
		return fmt.Errorf("unimplemented opcode: 0x%02x", uint64(op.Opcode))
	}
	// TODO: only add this for testing, not production.
	tr.instructions = append(tr.instructions, prover.Instruction{
		Name:     "EBREAK",
		Operands: []string{},
	})

	// Record the mapping for this EVM opcode
	generatedInstructions := tr.instructions[startInstructionCount:]
	dataVars := tr.getDataSectionSnapshot()
	tr.debugMappings = append(tr.debugMappings, EvmToRiscVMapping{
		EvmOpcode:         op.Opcode.String(),
		RiscVInstructions: make([]prover.Instruction, len(generatedInstructions)),
		DataVariables:     dataVars,
		CallDepth:         tr.currentDepth,
	})
	copy(tr.debugMappings[len(tr.debugMappings)-1].RiscVInstructions, generatedInstructions)

	return nil
}

func (tr *transpiler) pushOpcode(value int32) []prover.Instruction {
	return []prover.Instruction{
		{
			Name:     "addi",
			Operands: []string{"sp", "sp", "-32"},
		},
		{
			Name:     "li",
			Operands: []string{"t0", strconv.FormatInt(int64(value), 10)},
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

// nolint:unused
func (tr *transpiler) mstore256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "addi", Operands: []string{"a1", "sp", "32"}},
		{Name: "call", Operands: []string{"mstore256_stack_scratch"}},
	}
}

// nolint:unused
func (tr *transpiler) mload256Call() []prover.Instruction {
	return []prover.Instruction{
		{Name: "addi", Operands: []string{"a0", "sp", "0"}},
		{Name: "call", Operands: []string{"mload256_stack_scratch"}},
	}
}

// nolint:unused
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

// nolint:unused
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

// nolint:unused
func (tr *transpiler) returndatacopyCall(destOffset, returnDataOffset, length uint64, returnData []byte) []prover.Instruction {
	var instructions []prover.Instruction

	returnDataToCopy := make([]byte, length)
	if returnDataOffset < uint64(len(returnData)) {
		end := returnDataOffset + length
		if end > uint64(len(returnData)) {
			end = uint64(len(returnData))
		}
		copy(returnDataToCopy, returnData[returnDataOffset:end])
	}

	for i := uint64(0); i < length; i += 32 {
		chunk := make([]byte, 32)
		if i < uint64(len(returnDataToCopy)) {
			copy(chunk, returnDataToCopy[i:])
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

func (tr *transpiler) saveStackContext() []prover.Instruction {
	return []prover.Instruction{

		{
			Name:     "addi",
			Operands: []string{"s1", "s1", "-4"},
		},
		{
			Name:     "sw",
			Operands: []string{"sp", "0(s1)"},
		},

		{
			Name:     "addi",
			Operands: []string{"s1", "s1", "-4"},
		},
		{
			Name:     "sw",
			Operands: []string{"s3", "0(s1)"},
		},
	}
}

func (tr *transpiler) createNewStackFrame() []prover.Instruction {
	return []prover.Instruction{
		{
			Name:     "addi",
			Operands: []string{"sp", "sp", "-1024"},
		},
		{
			Name:     "addi",
			Operands: []string{"s3", "sp", "0"},
		},
	}
}

func (tr *transpiler) restoreStackContext() []prover.Instruction {
	return []prover.Instruction{

		{
			Name:     "lw",
			Operands: []string{"s3", "0(s1)"},
		},
		{
			Name:     "addi",
			Operands: []string{"s1", "s1", "4"},
		},

		{
			Name:     "lw",
			Operands: []string{"sp", "0(s1)"},
		},
		{
			Name:     "addi",
			Operands: []string{"s1", "s1", "4"},
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

func (tr *transpiler) GetDebugMappings() []EvmToRiscVMapping {
	return tr.debugMappings
}

func (tr *transpiler) SaveDebugMappings(filename string) error {
	data, err := json.MarshalIndent(tr.debugMappings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (tr *transpiler) getDataSectionSnapshot() []prover.DataVariable {
	dataVars := make([]prover.DataVariable, 0)
	for _, dataVar := range tr.dataSection.Iter() {
		dataVars = append(dataVars, prover.DataVariable{
			Name:  dataVar.Name,
			Value: dataVar.Value,
		})
	}
	return dataVars
}

func (tr *transpiler) resultFromTraceCall(resultStack *[]uint256.Int, numArgs int, opName string) ([]prover.Instruction, error) {
	var instructions []prover.Instruction

	for i := 0; i < numArgs; i++ {
		instructions = append(instructions, tr.popStack()...)
	}

	if resultStack == nil {
		return nil, fmt.Errorf("%s requires result stack but none provided", opName)
	}

	a := *resultStack
	result := a[len(a)-1]
	varName := tr.dataSection.Add(&result)
	instructions = append(instructions, tr.loadFromDataSection(varName)...)

	return instructions, nil
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
