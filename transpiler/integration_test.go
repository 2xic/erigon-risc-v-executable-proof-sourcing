package transpiler

import (
	"fmt"
	"testing"

	"erigon-transpiler-risc-v/prover"

	"github.com/erigontech/erigon/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func assertStackEqual(t *testing.T, expected, actual []uint256.Int, message string) {
	assert.Equal(t, len(expected), len(actual), message)

	for i := range expected {
		assert.True(t, expected[i].Eq(&actual[i]), fmt.Sprintf("%s: expected %s but got %s at index %d", message, expected[i].String(), actual[i].String(), i))
	}

}

func TestAddOpcode(t *testing.T) {
	bytecode := []byte{
		byte(vm.PUSH1), 0x42,
		byte(vm.PUSH1), 0x01,
		byte(vm.ADD),
	}
	assembly, _, err := NewTestRunner(bytecode).Execute()
	assert.NoError(t, err)

	bytecode, err = assembly.ToBytecode()
	assert.NoError(t, err)

	execution, err := prover.NewUnicornRunner()
	assert.NoError(t, err)
	snapshot, err := execution.Execute(bytecode)
	assert.NoError(t, err)

	// Verify that the stack is as expected at each step of the execution
	snapShot := *snapshot.StackSnapshots
	assert.Len(t, snapShot, 3)
	assert.Equal(t, []uint256.Int{*uint256.NewInt(0x42)}, snapShot[0])
	assert.Equal(t, []uint256.Int{*uint256.NewInt(0x42), *uint256.NewInt(0x1)}, snapShot[1])
	assert.Equal(t, []uint256.Int{*uint256.NewInt(0x43)}, snapShot[2])

	// Verify that we can run the Zk prover on the assembly
	content, err := assembly.ToToolChainCompatibleAssembly()
	assert.NoError(t, err)
	zkVm := prover.NewZkProver(content)
	output, err := zkVm.TestRun()
	assert.NoError(t, err)
	// All zero as we don't write any of the output.
	assert.Equal(t, "Execution output: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]", output)

	// output, err = zkVm.Prove()
	// assert.NoError(t, err)
	//	assert.Contains(t, "app_pk commit: 0x0094295cb5d90deb2b28cab4d658dab0fdc2922c4e9c10305bbf277c8d29d881\n", output)
	//	assert.Contains(t, "exe commit: 0x0086d334e8f5715dd186700497c4b3d3c667cd812fda3135c6414c66eb0fc0e3\n", output)
}

func TestSimpleOpcodes(t *testing.T) {
	tests := []struct {
		name     string
		bytecode []byte
	}{
		{
			name:     "PUSH0",
			bytecode: []byte{byte(vm.PUSH0), byte(vm.PUSH0), byte(vm.ADD)},
		},
		{
			name: "ADD_256bit",
			bytecode: []byte{
				byte(vm.PUSH4), 0xFF, 0xFF, 0xFF, 0xFF,
				byte(vm.PUSH1), 0x1,
				byte(vm.ADD),
				byte(vm.PUSH4), 0xFF, 0xFF, 0xFF, 0xFF,
				byte(vm.ADD),
			},
		},
		{
			name:     "PUSH4",
			bytecode: []byte{byte(vm.PUSH4), 0x42, 0x43, 0x42, 0x41},
		},
		{
			name:     "ADD",
			bytecode: []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x2, byte(vm.ADD)},
		},
		{
			name:     "SLT",
			bytecode: []byte{byte(vm.PUSH0), byte(vm.PUSH1), 0x1, byte(vm.SLT)},
		},
		{
			name:     "SHR",
			bytecode: []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH0), byte(vm.SHR)},
		},
		{
			name:     "SHL_simple",
			bytecode: []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x2, byte(vm.SHL)},
		},
		{
			name:     "SHL_zero_shift",
			bytecode: []byte{byte(vm.PUSH1), 0x0, byte(vm.PUSH1), 0x42, byte(vm.SHL)},
		},
		{
			name:     "SHL_zero_value",
			bytecode: []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH0), byte(vm.SHL)},
		},
		{
			name:     "EQ",
			bytecode: []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH0), byte(vm.EQ)},
		},
		{
			name:     "EQ",
			bytecode: []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x1, byte(vm.EQ)},
		},
		{
			name:     "LT",
			bytecode: []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x1, byte(vm.LT)},
		},
		{
			name:     "GT",
			bytecode: []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x1, byte(vm.GT)},
		},
		{
			name:     "JUMPDEST",
			bytecode: []byte{byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x1, byte(vm.JUMPDEST)},
		},
		{
			name: "JUMPI",
			bytecode: []byte{
				byte(vm.PUSH1), 0,
				byte(vm.PUSH1), 10,
				byte(vm.JUMPI),
				byte(vm.PUSH1), 1,
				byte(vm.PUSH1), 12,
				byte(vm.JUMPI),
				byte(vm.JUMPDEST),
				byte(vm.INVALID),
				byte(vm.JUMPDEST),
				byte(vm.PUSH1), 1,
			},
		},
		{
			name:     "DUP1",
			bytecode: []byte{byte(vm.PUSH0), byte(vm.DUP1), byte(vm.ADD)},
		},
		{
			name:     "DUP2",
			bytecode: []byte{byte(vm.PUSH1), 0x2, byte(vm.PUSH1), 0x1, byte(vm.DUP2), byte(vm.ADD)},
		},
		{
			name:     "DUP3",
			bytecode: []byte{byte(vm.PUSH1), 0x2, byte(vm.PUSH1), 0x1, byte(vm.PUSH1), 0x0, byte(vm.DUP3), byte(vm.ADD)},
		},

		{
			name:     "SWAP1",
			bytecode: []byte{byte(vm.PUSH1), 0x2, byte(vm.PUSH1), 0x1, byte(vm.DUP2), byte(vm.SWAP1), byte(vm.ADD)},
		},
		{
			name:     "SWAP2",
			bytecode: []byte{byte(vm.PUSH1), 0x2, byte(vm.PUSH1), 0x1, byte(vm.DUP2), byte(vm.SWAP2), byte(vm.ADD)},
		},
		{
			name:     "POP",
			bytecode: []byte{byte(vm.PUSH1), 0x2, byte(vm.PUSH1), 0x1, byte(vm.DUP2), byte(vm.POP), byte(vm.ADD)},
		},
		{
			name:     "MSTORE",
			bytecode: []byte{byte(vm.PUSH0), byte(vm.PUSH1), 0x42, byte(vm.MSTORE)},
		},
		{
			name:     "MLOAD",
			bytecode: []byte{byte(vm.PUSH0), byte(vm.PUSH1), 0x42, byte(vm.MSTORE), byte(vm.PUSH0), byte(vm.MLOAD)},
		},
		{
			name:     "ISZERO",
			bytecode: []byte{byte(vm.PUSH0), byte(vm.PUSH1), 0x42, byte(vm.MSTORE), byte(vm.PUSH0), byte(vm.MLOAD), byte(vm.ISZERO)},
		},
		{
			name:     "NOT_simple",
			bytecode: []byte{byte(vm.PUSH1), 0x1, byte(vm.NOT)},
		},
		{
			name:     "NOT_zero",
			bytecode: []byte{byte(vm.PUSH0), byte(vm.NOT)},
		},
		{
			name:     "NOT_max_32bit",
			bytecode: []byte{byte(vm.PUSH4), 0xFF, 0xFF, 0xFF, 0xFF, byte(vm.NOT)},
		},
		{
			name: "NOT_double",
			bytecode: []byte{
				byte(vm.PUSH1), 0x42,
				byte(vm.NOT),
				byte(vm.NOT),
			},
		},
		{
			name: "NOT_with_add",
			bytecode: []byte{
				byte(vm.PUSH1), 0x1,
				byte(vm.NOT),
				byte(vm.PUSH1), 0x1,
				byte(vm.ADD),
			},
		},
		{
			name:     "CALLVALUE",
			bytecode: []byte{byte(vm.CALLVALUE)},
		},
		{
			name:     "CALLDATASIZE",
			bytecode: []byte{byte(vm.CALLDATASIZE)},
		},
		{
			name: "CALLVALUE_with_ADD",
			bytecode: []byte{
				byte(vm.CALLVALUE),
				byte(vm.PUSH1), 0x1,
				byte(vm.ADD),
			},
		},
		{
			name: "CALLDATASIZE_with_ADD",
			bytecode: []byte{
				byte(vm.CALLDATASIZE),
				byte(vm.PUSH1), 0x1,
				byte(vm.ADD),
			},
		},
		{
			name:     "SSTORE",
			bytecode: []byte{byte(vm.PUSH1), 0x42, byte(vm.PUSH0), byte(vm.SSTORE)},
		},
		{
			name:     "SLOAD",
			bytecode: []byte{byte(vm.PUSH1), 0x42, byte(vm.PUSH0), byte(vm.SSTORE), byte(vm.PUSH0), byte(vm.SLOAD)},
		},
	}

	for _, tc := range tests {
		bytecode := tc.bytecode
		assembly, evmSnapshot, err := NewTestRunner(bytecode).Execute()
		assert.NoError(t, err)
		bytecode, err = assembly.ToBytecode()
		assert.NoError(t, err)

		execution, err := prover.NewUnicornRunner()
		assert.NoError(t, err)
		snapshot, err := execution.Execute(bytecode)
		assert.NoError(t, err)

		// Verify that the stack is as expected at each step of the execution
		snapShot := *snapshot.StackSnapshots
		assert.Len(t, snapShot, len(evmSnapshot.Snapshots), fmt.Sprintf("Failed on %s (snapshot length)", tc.name))

		/*
			if len(evmSnapshot.Snapshots) != len(snapShot) {
				for i := range evmSnapshot.Snapshots {
					fmt.Println(evmSnapshot.Snapshots[i])
				}
				fmt.Println("=====")
				for i := range snapShot {
					fmt.Println(snapShot[i])
				}
			}
		*/

		for i := range evmSnapshot.Snapshots {
			assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Failed on %s (instructions %d)", tc.name, i))
		}
	}
}

func TestCallValue(t *testing.T) {
	testValue := uint256.NewInt(0x42)

	bytecode := []byte{byte(vm.CALLVALUE)}

	assembly, evmSnapshot, err := NewTestRunnerWithConfig(bytecode, TestConfig{
		CallValue: testValue,
	}).Execute()

	assert.NoError(t, err)
	assert.NotNil(t, assembly, "Assembly should not be nil")

	bytecodeResult, err := assembly.ToBytecode()
	assert.NoError(t, err)

	execution, err := prover.NewUnicornRunner()
	assert.NoError(t, err)
	snapshot, err := execution.Execute(bytecodeResult)
	assert.NoError(t, err)

	snapShot := *snapshot.StackSnapshots
	assert.Len(t, snapShot, len(evmSnapshot.Snapshots))

	for i := range evmSnapshot.Snapshots {
		assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Failed on CALLVALUE_max (instruction %d)", i))
	}
}

func TestCallDataSize(t *testing.T) {
	testCallData := []byte{0x01, 0x02, 0x03, 0x04}

	bytecode := []byte{byte(vm.CALLDATASIZE)}

	assembly, evmSnapshot, err := NewTestRunnerWithConfig(bytecode, TestConfig{
		CallData: testCallData,
	}).Execute()

	assert.NoError(t, err)
	assert.NotNil(t, assembly, "Assembly should not be nil")

	bytecodeResult, err := assembly.ToBytecode()
	assert.NoError(t, err)

	execution, err := prover.NewUnicornRunner()
	assert.NoError(t, err)
	snapshot, err := execution.Execute(bytecodeResult)
	assert.NoError(t, err)

	snapShot := *snapshot.StackSnapshots
	assert.Len(t, snapShot, len(evmSnapshot.Snapshots))

	for i := range evmSnapshot.Snapshots {
		assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Failed on CALLDATASIZE (instruction %d)", i))
	}
}

func TestCallDataLoad(t *testing.T) {
	testCallData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	tests := []struct {
		name   string
		offset byte
	}{
		{"offset_0", 0x00},
		{"offset_4", 0x04},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bytecode := []byte{
				byte(vm.PUSH1), tc.offset,
				byte(vm.CALLDATALOAD),
			}

			assembly, evmSnapshot, err := NewTestRunnerWithConfig(bytecode, TestConfig{
				CallData: testCallData,
			}).Execute()

			assert.NoError(t, err)
			assert.NotNil(t, assembly, "Assembly should not be nil")

			bytecodeResult, err := assembly.ToBytecode()
			assert.NoError(t, err)

			execution, err := prover.NewUnicornRunner()
			assert.NoError(t, err)
			snapshot, err := execution.Execute(bytecodeResult)
			assert.NoError(t, err)

			snapShot := *snapshot.StackSnapshots
			assert.Len(t, snapShot, len(evmSnapshot.Snapshots))

			for i := range evmSnapshot.Snapshots {
				assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Failed on CALLDATALOAD %s (instruction %d)", tc.name, i))
			}
		})
	}
}

func TestCodeCopy(t *testing.T) {
	// Create bytecode that copies itself to memory, then loads it back
	bytecode := []byte{
		byte(vm.PUSH1), 0x08,
		byte(vm.PUSH1), 0x00,
		byte(vm.PUSH1), 0x00,
		byte(vm.CODECOPY),
		byte(vm.PUSH1), 0x00,
		byte(vm.MLOAD),
	}

	assembly, evmSnapshot, err := NewTestRunner(bytecode).Execute()

	assert.NoError(t, err)
	assert.NotNil(t, assembly, "Assembly should not be nil")

	bytecodeResult, err := assembly.ToBytecode()
	assert.NoError(t, err)

	execution, err := prover.NewUnicornRunner()
	assert.NoError(t, err)
	snapshot, err := execution.Execute(bytecodeResult)
	assert.NoError(t, err)

	snapShot := *snapshot.StackSnapshots
	assert.Len(t, snapShot, len(evmSnapshot.Snapshots))

	for i := range evmSnapshot.Snapshots {
		assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Failed on CODECOPY+MLOAD (instruction %d)", i))
	}
}

func TestHaltingOpcodes(t *testing.T) {
	tests := []struct {
		name           string
		bytecode       []byte
		expectedStatus int
	}{
		{
			name:     "RETURN",
			bytecode: []byte{byte(vm.PUSH0), byte(vm.PUSH0), byte(vm.RETURN)},
		},
		{
			name:     "REVERT",
			bytecode: []byte{byte(vm.PUSH0), byte(vm.PUSH0), byte(vm.REVERT)},
		},
		{
			name:     "INVALID",
			bytecode: []byte{byte(vm.INVALID)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assembly, evmSnapshot, err := NewTestRunner(tc.bytecode).Execute()
			assert.NoError(t, err)

			bytecode, err := assembly.ToBytecode()
			assert.NoError(t, err)

			execution, err := prover.NewUnicornRunner()
			assert.NoError(t, err)
			snapshot, err := execution.Execute(bytecode)
			assert.NoError(t, err)

			snapShot := *snapshot.StackSnapshots
			assert.Len(t, snapShot, len(evmSnapshot.Snapshots))

			for i := range evmSnapshot.Snapshots {
				assertStackEqual(t, evmSnapshot.Snapshots[i], snapShot[i], fmt.Sprintf("Failed on %s (instruction %d)", tc.name, i))
			}
		})
	}
}
