package server

import (
	"fmt"
	"github.com/ValentinKolb/dKV/lib/lockmgr"
	"github.com/ValentinKolb/dKV/lib/store"
	"github.com/ValentinKolb/dKV/rpc/common"
)

func NewLockManagerServerAdapter() IRPCServerAdapter {
	return &lockMgrServerServerAdapter{}
}

type lockMgrServerServerAdapter struct{}

func (adapter *lockMgrServerServerAdapter) Handle(req *common.Message, store store.IStore) (resp *common.Message) {

	// Check for nil store
	if store == nil {
		return common.NewErrorResponse("handler: store is nil")
	}

	// Create lock manager
	locks := lockmgr.NewLockManager(store)

	// Handle different message types
	switch req.MsgType {
	case common.MsgTLCKAcquire:
		ok, ownerID, err := locks.AcquireLock(req.Key, req.DeleteIn)
		return common.NewAcquireResponse(ok, ownerID, err)
	case common.MsgTLCKRelease:
		ok, err := locks.ReleaseLock(req.Key, req.Value)
		return common.NewReleaseResponse(ok, err)
	default:
		return common.NewErrorResponse(fmt.Sprintf("RPC LockManagerAdapter - Unsuported message type: %s", req.MsgType))
	}
}
