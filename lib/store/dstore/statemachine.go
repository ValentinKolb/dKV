package dstore

import (
	"fmt"
	"github.com/ValentinKolb/dKV/lib/db"
	"github.com/ValentinKolb/dKV/lib/store"
	"github.com/ValentinKolb/dKV/lib/store/dstore/internal"
	sm "github.com/lni/dragonboat/v4/statemachine"
	"io"
	"time"
)

// --------------------------------------------------------------------------
// State Machine Implementation
// --------------------------------------------------------------------------

// KVStateMachine is a state machine implementation for Dragonboat RAFT
type KVStateMachine struct {
	replicaID uint64
	shardID   uint64
	database  db.KVDB // the actual dataStorage
}

// CreateStateMaschineFactory returns a function that can be used by dragenboat to create a new standmaschine for a node host
// The factory pattern is used to enable the caller to pass an interchangeable dbFactory
func CreateStateMaschineFactory(dbFactory store.DBFactory) func(shardID uint64, replicaID uint64) sm.IConcurrentStateMachine {
	return func(shardID uint64, replicaID uint64) sm.IConcurrentStateMachine {
		return &KVStateMachine{
			replicaID: replicaID,
			shardID:   shardID,
			database:  dbFactory(),
		}
	}
}

// Lookup handles read-only queries by mapping each Query operation to the corresponding KVDB method.
func (fsm *KVStateMachine) Lookup(itf interface{}) (interface{}, error) {

	// try to parse Query into Query struct
	q, ok := itf.(internal.Query)
	if !ok {
		return nil, store.NewError(store.RetCInternalError, fmt.Sprintf("invalid Query type: %T", itf))
	}

	// Handle different Query types
	switch q.Type {
	case internal.QueryTGet:
		if !fsm.database.SupportsFeature(db.FeatureGet) {
			return nil, store.NewError(store.RetCUnsupportedOperation, "Get operation is not supported")
		}
		val, ok := fsm.database.Get(q.Key)
		return internal.QueryResult{
			Value: val,
			Ok:    ok,
		}, nil
	case internal.QueryTHas:
		if !fsm.database.SupportsFeature(db.FeatureHas) {
			return nil, store.NewError(store.RetCUnsupportedOperation, "Has operation is not supported")
		}
		return fsm.database.Has(q.Key), nil
	case internal.QueryTGetDBInfo:
		return fsm.database.GetInfo(), nil
	default:
		return nil, store.NewError(store.RetCInvalidOperation, fmt.Sprintf("unknown Query operation: %d", q.Type))
	}
}

// Update handles write commands on the KVDB instance
// All write operations are serialized into []byte and are accessible via the entries struct
func (fsm *KVStateMachine) Update(entries []sm.Entry) ([]sm.Entry, error) {

	// Nothing to do
	if len(entries) == 0 {
		return entries, nil
	}

	// Stats
	start := time.Now()

	for idx, e := range entries {
		// Handle each entry
		if len(e.Cmd) == 0 {
			entries[idx].Result = sm.Result{Value: uint64(store.RetCInvalidOperation), Data: []byte("empty command ignored")}
		}
		// Deserialize the command
		cmd := internal.Command{}
		err := cmd.Deserialize(e.Cmd)
		if err != nil {
			entries[idx].Result = sm.Result{Value: uint64(store.RetCInternalError), Data: []byte(fmt.Sprintf("failed to deserialize command: %v", err))}
		}

		// Check if the db supports the operation
		feat, err := cmd.Type.ToDBFeature()
		if err != nil {
			entries[idx].Result = sm.Result{
				Value: uint64(store.RetCInvalidOperation),
				Data:  []byte(fmt.Sprintf("unknown Command operation: %s", cmd.Type)),
			}
			continue
		}
		if !fsm.database.SupportsFeature(feat) {
			entries[idx].Result = sm.Result{
				Value: uint64(store.RetCUnsupportedOperation),
				Data:  []byte(fmt.Sprintf("%s operation is not suported", cmd.Type)),
			}
			continue
		}

		switch cmd.Type {
		case internal.CommandTSet:
			fsm.database.Set(cmd.Key, cmd.Value, e.Index)
			entries[idx].Result = sm.Result{
				Value: uint64(store.RetCSuccess),
				Data:  []byte(fmt.Sprintf("set: key=%s", cmd.Key)),
			}
		case internal.CommandTSetE:
			fsm.database.SetE(cmd.Key, cmd.Value, e.Index, cmd.ExpireIn, cmd.DeleteIn)
			entries[idx].Result = sm.Result{
				Value: uint64(store.RetCSuccess),
				Data:  []byte(fmt.Sprintf("set: key=%s", cmd.Key)),
			}
		case internal.CommandTSetIfUnset:
			fsm.database.SetEIfUnset(cmd.Key, cmd.Value, e.Index, cmd.ExpireIn, cmd.DeleteIn)
			entries[idx].Result = sm.Result{
				Value: uint64(store.RetCSuccess),
				Data:  []byte(fmt.Sprintf("setIfUnset: key=%s", cmd.Key)),
			}
		case internal.CommandTExpire:
			fsm.database.Expire(cmd.Key, e.Index)
			entries[idx].Result = sm.Result{
				Value: uint64(store.RetCSuccess),
				Data:  []byte(fmt.Sprintf("expired key=%s", cmd.Key)),
			}
		case internal.CommandTDelete:
			fsm.database.Delete(cmd.Key, e.Index)
			entries[idx].Result = sm.Result{
				Value: uint64(store.RetCSuccess),
				Data:  []byte(fmt.Sprintf("deleted key=%s", cmd.Key)),
			}
		default:
			entries[idx].Result = sm.Result{
				Value: uint64(store.RetCInvalidOperation),
				Data:  []byte(fmt.Sprintf("unknown Command operation: %s", cmd.Type)),
			}
		}
	}

	// Log if the update took long
	if elapsed := time.Since(start); elapsed > time.Millisecond {
		log.Infof("Statemashine took long to update. Batch updated %d entries, took %.2fms:", len(entries), float64(elapsed)/float64(time.Millisecond))
	}
	return entries, nil
}

// PrepareSnapshot is not used. We don't need to prepare anything since we use fuzzy snapshotting
func (fsm *KVStateMachine) PrepareSnapshot() (interface{}, error) {
	return nil, nil
}

// SaveSnapshot saves a fuzzy db snapshot to the writer
func (fsm *KVStateMachine) SaveSnapshot(_ interface{}, writer io.Writer, _ sm.ISnapshotFileCollection, _ <-chan struct{}) error {
	if !fsm.database.SupportsFeature(db.FeatureSave) {
		return fmt.Errorf("the used KVDB implemantation does not supports Save() operations")
	}
	return fsm.database.Save(writer)
}

// RecoverFromSnapshot delegates snapshot recovery to the Bforge layer.
func (fsm *KVStateMachine) RecoverFromSnapshot(r io.Reader, _ []sm.SnapshotFile, _ <-chan struct{}) error {
	if !fsm.database.SupportsFeature(db.FeatureLoad) {
		return fmt.Errorf("the used KVDB implemantation does not supports Load() operations")
	}
	return fsm.database.Load(r)
}

// Close performs any necessary cleanup.
func (fsm *KVStateMachine) Close() error {
	return fsm.database.Close()
}
