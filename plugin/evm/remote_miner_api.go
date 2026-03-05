package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ava-labs/coreth/core"
	"github.com/ava-labs/coreth/core/txpool"
	"github.com/ava-labs/libevm/common/hexutil"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/log"
	"github.com/ava-labs/libevm/rlp"
)

const (
	// Maximum transaction size for validation (128KB, same as legacy pool).
	remoteMinerTxMaxSize = 4 * 32 * 1024
)

// RemoteMinerAPI provides an RPC endpoint for the external miner to submit transactions.
// The responder node validates and stores the transactions; the requester node
// later retrieves them and builds a fully executed block locally.
type RemoteMinerAPI struct {
	responder  *RemoteMinerResponder
	blockchain *core.BlockChain
}

// NewRemoteMinerAPI creates a new RemoteMinerAPI.
func NewRemoteMinerAPI(responder *RemoteMinerResponder, blockchain *core.BlockChain) *RemoteMinerAPI {
	return &RemoteMinerAPI{
		responder:  responder,
		blockchain: blockchain,
	}
}

// SubmitTransactions accepts a list of RLP-encoded signed transactions from the external miner.
// It decodes and validates each transaction, then stores the list for relay to the requester.
func (api *RemoteMinerAPI) SubmitTransactions(_ context.Context, rawTxs []hexutil.Bytes) error {
	if len(rawTxs) == 0 {
		return fmt.Errorf("no transactions provided")
	}

	// 1. Decode transactions
	txs := make([]*types.Transaction, len(rawTxs))
	for i, raw := range rawTxs {
		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(raw); err != nil {
			return fmt.Errorf("invalid transaction %d: %w", i, err)
		}
		txs[i] = tx
	}

	// 2. Validate each transaction syntactically
	chainConfig := api.blockchain.Config()
	head := api.blockchain.CurrentHeader()
	signer := types.LatestSigner(chainConfig)

	opts := &txpool.ValidationOptions{
		Config:  chainConfig,
		Accept:  1<<types.LegacyTxType | 1<<types.AccessListTxType | 1<<types.DynamicFeeTxType,
		MaxSize: remoteMinerTxMaxSize,
		MinTip:  big.NewInt(0),
	}

	for i, tx := range txs {
		if err := txpool.ValidateTransaction(tx, head, signer, opts); err != nil {
			return fmt.Errorf("invalid transaction %d: %w", i, err)
		}
	}

	// 3. RLP-encode the transaction list and enqueue for the requester
	txsRLP, err := rlp.EncodeToBytes(txs)
	if err != nil {
		return fmt.Errorf("failed to encode transactions: %w", err)
	}

	api.responder.Enqueue(txsRLP)
	log.Info("RemoteMinerAPI: transactions submitted", "count", len(txs))
	return nil
}
