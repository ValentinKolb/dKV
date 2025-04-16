package server

import (
	"fmt"
	"github.com/ValentinKolb/dKV/lib/store"
	"github.com/ValentinKolb/dKV/rpc/common"
)

func NewIStoreServerAdapter() IRPCServerAdapter {
	return &iStoreServerAdapterImpl{}
}

type iStoreServerAdapterImpl struct{}

func (adapter *iStoreServerAdapterImpl) Handle(req *common.Message, store store.IStore) *common.Message {
	// Check for nil store
	if store == nil {
		return common.NewErrorResponse("handler: store is nil")
	}

	// Handle different message types
	switch req.MsgType {
	case common.MsgTKVSet:
		err := store.Set(req.Key, req.Value)
		return common.NewSetResponse(err)
	case common.MsgTKVSetE:
		err := store.SetE(req.Key, req.Value, req.ExpireIn, req.DeleteIn)
		return common.NewSetEResponse(err)
	case common.MsgTKVSetEIfUnset:
		err := store.SetEIfUnset(req.Key, req.Value, req.ExpireIn, req.DeleteIn)
		return common.NewSetEIfUnsetResponse(err)
	case common.MsgTKVExpire:
		err := store.Expire(req.Key)
		return common.NewExpireResponse(err)
	case common.MsgTKVDelete:
		err := store.Delete(req.Key)
		return common.NewDeleteResponse(err)
	case common.MsgTKVGet:
		val, ok, err := store.Get(req.Key)
		return common.NewGetResponse(val, ok, err)
	case common.MsgTKVHas:
		ok, err := store.Has(req.Key)
		return common.NewHasResponse(ok, err)
	default:
		return common.NewErrorResponse(
			fmt.Sprintf("RPC IStoreAdapter - Unsuported message type: %s", req.MsgType),
		)
	}
}

type MessageHandler func(req *common.Message) (resp *common.Message)

type RegisterMessageHandler func(handler MessageHandler)
