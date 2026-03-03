package evm

import (
	"context"
	"sync"
	"time"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/coreth/network"
	"github.com/ava-labs/coreth/plugin/evm/message"
	"github.com/ava-labs/libevm/log"
)

const (
	remoteMiningQueueSize    = 10
	remoteMiningPollInterval = 2 * time.Second
)

// RemoteMiner periodically requests built containers from a remote mining node
// and stores them in a bounded queue for consumption by the block builder.
type RemoteMiner struct {
	network      network.Network
	codec        codec.Manager
	remoteNodeID ids.NodeID

	mu    sync.Mutex
	queue [][]byte
	head  int
	tail  int
	count int

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewRemoteMiner creates a new RemoteMiner that polls the given remote node.
func NewRemoteMiner(net network.Network, codec codec.Manager, remoteNodeID ids.NodeID) *RemoteMiner {
	return &RemoteMiner{
		network:      net,
		codec:        codec,
		remoteNodeID: remoteNodeID,
		queue:        make([][]byte, remoteMiningQueueSize),
		stopCh:       make(chan struct{}),
	}
}

// Start begins the periodic polling goroutine.
func (rm *RemoteMiner) Start() {
	rm.wg.Add(1)
	go rm.pollLoop()
}

// Stop signals the polling goroutine to stop and waits for it to finish.
func (rm *RemoteMiner) Stop() {
	close(rm.stopCh)
	rm.wg.Wait()
}

// Dequeue returns the next container from the queue, if available.
func (rm *RemoteMiner) Dequeue() ([]byte, bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.count == 0 {
		return nil, false
	}

	container := rm.queue[rm.head]
	rm.queue[rm.head] = nil // allow GC
	rm.head = (rm.head + 1) % len(rm.queue)
	rm.count--
	return container, true
}

func (rm *RemoteMiner) enqueue(container []byte) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.count == len(rm.queue) {
		// Drop oldest: advance head
		rm.queue[rm.head] = nil
		rm.head = (rm.head + 1) % len(rm.queue)
		rm.count--
	}

	rm.queue[rm.tail] = container
	rm.tail = (rm.tail + 1) % len(rm.queue)
	rm.count++
}

func (rm *RemoteMiner) pollLoop() {
	defer rm.wg.Done()

	ticker := time.NewTicker(remoteMiningPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rm.stopCh:
			return
		case <-ticker.C:
			rm.poll()
		}
	}
}

func (rm *RemoteMiner) poll() {
	request := message.RemoteContainerRequest{}
	requestBytes, err := message.RequestToBytes(rm.codec, request)
	if err != nil {
		log.Error("RemoteMiner: failed to marshal request", "err", err)
		return
	}

	handler := &remoteMinerResponseHandler{rm: rm}
	if err := rm.network.SendAppRequest(context.TODO(), rm.remoteNodeID, requestBytes, handler); err != nil {
		log.Warn("RemoteMiner: failed to send request", "nodeID", rm.remoteNodeID, "err", err)
	}
}

// remoteMinerResponseHandler implements message.ResponseHandler.
type remoteMinerResponseHandler struct {
	rm *RemoteMiner
}

func (h *remoteMinerResponseHandler) OnResponse(responseBytes []byte) error {
	var response message.RemoteContainerResponse
	if _, err := h.rm.codec.Unmarshal(responseBytes, &response); err != nil {
		log.Warn("RemoteMiner: failed to unmarshal response", "err", err)
		return nil
	}

	if len(response.Container) == 0 {
		log.Debug("RemoteMiner: received empty container (no block available)", "nodeID", h.rm.remoteNodeID)
		return nil
	}

	h.rm.enqueue(response.Container)
	log.Info("RemoteMiner: enqueued container", "nodeID", h.rm.remoteNodeID, "size", len(response.Container))
	return nil
}

func (h *remoteMinerResponseHandler) OnFailure() error {
	log.Warn("RemoteMiner: request failed", "nodeID", h.rm.remoteNodeID)
	return nil
}
