package main

import (
	"erigon-transpiler-risc-v/prover"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var resultsFile string

	cmd := &cobra.Command{
		Use:   "proof-verify",
		Short: "Verify OpenVM proofs from results file",
		Long:  "Verify OpenVM app proofs using cargo openvm verify app command with data from results.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := prover.VerifyFromResults(resultsFile)
			if err != nil {
				fmt.Printf("Verification failed: %v\n", err)
				fmt.Printf("Output: %s\n", result.Stdout)
				return err
			}

			if result.Valid {
				fmt.Println("✓ Verification successful!")
			} else {
				fmt.Println("✗ Verification failed!")
			}
			fmt.Printf("Output: %s\n", result.Stdout)

			return nil
		},
	}

	cmd.Flags().StringVar(&resultsFile, "results", "results.json", "Path to results file containing AppVK and Proof data")

	if err := cmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}