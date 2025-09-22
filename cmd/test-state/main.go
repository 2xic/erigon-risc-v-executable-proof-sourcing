package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"erigon-transpiler-risc-v/tracer"

	"github.com/erigontech/erigon-lib/common"
)

func main() {
	fmt.Println("State lookup")
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <chaindata-path>\n", os.Args[0])
		os.Exit(1)
	}

	chainDataPath := os.Args[1]

	fmt.Printf("Testing state database: %s\n", chainDataPath)
	latestsBlock, err := tracer.GetLatestsBLockInStateDb(chainDataPath)
	if err != nil {
		panic(err)
	}
	fmt.Println("latests block", latestsBlock)
	bytes, err := hex.DecodeString("1f98431c8ad98523631ae4a59f267346ea31f984")
	if err != nil {
		panic(err)
	}
	contractCode, err := tracer.GetContractCodeViaState(chainDataPath, common.BytesToAddress(bytes))
	if err != nil {
		panic(err)
	}
	fmt.Println(contractCode)
	contractCodeV3, err := tracer.GetContractCodeWithReaderV3(chainDataPath, common.BytesToAddress(bytes))
	if err != nil {
		panic(err)
	}
	fmt.Println(contractCodeV3)

	//err = tracer.GetLatestsTransactionInfo(chainDataPath)
	//fmt.Println(err)

	fmt.Printf("Testing state database: %s\n", chainDataPath)

	state, err := tracer.NewStateDbAA(chainDataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Success! State database loaded.\n")

	// Test a well-known address - Ethereum Foundation donation address
	a, err := hex.DecodeString("9c33eacc2f50e39940d3afaf2c7b8246b681a374")
	if err != nil {
		fmt.Println("Bad hex decode")
		os.Exit(1)
	}
	addr := common.Address(common.BytesToAddress(a))

	balance, err := state.GetBalance(addr)
	if err != nil {
		fmt.Printf("Error getting balance: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Address %s balance: %s\n", addr.Hex(), balance.String())

	nonce, err := state.GetNonce(addr)
	if err != nil {
		fmt.Printf("Error getting nonce: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Address nonce: %d\n", nonce)

	code, err := state.GetCode(addr)
	if err != nil {
		fmt.Printf("Error getting code: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Address code: %d\n", code)
}
