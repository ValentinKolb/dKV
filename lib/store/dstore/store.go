package dstore

import (
	"context"
	"errors"
	"fmt"
	"github.com/ValentinKolb/dKV/lib/db"
	"github.com/ValentinKolb/dKV/lib/store"
	"github.com/ValentinKolb/dKV/lib/store/dstore/internal"
	"github.com/lni/dragonboat/v4/logger"
	"time"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/client"
)

var (
	retries = 5
	log     = logger.GetLogger("store")
)

// storeImpl is the concrete implementation of the RemoteStore interface.
// It encapsulates a Dragonboat NodeHost which is used to communicate with the state machine.
type storeImpl struct {
	nh      *dragonboat.NodeHost
	shardID uint64
	cs      *client.Session
	timeout time.Duration
}

// NewDistributedStore creates a new distributed store instance which uses raft consensus to ensure strict linearizability
// across multiple nodes.
func NewDistributedStore(nh *dragonboat.NodeHost, shardID uint64, timeout time.Duration) store.IStore {
	cs := nh.GetNoOPSession(shardID)
	return &storeImpl{
		nh:      nh,
		shardID: shardID,
		cs:      cs,
		timeout: timeout,
	}
}

// --------------------------------------------------------------------------
// Internal write and read operations (used by interface methods)
// --------------------------------------------------------------------------

// write marshals a Command as JSON and sends it via SyncPropose.
// It returns a *RemoteStoreError if an error occurs, or nil on success.
func (s *storeImpl) write(cmd internal.Command) error {
	for i := 0; i < retries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)

		res, err := s.nh.SyncPropose(ctx, s.cs, cmd.Serialize())
		cancel()

		// Check for system busy errors
		if errors.Is(err, dragonboat.ErrSystemBusy) {
			log.Infof("SyncPropose: System busy, retrying (%d/%d)...", i+1, retries)
			time.Sleep(s.timeout / 10)
			continue
		}

		if err != nil {
			return store.NewError(store.RetCInternalError, err.Error())
		}
		if res.Value != uint64(store.RetCSuccess) {
			return store.NewError(store.RetCode(res.Value), string(res.Data))
		}
		return nil
	}
	return store.NewError(store.RetCInternalError, "timeout")
}

// read is a generic helper function queries the statemachine
// and attempts to convert the response into the expected type R.
//
// This function uses the SyncRead function (dragenboat) by default to Query the state machine.
// If linearizability is not required, the stale parameter can be set to true to use the faster StaleRead function.
//
// Is the read operation fails due to a system busy error, the function retries up to 5 times.
//
// It returns the response of type R and a error (nil on success).
func read[R any](r *storeImpl, q internal.Query, stale bool) (R, error) {
	var zero R
	for i := 0; i < retries; i++ {

		var res interface{}
		var err error

		// Query the standmaschine, use StaleRead if stale is set otherwise use SyncRead (default)
		if stale {
			res, err = r.nh.StaleRead(r.shardID, q)
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
			res, err = r.nh.SyncRead(ctx, r.shardID, q)
			cancel()
		}

		// Check for system busy errors
		if errors.Is(err, dragonboat.ErrSystemBusy) {
			log.Infof("SyncRead: System busy, retrying (%d/%d)...", i+1, retries)
			time.Sleep(r.timeout / 10)
			continue
		}

		if err != nil {
			var rse error
			if errors.As(err, &rse) {
				return zero, rse
			}
			return zero, store.NewError(store.RetCInternalError, err.Error())
		}

		// The state machine is expected to return the response in the expected type R.
		casted, ok := res.(R)
		if !ok {
			return zero, store.NewError(store.RetCInternalError,
				fmt.Sprintf("unexpected type: received %T, expected %T", res, zero))
		}
		return casted, nil
	}
	return zero, store.NewError(store.RetCInternalError, "timeout")
}

// --------------------------------------------------------------------------
// Interface Methods (docs see store/interface.go)
// --------------------------------------------------------------------------

func (s *storeImpl) Set(key string, value []byte) error {
	return s.write(internal.Command{
		Type:  internal.CommandTSet,
		Key:   key,
		Value: value,
	})
}

func (s *storeImpl) SetE(key string, value []byte, expireIn, deleteIn uint64) error {
	return s.write(internal.Command{
		Type:     internal.CommandTSetE,
		Key:      key,
		Value:    value,
		ExpireIn: expireIn,
		DeleteIn: deleteIn,
	})
}

func (s *storeImpl) SetEIfUnset(key string, value []byte, expireIn, deleteIn uint64) error {
	return s.write(internal.Command{
		Type:     internal.CommandTSetIfUnset,
		Key:      key,
		Value:    value,
		ExpireIn: expireIn,
		DeleteIn: deleteIn,
	})
}

func (s *storeImpl) Expire(key string) error {
	return s.write(internal.Command{
		Type: internal.CommandTExpire,
		Key:  key,
	})
}

func (s *storeImpl) Delete(key string) error {
	return s.write(
		internal.Command{
			Type: internal.CommandTDelete,
			Key:  key,
		},
	)
}

func (s *storeImpl) Get(key string) ([]byte, bool, error) {
	res, err := read[internal.QueryResult](s, internal.Query{
		Type: internal.QueryTGet,
		Key:  key,
	}, false)
	if err != nil {
		return nil, false, err
	}
	return res.Value, res.Ok, nil
}

func (s *storeImpl) Has(key string) (bool, error) {
	return read[bool](s, internal.Query{
		Type: internal.QueryTHas,
		Key:  key,
	}, false)
}

func (s *storeImpl) GetDBInfo() (db.DatabaseInfo, error) {
	return read[db.DatabaseInfo](
		s,
		internal.Query{
			Type: internal.QueryTGetDBInfo,
		},
		true, // Note: allow for stale reads
	)
}
