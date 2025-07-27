package main

import (
	"testing"

	"github.com/erigontech/erigon/core/vm"
	"github.com/stretchr/testify/assert"
)

func TestAddOpcode(t *testing.T) {
	bytecode := []byte{
		byte(vm.PUSH1), 0x42,
		byte(vm.PUSH1), 0x01,
		byte(vm.ADD),
	}
	assembly, err := NewTestRunner(bytecode).Execute()
	assert.NoError(t, err)
	bytecode, err = assembly.toBytecode()
	assert.NoError(t, err)

	execution, err := NewVmRunner()
	assert.NoError(t, err)
	snapshot, err := execution.Execute(bytecode)
	assert.NoError(t, err)

	// Verify that the stack is as expected at each step of the execution
	snapShot := *snapshot.stackSnapshots
	assert.Len(t, snapShot, 3)
	assert.Equal(t, []uint64{0x42}, snapShot[0])
	assert.Equal(t, []uint64{0x01, 0x42}, snapShot[1])
	assert.Equal(t, []uint64{0x43}, snapShot[2])

	// Verify that we can run the Zk prover on the assembly
	content, err := assembly.toToolChainCompatibleAssembly()
	assert.NoError(t, err)
	zkVm := NewZkProver(content)
	output, err := zkVm.TestRun()
	assert.NoError(t, err)
	// All zero as we don't write any of the output.
	assert.Equal(t, "Execution output: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]\n", output)

	// output, err = zkVm.Prove()
	// assert.NoError(t, err)
	//	assert.Contains(t, "app_pk commit: 0x0094295cb5d90deb2b28cab4d658dab0fdc2922c4e9c10305bbf277c8d29d881\n", output)
	//	assert.Contains(t, "exe commit: 0x0086d334e8f5715dd186700497c4b3d3c667cd812fda3135c6414c66eb0fc0e3\n", output)
}
