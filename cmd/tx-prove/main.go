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

	"github.com/erigontech/erigon-lib/common"
	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/jsonstream"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon/cmd/rpcdaemon/cli"
	"github.com/erigontech/erigon/core/tracing"
	"github.com/erigontech/erigon/eth/tracers"
	"github.com/erigontech/erigon/eth/tracers/config"
	"github.com/erigontech/erigon/rpc"
	"github.com/erigontech/erigon/rpc/jsonrpc"
	"github.com/erigontech/erigon/turbo/debug"
	"github.com/holiman/uint256"
	"github.com/spf13/cobra"
)

func newTracer(code string, ctx *tracers.Context, cfg json.RawMessage) (*tracers.Tracer, error) {
	return &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnEnter: func(depth int, typ byte, from common.Address, to common.Address, precompile bool, input []byte, gas uint64, value *uint256.Int, code []byte) {
				fmt.Println("hello from tracer ...")
				fmt.Println(from.Hex())
			},
			OnOpcode: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
				fmt.Println("hello from tracer ...")
				fmt.Println(pc)
			},
		},
		Stop: func(err error) {
			fmt.Println(err.Error())
		},
		GetResult: func() (json.RawMessage, error) {
			return nil, nil
		},
	}, nil
}

type StructMinimalResults struct {
	AppVK string
	Proof string
}

func newTracerHooks(code string, ctx *tracers.Context, cfg json.RawMessage) (*tracers.Tracer, error) {
	newTracer := tracer.NewStateTracer()
	return &tracers.Tracer{
		Hooks: newTracer.Hooks(),
		Stop: func(err error) {
			fmt.Println(err.Error())
		},
		GetResult: func() (json.RawMessage, error) {
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
			zkVm := prover.NewZkProver(content)
			output, err := zkVm.Prove()
			if err != nil {
				return nil, err
			}
			data, err := json.Marshal(StructMinimalResults{
				Proof: hex.EncodeToString(output.Proof),
				AppVK: hex.EncodeToString(output.AppVK),
			})
			if err != nil {
				return nil, err
			}

			return data, nil
		},
	}, nil
}

func main() {
	cmd, cfg := cli.RootCommand()
	rootCtx, rootCancel := common.RootContext()

	var txHash string
	cmd.Flags().StringVar(&txHash, "tx-hash", "04d3d48f42983eb155be1ff4b66d5c5af8ed1cedecac055083a00f6e863603d2", "Transaction hash to trace (required)")
	// cmd.MarkFlagRequired("tx-hash")

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

		tracers.RegisterLookup(true, newTracerHooks)
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
		}

		fmt.Println("Results: ", buf.String())
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
