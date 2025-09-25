package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/erigontech/erigon-lib/common"
	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/jsonstream"
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

func main() {
	// Use the exact same command setup as the real rpcdaemon
	cmd, cfg := cli.RootCommand()
	rootCtx, rootCancel := common.RootContext()

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		logger := debug.SetupCobra(cmd, "rpcdaemon")

		// Initialize services exactly like the real rpcdaemon
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

		// Create the exact same API list as the real rpcdaemon
		apiList := jsonrpc.APIList(db, backend, txPool, mining, ff, stateCache, blockReader, cfg, engine, logger, bridgeReader, heimdallReader)

		// Now you have access to all the same components the RPC daemon uses
		// You can either start the RPC server or use the components directly

		// Option 1: Start RPC server (same as real rpcdaemon)
		/*
			if err := cli.StartRpcServer(ctx, cfg, apiList, logger); err != nil {
				logger.Error(err.Error())
				return nil
			}*/

		// Option 2: Use components directly for your custom logic
		// Example: Get contract code using the same backend the RPC uses
		ethAPI := findEthAPI(apiList)
		address := libcommon.HexToAddress("1f98431c8ad98523631ae4a59f267346ea31f984")
		code, err := ethAPI.GetCode(ctx, address, rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
		fmt.Println("code length", len(code))

		debugAPI := findDebug(apiList)

		var buf bytes.Buffer
		stream := jsonstream.New(&buf)

		tracers.RegisterLookup(true, newTracer)

		tracer := "bagel"
		err = debugAPI.TraceTransaction(
			context.Background(),
			libcommon.HexToHash("04d3d48f42983eb155be1ff4b66d5c5af8ed1cedecac055083a00f6e863603d2"),
			&config.TraceConfig{
				Tracer: &tracer,
			},
			stream,
		)
		//		fmt.Println("code length", len(debug.trace))
		if err != nil {
			fmt.Println("failed to do trace", err.Error())
		}

		return nil
	}

	if err := cmd.ExecuteContext(rootCtx); err != nil {
		fmt.Printf("ExecuteContext: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("peace out")
}

// Helper function to extract the ETH API from the API list
func findEthAPI(apiList []rpc.API) *jsonrpc.APIImpl {
	for _, api := range apiList {
		if api.Namespace == "eth" {
			if ethAPI, ok := api.Service.(*jsonrpc.APIImpl); ok {
				return ethAPI
			}
		}
	}
	return nil
}

// Helper function to extract the ETH API from the API list
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
