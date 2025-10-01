package transpiler

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

type TestRunner struct {
	program []byte
	config  *TestConfig
	runner  *tracer.SimpleTracer
}

func NewTestRunner(program []byte) *TestRunner {
	runner := tracer.NewSimpleTracer()

	return &TestRunner{
		program: program,
		config: &TestConfig{
			CallValue: uint256.NewInt(0),
		},
		runner: runner,
	}
}

func NewTestRunnerWithConfig(program []byte, config TestConfig) *TestRunner {
	if config.CallValue == nil {
		config.CallValue = uint256.NewInt(0)
	}
	runner := tracer.NewSimpleTracer()

	return &TestRunner{
		program: program,
		config:  &config,
		runner:  runner,
	}
}

type EvmStackSnapshot struct {
	Snapshots [][]uint256.Int
}

func (t *TestRunner) Execute() (*prover.AssemblyFile, *EvmStackSnapshot, error) {
	contractAddr := libcommon.HexToAddress(CONTRACT_ADDRESS)

	err := t.runner.DeployContract(contractAddr, t.program, uint256.NewInt(1000))
	if err != nil {
		return nil, nil, err
	}

	callData := t.config.CallData
	if callData == nil {
		callData = []byte{}
	}
	instructions, executionState, _, err := t.runner.ExecuteContract(contractAddr, callData, 100000, t.config.CallValue)
	if err != nil {
		return nil, nil, err
	}
	transpiler := NewTranspiler()
	snapshot, err := transpiler.ProcessExecution(instructions, executionState)
	if err != nil {
		return nil, nil, err
	}

	assembly := transpiler.ToAssembly()
	return assembly, &snapshot, nil
}

func (t *TestRunner) DeployContract(contractAddr libcommon.Address, bytecode []byte) error {
	err := t.runner.DeployContract(contractAddr, bytecode, uint256.NewInt(1000))
	return err
}
