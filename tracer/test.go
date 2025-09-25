package tracer

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/common/datadir"
	"github.com/erigontech/erigon-lib/config3"
	"github.com/erigontech/erigon-lib/kv"
	"github.com/erigontech/erigon-lib/kv/kvcache"
	"github.com/erigontech/erigon-lib/kv/mdbx"
	"github.com/erigontech/erigon-lib/kv/temporal"
	"github.com/erigontech/erigon-lib/log/v3"
	libstate "github.com/erigontech/erigon-lib/state"
	"github.com/erigontech/erigon/eth/ethconfig"
	"github.com/erigontech/erigon/polygon/heimdall"
	"github.com/erigontech/erigon/rpc"
	"github.com/erigontech/erigon/rpc/rpchelper"
	"github.com/erigontech/erigon/turbo/snapshotsync/freezeblocks"
)

func CreateStateReaderFromChaindata(chaindataDir string, addr common.Address) ([]byte, error) {
	ctx := context.Background()
	logger := log.New()

	// Initialize directories structure
	dirs := datadir.New(filepath.Dir(chaindataDir))

	// Open the chaindata database
	rawDB, err := mdbx.New(kv.ChainDB, logger).Path(chaindataDir).Readonly(true).Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open chaindata: %w", err)
	}
	defer rawDB.Close()

	// Initialize snapshots
	snapCfg := ethconfig.Defaults.Snapshot
	snapCfg.ChainName = "mainnet"
	allSnapshots := freezeblocks.NewRoSnapshots(snapCfg, dirs.Snap, 0, logger)
	allBorSnapshots := heimdall.NewRoSnapshots(snapCfg, dirs.Snap, 0, logger)

	// Open snapshots
	if err := allSnapshots.OpenFiles(); err != nil {
		return nil, fmt.Errorf("failed to reopen snapshots: %w", err)
	}
	if err := allBorSnapshots.OpenFiles(); err != nil {
		return nil, fmt.Errorf("failed to reopen bor snapshots: %w", err)
	}

	// Initialize block reader
	blockReader := freezeblocks.NewBlockReader(allSnapshots, allBorSnapshots)
	txNumsReader := blockReader.TxnumReader(ctx)

	// Initialize aggregator
	agg, err := libstate.NewAggregator(ctx, dirs, config3.DefaultStepSize, rawDB, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create aggregator: %w", err)
	}
	defer agg.Close()

	if err := agg.OpenFolder(); err != nil {
		return nil, fmt.Errorf("failed to open aggregator folder: %w", err)
	}

	// Create temporal database
	db, err := temporal.New(rawDB, agg)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporal db: %w", err)
	}
	defer db.Close()

	// Begin temporal transaction
	tx, err := db.BeginTemporalRo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin temporal tx: %w", err)
	}
	defer tx.Rollback()

	// Initialize state cache
	stateCache := kvcache.New(kvcache.DefaultCoherentConfig)

	// Initialize filters
	filters := rpchelper.New(ctx, rpchelper.DefaultFiltersConfig, nil, nil, nil, func() {}, logger)

	// Now call CreateStateReader with all properly initialized components
	reader, err := rpchelper.CreateStateReader(ctx, tx, blockReader, rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber), 0, filters, stateCache, txNumsReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create state reader: %w", err)
	}
	a, _ := reader.ReadAccountData(addr)
	fmt.Println("hash", a.CodeHash.Hex())
	fmt.Println("nonce", a.Nonce)

	code, err := reader.ReadAccountCodeSize(addr)
	fmt.Println("code", code)

	codex, err := tx.GetOne(kv.Code, a.CodeHash[:])
	fmt.Println("direct read ", len(codex))

	fmt.Println("lets goooo")
	header, err := blockReader.HeaderByNumber(ctx, tx, rpc.LatestBlockNumber.Uint64())
	if err != nil || header == nil {
		fmt.Println("header is nil", header == nil)
		return nil, err
	}
	fmt.Println(" gas usage ... ", header.GasUsed)

	return reader.ReadAccountCode(addr)
}
