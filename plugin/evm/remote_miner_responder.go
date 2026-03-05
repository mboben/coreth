package evm

import "sync"

const remoteMinerResponderQueueSize = 10

// RemoteMinerResponder stores RLP-encoded blocks pushed by an external miner
// and serves them to requester nodes via HandleRemoteContainerRequest.
type RemoteMinerResponder struct {
	mu    sync.Mutex
	queue [][]byte
	head  int
	tail  int
	count int
}

// NewRemoteMinerResponder creates a new responder with a bounded queue.
func NewRemoteMinerResponder(size int) *RemoteMinerResponder {
	return &RemoteMinerResponder{
		queue: make([][]byte, size),
	}
}

// Enqueue adds a container to the queue. If the queue is full, the oldest entry is dropped.
func (r *RemoteMinerResponder) Enqueue(container []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.count == len(r.queue) {
		// Drop oldest: advance head
		r.queue[r.head] = nil
		r.head = (r.head + 1) % len(r.queue)
		r.count--
	}

	r.queue[r.tail] = container
	r.tail = (r.tail + 1) % len(r.queue)
	r.count++
}

// Dequeue returns the next container from the queue, if available.
func (r *RemoteMinerResponder) Dequeue() ([]byte, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.count == 0 {
		return nil, false
	}

	container := r.queue[r.head]
	r.queue[r.head] = nil // allow GC
	r.head = (r.head + 1) % len(r.queue)
	r.count--
	return container, true
}
