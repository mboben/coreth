// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/network/p2p"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/coreth/plugin/evm/customrawdb"
	ethcommon "github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/ethdb"
	"github.com/ava-labs/libevm/log"
	"github.com/ava-labs/libevm/rlp"
)

const (
	// BlobSidecarRequestHandlerID is the p2p protocol ID for blob sidecar requests.
	// Uses a custom ID outside avalanchego's reserved range.
	BlobSidecarRequestHandlerID uint64 = 0x100
)

// BlobSidecarRequest is the request message for blob sidecars.
type BlobSidecarRequest struct {
	BlockHash ethcommon.Hash
}

// BlobSidecarResponse is the response message containing blob sidecars.
type BlobSidecarResponse struct {
	Found   bool
	Entries []customrawdb.BlobSidecarEntry
}

// BlobSidecarHandler serves blob sidecar requests from peers.
type BlobSidecarHandler struct {
	db ethdb.KeyValueStore
}

var _ p2p.Handler = (*BlobSidecarHandler)(nil)

func (h *BlobSidecarHandler) AppGossip(_ context.Context, _ ids.NodeID, _ []byte) {
	// No-op for pull-based protocol
}

func (h *BlobSidecarHandler) AppRequest(_ context.Context, _ ids.NodeID, _ time.Time, requestBytes []byte) ([]byte, *common.AppError) {
	var req BlobSidecarRequest
	if err := rlp.DecodeBytes(requestBytes, &req); err != nil {
		return nil, &common.AppError{
			Code:    1,
			Message: fmt.Sprintf("failed to decode blob sidecar request: %v", err),
		}
	}

	entries, err := customrawdb.ReadBlobSidecars(h.db, req.BlockHash)
	if err != nil {
		// Not found - return empty response
		resp := BlobSidecarResponse{Found: false}
		respBytes, _ := rlp.EncodeToBytes(resp)
		return respBytes, nil
	}

	resp := BlobSidecarResponse{
		Found:   true,
		Entries: entries,
	}
	respBytes, err := rlp.EncodeToBytes(resp)
	if err != nil {
		log.Error("Failed to encode blob sidecar response", "err", err)
		return nil, &common.AppError{
			Code:    2,
			Message: fmt.Sprintf("failed to encode response: %v", err),
		}
	}
	return respBytes, nil
}
