package lstore

import (
	"github.com/ValentinKolb/dKV/lib/db"
	"github.com/ValentinKolb/dKV/lib/store"
	"sync/atomic"
)

type storeImpl struct {
	db    db.KVDB
	index atomic.Uint64
}

// NewLocalStore creates a new local store instance.
// This store implementation is not distributed and only works on a single node.
// This works by using the maple engine from the db package directly.
func NewLocalStore(factory store.DBFactory) store.IStore {
	return &storeImpl{
		db:    factory(),
		index: atomic.Uint64{},
	}
}

// incAndGetIndex increments the index and returns the new value.
// It is used to ensure that each write operation has a unique index.
//
// Thread-safety: This method is thread-safe since it uses atomic operations.
func (s *storeImpl) incAndGetIndex() uint64 {
	return s.index.Add(1)
}

// --------------------------------------------------------------------------
// Interface Methods (docu see store/interface.go)
// --------------------------------------------------------------------------

func (s *storeImpl) Set(key string, value []byte) error {
	if !s.db.SupportsFeature(db.FeatureSet) {
		return store.NewError(store.RetCUnsupportedOperation, "Set operation is not supported")
	}
	s.db.Set(key, value, s.incAndGetIndex())
	return nil
}

func (s *storeImpl) SetE(key string, value []byte, expireIn, deleteIn uint64) error {
	if !s.db.SupportsFeature(db.FeatureSetE) {
		return store.NewError(store.RetCUnsupportedOperation, "SetE operation is not supported")
	}
	s.db.SetE(key, value, s.incAndGetIndex(), expireIn, deleteIn)
	return nil
}

func (s *storeImpl) SetEIfUnset(key string, value []byte, expireIn, deleteIn uint64) error {
	if !s.db.SupportsFeature(db.FeatureSetEIfUnset) {
		return store.NewError(store.RetCUnsupportedOperation, "SetEIfUnset operation is not supported")
	}
	s.db.SetEIfUnset(key, value, s.incAndGetIndex(), expireIn, deleteIn)
	return nil
}

func (s *storeImpl) Expire(key string) error {
	if !s.db.SupportsFeature(db.FeatureExpire) {
		return store.NewError(store.RetCUnsupportedOperation, "Expire operation is not supported")
	}
	s.db.Expire(key, s.incAndGetIndex())
	return nil
}

func (s *storeImpl) Delete(key string) error {
	if !s.db.SupportsFeature(db.FeatureDelete) {
		return store.NewError(store.RetCUnsupportedOperation, "Delete operation is not supported")
	}
	s.db.Delete(key, s.incAndGetIndex())
	return nil
}

func (s *storeImpl) Get(key string) ([]byte, bool, error) {
	if !s.db.SupportsFeature(db.FeatureGet) {
		return nil, false, store.NewError(store.RetCUnsupportedOperation, "Get operation is not supported")
	}
	val, ok := s.db.Get(key)
	return val, ok, nil
}

func (s *storeImpl) Has(key string) (bool, error) {
	if !s.db.SupportsFeature(db.FeatureHas) {
		return false, store.NewError(store.RetCUnsupportedOperation, "Has operation is not supported")
	}
	return s.db.Has(key), nil
}

func (s *storeImpl) GetDBInfo() (db.DatabaseInfo, error) {
	return s.db.GetInfo(), nil
}
