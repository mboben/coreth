package evm

import (
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/coreth/plugin/evm/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteMinerQueueBasic(t *testing.T) {
	rm := &RemoteMiner{
		queue: make([][]byte, 3),
	}

	// Dequeue from empty queue
	_, ok := rm.Dequeue()
	assert.False(t, ok)

	// Enqueue and dequeue
	rm.enqueue([]byte("a"))
	rm.enqueue([]byte("b"))

	val, ok := rm.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("a"), val)

	val, ok = rm.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("b"), val)

	_, ok = rm.Dequeue()
	assert.False(t, ok)
}

func TestRemoteMinerQueueDropOldest(t *testing.T) {
	rm := &RemoteMiner{
		queue: make([][]byte, 3),
	}

	rm.enqueue([]byte("a"))
	rm.enqueue([]byte("b"))
	rm.enqueue([]byte("c"))

	// Queue is full, this should drop "a"
	rm.enqueue([]byte("d"))

	val, ok := rm.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("b"), val)

	val, ok = rm.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("c"), val)

	val, ok = rm.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("d"), val)

	_, ok = rm.Dequeue()
	assert.False(t, ok)
}

func TestRemoteMinerQueueWrapAround(t *testing.T) {
	rm := &RemoteMiner{
		queue: make([][]byte, 3),
	}

	// Fill and drain to advance head/tail
	rm.enqueue([]byte("x"))
	rm.enqueue([]byte("y"))
	rm.Dequeue()
	rm.Dequeue()

	// Now head=2, tail=2, count=0
	rm.enqueue([]byte("a"))
	rm.enqueue([]byte("b"))
	rm.enqueue([]byte("c"))

	val, ok := rm.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("a"), val)

	val, ok = rm.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("b"), val)

	val, ok = rm.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("c"), val)
}

func TestRemoteMinerStartStop(t *testing.T) {
	nodeID := ids.GenerateTestNodeID()
	rm := NewRemoteMiner(nil, message.Codec, nodeID)
	// Don't actually start polling (no network), just test lifecycle
	// Start and stop should not panic
	rm.stopCh = make(chan struct{})
	close(rm.stopCh) // pre-close so pollLoop exits immediately
	rm.wg.Add(1)
	go rm.pollLoop()
	rm.wg.Wait()
}

func TestRemoteMinerResponseHandler(t *testing.T) {
	rm := &RemoteMiner{
		queue: make([][]byte, 3),
		codec: message.Codec,
	}

	// Test successful response
	container := []byte("test-block-bytes")
	response := message.RemoteContainerResponse{Container: container}
	responseBytes, err := message.Codec.Marshal(message.Version, &response)
	require.NoError(t, err)

	handler := &remoteMinerResponseHandler{rm: rm}
	err = handler.OnResponse(responseBytes)
	require.NoError(t, err)

	val, ok := rm.Dequeue()
	require.True(t, ok)
	assert.Equal(t, container, val)
}

func TestRemoteMinerResponseHandlerEmptyContainer(t *testing.T) {
	rm := &RemoteMiner{
		queue: make([][]byte, 3),
		codec: message.Codec,
	}

	// Empty container should not be enqueued
	response := message.RemoteContainerResponse{Container: nil}
	responseBytes, err := message.Codec.Marshal(message.Version, &response)
	require.NoError(t, err)

	handler := &remoteMinerResponseHandler{rm: rm}
	err = handler.OnResponse(responseBytes)
	require.NoError(t, err)

	_, ok := rm.Dequeue()
	assert.False(t, ok)
}

func TestRemoteMinerResponseHandlerInvalidBytes(t *testing.T) {
	rm := &RemoteMiner{
		queue: make([][]byte, 3),
		codec: message.Codec,
	}

	handler := &remoteMinerResponseHandler{rm: rm}
	err := handler.OnResponse([]byte("invalid"))
	require.NoError(t, err) // should not return error, just log warning

	_, ok := rm.Dequeue()
	assert.False(t, ok)
}

func TestRemoteMinerResponseHandlerOnFailure(t *testing.T) {
	nodeID := ids.GenerateTestNodeID()
	rm := &RemoteMiner{
		queue:        make([][]byte, 3),
		remoteNodeID: nodeID,
	}

	handler := &remoteMinerResponseHandler{rm: rm}
	err := handler.OnFailure()
	require.NoError(t, err) // should not return error, just log warning
}
