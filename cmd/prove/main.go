package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"erigon-transpiler-risc-v/prover"
	"erigon-transpiler-risc-v/transpiler"

	"github.com/alexflint/go-arg"
	"github.com/holiman/uint256"
)

var args struct {
	Bytecode   string `arg:"-b,--bytecode,required" help:"Contract bytecode (hex string, with or without 0x prefix)"`
	Calldata   string `arg:"-c,--calldata" help:"Call data (hex string, with or without 0x prefix)"`
	OutputFile string `arg:"-o,--output" default:"test.proof" help:"Output file path"`
}

func main() {
	arg.MustParse(&args)

	bytecodeHex := strings.TrimPrefix(args.Bytecode, "0x")
	calldataHex := strings.TrimPrefix(args.Calldata, "0x")

	bytecode, err := hex.DecodeString(bytecodeHex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding bytecode: %v\n", err)
		os.Exit(1)
	}

	var calldata []byte
	if calldataHex != "" {
		calldata, err = hex.DecodeString(calldataHex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding calldata: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Fprintf(os.Stderr, "Running transpilation...\n")
	assembly, _, err := transpiler.NewTestRunnerWithConfig(bytecode, transpiler.TestConfig{
		CallValue: uint256.NewInt(0),
		CallData:  calldata,
	}).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during transpilation: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Converting to toolchain-compatible assembly...\n")
	content, err := assembly.ToToolChainCompatibleAssembly()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting assembly: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Generating proof...\n")
	zkVm := prover.NewZkProver(content)

	output, err := zkVm.Prove()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating proof: %v\n", err)
		os.Exit(1)
	}

	//	encoded := hex.EncodeToString(output.Proof)
	proofFile := args.OutputFile + ".proof"
	appVkFile := args.OutputFile + ".vk"
	os.WriteFile(proofFile, []byte(output.Proof), 0644)
	os.WriteFile(appVkFile, []byte(output.AppVK), 0644)
	fmt.Println("You can verify it by doing the following:")

	verifyCmd := fmt.Sprintf("cargo openvm verify app --app-vk %s --proof %s", appVkFile, proofFile)
	fmt.Println(verifyCmd)
}
