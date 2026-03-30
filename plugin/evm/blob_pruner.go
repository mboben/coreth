// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"time"

	"github.com/ava-labs/coreth/core"
	"github.com/ava-labs/coreth/plugin/evm/customrawdb"
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/ethdb"
	"github.com/ava-labs/libevm/log"
)

// BlobPruner periodically deletes blob sidecars older than the retention window.
type BlobPruner struct {
	db         ethdb.KeyValueStore
	blockchain *core.BlockChain
	retention  uint64        // number of blocks to retain
	interval   time.Duration // pruning frequency
}

// Run starts the pruning loop, which runs periodically until the context is cancelled.
func (p *BlobPruner) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.prune()
		}
	}
}

func (p *BlobPruner) prune() {
	head := p.blockchain.CurrentBlock()
	if head == nil || head.Number == nil {
		return
	}
	headNum := head.Number.Uint64()
	if headNum <= p.retention {
		return
	}
	cutoff := headNum - p.retention

	pruned := 0
	batch := p.db.NewBatch()

	customrawdb.IterateBlobSidecarNumbers(p.db, func(number uint64, hash common.Hash) bool {
		if number > cutoff {
			return false // stop iteration
		}
		if err := customrawdb.DeleteBlobSidecars(batch, hash, number); err != nil {
			log.Warn("Failed to delete blob sidecars", "block", number, "err", err)
			return true // continue
		}
		pruned++

		// Write batch periodically to avoid excessive memory use
		if batch.ValueSize() > 1024*1024 { // 1MB
			if err := batch.Write(); err != nil {
				log.Error("Failed to write blob pruning batch", "err", err)
				return false
			}
			batch.Reset()
		}
		return true
	})

	if batch.ValueSize() > 0 {
		if err := batch.Write(); err != nil {
			log.Error("Failed to write blob pruning batch", "err", err)
		}
	}

	if pruned > 0 {
		log.Info("Pruned old blob sidecars", "pruned", pruned, "cutoff", cutoff)
	}
}
