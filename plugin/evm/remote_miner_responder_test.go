package evm

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteMinerResponderBasic(t *testing.T) {
	r := NewRemoteMinerResponder(3)

	// Dequeue from empty queue
	_, ok := r.Dequeue()
	assert.False(t, ok)

	// Enqueue and dequeue
	r.Enqueue([]byte("a"))
	r.Enqueue([]byte("b"))

	val, ok := r.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("a"), val)

	val, ok = r.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("b"), val)

	_, ok = r.Dequeue()
	assert.False(t, ok)
}

func TestRemoteMinerResponderDropOldest(t *testing.T) {
	r := NewRemoteMinerResponder(3)

	r.Enqueue([]byte("a"))
	r.Enqueue([]byte("b"))
	r.Enqueue([]byte("c"))

	// Queue is full, this should drop "a"
	r.Enqueue([]byte("d"))

	val, ok := r.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("b"), val)

	val, ok = r.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("c"), val)

	val, ok = r.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("d"), val)

	_, ok = r.Dequeue()
	assert.False(t, ok)
}

func TestRemoteMinerResponderWrapAround(t *testing.T) {
	r := NewRemoteMinerResponder(3)

	// Fill and drain to advance head/tail
	r.Enqueue([]byte("x"))
	r.Enqueue([]byte("y"))
	r.Dequeue()
	r.Dequeue()

	// Now head=2, tail=2, count=0
	r.Enqueue([]byte("a"))
	r.Enqueue([]byte("b"))
	r.Enqueue([]byte("c"))

	val, ok := r.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("a"), val)

	val, ok = r.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("b"), val)

	val, ok = r.Dequeue()
	require.True(t, ok)
	assert.Equal(t, []byte("c"), val)
}

func TestRemoteMinerResponderConcurrency(t *testing.T) {
	r := NewRemoteMinerResponder(100)
	var wg sync.WaitGroup

	// Concurrent enqueues
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r.Enqueue([]byte{byte(i)})
		}(i)
	}

	// Concurrent dequeues
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Dequeue()
		}()
	}

	wg.Wait()
	// Just verify no panic/deadlock occurred
}
