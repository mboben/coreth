package message

import (
	"context"

	"github.com/ava-labs/avalanchego/ids"
)

var _ Request = (*RemoteContainerRequest)(nil)

// RemoteContainerRequest is a request to retrieve a built container (C-chain block)
// from a remote mining node.
type RemoteContainerRequest struct{}

func (r RemoteContainerRequest) String() string {
	return "RemoteContainerRequest()"
}

func (r RemoteContainerRequest) Handle(ctx context.Context, nodeID ids.NodeID, requestID uint32, handler RequestHandler) ([]byte, error) {
	return handler.HandleRemoteContainerRequest(ctx, nodeID, requestID, r)
}

// RemoteContainerResponse is a response to a RemoteContainerRequest
// containing the built container bytes.
type RemoteContainerResponse struct {
	Container []byte `serialize:"true"`
}
