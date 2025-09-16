package tracer

import (
	"fmt"
	"math/big"
	"strings"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon/core/tracing"
	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/core/vm"
	"github.com/erigontech/erigon/core/vm/evmtypes"
	"github.com/erigontech/erigon/params"
	"github.com/holiman/uint256"
)

type EvmExecutionState struct {
	CallValue *uint256.Int
	CallData  []byte
	CodeData  []byte
	Gas       *uint256.Int
	Address   libcommon.Address
	Timestamp *uint256.Int
	ChainId   *uint256.Int
	CodeSizes map[libcommon.Address]uint64
}

type EvmInstructionMetadata struct {
	Opcode        vm.OpCode
	Arguments     []byte
	StackSnapshot []uint256.Int
}

// =============================================================================
// STATE TRACER
// =============================================================================

type StateTracer struct {
	jumpTable       *vm.JumpTable
	evmInstructions []*EvmInstructionMetadata
	executionState  *EvmExecutionState
	blockTime       uint64
	chainId         *uint256.Int
}

func (t *StateTracer) Hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnTxStart: func(vm *tracing.VMContext, txn types.Transaction, from libcommon.Address) {
			fmt.Println("hello!!")
		},
		OnOpcode: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
			fmt.Printf("🔥 OnOpcode hook: pc=%d op=%s gas=%d\n", pc, vm.OpCode(op).String(), gas)
		},
	}

}

func (t *StateTracer) TracingHooks() *tracing.Hooks {
	return t.Hooks()

}

func NewStateTracer() *StateTracer {
	tracer := &StateTracer{
		jumpTable: nil,
	}
	// Set Hooks to point to itself since StateTracer implements vm.EVMLogger
	// tracer.Hooks = tracer
	return tracer
}

func (t *StateTracer) setJumpTable(jt *vm.JumpTable) {
	t.jumpTable = jt
}

func (t *StateTracer) CaptureTxStart(gasLimit uint64) {
	fmt.Println("CaptureTxStart")
}
func (t *StateTracer) CaptureTxEnd(restGas uint64) {
	fmt.Println("CaptureTxEnd")
}
func (t *StateTracer) CaptureStart(env *vm.EVM, from, to libcommon.Address, precompile, create bool, input []byte, gas uint64, value *uint256.Int, code []byte) {
	fmt.Println("CaptureStart")
	t.blockTime = env.Context.Time
	t.chainId = new(uint256.Int)

	// Safely access chain config
	if env.ChainConfig() != nil && env.ChainConfig().ChainID != nil {
		t.chainId.SetFromBig(env.ChainConfig().ChainID)
	} else {
		t.chainId.SetUint64(1) // Default to mainnet if unavailable
	}
}
func (t *StateTracer) CaptureEnd(output []byte, usedGas uint64, err error) {
	fmt.Println("CaptureEnd")
}
func (t *StateTracer) CaptureEnter(typ vm.OpCode, from, to libcommon.Address, precompile, create bool, input []byte, gas uint64, value *uint256.Int, code []byte) {
	fmt.Println("CaptureEnter")
}
func (t *StateTracer) CaptureExit(output []byte, usedGas uint64, err error) {
	fmt.Println("CaptureExit")
}
func (t *StateTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, opDepth int, err error) {
	fmt.Println("CaptureFault")
}

func (t *StateTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, opDepth int, err error) {
	fmt.Printf("💫 CaptureState: pc=%d op=%s gas=%d stack_len=%d\n", pc, op.String(), gas, scope.Stack.Len())
	log.Debug("PC:%d %s Gas:%d len(Stack):%d", pc, op.String(), gas, scope.Stack.Len())

	arguments := []byte{}
	if op.IsPushWithImmediateArgs() {
		size := uint64(op) - uint64(vm.PUSH1-1)
		arguments = make([]byte, size)
		index := 0
		for i := pc + 1; i <= pc+size; i++ {
			arguments[index] = scope.Contract.Code[i]
			index++
		}
	}

	snapshot := make([]uint256.Int, len(scope.Stack.Data))
	for i := range len(scope.Stack.Data) {
		snapshot[i] = scope.Stack.Data[i]
	}

	// TODO: this should likely not be re-computed
	codeSizes := make(map[libcommon.Address]uint64)
	codeSizes[scope.Contract.Address()] = uint64(len(scope.Contract.Code))

	t.executionState = &EvmExecutionState{
		CallValue: scope.Contract.Value(),
		CallData:  scope.Contract.Input,
		CodeData:  scope.Contract.Code,
		Gas:       uint256.NewInt(gas),
		Address:   scope.Contract.Address(),
		Timestamp: uint256.NewInt(t.blockTime),
		ChainId:   t.chainId,
		CodeSizes: codeSizes,
	}

	t.evmInstructions = append(t.evmInstructions, &EvmInstructionMetadata{
		Opcode:        op,
		Arguments:     arguments,
		StackSnapshot: snapshot,
	})
}

// GetInstructions returns all captured instructions
func (t *StateTracer) GetInstructions() []*EvmInstructionMetadata {
	return t.evmInstructions
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
		BlockNumber: 23041867,
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

	evm := vm.NewEVM(blockCtx, txCtx, state, params.AllProtocolChanges, vmConfig)
	in := vm.NewEVMInterpreter(evm, evm.Config())
	tracer.setJumpTable(in.JT)

	return &SimpleTracer{
		state:  state,
		tracer: tracer,
		evm:    evm,
	}
}

func (tr *SimpleTracer) DeployContract(addr libcommon.Address, bytecode []byte, balance *uint256.Int) error {
	return tr.state.SetupContract(addr, bytecode, balance)
}

func (tr *SimpleTracer) ExecuteContract(contractAddr libcommon.Address, input []byte, gasLimit uint64, callValue *uint256.Int) ([]*EvmInstructionMetadata, *EvmExecutionState, uint64, error) {
	if callValue == nil {
		callValue = uint256.NewInt(0)
	}
	callerAddr := libcommon.HexToAddress("0xabcd")
	caller := vm.AccountRef(callerAddr)

	if callValue.Cmp(uint256.NewInt(0)) > 0 {
		currentBalance, err := tr.state.GetBalance(callerAddr)
		if err != nil {
			return nil, nil, 0, err
		}
		if currentBalance.Cmp(callValue) < 0 {
			neededBalance := new(uint256.Int).Add(callValue, uint256.NewInt(1000000))
			tr.state.AddBalance(callerAddr, neededBalance, tracing.BalanceChangeUnspecified)
		}
	}

	_, gasLeft, err := tr.evm.Call(caller, contractAddr, input, gasLimit, callValue, false)
	// We don't want these errors to propagate
	if err == vm.ErrExecutionReverted || (err != nil && strings.Contains(err.Error(), "invalid opcode:")) {
		log.Warn("vm error: %w", err)
		err = nil
	}
	return tr.tracer.evmInstructions, tr.tracer.executionState, gasLimit - gasLeft, err
}

func (tr *SimpleTracer) GetStorageAt(addr libcommon.Address, key libcommon.Hash) (*uint256.Int, error) {
	var value uint256.Int
	err := tr.state.GetState(addr, &key, &value)
	return &value, err
}
