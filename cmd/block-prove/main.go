package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"erigon-transpiler-risc-v/prover"
	"erigon-transpiler-risc-v/tracer"
	"erigon-transpiler-risc-v/transpiler"
	"fmt"
	"os"
	"strconv"

	"github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/jsonstream"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon/cmd/rpcdaemon/cli"
	"github.com/erigontech/erigon/eth/tracers"
	"github.com/erigontech/erigon/eth/tracers/config"
	"github.com/erigontech/erigon/rpc"
	"github.com/erigontech/erigon/rpc/ethapi"
	"github.com/erigontech/erigon/rpc/jsonrpc"
	"github.com/erigontech/erigon/turbo/debug"
	"github.com/spf13/cobra"
)

type ProofResult struct {
	TransactionHash  string `json:"transaction_hash"`
	TransactionIndex int    `json:"transaction_index"`
	InstructionCount int    `json:"instruction_count"`
	AppVK            string `json:"app_vk"`
	Proof            string `json:"proof"`
}

func main() {
	cmd, cfg := cli.RootCommand()
	rootCtx, rootCancel := common.RootContext()

	var blockNumber string
	var debugAssembly bool
	var assemblyFile string
	cmd.Flags().StringVar(&blockNumber, "block-number", "", "Block number to trace all transactions (required)")
	cmd.Flags().BoolVar(&debugAssembly, "debug-assembly", false, "Write transpiled assembly to disk for debugging")
	cmd.Flags().StringVar(&assemblyFile, "assembly-file", "transpiled_block.s", "Assembly output file path (used with --debug-assembly)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if blockNumber == "" {
			return fmt.Errorf("block-number is required")
		}

		ctx := cmd.Context()
		logger := debug.SetupCobra(cmd, "rpcdaemon")
		logger.Enabled(ctx, log.LvlCrit)
		db, backend, txPool, mining, stateCache, blockReader, engine, ff, bridgeReader, heimdallReader, err := cli.RemoteServices(ctx, cfg, logger, rootCancel)
		if err != nil {
			logger.Error("Could not connect to DB", "err", err)
			return nil
		}
		defer db.Close()
		defer engine.Close()
		if bridgeReader != nil {
			defer bridgeReader.Close()
		}
		if heimdallReader != nil {
			defer heimdallReader.Close()
		}

		apiList := jsonrpc.APIList(db, backend, txPool, mining, ff, stateCache, blockReader, cfg, engine, logger, bridgeReader, heimdallReader)
		debugAPI := findDebug(apiList)
		ethApi := findEth(apiList)

		blockNum, err := strconv.ParseUint(blockNumber, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid block number: %v", err)
		}

		blockData, err := ethApi.GetBlockByNumber(ctx, rpc.BlockNumber(blockNum), true)
		if err != nil {
			return fmt.Errorf("failed to get block: %v", err)
		}

		if blockData == nil {
			return fmt.Errorf("block not found")
		}

		txsInterface, ok := blockData["transactions"]
		if !ok {
			return fmt.Errorf("block has no transactions field")
		}

		txs, ok := txsInterface.([]interface{})
		if !ok {
			return fmt.Errorf("transactions field is not an array")
		}

		fmt.Printf("Tracing block %d with %d transactions\n", blockNum, len(txs))

		for i, txInterface := range txs {
			tx, ok := txInterface.(*ethapi.RPCTransaction)
			if !ok {
				fmt.Printf("Skipping invalid transaction %d (type: %T)\n", i+1, txInterface)
				continue
			}

			txHash := tx.Hash
			
			// Check if transaction already exists
			outputFolder := fmt.Sprintf("%d", blockNum)
			txOutputFile := fmt.Sprintf("%s/tx_%d.json", outputFolder, i+1)
			if _, err := os.Stat(txOutputFile); err == nil {
				fmt.Printf("Transaction %d/%d already exists, skipping: %s\n", i+1, len(txs), txHash.String())
				continue
			}
			
			fmt.Printf("Processing transaction %d/%d: %s\n", i+1, len(txs), txHash.String())

			var buf bytes.Buffer
			stream := jsonstream.New(&buf)

			ranTracer := false
			var txInstructions []*tracer.EvmInstructionMetadata
			var txExecutionState *tracer.EvmExecutionState

			customTracer := tracer.NewTracerHooks(
				func(newTracer *tracer.StateTracer) (*prover.ResultsFile, error) {
					ranTracer = true
					txInstructions = newTracer.GetInstructions()
					txExecutionState = newTracer.GetExecutionState()

					fmt.Printf("Transaction %d generated %d instructions\n", i+1, len(txInstructions))

					transpiler := transpiler.NewTranspiler()
					_, err := transpiler.ProcessExecution(txInstructions, txExecutionState)
					if err != nil {
						return nil, fmt.Errorf("failed to transpile transaction %s: %v", txHash.String(), err)
					}

					assembly := transpiler.ToAssembly()
					content, err := assembly.ToToolChainCompatibleAssembly()
					if err != nil {
						return nil, fmt.Errorf("failed to generate assembly for transaction %s: %v", txHash.String(), err)
					}

					debugFile := fmt.Sprintf("debug_mappings_tx_%d.json", i+1)
					err = transpiler.SaveDebugMappings(debugFile)
					if err != nil {
						fmt.Printf("Warning: Failed to write debug mappings to %s: %v\n", debugFile, err)
					} else {
						fmt.Printf("Debug mappings written to: %s\n", debugFile)
					}

					if debugAssembly {
						assemblyFile := fmt.Sprintf("transpiled_tx_%d.s", i+1)
						err := os.WriteFile(assemblyFile, []byte(content), 0644)
						if err != nil {
							fmt.Printf("Warning: Failed to write assembly to %s: %v\n", assemblyFile, err)
						} else {
							fmt.Printf("Transpiled assembly written to: %s\n", assemblyFile)
						}
					}

					zkVm := prover.NewZkProver(content)
					output, err := zkVm.Prove()
					if err != nil {
						return nil, fmt.Errorf("failed to prove transaction %s: %v", txHash.String(), err)
					}

					result := ProofResult{
						TransactionHash:  txHash.String(),
						TransactionIndex: i + 1,
						InstructionCount: len(txInstructions),
						AppVK:            hex.EncodeToString(output.AppVK),
						Proof:            hex.EncodeToString(output.Proof),
					}

					jsonData, err := json.MarshalIndent(result, "", "  ")
					if err != nil {
						return nil, fmt.Errorf("failed to marshal JSON for transaction %s: %v", txHash.String(), err)
					}

					outputFolder := fmt.Sprintf("%d", blockNum)
					err = os.MkdirAll(outputFolder, 0755)
					if err != nil {
						fmt.Printf("Error creating directory %s: %v\n", outputFolder, err)
					} else {
						txOutputFile := fmt.Sprintf("%s/tx_%d.json", outputFolder, i+1)
						err := os.WriteFile(txOutputFile, jsonData, 0644)
						if err != nil {
							fmt.Printf("Error writing to file %s: %v\n", txOutputFile, err)
						} else {
							fmt.Printf("Transaction %d results written to: %s\n", i+1, txOutputFile)
						}
					}

					return &prover.ResultsFile{}, nil
				},
			)

			tracers.RegisterLookup(true, customTracer)
			tracerName := "Mine"
			timeout := "10h"
			err = debugAPI.TraceTransaction(
				context.Background(),
				txHash,
				&config.TraceConfig{
					Tracer:  &tracerName,
					Timeout: &timeout,
				},
				stream,
			)
			if err != nil {
				fmt.Printf("Failed to trace transaction %s: %v\n", txHash.String(), err)
				continue
			}

			if !ranTracer {
				fmt.Printf("Warning: Tracer did not run for transaction %s\n", txHash.String())
			}
		}

		fmt.Printf("Completed processing %d transactions\n", len(txs))

		return nil
	}

	if err := cmd.ExecuteContext(rootCtx); err != nil {
		fmt.Printf("ExecuteContext: %v\n", err)
		os.Exit(1)
	}
}

func findEth(apiList []rpc.API) jsonrpc.EthAPI {
	for _, api := range apiList {
		if api.Namespace == "eth" {
			if ethAPI, ok := api.Service.(jsonrpc.EthAPI); ok {
				return ethAPI
			}
		}
	}
	return nil
}

func findDebug(apiList []rpc.API) *jsonrpc.DebugAPIImpl {
	for _, api := range apiList {
		if api.Namespace == "debug" {
			if ethAPI, ok := api.Service.(*jsonrpc.DebugAPIImpl); ok {
				return ethAPI
			}
		}
	}
	return nil
}
