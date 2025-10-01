package transpiler

import (
	"fmt"
	"testing"

	"erigon-transpiler-risc-v/prover"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestNestedCallABC(t *testing.T) {

	contractA := []byte{
		byte(vm.PUSH1), 0xAA,
		byte(vm.PUSH1), 0x20,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH20), 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		byte(vm.PUSH2), 0x27, 0x10,
		byte(vm.CALL),
		byte(vm.PUSH1), 0xDD,
		byte(vm.STOP),
	}

	contractB := []byte{
		byte(vm.PUSH1), 0xBB,
		byte(vm.PUSH1), 0xBB,
		byte(vm.PUSH1), 0x20,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH20), 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33,
		0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33,
		byte(vm.PUSH2), 0x27, 0x10,
		byte(vm.CALL),
		byte(vm.PUSH1), 0xEE,
		byte(vm.PUSH0),
		byte(vm.PUSH0),
		byte(vm.RETURN),
	}

	contractC := []byte{
		byte(vm.PUSH1), 0xCC,
		byte(vm.PUSH1), 0xFF,
		byte(vm.PUSH0),
		byte(vm.PUSH0),
		byte(vm.RETURN),
	}

	testRunner := NewTestRunnerWithConfig(contractA, TestConfig{
		CallValue: uint256.NewInt(0),
		CallData:  []byte{},
	})

	addrB := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	addrC := libcommon.HexToAddress("0x3333333333333333333333333333333333333333")

	err := testRunner.DeployContract(addrB, contractB)
	assert.NoError(t, err)

	err = testRunner.DeployContract(addrC, contractC)
	assert.NoError(t, err)

	assembly, evmSnapshot, err := testRunner.Execute()
	assert.NoError(t, err)

	riscvBytecode, err := assembly.ToBytecode()
	assert.NoError(t, err)

	execution, err := prover.NewUnicornRunner()
	assert.NoError(t, err)

	snapshot, err := execution.Execute(riscvBytecode)
	assert.NoError(t, err)

	snapShot := *snapshot.StackSnapshots
	assert.Len(t, snapShot, len(evmSnapshot.Snapshots), "Snapshot length should match")

	finalStack := snapShot[len(snapShot)-1]

	assert.Len(t, finalStack, 3, "Final stack should have 3 elements")
	assert.Equal(t, uint64(0xAA), finalStack[0].Uint64(), "First element should be 0xAA")
	assert.Equal(t, uint64(1), finalStack[1].Uint64(), "Second element should be success flag (1)")
	assert.Equal(t, uint64(0xDD), finalStack[2].Uint64(), "Third element should be 0xDD")

	for i := range evmSnapshot.Snapshots {
		assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Stack mismatch at instruction %d", i))
	}

}

func TestNestedCallDepth4(t *testing.T) {

	contractA := []byte{
		byte(vm.PUSH1), 0xAA,
		byte(vm.PUSH1), 0x20, byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00,
		byte(vm.PUSH20), 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		byte(vm.PUSH2), 0x27, 0x10,
		byte(vm.CALL),
		byte(vm.STOP),
	}

	contractB := []byte{
		byte(vm.PUSH1), 0xBB,
		byte(vm.PUSH1), 0x20, byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00,
		byte(vm.PUSH20), 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33,
		byte(vm.PUSH2), 0x27, 0x10,
		byte(vm.CALL),
		byte(vm.PUSH0), byte(vm.PUSH0), byte(vm.RETURN),
	}

	contractC := []byte{
		byte(vm.PUSH1), 0xCC,
		byte(vm.PUSH1), 0x20, byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00,
		byte(vm.PUSH20), 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44,
		byte(vm.PUSH2), 0x27, 0x10,
		byte(vm.CALL),
		byte(vm.PUSH0), byte(vm.PUSH0), byte(vm.RETURN),
	}

	contractD := []byte{
		byte(vm.PUSH1), 0xDD,
		byte(vm.PUSH0), byte(vm.PUSH0), byte(vm.RETURN),
	}

	testRunner := NewTestRunnerWithConfig(contractA, TestConfig{
		CallValue: uint256.NewInt(0),
		CallData:  []byte{},
	})

	err := testRunner.DeployContract(libcommon.HexToAddress("0x2222222222222222222222222222222222222222"), contractB)
	assert.NoError(t, err)
	err = testRunner.DeployContract(libcommon.HexToAddress("0x3333333333333333333333333333333333333333"), contractC)
	assert.NoError(t, err)
	err = testRunner.DeployContract(libcommon.HexToAddress("0x4444444444444444444444444444444444444444"), contractD)
	assert.NoError(t, err)

	assembly, evmSnapshot, err := testRunner.Execute()
	assert.NoError(t, err)

	riscvBytecode, err := assembly.ToBytecode()
	assert.NoError(t, err)

	execution, err := prover.NewUnicornRunner()
	assert.NoError(t, err)

	snapshot, err := execution.Execute(riscvBytecode)
	assert.NoError(t, err)

	snapShot := *snapshot.StackSnapshots
	assert.Len(t, snapShot, len(evmSnapshot.Snapshots), "Snapshot length should match")

	for i := range evmSnapshot.Snapshots {
		assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Stack mismatch at instruction %d", i))
	}

}

func TestDelegateCall(t *testing.T) {

	contractA := []byte{
		byte(vm.PUSH1), 0xAA,
		byte(vm.PUSH1), 0x20,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH20), 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		byte(vm.PUSH2), 0x27, 0x10,
		byte(vm.DELEGATECALL),
		byte(vm.PUSH1), 0xDD,
		byte(vm.STOP),
	}

	contractB := []byte{
		byte(vm.PUSH1), 0xBB,
		byte(vm.PUSH0),
		byte(vm.PUSH0),
		byte(vm.RETURN),
	}

	testRunner := NewTestRunnerWithConfig(contractA, TestConfig{
		CallValue: uint256.NewInt(0),
		CallData:  []byte{},
	})

	addrB := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	err := testRunner.DeployContract(addrB, contractB)
	assert.NoError(t, err)

	assembly, evmSnapshot, err := testRunner.Execute()
	assert.NoError(t, err)

	riscvBytecode, err := assembly.ToBytecode()
	assert.NoError(t, err)

	execution, err := prover.NewUnicornRunner()
	assert.NoError(t, err)

	snapshot, err := execution.Execute(riscvBytecode)
	assert.NoError(t, err)

	snapShot := *snapshot.StackSnapshots
	assert.Len(t, snapShot, len(evmSnapshot.Snapshots), "Snapshot length should match")

	for i := range evmSnapshot.Snapshots {
		assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Stack mismatch at instruction %d", i))
	}

}

func TestStaticCall(t *testing.T) {

	contractA := []byte{
		byte(vm.PUSH1), 0xAA,
		byte(vm.PUSH1), 0x20,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH20), 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		byte(vm.PUSH2), 0x27, 0x10,
		byte(vm.STATICCALL),
		byte(vm.PUSH1), 0xDD,
		byte(vm.STOP),
	}

	contractB := []byte{
		byte(vm.PUSH1), 0xBB,
		byte(vm.PUSH0),
		byte(vm.PUSH0),
		byte(vm.RETURN),
	}

	testRunner := NewTestRunnerWithConfig(contractA, TestConfig{
		CallValue: uint256.NewInt(0),
		CallData:  []byte{},
	})

	addrB := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	err := testRunner.DeployContract(addrB, contractB)
	assert.NoError(t, err)

	assembly, evmSnapshot, err := testRunner.Execute()
	assert.NoError(t, err)

	riscvBytecode, err := assembly.ToBytecode()
	assert.NoError(t, err)

	execution, err := prover.NewUnicornRunner()
	assert.NoError(t, err)

	snapshot, err := execution.Execute(riscvBytecode)
	assert.NoError(t, err)

	snapShot := *snapshot.StackSnapshots
	assert.Len(t, snapShot, len(evmSnapshot.Snapshots), "Snapshot length should match")

	for i := range evmSnapshot.Snapshots {
		assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Stack mismatch at instruction %d", i))
	}

}

func TestCallCode(t *testing.T) {

	contractA := []byte{
		byte(vm.PUSH1), 0xAA,
		byte(vm.PUSH1), 0x20,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH20), 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		byte(vm.PUSH2), 0x27, 0x10,
		byte(vm.CALLCODE),
		byte(vm.PUSH1), 0xDD,
		byte(vm.STOP),
	}

	contractB := []byte{
		byte(vm.PUSH1), 0xBB,
		byte(vm.PUSH0),
		byte(vm.PUSH0),
		byte(vm.RETURN),
	}

	testRunner := NewTestRunnerWithConfig(contractA, TestConfig{
		CallValue: uint256.NewInt(0),
		CallData:  []byte{},
	})

	addrB := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	err := testRunner.DeployContract(addrB, contractB)
	assert.NoError(t, err)

	assembly, evmSnapshot, err := testRunner.Execute()
	assert.NoError(t, err)

	riscvBytecode, err := assembly.ToBytecode()
	assert.NoError(t, err)

	execution, err := prover.NewUnicornRunner()
	assert.NoError(t, err)

	snapshot, err := execution.Execute(riscvBytecode)
	assert.NoError(t, err)

	snapShot := *snapshot.StackSnapshots
	assert.Len(t, snapShot, len(evmSnapshot.Snapshots), "Snapshot length should match")

	for i := range evmSnapshot.Snapshots {
		assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Stack mismatch at instruction %d", i))
	}

}

func TestNestedCallWithRevert(t *testing.T) {

	contractA := []byte{
		byte(vm.PUSH1), 0xAA,
		byte(vm.PUSH1), 0x20,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH20), 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		byte(vm.PUSH2), 0x27, 0x10,
		byte(vm.CALL),
		byte(vm.PUSH1), 0xDD,
		byte(vm.STOP),
	}

	contractB := []byte{
		byte(vm.PUSH1), 0xBB,
		byte(vm.PUSH0),
		byte(vm.PUSH0),
		byte(vm.REVERT),
	}

	testRunner := NewTestRunnerWithConfig(contractA, TestConfig{
		CallValue: uint256.NewInt(0),
		CallData:  []byte{},
	})

	addrB := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	err := testRunner.DeployContract(addrB, contractB)
	assert.NoError(t, err)

	assembly, evmSnapshot, err := testRunner.Execute()
	assert.NoError(t, err)

	riscvBytecode, err := assembly.ToBytecode()
	assert.NoError(t, err)

	execution, err := prover.NewUnicornRunner()
	assert.NoError(t, err)

	snapshot, err := execution.Execute(riscvBytecode)
	assert.NoError(t, err)

	snapShot := *snapshot.StackSnapshots
	assert.Len(t, snapShot, len(evmSnapshot.Snapshots), "Snapshot length should match")

	finalStack := snapShot[len(snapShot)-1]

	assert.Len(t, finalStack, 3, "Final stack should have 3 elements")
	assert.Equal(t, uint64(0xAA), finalStack[0].Uint64(), "First element should be 0xAA")
	assert.Equal(t, uint64(0), finalStack[1].Uint64(), "Second element should be failure flag (0)")
	assert.Equal(t, uint64(0xDD), finalStack[2].Uint64(), "Third element should be 0xDD")

	for i := range evmSnapshot.Snapshots {
		assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Stack mismatch at instruction %d", i))
	}

}

func TestNestedCallWithInvalid(t *testing.T) {

	contractA := []byte{
		byte(vm.PUSH1), 0xAA,
		byte(vm.PUSH1), 0x20,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH20), 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		byte(vm.PUSH2), 0x27, 0x10,
		byte(vm.CALL),
		byte(vm.PUSH1), 0xDD,
		byte(vm.STOP),
	}

	contractB := []byte{
		byte(vm.PUSH1), 0xBB,
		byte(vm.INVALID),
	}

	testRunner := NewTestRunnerWithConfig(contractA, TestConfig{
		CallValue: uint256.NewInt(0),
		CallData:  []byte{},
	})

	addrB := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	err := testRunner.DeployContract(addrB, contractB)
	assert.NoError(t, err)

	assembly, evmSnapshot, err := testRunner.Execute()
	assert.NoError(t, err)

	riscvBytecode, err := assembly.ToBytecode()
	assert.NoError(t, err)

	execution, err := prover.NewUnicornRunner()
	assert.NoError(t, err)

	snapshot, err := execution.Execute(riscvBytecode)
	assert.NoError(t, err)

	snapShot := *snapshot.StackSnapshots
	assert.Len(t, snapShot, len(evmSnapshot.Snapshots), "Snapshot length should match")

	finalStack := snapShot[len(snapShot)-1]

	assert.Len(t, finalStack, 3, "Final stack should have 3 elements")
	assert.Equal(t, uint64(0xAA), finalStack[0].Uint64(), "First element should be 0xAA")
	assert.Equal(t, uint64(0), finalStack[1].Uint64(), "Second element should be failure flag (0)")
	assert.Equal(t, uint64(0xDD), finalStack[2].Uint64(), "Third element should be 0xDD")

	for i := range evmSnapshot.Snapshots {
		assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Stack mismatch at instruction %d", i))
	}

}

func TestDelegateCallWithRevert(t *testing.T) {

	contractA := []byte{
		byte(vm.PUSH1), 0xAA,
		byte(vm.PUSH1), 0x20,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH20), 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
		byte(vm.PUSH2), 0x27, 0x10,
		byte(vm.DELEGATECALL),
		byte(vm.PUSH1), 0xDD,
		byte(vm.STOP),
	}

	contractB := []byte{
		byte(vm.PUSH1), 0xBB,
		byte(vm.PUSH0),
		byte(vm.PUSH0),
		byte(vm.REVERT),
	}

	testRunner := NewTestRunnerWithConfig(contractA, TestConfig{
		CallValue: uint256.NewInt(0),
		CallData:  []byte{},
	})

	addrB := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	err := testRunner.DeployContract(addrB, contractB)
	assert.NoError(t, err)

	assembly, evmSnapshot, err := testRunner.Execute()
	assert.NoError(t, err)

	riscvBytecode, err := assembly.ToBytecode()
	assert.NoError(t, err)

	execution, err := prover.NewUnicornRunner()
	assert.NoError(t, err)

	snapshot, err := execution.Execute(riscvBytecode)
	assert.NoError(t, err)

	snapShot := *snapshot.StackSnapshots
	assert.Len(t, snapShot, len(evmSnapshot.Snapshots), "Snapshot length should match")

	for i := range evmSnapshot.Snapshots {
		assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Stack mismatch at instruction %d", i))
	}

}

func TestCallOpcodeWithStackSeparation(t *testing.T) {
	callerBytecode := []byte{
		byte(vm.PUSH1), 0xAA,

		byte(vm.PUSH1), 0x20,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,

		byte(vm.PUSH20), 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11,
		byte(vm.PUSH2), 0x27, 0x10,

		byte(vm.CALL),

		byte(vm.PUSH1), 0xBB,

		byte(vm.STOP),
	}

	calledContractBytecode := []byte{
		byte(vm.PUSH0),
		byte(vm.PUSH0),
		byte(vm.RETURN),
	}

	testRunner := NewTestRunnerWithConfig(callerBytecode, TestConfig{
		CallValue: uint256.NewInt(0),
		CallData:  []byte{},
	})

	calledAddr := libcommon.HexToAddress("0x1111111111111111111111111111111111111111")
	err := testRunner.DeployContract(calledAddr, calledContractBytecode)
	assert.NoError(t, err)

	assembly, evmSnapshot, err := testRunner.Execute()
	assert.NoError(t, err)

	riscvBytecode, err := assembly.ToBytecode()
	assert.NoError(t, err)

	execution, err := prover.NewUnicornRunner()
	assert.NoError(t, err)
	snapshot, err := execution.Execute(riscvBytecode)
	assert.NoError(t, err)

	snapShot := *snapshot.StackSnapshots
	assert.Len(t, snapShot, len(evmSnapshot.Snapshots), "Snapshot length should match")

	for i := range evmSnapshot.Snapshots {
		assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Stack mismatch at instruction %d", i))
	}
}
