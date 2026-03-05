package evm

import (
	"context"
	"testing"

	"github.com/ava-labs/libevm/common/hexutil"
	"github.com/stretchr/testify/assert"
)

func TestRemoteMinerAPISubmitTransactionsInvalidRLP(t *testing.T) {
	responder := NewRemoteMinerResponder(10)
	api := &RemoteMinerAPI{
		responder: responder,
	}

	err := api.SubmitTransactions(context.Background(), []hexutil.Bytes{[]byte("not-valid-rlp")})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transaction 0")

	// Nothing should be enqueued
	_, ok := responder.Dequeue()
	assert.False(t, ok)
}

func TestRemoteMinerAPISubmitTransactionsEmpty(t *testing.T) {
	responder := NewRemoteMinerResponder(10)
	api := &RemoteMinerAPI{
		responder: responder,
	}

	err := api.SubmitTransactions(context.Background(), []hexutil.Bytes{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no transactions provided")

	_, ok := responder.Dequeue()
	assert.False(t, ok)
}
