package main

import (
	"fmt"
	"os"

	"github.com/erigontech/erigon-lib/common"
	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon/cmd/rpcdaemon/cli"
	"github.com/erigontech/erigon/rpc"
	"github.com/erigontech/erigon/rpc/jsonrpc"
	"github.com/erigontech/erigon/turbo/debug"
	"github.com/spf13/cobra"
)

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

		return nil
	}

	if err := cmd.ExecuteContext(rootCtx); err != nil {
		fmt.Printf("ExecuteContext: %v\n", err)
		os.Exit(1)
	}
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
