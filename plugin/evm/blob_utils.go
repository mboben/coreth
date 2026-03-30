// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"github.com/ava-labs/coreth/plugin/evm/customrawdb"
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/types"
)

// buildBlobSidecarEntries creates BlobSidecarEntry structs from the miner's
// sidecars and the corresponding transaction hashes. The txIndex for each
// entry is determined by finding the blob transaction in the block's tx list.
func buildBlobSidecarEntries(block *types.Block, sidecars []*types.BlobTxSidecar, txHashes []common.Hash) []customrawdb.BlobSidecarEntry {
	// Build a map of tx hash -> index in block
	txIndexMap := make(map[common.Hash]uint64)
	for i, tx := range block.Transactions() {
		if tx.Type() == types.BlobTxType {
			txIndexMap[tx.Hash()] = uint64(i)
		}
	}

	entries := make([]customrawdb.BlobSidecarEntry, 0, len(sidecars))
	for i, sc := range sidecars {
		if i >= len(txHashes) {
			break
		}
		txHash := txHashes[i]
		txIdx, ok := txIndexMap[txHash]
		if !ok {
			continue
		}
		entries = append(entries, customrawdb.BlobSidecarEntry{
			TxHash:  txHash,
			TxIndex: txIdx,
			Sidecar: sc,
		})
	}
	return entries
}
