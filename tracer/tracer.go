package tracer

import (
	"encoding/json"
	"erigon-transpiler-risc-v/prover"
	"fmt"
	"math/big"
	"strings"

	"github.com/erigontech/erigon-lib/chain"
	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon-lib/types"
	"github.com/erigontech/erigon-lib/types/accounts"
	"github.com/erigontech/erigon/core/state"
	"github.com/erigontech/erigon/core/tracing"
	"github.com/erigontech/erigon/core/vm"
	"github.com/erigontech/erigon/core/vm/evmtypes"
	"github.com/erigontech/erigon/eth/tracers"
	"github.com/holiman/uint256"
)

type EvmExecutionState struct {
	CallValue *uint256.Int
	CallData  []byte
	CodeData  []byte
	Gas       *uint256.Int
	Address   libcommon.Address
	Caller    libcommon.Address
	Timestamp *uint256.Int
	ChainId   *uint256.Int
	CodeSizes map[libcommon.Address]uint64
}

type EvmInstructionMetadata struct {
	Opcode        vm.OpCode
	Arguments     []byte
	StackSnapshot []uint256.Int
	Result        *uint256.Int // Stores the result of operations like KECCAK256
}

// =============================================================================
// MOCK STATE READER
// =============================================================================

type MockStateReader struct{}

func (m *MockStateReader) ReadAccountData(address libcommon.Address) (*accounts.Account, error) {
	return &accounts.Account{}, nil
}

func (m *MockStateReader) ReadAccountDataForDebug(address libcommon.Address) (*accounts.Account, error) {
	return &accounts.Account{}, nil
}

func (m *MockStateReader) ReadAccountStorage(address libcommon.Address, key libcommon.Hash) (uint256.Int, bool, error) {
	return *uint256.NewInt(0), false, nil
}

func (m *MockStateReader) HasStorage(address libcommon.Address) (bool, error) {
	return false, nil
}

func (m *MockStateReader) ReadAccountCode(address libcommon.Address) ([]byte, error) {
	return nil, nil
}

func (m *MockStateReader) ReadAccountCodeSize(address libcommon.Address) (int, error) {
	return 0, nil
}

func (m *MockStateReader) ReadAccountIncarnation(address libcommon.Address) (uint64, error) {
	return 0, nil
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

func (t *StateTracer) CaptureTxStart(vm *tracing.VMContext, tx types.Transaction, from libcommon.Address) {
	t.blockTime = vm.Time
	t.chainId = new(uint256.Int)
	t.chainId.SetFromBig(vm.ChainConfig.ChainID)
}
func (t *StateTracer) CaptureTxEnd(receipt *types.Receipt, err error) {}
func (t *StateTracer) CaptureEnter(depth int, typ byte, from libcommon.Address, to libcommon.Address, precompile bool, input []byte, gas uint64, value *uint256.Int, code []byte) {
}
func (t *StateTracer) CaptureExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
}
func (t *StateTracer) CaptureFault(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
}

func (t *StateTracer) CaptureState(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	stackData := scope.StackData()

	arguments := []byte{}
	opCode := vm.OpCode(op)
	if opCode.IsPushWithImmediateArgs() {
		size := uint64(op) - uint64(vm.PUSH1-1)
		arguments = make([]byte, size)
		code := scope.Code()
		for i := uint64(0); i < size; i++ {
			if pc+1+i < uint64(len(code)) {
				arguments[i] = code[pc+1+i]
			}
		}
	}

	snapshot := make([]uint256.Int, len(stackData))
	copy(snapshot, stackData)

	// TODO: this should likely not be re-computed
	codeSizes := make(map[libcommon.Address]uint64)
	codeSizes[scope.Address()] = uint64(len(scope.Code()))

	t.executionState = &EvmExecutionState{
		CallValue: scope.CallValue(),
		CallData:  scope.CallInput(),
		CodeData:  scope.Code(),
		Gas:       uint256.NewInt(gas),
		Address:   scope.Address(),
		Caller:    scope.Caller(),
		Timestamp: uint256.NewInt(t.blockTime),
		ChainId:   t.chainId,
		CodeSizes: codeSizes,
	}

	t.evmInstructions = append(t.evmInstructions, &EvmInstructionMetadata{
		Opcode:        vm.OpCode(op),
		Arguments:     arguments,
		StackSnapshot: snapshot,
	})
}

// GetInstructions returns all captured instructions
func (t *StateTracer) GetInstructions() []*EvmInstructionMetadata {
	return t.evmInstructions
}

func (t *StateTracer) GetExecutionState() *EvmExecutionState {
	return t.executionState
}

func (t *StateTracer) Hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnOpcode:  t.CaptureState,
		OnTxStart: t.CaptureTxStart,
		OnTxEnd:   t.CaptureTxEnd,
		OnEnter:   t.CaptureEnter,
		OnExit:    t.CaptureExit,
		OnFault:   t.CaptureFault,
	}
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
	statedbInMemory := state.New(&MockStateReader{})

	blockCtx := evmtypes.BlockContext{
		CanTransfer: func(db evmtypes.IntraBlockState, addr libcommon.Address, amount *uint256.Int) (bool, error) {
			balance, _ := db.GetBalance(addr)
			return balance.Cmp(amount) >= 0, nil
		},
		Transfer: func(db evmtypes.IntraBlockState, sender, recipient libcommon.Address, amount *uint256.Int, bailout bool) error {
			return nil
		},
		GetHash:     func(uint64) (libcommon.Hash, error) { return libcommon.Hash{}, nil },
		Coinbase:    libcommon.Address{},
		GasLimit:    1000000,
		BlockNumber: 23041867,
		Time:        1,
		Difficulty:  big.NewInt(1),
	}

	chainConfig := chain.AllProtocolChanges
	chainConfig.ChainID = big.NewInt(1337)

	txCtx := evmtypes.TxContext{
		Origin:   libcommon.HexToAddress("0xabcd"),
		GasPrice: uint256.NewInt(1),
	}

	tracer := NewStateTracer()

	// TODO: for some reason capture tx start does not trigger.
	tracer.blockTime = blockCtx.Time
	tracer.chainId = new(uint256.Int)
	tracer.chainId.SetFromBig(chainConfig.ChainID)

	hooks := tracer.Hooks()
	vmConfig := vm.Config{
		Tracer: hooks,
	}

	evm := vm.NewEVM(blockCtx, txCtx, statedbInMemory, chainConfig, vmConfig)
	in := vm.NewEVMInterpreter(evm, vmConfig)
	tracer.setJumpTable(in.JT())

	return &SimpleTracer{
		state:  NewMockState(),
		tracer: tracer,
		evm:    evm,
	}
}

func (tr *SimpleTracer) DeployContract(addr libcommon.Address, bytecode []byte, balance *uint256.Int) error {
	tr.evm.IntraBlockState().CreateAccount(addr, true)
	tr.evm.IntraBlockState().SetCode(addr, bytecode)
	if balance != nil && balance.Sign() > 0 {
		tr.evm.IntraBlockState().SetBalance(addr, *balance, tracing.BalanceChangeUnspecified)
	}
	return nil
}
func (tr *SimpleTracer) ExecuteContract(contractAddr libcommon.Address, input []byte, gasLimit uint64, callValue *uint256.Int) ([]*EvmInstructionMetadata, *EvmExecutionState, uint64, error) {
	if callValue == nil {
		callValue = uint256.NewInt(0)
	}
	callerAddr := libcommon.HexToAddress("0xabcd")
	caller := vm.AccountRef(callerAddr)

	neededBalance := new(uint256.Int).Add(callValue, uint256.NewInt(1000000))
	tr.evm.IntraBlockState().SetBalance(callerAddr, *neededBalance, tracing.BalanceChangeUnspecified)

	tr.evm.IntraBlockState().SetHooks(tr.evm.Config().Tracer)
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

// Signature to match RegisterLookup
func NewTracerHooks(createResults func(newTracer *StateTracer) (*prover.ResultsFile, error)) func(code string, ctx *tracers.Context, cfg json.RawMessage) (*tracers.Tracer, error) {
	return func(code string, ctx *tracers.Context, cfg json.RawMessage) (*tracers.Tracer, error) {
		newTracer := NewStateTracer()
		return &tracers.Tracer{
			Hooks: newTracer.Hooks(),
			Stop: func(err error) {
				fmt.Println(err.Error())
			},
			GetResult: func() (json.RawMessage, error) {
				results, err := createResults(newTracer)
				if err != nil {
					return nil, err
				}
				data, err := json.Marshal(results)
				if err != nil {
					return nil, err
				}
				return data, nil
			},
		}, nil
	}
}
