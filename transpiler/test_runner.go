package main

import (
	"erigon-transpiler-risc-v/prover"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/holiman/uint256"
)

type TestRunner struct {
	program []byte
}

func NewTestRunner(program []byte) *TestRunner {
	return &TestRunner{
		program: program,
	}
}

func (t *TestRunner) Execute() (*prover.AssemblyFile, error) {
	contractAddr := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")

	runner := NewSimpleTracer()
	err := runner.DeployContract(contractAddr, t.program, uint256.NewInt(1000))
	if err != nil {
		return nil, err
	}

	transpiler, _, err := runner.ExecuteContract(contractAddr, nil, 100000)
	if err != nil {
		return nil, err
	}

	assembly := transpiler.toAssembly()
	return assembly, nil
}
