// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"fmt"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/network/p2p"
	"github.com/ava-labs/coreth/plugin/evm/customrawdb"
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/ethdb"
	"github.com/ava-labs/libevm/log"
	"github.com/ava-labs/libevm/rlp"
)

// BlobFetcher requests blob sidecars from peers via the p2p protocol.
type BlobFetcher struct {
	client *p2p.Client
	db     ethdb.KeyValueStore
}

// FetchBlobSidecars requests blob sidecars for a block from peers and stores them.
// Returns nil if sidecars were successfully fetched and stored, or if they already exist.
func (f *BlobFetcher) FetchBlobSidecars(ctx context.Context, blockHash common.Hash, blockNumber uint64) error {
	if customrawdb.HasBlobSidecars(f.db, blockHash) {
		return nil // Already have them
	}

	reqBytes, err := rlp.EncodeToBytes(BlobSidecarRequest{BlockHash: blockHash})
	if err != nil {
		return err
	}

	type result struct {
		respBytes []byte
		err       error
	}
	ch := make(chan result, 1)

	err = f.client.AppRequestAny(ctx, reqBytes, func(_ context.Context, _ ids.NodeID, responseBytes []byte, respErr error) {
		ch <- result{respBytes: responseBytes, err: respErr}
	})
	if err != nil {
		return fmt.Errorf("failed to send blob sidecar request: %w", err)
	}

	// Wait for the response
	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-ch:
		if res.err != nil {
			return fmt.Errorf("blob sidecar request failed: %w", res.err)
		}
		return f.handleResponse(res.respBytes, blockHash, blockNumber)
	}
}

func (f *BlobFetcher) handleResponse(respBytes []byte, blockHash common.Hash, blockNumber uint64) error {
	var resp BlobSidecarResponse
	if err := rlp.DecodeBytes(respBytes, &resp); err != nil {
		return fmt.Errorf("failed to decode blob sidecar response: %w", err)
	}

	if !resp.Found || len(resp.Entries) == 0 {
		log.Debug("No blob sidecars available from peers", "block", blockNumber, "hash", blockHash)
		return nil
	}

	if err := customrawdb.WriteBlobSidecars(f.db, blockHash, blockNumber, resp.Entries); err != nil {
		return err
	}

	log.Debug("Fetched and stored blob sidecars", "block", blockNumber, "hash", blockHash, "entries", len(resp.Entries))
	return nil
}
