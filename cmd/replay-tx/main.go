package main

import (
	"fmt"
	"os"

	"erigon-transpiler-risc-v/tracer"

	libcommon "github.com/erigontech/erigon-lib/common"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <chaindata-path> <tx-hash>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s /path/to/chaindata 0x1234...\n", os.Args[0])
		os.Exit(1)
	}

	chainDataPath := os.Args[1]
	var txnHashStr string
	if len(os.Args) > 2 {
		txnHashStr = os.Args[2]
	} else {
		txnHashStr = "84c9fc3bc6856d53f49ea3bea242201c96b8ac4ae5b0f6aa8d371921f7fb1314"
	}

	// Parse transaction hash
	txHash := libcommon.HexToHash(txnHashStr)

	fmt.Printf("Replaying transaction: %s\n", txHash.Hex())
	fmt.Printf("Using chaindata: %s\n", chainDataPath)

	// Create your custom tracer
	customTracer := tracer.NewStateTracer()

	// Replay the transaction using the new otterscan approach
	err := tracer.SimpleReplayTransaction(chainDataPath, txHash, customTracer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Transaction replayed successfully!")
	fmt.Println("Your custom tracer was called during execution")
}
