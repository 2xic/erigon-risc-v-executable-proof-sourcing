package main

import (
	"encoding/json"
	"fmt"
	"os"

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

	// Read debug mappings
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

	// Binary search to find the problematic instruction
	left := 0
	right := len(mappings) - 1
	lastWorkingIndex := -1

	for left <= right {
		mid := (left + right) / 2
		fmt.Printf("Testing range 0-%d (%d EVM opcodes)...", mid, mid+1)

		// Build assembly up to mid index
		content, err := buildAssemblyUpTo(mappings, mid)
		if err != nil {
			fmt.Printf(" FAILED at assembly generation: %v\n", err)
			right = mid - 1
		} else {
			// Try to compile with prover
			zkVm := prover.NewZkProver(content)
			_, err := zkVm.Prove()
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

	if lastWorkingIndex == -1 {
		fmt.Printf("\n❌ Even the first EVM opcode fails!\n")
		fmt.Printf("Problematic EVM opcode: %s\n", mappings[0].EvmOpcode)
	} else if lastWorkingIndex == len(mappings)-1 {
		fmt.Printf("\n✅ All EVM opcodes compile successfully!\n")
	} else {
		problemIndex := lastWorkingIndex + 1
		fmt.Printf("\n🎯 Found problematic EVM opcode!\n")
		fmt.Printf("Last working index: %d (%s) [depth: %d]\n", lastWorkingIndex, mappings[lastWorkingIndex].EvmOpcode, mappings[lastWorkingIndex].CallDepth)
		fmt.Printf("Problematic EVM opcode at index %d: %s [depth: %d]\n", problemIndex, mappings[problemIndex].EvmOpcode, mappings[problemIndex].CallDepth)
		
		// Save working assembly for inspection
		workingContent, _ := buildAssemblyUpTo(mappings, lastWorkingIndex)
		os.WriteFile("debug_working.s", []byte(workingContent), 0644)
		fmt.Printf("✓ Working assembly saved to: debug_working.s\n")
		
		// Save problematic assembly for inspection  
		problemContent, _ := buildAssemblyUpTo(mappings, problemIndex)
		os.WriteFile("debug_problem.s", []byte(problemContent), 0644)
		fmt.Printf("✓ Problematic assembly saved to: debug_problem.s\n")
	}
}

func buildAssemblyUpTo(mappings []transpiler.EvmToRiscVMapping, endIndex int) (string, error) {
	// Create assembly file with instructions and data up to endIndex
	var allInstructions []prover.Instruction
	dataVarMap := make(map[string]prover.DataVariable) // Deduplicate by name
	
	for i := 0; i <= endIndex && i < len(mappings); i++ {
		allInstructions = append(allInstructions, mappings[i].RiscVInstructions...)
		
		// Add data variables, avoiding duplicates
		for _, dataVar := range mappings[i].DataVariables {
			dataVarMap[dataVar.Name] = dataVar
		}
	}
	
	// Convert map to slice
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