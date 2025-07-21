package main

import (
	"fmt"
	"math/big"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon/core/vm"
	"github.com/erigontech/erigon/core/vm/evmtypes"
	"github.com/erigontech/erigon/params"
	"github.com/holiman/uint256"
)

// =============================================================================
// STATE TRACER
// =============================================================================

type StateTracer struct {
	jumpTable  *vm.JumpTable
	transpiler *Transpiler
}

func NewStateTracer() *StateTracer {
	return &StateTracer{
		jumpTable:  nil,
		transpiler: NewTranspiler(),
	}
}

func (t *StateTracer) setJumpTable(jt *vm.JumpTable) {
	t.jumpTable = jt
}

func (t *StateTracer) CaptureTxStart(gasLimit uint64) {}
func (t *StateTracer) CaptureTxEnd(restGas uint64)    {}
func (t *StateTracer) CaptureStart(env *vm.EVM, from, to libcommon.Address, precompile, create bool, input []byte, gas uint64, value *uint256.Int, code []byte) {
}
func (t *StateTracer) CaptureEnd(output []byte, usedGas uint64, err error) {}
func (t *StateTracer) CaptureEnter(typ vm.OpCode, from, to libcommon.Address, precompile, create bool, input []byte, gas uint64, value *uint256.Int, code []byte) {
}
func (t *StateTracer) CaptureExit(output []byte, usedGas uint64, err error) {}
func (t *StateTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, opDepth int, err error) {
}

func (t *StateTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, opDepth int, err error) {
	fmt.Printf("PC:%d %s Gas:%d len(Stack):%d", pc, op.String(), gas, scope.Stack.Len())
	fmt.Println()

	numPop := t.jumpTable[op].NumPop
	numPush := t.jumpTable[op].NumPush

	arguments := []byte{}
	if op.IsPushWithImmediateArgs() {
		size := uint64(op) - uint64(vm.PUSH1-1)
		arguments = make([]byte, size)
		index := 0
		for i := pc + 1; i <= pc+size; i++ {
			arguments[index] = scope.Contract.Code[i]
			index += 1
		}

	}

	t.transpiler.AddInstruction(&evmInstructionMetadata{
		opcode:    op,
		numPush:   uint64(numPush),
		numPop:    uint64(numPop),
		arguments: arguments,
	})
}

// =============================================================================
// VM construction
// =============================================================================

type SimpleTracer struct {
	state  *MockState
	tracer *StateTracer
	evm    *vm.EVM
}

func NewSimpleTracer() *SimpleTracer {
	state := NewMockState()

	blockCtx := evmtypes.BlockContext{
		CanTransfer: func(db evmtypes.IntraBlockState, addr libcommon.Address, amount *uint256.Int) (bool, error) {
			balance, _ := db.GetBalance(addr)
			return balance.Cmp(amount) >= 0, nil
		},
		Transfer: func(db evmtypes.IntraBlockState, sender, recipient libcommon.Address, amount *uint256.Int, bailout bool) error {
			return nil
		},
		GetHash:     func(uint64) libcommon.Hash { return libcommon.Hash{} },
		Coinbase:    libcommon.Address{},
		GasLimit:    1000000,
		BlockNumber: 1,
		Time:        1,
		Difficulty:  big.NewInt(1),
	}

	txCtx := evmtypes.TxContext{
		Origin:   libcommon.HexToAddress("0xabcd"),
		GasPrice: uint256.NewInt(1),
	}

	tracer := NewStateTracer()
	vmConfig := vm.Config{
		Tracer: tracer,
		Debug:  true,
	}

	evm := vm.NewEVM(blockCtx, txCtx, state, params.TestChainConfig, vmConfig)
	intrp := vm.NewEVMInterpreter(evm, evm.Config())
	tracer.setJumpTable(intrp.JT)

	return &SimpleTracer{
		state:  state,
		tracer: tracer,
		evm:    evm,
	}
}

func (tr *SimpleTracer) DeployContract(addr libcommon.Address, bytecode []byte, balance *uint256.Int) error {
	return tr.state.SetupContract(addr, bytecode, balance)
}

func (tr *SimpleTracer) ExecuteContract(contractAddr libcommon.Address, input []byte, gasLimit uint64) (*Transpiler, uint64, error) {
	caller := vm.AccountRef(libcommon.HexToAddress("0xabcd"))
	_, gasLeft, err := tr.evm.Call(caller, contractAddr, input, gasLimit, uint256.NewInt(0), false)
	return tr.tracer.transpiler, gasLimit - gasLeft, err
}

func (tr *SimpleTracer) GetStorageAt(addr libcommon.Address, key libcommon.Hash) (*uint256.Int, error) {
	var value uint256.Int
	err := tr.state.GetState(addr, &key, &value)
	return &value, err
}
