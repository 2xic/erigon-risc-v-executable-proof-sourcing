package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"erigon-transpiler-risc-v/prover"
	"erigon-transpiler-risc-v/tracer"
	"erigon-transpiler-risc-v/transpiler"
	"fmt"
	"os"
	"time"

	"github.com/erigontech/erigon-lib/common"
	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/jsonstream"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon/cmd/rpcdaemon/cli"
	"github.com/erigontech/erigon/eth/tracers"
	"github.com/erigontech/erigon/eth/tracers/config"
	"github.com/erigontech/erigon/rpc"
	"github.com/erigontech/erigon/rpc/jsonrpc"
	"github.com/erigontech/erigon/turbo/debug"
	"github.com/spf13/cobra"
)

func main() {
	cmd, cfg := cli.RootCommand()
	rootCtx, rootCancel := common.RootContext()

	var txHash string
	var outputFile string
	var debugAssembly bool
	var debugMode bool
	var skipProving bool
	var assemblyFile string
	cmd.Flags().StringVar(&txHash, "tx-hash", "0x04d3d48f42983eb155be1ff4b66d5c5af8ed1cedecac055083a00f6e863603d2", "Transaction hash to trace (required)")
	cmd.Flags().StringVar(&outputFile, "output", "", "Output file path (optional, defaults to stdout)")
	cmd.Flags().BoolVar(&debugAssembly, "debug-assembly", false, "Write transpiled assembly to disk for debugging")
	cmd.Flags().BoolVar(&debugMode, "debug-mode", false, "Enable debug transpiler with detailed mappings")
	cmd.Flags().StringVar(&assemblyFile, "assembly-file", "transpiled.s", "Assembly output file path (used with --debug-assembly)")
	cmd.Flags().BoolVar(&skipProving, "skip-proving", false, "Skip proof generation")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
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
		var buf bytes.Buffer
		stream := jsonstream.New(&buf)
		debugAPI := findDebug(apiList)

		ranTracer := false
		customTracer := tracer.NewTracerHooks(
			func(newTracer *tracer.StateTracer) (*prover.ResultsFile, error) {
				ranTracer = true
				fmt.Println("hello")
				transpiler := transpiler.NewTranspiler()
				instructions := newTracer.GetInstructions()
				executionState := newTracer.GetExecutionState()
				_, err := transpiler.ProcessExecution(instructions, executionState)
				if err != nil {
					return nil, err
				}
				assembly := transpiler.ToAssembly()
				content, err := assembly.ToToolChainCompatibleAssembly()
				if err != nil {
					return nil, err
				}

				if debugMode {
					debugFile := "debug_mappings.json"
					err = transpiler.SaveDebugMappings(debugFile)
					if err != nil {
						fmt.Printf("Warning: Failed to write debug mappings to %s: %v\n", debugFile, err)
					} else {
						fmt.Printf("Debug mappings written to: %s\n", debugFile)
					}
				}

				if debugAssembly {
					err := os.WriteFile(assemblyFile, []byte(content), 0644)
					if err != nil {
						fmt.Printf("Warning: Failed to write assembly to %s: %v\n", assemblyFile, err)
					} else {
						fmt.Printf("Transpiled assembly written to: %s\n", assemblyFile)
					}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()

				if skipProving {
					fmt.Println("Skipping proving as per --skip-proving flag.")
					return &prover.ResultsFile{
						AppVK: "skipped",
						Proof: "skipped",
					}, nil
				}
				zkVm := prover.NewZkProver(content)
				output, err := zkVm.Prove(ctx)
				if err != nil {
					return nil, err
				}

				return &prover.ResultsFile{
					AppVK: hex.EncodeToString(output.AppVK),
					Proof: hex.EncodeToString(output.Proof),
				}, nil
			},
		)

		tracers.RegisterLookup(true, customTracer)
		tracer := "Mine"
		timeout := "10h"
		err = debugAPI.TraceTransaction(
			context.Background(),
			libcommon.HexToHash(txHash),
			&config.TraceConfig{
				Tracer:  &tracer,
				Timeout: &timeout,
			},
			stream,
		)
		if err != nil {
			fmt.Println("failed to do trace", err.Error())
			os.Exit(1)
		}

		if !ranTracer {
			fmt.Println("failed to do trace ... Transaction not found?")
			os.Exit(1)
		}

		// Output results to file or stdout
		results := buf.String()
		if outputFile != "" {
			err := os.WriteFile(outputFile, []byte(results), 0644)
			if err != nil {
				fmt.Printf("Error writing to file %s: %v\n", outputFile, err)
				return err
			}
			fmt.Printf("Results written to: %s\n", outputFile)
		} else {
			fmt.Println("Results: ", results)
		}
		os.Exit(0)

		return nil
	}

	if err := cmd.ExecuteContext(rootCtx); err != nil {
		fmt.Printf("ExecuteContext: %v\n", err)
		os.Exit(1)
	}
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
