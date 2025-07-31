package main

import (
	"erigon-transpiler-risc-v/prover"
	"erigon-transpiler-risc-v/tracer"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/holiman/uint256"
)

const CONTRACT_ADDRESS = "0x1234567890123456789012345678901234567890"

type testRunner struct {
	program []byte
}

func NewTestRunner(program []byte) *testRunner {
	return &testRunner{
		program: program,
	}
}

type EvmStackSnapshot struct {
	Snapshots [][]uint256.Int
}

func (t *testRunner) Execute() (*prover.AssemblyFile, *EvmStackSnapshot, error) {
	contractAddr := libcommon.HexToAddress(CONTRACT_ADDRESS)

	runner := tracer.NewSimpleTracer()
	err := runner.DeployContract(contractAddr, t.program, uint256.NewInt(1000))
	if err != nil {
		return nil, nil, err
	}

	instructions, _, err := runner.ExecuteContract(contractAddr, nil, 100000)
	if err != nil {
		return nil, nil, err
	}
	transpiler := NewTranspiler()
	snapshot := EvmStackSnapshot{
		Snapshots: make([][]uint256.Int, 0),
	}
	for i := range instructions {
		transpiler.AddInstruction(instructions[i])
		if i > 0 {
			snapshot.Snapshots = append(snapshot.Snapshots, instructions[i].StackSnapshot)
		}
	}

	assembly := transpiler.toAssembly()
	return assembly, &snapshot, nil
}
