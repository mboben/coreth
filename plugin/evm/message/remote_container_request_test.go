package message

import (
	"context"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteContainerRequestMarshalRoundtrip(t *testing.T) {
	request := RemoteContainerRequest{}

	// Marshal as concrete type
	requestBytes, err := Codec.Marshal(Version, request)
	require.NoError(t, err)
	require.NotEmpty(t, requestBytes)

	// Unmarshal as concrete type
	var parsed RemoteContainerRequest
	_, err = Codec.Unmarshal(requestBytes, &parsed)
	require.NoError(t, err)
}

func TestRemoteContainerRequestToBytes(t *testing.T) {
	request := RemoteContainerRequest{}

	// RequestToBytes marshals as Request interface (with type prefix)
	requestBytes, err := RequestToBytes(Codec, request)
	require.NoError(t, err)
	require.NotEmpty(t, requestBytes)

	// Should unmarshal back as Request interface
	var parsed Request
	_, err = Codec.Unmarshal(requestBytes, &parsed)
	require.NoError(t, err)

	_, ok := parsed.(RemoteContainerRequest)
	assert.True(t, ok)
}

func TestRemoteContainerResponseMarshalRoundtrip(t *testing.T) {
	container := []byte("test-container-data")
	response := RemoteContainerResponse{Container: container}
	responseBytes, err := Codec.Marshal(Version, &response)
	require.NoError(t, err)
	require.NotEmpty(t, responseBytes)

	var parsed RemoteContainerResponse
	_, err = Codec.Unmarshal(responseBytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, container, parsed.Container)
}

type mockRemoteContainerHandler struct {
	called bool
}

func (m *mockRemoteContainerHandler) HandleLeafsRequest(_ context.Context, _ ids.NodeID, _ uint32, _ LeafsRequest) ([]byte, error) {
	return nil, nil
}

func (m *mockRemoteContainerHandler) HandleBlockRequest(_ context.Context, _ ids.NodeID, _ uint32, _ BlockRequest) ([]byte, error) {
	return nil, nil
}

func (m *mockRemoteContainerHandler) HandleCodeRequest(_ context.Context, _ ids.NodeID, _ uint32, _ CodeRequest) ([]byte, error) {
	return nil, nil
}

func (m *mockRemoteContainerHandler) HandleRemoteContainerRequest(_ context.Context, _ ids.NodeID, _ uint32, _ RemoteContainerRequest) ([]byte, error) {
	m.called = true
	return []byte("response"), nil
}

func TestRemoteContainerRequestHandle(t *testing.T) {
	handler := &mockRemoteContainerHandler{}
	request := RemoteContainerRequest{}
	nodeID := ids.GenerateTestNodeID()

	result, err := request.Handle(context.Background(), nodeID, 1, handler)
	require.NoError(t, err)
	assert.True(t, handler.called)
	assert.Equal(t, []byte("response"), result)
}
