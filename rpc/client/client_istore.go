package client

import (
	"fmt"
	"github.com/ValentinKolb/dKV/lib/db"
	"github.com/ValentinKolb/dKV/lib/store"
	"github.com/ValentinKolb/dKV/rpc/common"
	"github.com/ValentinKolb/dKV/rpc/serializer"
	"github.com/ValentinKolb/dKV/rpc/transport"
)

// NewRPCStore creates a new RPC store
// The function takes a shard ID, a util, a transport and a serializer as parameters
// It returns a store.IStore and an error
func NewRPCStore(
	shardId uint64,
	config common.ClientConfig,
	transport transport.IRPCClientTransport,
	serializer serializer.IRPCSerializer,
) (store.IStore, error) {

	// Connect the transport
	err := transport.Connect(config)
	if err != nil {
		return nil, err
	}

	// Create a new RPC store
	s := rpcStore{
		rpcClientAdapter{
			shardId:    shardId,
			config:     config,
			transport:  transport,
			serializer: serializer,
		},
	}

	// Return the RPC store
	return &s, nil
}

type rpcStore struct {
	rpcClientAdapter
}

// --------------------------------------------------------------------------
// Interface Methods (docu see the store package in interface.go)
// --------------------------------------------------------------------------

func (i *rpcStore) Set(key string, value []byte) (err error) {
	req := common.NewSetRequest(key, value)
	_, err = invokeRPCRequest(i.shardId, req, i.transport, i.serializer)
	return err
}

func (i *rpcStore) SetE(key string, value []byte, expireIn, deleteIn uint64) (err error) {
	req := common.NewSetERequest(key, value, expireIn, deleteIn)
	_, err = invokeRPCRequest(i.shardId, req, i.transport, i.serializer)
	return err
}

func (i *rpcStore) SetEIfUnset(key string, value []byte, expireIn, deleteIn uint64) (err error) {
	req := common.NewSetEIfUnsetRequest(key, value, expireIn, deleteIn)
	_, err = invokeRPCRequest(i.shardId, req, i.transport, i.serializer)
	return err
}

func (i *rpcStore) Expire(key string) (err error) {
	req := common.NewExpireRequest(key)
	_, err = invokeRPCRequest(i.shardId, req, i.transport, i.serializer)
	return err
}

func (i *rpcStore) Delete(key string) (err error) {
	req := common.NewDeleteRequest(key)
	_, err = invokeRPCRequest(i.shardId, req, i.transport, i.serializer)
	return err
}

func (i *rpcStore) Get(key string) (value []byte, loaded bool, err error) {
	req := common.NewGetRequest(key)
	resp, err := invokeRPCRequest(i.shardId, req, i.transport, i.serializer)
	if err != nil {
		return nil, false, err
	}
	return resp.Value, resp.Ok, nil
}

func (i *rpcStore) Has(key string) (loaded bool, err error) {
	req := common.NewHasRequest(key)
	resp, err := invokeRPCRequest(i.shardId, req, i.transport, i.serializer)
	if err != nil {
		return false, err
	}
	return resp.Ok, nil
}

// GetDBInfo is not implemented for rpc
func (i *rpcStore) GetDBInfo() (info db.DatabaseInfo, err error) {
	return db.DatabaseInfo{}, fmt.Errorf("the GetDBInfo() method is not implemented in the rpc client adapter")
}
