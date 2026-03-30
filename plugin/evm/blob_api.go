// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"fmt"

	"github.com/ava-labs/coreth/core"
	"github.com/ava-labs/coreth/plugin/evm/customrawdb"
	"github.com/ava-labs/coreth/rpc"
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/common/hexutil"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/ethdb"
)

// BlobAPI provides an API to access blob sidecar data for blocks.
type BlobAPI struct {
	db         ethdb.KeyValueStore
	blockchain *core.BlockChain
}

// BlobsResult contains the blob data for a single block.
type BlobsResult struct {
	BlockHash   common.Hash    `json:"blockHash"`
	BlockNumber hexutil.Uint64 `json:"blockNumber"`
	Blobs       []BlobResult   `json:"blobs"`
}

// BlobResult contains a single blob with its metadata.
type BlobResult struct {
	TxHash     common.Hash    `json:"txHash"`
	TxIndex    hexutil.Uint64 `json:"txIndex"`
	BlobIndex  hexutil.Uint64 `json:"blobIndex"`
	Blob       hexutil.Bytes  `json:"blob"`
	Commitment hexutil.Bytes  `json:"kzgCommitment"`
	Proof      hexutil.Bytes  `json:"kzgProof"`
}

// GetBlobs returns blob data for a given block identified by number or hash.
func (api *BlobAPI) GetBlobs(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*BlobsResult, error) {
	var block *types.Block

	if blockNr, ok := blockNrOrHash.Number(); ok {
		switch blockNr {
		case rpc.LatestBlockNumber, rpc.PendingBlockNumber:
			header := api.blockchain.CurrentBlock()
			if header != nil {
				block = api.blockchain.GetBlock(header.Hash(), header.Number.Uint64())
			}
		case rpc.FinalizedBlockNumber, rpc.SafeBlockNumber:
			header := api.blockchain.CurrentBlock()
			if header != nil {
				block = api.blockchain.GetBlock(header.Hash(), header.Number.Uint64())
			}
		case rpc.EarliestBlockNumber:
			block = api.blockchain.GetBlockByNumber(0)
		default:
			block = api.blockchain.GetBlockByNumber(uint64(blockNr))
		}
	} else if hash, ok := blockNrOrHash.Hash(); ok {
		block = api.blockchain.GetBlockByHash(hash)
	}

	if block == nil {
		return nil, fmt.Errorf("block not found")
	}

	entries, err := customrawdb.ReadBlobSidecars(api.db, block.Hash())
	if err != nil {
		return nil, fmt.Errorf("blob sidecars not available for block %d", block.NumberU64())
	}

	result := &BlobsResult{
		BlockHash:   block.Hash(),
		BlockNumber: hexutil.Uint64(block.NumberU64()),
		Blobs:       make([]BlobResult, 0),
	}

	for _, entry := range entries {
		if entry.Sidecar == nil {
			continue
		}
		for i := range entry.Sidecar.Blobs {
			result.Blobs = append(result.Blobs, BlobResult{
				TxHash:     entry.TxHash,
				TxIndex:    hexutil.Uint64(entry.TxIndex),
				BlobIndex:  hexutil.Uint64(i),
				Blob:       entry.Sidecar.Blobs[i][:],
				Commitment: entry.Sidecar.Commitments[i][:],
				Proof:      entry.Sidecar.Proofs[i][:],
			})
		}
	}

	return result, nil
}
