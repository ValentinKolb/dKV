package client

import (
	"github.com/ValentinKolb/dKV/lib/lockmgr"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/serializer"
	"github.com/ValentinKolb/dKV/rpc/transport"
)

// NewRPCLockMgr creates a new RPC ILockManager
// The function takes a shard ID, a util, a transport and a serializer as parameters
// It returns a store.IStore and an error
func NewRPCLockMgr(
	shardId uint64,
	config common.ClientConfig,
	transport transport.IRPCClientTransport,
	serializer serializer.IRPCSerializer,
) (lockmgr.ILockManager, error) {

	// Connect the transport
	err := transport.Connect(config)
	if err != nil {
		return nil, err
	}

	// Create a new RPC store
	l := rpcLockMgr{
		rpcClientAdapter{
			shardId:    shardId,
			config:     config,
			transport:  transport,
			serializer: serializer,
		},
	}

	// Return the RPC store
	return &l, nil
}

type rpcLockMgr struct {
	rpcClientAdapter
}

// --------------------------------------------------------------------------
// Interface Methods (docu see the lockmgr package in interface.go)
// --------------------------------------------------------------------------

func (i *rpcLockMgr) AcquireLock(key string, timeout uint64) (ok bool, ownerID []byte, err error) {
	req := common.NewAcquireRequest(key, timeout)
	resp, err := invokeRPCRequest(i.shardId, req, i.transport, i.serializer)
	if err != nil {
		return false, nil, err
	}
	return resp.Ok, resp.Value, nil
}

func (i *rpcLockMgr) ReleaseLock(key string, ownerID []byte) (ok bool, err error) {
	req := common.NewReleaseRequest(key, ownerID)
	resp, err := invokeRPCRequest(i.shardId, req, i.transport, i.serializer)
	if err != nil {
		return false, err
	}
	return resp.Ok, nil
}
