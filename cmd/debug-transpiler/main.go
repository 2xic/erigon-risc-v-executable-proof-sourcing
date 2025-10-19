package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"erigon-transpiler-risc-v/prover"
	"erigon-transpiler-risc-v/transpiler"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <debug_mappings.json>\n", os.Args[0])
		fmt.Printf("This tool will binary search to find the problematic EVM opcode using existing debug mappings\n")
		os.Exit(1)
	}
	filename := os.Args[1]

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	var mappings []transpiler.EvmToRiscVMapping
	err = json.Unmarshal(data, &mappings)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Total EVM opcodes: %d\n", len(mappings))
	fmt.Printf("Starting binary search to find problematic assembly...\n\n")

	left := 0
	right := len(mappings) - 1
	lastWorkingIndex := -1

	content := ""

	for left <= right {
		mid := (left + right) / 2
		fmt.Printf("Testing range 0-%d (%d EVM opcodes)...", mid, mid+1)

		content, err = buildAssemblyUpTo(mappings, mid)
		if err != nil {
			fmt.Printf(" FAILED at assembly generation: %v\n", err)
			right = mid - 1
		} else {
			start := time.Now()
			zkVm := prover.NewZkProver(content)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			_, err := zkVm.Prove(ctx)

			end := time.Now()
			duration := end.Sub(start)
			fmt.Printf(" Proving took %s...", duration.String())
			if err != nil {
				fmt.Printf(" FAILED at proving: %v\n", err)
				right = mid - 1
			} else {
				fmt.Printf(" SUCCESS\n")
				lastWorkingIndex = mid
				left = mid + 1
			}
		}
	}

	assemblyFile := "debug_transpiler_assembly.s"
	err = os.WriteFile(assemblyFile, []byte(content), 0644)
	if err != nil {
		fmt.Printf("Warning: Failed to write assembly to %s: %v\n", assemblyFile, err)
	} else {
		fmt.Printf("Transpiled assembly written to: %s\n", assemblyFile)
	}

	if lastWorkingIndex == -1 {
		fmt.Printf("\n❌ First EVM opcode fails!\n")
		fmt.Printf("Problematic EVM opcode: %s\n", mappings[0].EvmOpcode)
	} else if lastWorkingIndex == len(mappings)-1 {
		fmt.Printf("\n✅ All EVM opcodes compile successfully!\n")
	} else {
		problemIndex := lastWorkingIndex + 1
		fmt.Printf("\n🎯 Found problematic EVM opcode!\n")
		// fmt.Printf("Last working index: %d (%s) [depth: %d]\n", lastWorkingIndex, mappings[lastWorkingIndex].EvmOpcode, mappings[lastWorkingIndex].CallDepth)
		fmt.Printf("Problematic EVM opcode at index %d: %s [depth: %d]\n", problemIndex, mappings[problemIndex].EvmOpcode, mappings[problemIndex].CallDepth)
	}
}

func buildAssemblyUpTo(mappings []transpiler.EvmToRiscVMapping, endIndex int) (string, error) {
	var allInstructions []prover.Instruction
	dataVarMap := make(map[string]prover.DataVariable)

	for i := 0; i <= endIndex && i < len(mappings); i++ {
		allInstructions = append(allInstructions, mappings[i].RiscVInstructions...)

		for _, dataVar := range mappings[i].DataVariables {
			dataVarMap[dataVar.Name] = dataVar
		}
	}

	var allDataVars []prover.DataVariable
	for _, dataVar := range dataVarMap {
		allDataVars = append(allDataVars, dataVar)
	}

	assembly := &prover.AssemblyFile{
		Instructions: allInstructions,
		DataSection:  allDataVars,
	}

	return assembly.ToToolChainCompatibleAssembly()
}
