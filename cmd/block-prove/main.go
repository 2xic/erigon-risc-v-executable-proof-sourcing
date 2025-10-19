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
	"sync"
	"time"

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
	var transpileBlock bool
	var debugMode bool
	var skipProof bool
	var maxTxs int
	cmd.Flags().StringVar(&blockNumber, "block-number", "", "Block number to trace all transactions (required)")
	cmd.Flags().BoolVar(&debugAssembly, "debug-assembly", false, "Write transpiled assembly to disk for debugging")
	cmd.Flags().StringVar(&assemblyFile, "assembly-file", "transpiled_block.s", "Assembly output file path (used with --debug-assembly)")
	cmd.Flags().BoolVar(&transpileBlock, "transpile-block", true, "Transpile entire block as single unit with transaction boundaries")
	cmd.Flags().BoolVar(&debugMode, "debug-mode", false, "Enable debug transpiler with detailed mappings")
	cmd.Flags().BoolVar(&skipProof, "skip-proof", false, "Skip ZK proof generation to save memory")
	cmd.Flags().IntVar(&maxTxs, "max-txs", 0, "Limit to first N transactions (0 = all transactions, useful for binary search debugging)")

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

		if transpileBlock {
			return processBlockAsUnit(ctx, debugAPI, blockNum, txs, debugAssembly, assemblyFile, debugMode, skipProof, maxTxs)
		}

		// Transaction context for tracer callback
		var currentTxIndex int
		var currentTxHash common.Hash
		var ranTracer bool

		// Register tracer once
		customTracer := tracer.NewTracerHooks(
			func(newTracer *tracer.StateTracer) (*prover.ResultsFile, error) {
				ranTracer = true
				txInstructions := newTracer.GetInstructions()
				txExecutionState := newTracer.GetExecutionState()

				fmt.Printf("Transaction %d generated %d instructions\n", currentTxIndex+1, len(txInstructions))

				transpiler := transpiler.NewTranspiler()
				_, err := transpiler.ProcessExecution(txInstructions, txExecutionState)
				if err != nil {
					return nil, fmt.Errorf("failed to transpile transaction %s: %v", currentTxHash.String(), err)
				}

				assembly := transpiler.ToAssembly()
				content, err := assembly.ToToolChainCompatibleAssembly()
				if err != nil {
					return nil, fmt.Errorf("failed to generate assembly for transaction %s: %v", currentTxHash.String(), err)
				}

				zkVm := prover.NewZkProver(content)
				output, err := zkVm.Prove(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to prove transaction %s: %v", currentTxHash.String(), err)
				}

				result := ProofResult{
					TransactionHash:  currentTxHash.String(),
					TransactionIndex: currentTxIndex + 1,
					InstructionCount: len(txInstructions),
					AppVK:            hex.EncodeToString(output.AppVK),
					Proof:            hex.EncodeToString(output.Proof),
				}

				jsonData, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return nil, fmt.Errorf("failed to marshal JSON for transaction %s: %v", currentTxHash.String(), err)
				}

				outputFolder := fmt.Sprintf("%d", blockNum)
				err = os.MkdirAll(outputFolder, 0755)
				if err != nil {
					fmt.Printf("Error creating directory %s: %v\n", outputFolder, err)
				} else {
					txOutputFile := fmt.Sprintf("%s/tx_%d.json", outputFolder, currentTxIndex+1)
					err := os.WriteFile(txOutputFile, jsonData, 0644)
					if err != nil {
						fmt.Printf("Error writing to file %s: %v\n", txOutputFile, err)
					} else {
						fmt.Printf("Transaction %d results written to: %s\n", currentTxIndex+1, txOutputFile)
					}
				}

				return &prover.ResultsFile{}, nil
			},
		)
		tracers.RegisterLookup(false, customTracer)

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

			// Set current transaction context for tracer callback
			currentTxIndex = i
			currentTxHash = txHash
			ranTracer = false

			var buf bytes.Buffer
			stream := jsonstream.New(&buf)

			tracerName := "Mine"
			timeout := "2m"
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

func traceTransaction(ctx context.Context, debugAPI *jsonrpc.DebugAPIImpl, txHash common.Hash) ([]*tracer.EvmInstructionMetadata, *tracer.EvmExecutionState, error) {
	var tracerResult *tracer.StateTracer

	customTracer := tracer.NewTracerHooks(
		func(newTracer *tracer.StateTracer) (*prover.ResultsFile, error) {
			tracerResult = newTracer
			return &prover.ResultsFile{}, nil
		},
	)
	tracers.RegisterLookup(false, customTracer)

	var buf bytes.Buffer
	stream := jsonstream.New(&buf)

	tracerName := "Mine"
	timeout := "10m"
	err := debugAPI.TraceTransaction(
		ctx,
		txHash,
		&config.TraceConfig{
			Tracer:  &tracerName,
			Timeout: &timeout,
		},
		stream,
	)

	if err != nil {
		return nil, nil, err
	}

	if tracerResult == nil {
		return nil, nil, fmt.Errorf("no tracer result received")
	}

	return tracerResult.GetInstructions(), tracerResult.GetExecutionState(), nil
}

func processBlockAsUnit(ctx context.Context, debugAPI *jsonrpc.DebugAPIImpl, blockNum uint64, txs []interface{}, debugAssembly bool, assemblyFile string, debugMode bool, skipProof bool, maxTxs int) error {
	fmt.Printf("Processing block %d with %d transactions using parallel tracing...\n", blockNum, len(txs))

	// Define structures for parallel processing
	type TraceJob struct {
		Index    int
		TxIndex  int
		TxHash   common.Hash
		TxObject *ethapi.RPCTransaction
	}

	type TraceResult struct {
		Index        int
		TxIndex      int
		TxHash       common.Hash
		Instructions []*tracer.EvmInstructionMetadata
		State        *tracer.EvmExecutionState
		Error        error
	}

	// Collect all valid transactions (limited by maxTxs if specified)
	var jobs []TraceJob
	for i, txInterface := range txs {
		if maxTxs > 0 && i >= maxTxs {
			fmt.Printf("Limiting to first %d transactions for debugging\n", maxTxs)
			break
		}

		tx, ok := txInterface.(*ethapi.RPCTransaction)
		if !ok {
			fmt.Printf("Skipping invalid transaction %d (type: %T)\n", i+1, txInterface)
			continue
		}
		jobs = append(jobs, TraceJob{
			Index:    len(jobs),
			TxIndex:  i,
			TxHash:   tx.Hash,
			TxObject: tx,
		})
	}

	// Parallel tracing with semaphore to limit concurrency
	maxConcurrent := 5
	semaphore := make(chan struct{}, maxConcurrent)
	results := make([]TraceResult, len(jobs))
	var wg sync.WaitGroup

	fmt.Printf("Tracing %d transactions in parallel (max %d concurrent)...\n", len(jobs), maxConcurrent)

	// Launch all tracing goroutines
	for _, job := range jobs {
		wg.Add(1)
		go func(j TraceJob) {
			defer wg.Done()

			// Acquire semaphore to limit concurrency
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fmt.Printf("Tracing transaction %d/%d: %s\n", j.TxIndex+1, len(txs), j.TxHash.String())

			instructions, state, err := traceTransaction(ctx, debugAPI, j.TxHash)

			results[j.Index] = TraceResult{
				Index:        j.Index,
				TxIndex:      j.TxIndex,
				TxHash:       j.TxHash,
				Instructions: instructions,
				State:        state,
				Error:        err,
			}

			if err != nil {
				fmt.Printf("Failed to trace transaction %d: %v\n", j.TxIndex+1, err)
			} else {
				fmt.Printf("Completed trace for transaction %d/%d with %d instructions\n",
					j.TxIndex+1, len(txs), len(instructions))
			}
		}(job)
	}

	// Wait for all traces to complete
	fmt.Printf("Waiting for all %d transactions to complete tracing...\n", len(jobs))
	wg.Wait()
	fmt.Printf("All transactions traced successfully!\n")

	// Process all transactions with boundaries in one transpiler
	fmt.Printf("Processing all %d traced transactions with boundaries...\n", len(results))
	blockTranspiler := transpiler.NewTranspiler() // Use working version for now

	var allTxResults []ProofResult

	for i, result := range results {
		if result.Error != nil {
			fmt.Printf("Skipping transaction %d due to trace error: %v\n", result.TxIndex+1, result.Error)
			continue
		}

		fmt.Printf("Transpiling transaction %d/%d with %d instructions\n",
			result.TxIndex+1, len(txs), len(result.Instructions))

		_, err := blockTranspiler.ProcessExecution(result.Instructions, result.State)
		if err != nil {
			return fmt.Errorf("failed to transpile transaction %s: %v", result.TxHash.String(), err)
		}

		// Add transaction boundary (except for the last transaction)
		if i < len(results)-1 {
			blockTranspiler.AddTransactionBoundary()
		}

		allTxResults = append(allTxResults, ProofResult{
			TransactionHash:  result.TxHash.String(),
			TransactionIndex: result.TxIndex + 1,
			InstructionCount: len(result.Instructions),
		})
	}

	// Generate assembly for the entire block with transaction boundaries
	fmt.Printf("Generating assembly for block...\n")
	assemblyStart := time.Now()
	assembly := blockTranspiler.ToAssembly()
	content, err := assembly.ToToolChainCompatibleAssembly()
	assemblyTime := time.Since(assemblyStart)
	fmt.Printf("Assembly generation completed in %v\n", assemblyTime)

	if err != nil {
		return fmt.Errorf("failed to generate assembly for block: %v", err)
	}

	// Write debug assembly if requested
	if debugAssembly {
		err := os.WriteFile(assemblyFile, []byte(content), 0644)
		if err != nil {
			fmt.Printf("Error writing assembly file %s: %v\n", assemblyFile, err)
		} else {
			fmt.Printf("Block assembly written to: %s\n", assemblyFile)
		}
	}

	var output prover.ProofGeneration
	if skipProof {
		fmt.Printf("Skipping ZK proof generation (--skip-proof enabled)\n")

		if debugMode {
			debugFile := fmt.Sprintf("debug_mappings_block_%d.json", blockNum)
			if saveErr := blockTranspiler.SaveDebugMappings(debugFile); saveErr != nil {
				fmt.Printf("Failed to save debug mappings: %v\n", saveErr)
			} else {
				fmt.Printf("Debug mappings saved to: %s\n", debugFile)
			}
		}
	} else {
		fmt.Printf("Starting ZK proof generation for combined block...\n")
		proveStart := time.Now()
		zkVm := prover.NewZkProver(content)
		var err error
		output, err = zkVm.Prove(ctx)
		proveTime := time.Since(proveStart)

		if err != nil {
			fmt.Printf("ZK proof failed, saving debug info...\n")
			// Save debug mappings for analysis
			debugFile := fmt.Sprintf("debug_mappings_block_%d.json", blockNum)
			if saveErr := blockTranspiler.SaveDebugMappings(debugFile); saveErr != nil {
				fmt.Printf("Failed to save debug mappings: %v\n", saveErr)
			} else {
				fmt.Printf("Debug mappings saved to: %s\n", debugFile)
			}

			// If not in debug mode, suggest running with debug mode
			if !debugMode {
				fmt.Printf("Re-run with --debug-mode for detailed transpilation analysis\n")
			}

			return fmt.Errorf("failed to prove block: %v", err)
		}

		fmt.Printf("ZK proof generation completed in %v\n", proveTime)
	}

	blockResult := struct {
		BlockNumber       uint64        `json:"block_number"`
		TransactionCount  int           `json:"transaction_count"`
		Transactions      []ProofResult `json:"transactions"`
		TotalInstructions int           `json:"total_instructions"`
		AssemblyTimeMs    int64         `json:"assembly_time_ms"`
		ProofTimeMs       int64         `json:"proof_time_ms"`
		TotalTimeMs       int64         `json:"total_time_ms"`
		AppVK             string        `json:"app_vk"`
		Proof             string        `json:"proof"`
		Timestamp         string        `json:"timestamp"`
	}{
		BlockNumber:      blockNum,
		TransactionCount: len(allTxResults),
		Transactions:     allTxResults,
		AppVK:            hex.EncodeToString(output.AppVK),
		Proof:            hex.EncodeToString(output.Proof),
		AssemblyTimeMs:   assemblyTime.Milliseconds(),
		ProofTimeMs:      proveTime.Milliseconds(),
		TotalTimeMs:      (assemblyTime + proveTime).Milliseconds(),
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
	}

	for _, txResult := range allTxResults {
		blockResult.TotalInstructions += txResult.InstructionCount
	}

	jsonData, err := json.MarshalIndent(blockResult, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON for block: %v", err)
	}

	outputFile := fmt.Sprintf("block_%d.json", blockNum)
	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing to file %s: %v\n", outputFile, err)
	} else {
		fmt.Printf("Block %d results written to: %s\n", blockNum, outputFile)
	}

	fmt.Printf("Completed block transpilation with %d transactions and %d total instructions\n",
		len(allTxResults), blockResult.TotalInstructions)

	return nil
}
