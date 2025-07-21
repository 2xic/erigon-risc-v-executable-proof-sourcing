package main

import (
	chain2 "github.com/erigontech/erigon-lib/chain"
	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon/core/tracing"
	"github.com/erigontech/erigon/core/types"
	"github.com/holiman/uint256"
)

// =============================================================================
// MOCK STATE DB
// =============================================================================

type MockState struct {
	balances map[libcommon.Address]*uint256.Int
	storage  map[libcommon.Address]map[libcommon.Hash]*uint256.Int
	code     map[libcommon.Address][]byte
	nonces   map[libcommon.Address]uint64
}

func NewMockState() *MockState {
	return &MockState{
		balances: make(map[libcommon.Address]*uint256.Int),
		storage:  make(map[libcommon.Address]map[libcommon.Hash]*uint256.Int),
		code:     make(map[libcommon.Address][]byte),
		nonces:   make(map[libcommon.Address]uint64),
	}
}

func (s *MockState) GetBalance(addr libcommon.Address) (*uint256.Int, error) {
	if balance, ok := s.balances[addr]; ok {
		return balance, nil
	}
	return uint256.NewInt(0), nil
}

func (s *MockState) AddBalance(addr libcommon.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) error {
	current, _ := s.GetBalance(addr)
	s.balances[addr] = new(uint256.Int).Add(current, amount)
	return nil
}

func (s *MockState) SubBalance(addr libcommon.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) error {
	current, _ := s.GetBalance(addr)
	s.balances[addr] = new(uint256.Int).Sub(current, amount)
	return nil
}

func (s *MockState) GetState(addr libcommon.Address, key *libcommon.Hash, value *uint256.Int) error {
	if storage, ok := s.storage[addr]; ok {
		if val, exists := storage[*key]; exists {
			value.Set(val)
			return nil
		}
	}
	value.Clear()
	return nil
}

func (s *MockState) SetState(addr libcommon.Address, key *libcommon.Hash, value uint256.Int) error {
	if _, ok := s.storage[addr]; !ok {
		s.storage[addr] = make(map[libcommon.Hash]*uint256.Int)
	}
	s.storage[addr][*key] = &value
	return nil
}

func (s *MockState) GetCode(addr libcommon.Address) ([]byte, error) {
	if code, ok := s.code[addr]; ok {
		return code, nil
	}
	return nil, nil
}

func (s *MockState) SetCode(addr libcommon.Address, code []byte) error {
	s.code[addr] = code
	return nil
}

func (s *MockState) GetNonce(addr libcommon.Address) (uint64, error) {
	return s.nonces[addr], nil
}

func (s *MockState) SetNonce(addr libcommon.Address, nonce uint64) error {
	s.nonces[addr] = nonce
	return nil
}

func (s *MockState) SetupContract(addr libcommon.Address, code []byte, balance *uint256.Int) error {
	s.CreateAccount(addr, true)
	s.SetCode(addr, code)
	if balance != nil {
		s.AddBalance(addr, balance, tracing.BalanceChangeUnspecified)
	}
	return nil
}

// Required interface methods (minimal implementations)
func (s *MockState) CreateAccount(addr libcommon.Address, contractCreation bool) error {
	s.balances[addr] = uint256.NewInt(0)
	s.nonces[addr] = 0
	return nil
}
func (s *MockState) GetCodeHash(addr libcommon.Address) (libcommon.Hash, error) {
	return libcommon.Hash{}, nil
}
func (s *MockState) GetCodeSize(addr libcommon.Address) (int, error) {
	code, _ := s.GetCode(addr)
	return len(code), nil
}
func (s *MockState) Exist(addr libcommon.Address) (bool, error) { return true, nil }
func (s *MockState) Empty(addr libcommon.Address) (bool, error) { return false, nil }
func (s *MockState) AddRefund(uint64)                           {}
func (s *MockState) SubRefund(uint64)                           {}
func (s *MockState) GetRefund() uint64                          { return 0 }
func (s *MockState) GetCommittedState(addr libcommon.Address, key *libcommon.Hash, value *uint256.Int) error {
	return s.GetState(addr, key, value)
}
func (s *MockState) GetTransientState(addr libcommon.Address, key libcommon.Hash) uint256.Int {
	return *uint256.NewInt(0)
}
func (s *MockState) SetTransientState(addr libcommon.Address, key libcommon.Hash, value uint256.Int) {
}
func (s *MockState) Selfdestruct(addr libcommon.Address) (bool, error)      { return false, nil }
func (s *MockState) HasSelfdestructed(addr libcommon.Address) (bool, error) { return false, nil }
func (s *MockState) Selfdestruct6780(addr libcommon.Address) error          { return nil }
func (s *MockState) Snapshot() int                                          { return 0 }
func (s *MockState) RevertToSnapshot(int)                                   {}
func (s *MockState) AddAddressToAccessList(addr libcommon.Address) bool     { return false }
func (s *MockState) AddSlotToAccessList(addr libcommon.Address, slot libcommon.Hash) (bool, bool) {
	return false, false
}
func (s *MockState) AddressInAccessList(addr libcommon.Address) bool { return true }
func (s *MockState) ResolveCodeHash(addr libcommon.Address) (libcommon.Hash, error) {
	return s.GetCodeHash(addr)
}
func (s *MockState) ResolveCode(addr libcommon.Address) ([]byte, error) { return s.GetCode(addr) }
func (s *MockState) GetDelegatedDesignation(addr libcommon.Address) (libcommon.Address, bool, error) {
	return libcommon.Address{}, false, nil
}
func (s *MockState) AddLog(*types.Log)       {}
func (s *MockState) SetHooks(*tracing.Hooks) {}
func (s *MockState) Prepare(rules *chain2.Rules, sender, coinbase libcommon.Address, dest *libcommon.Address, precompiles []libcommon.Address, txAccesses types.AccessList, authorities []libcommon.Address) error {
	return nil
}
