package main

import (
	"erigon-transpiler-risc-v/prover"
	"erigon-transpiler-risc-v/tracer"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/holiman/uint256"
)

const CONTRACT_ADDRESS = "0x1234567890123456789012345678901234567890"

type TestConfig struct {
	CallValue *uint256.Int
	CallData  []byte
}

type testRunner struct {
	program []byte
	config  *TestConfig
}

func NewTestRunner(program []byte) *testRunner {
	return &testRunner{
		program: program,
		config: &TestConfig{
			CallValue: uint256.NewInt(0),
		},
	}
}

func NewTestRunnerWithConfig(program []byte, config TestConfig) *testRunner {
	if config.CallValue == nil {
		config.CallValue = uint256.NewInt(0)
	}

	return &testRunner{
		program: program,
		config:  &config,
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

	callData := t.config.CallData
	if callData == nil {
		callData = []byte{}
	}
	instructions, executionState, _, err := runner.ExecuteContract(contractAddr, callData, 100000, t.config.CallValue)
	if err != nil {
		return nil, nil, err
	}
	transpiler := NewTranspiler()
	snapshot := EvmStackSnapshot{
		Snapshots: make([][]uint256.Int, 0),
	}

	for i := range instructions {
		transpiler.AddInstruction(instructions[i], executionState)
		if i > 0 {
			snapshot.Snapshots = append(snapshot.Snapshots, instructions[i].StackSnapshot)
		}
	}

	assembly := transpiler.toAssembly()
	return assembly, &snapshot, nil
}
