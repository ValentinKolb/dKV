package client

import (
	"fmt"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/serializer"
	"github.com/ValentinKolb/dKV/rpc/transport"
	"github.com/lni/dragonboat/v4/logger"
)

var (
	Logger = logger.GetLogger("rpc")
)

// rpcClientAdapter is a struct that stores all data needed for an implementation if an RPC client
// Used by the RPCStore and RPCLockMgr with composition pattern
type rpcClientAdapter struct {
	shardId    uint64
	config     common.ClientConfig
	transport  transport.IRPCClientTransport
	serializer serializer.IRPCSerializer
}

// invokeRPCRequest is a helper function used for all RPC Clients to send requests
// It takes a shard ID, a request message, a transport layer and a serializer as parameters
// It returns a response message and an error if any occurs
// This method also checks if the response is an error response and if the type of the response is the expected type
func invokeRPCRequest(shardId uint64, req *common.Message, transport transport.IRPCClientTransport, serializer serializer.IRPCSerializer) (*common.Message, error) {
	// Serialize the request
	reqBytes, err := serializer.Serialize(*req)
	if err != nil {
		return nil, err
	}

	// Send the handler
	respBytes, err := transport.Send(shardId, reqBytes)
	if err != nil {
		return nil, err
	}

	// Deserialize the response
	resp := &common.Message{}
	err = serializer.Deserialize(respBytes, resp)
	if err != nil {
		return nil, fmt.Errorf("RPC IStoreAdapter - Error: %s", err)
	}

	// Check if the response is an error response
	if resp.MsgType == common.MsgTError || resp.Err != "" {
		return nil, fmt.Errorf("RPC IStoreAdapter - Error: %s", resp.Err)
	}

	// Check if the type of the response is the expected type
	if resp.MsgType != req.MsgType {
		return nil, fmt.Errorf("RPC IStoreAdapter - Unexpected message type: %s, exected %s", resp.MsgType, req.MsgType)
	}

	// Return the response
	return resp, nil
}
