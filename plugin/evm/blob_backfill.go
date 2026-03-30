// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"time"

	"github.com/ava-labs/coreth/core"
	"github.com/ava-labs/coreth/plugin/evm/customrawdb"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/ethdb"
	"github.com/ava-labs/libevm/log"
)

// BlobBackfiller scans recent blocks and fetches any missing blob sidecars from peers.
// This is used after a node restart to catch up on blobs that were missed while offline.
type BlobBackfiller struct {
	blockchain *core.BlockChain
	db         ethdb.KeyValueStore
	fetcher    *BlobFetcher
	retention  uint64 // number of blocks to retain
}

// Run scans from the current head backward through the retention window,
// fetching any missing blob sidecars. It rate-limits requests to avoid
// overwhelming peers.
func (b *BlobBackfiller) Run(ctx context.Context) {
	head := b.blockchain.CurrentBlock()
	if head == nil || head.Number == nil {
		return
	}
	headNum := head.Number.Uint64()
	var cutoff uint64
	if headNum > b.retention {
		cutoff = headNum - b.retention
	}

	log.Info("Starting blob sidecar backfill", "head", headNum, "cutoff", cutoff)
	fetched := 0

	for num := headNum; num > cutoff; num-- {
		select {
		case <-ctx.Done():
			log.Info("Blob backfill interrupted", "fetched", fetched)
			return
		default:
		}

		block := b.blockchain.GetBlockByNumber(num)
		if block == nil {
			continue
		}
		if !blockHasBlobTxs(block) {
			continue
		}
		if customrawdb.HasBlobSidecars(b.db, block.Hash()) {
			continue
		}

		if err := b.fetcher.FetchBlobSidecars(ctx, block.Hash(), num); err != nil {
			log.Warn("Failed to fetch blob sidecars during backfill", "block", num, "err", err)
		} else {
			fetched++
		}

		// Rate limit: small delay between requests to avoid overwhelming peers
		select {
		case <-ctx.Done():
			return
		case <-time.After(50 * time.Millisecond):
		}
	}

	log.Info("Blob sidecar backfill complete", "fetched", fetched)
}

// blockHasBlobTxs checks if a block contains any blob transactions.
func blockHasBlobTxs(block *types.Block) bool {
	for _, tx := range block.Transactions() {
		if tx.Type() == types.BlobTxType {
			return true
		}
	}
	return false
}
