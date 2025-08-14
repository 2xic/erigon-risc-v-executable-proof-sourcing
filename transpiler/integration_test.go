package main

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

	execution, err := prover.NewVmRunner()
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
	}

	for _, tc := range tests {
		bytecode := tc.bytecode
		assembly, evmSnapshot, err := NewTestRunner(bytecode).Execute()
		assert.NoError(t, err)
		bytecode, err = assembly.ToBytecode()
		assert.NoError(t, err)

		execution, err := prover.NewVmRunner()
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
