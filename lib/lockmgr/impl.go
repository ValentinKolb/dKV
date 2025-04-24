package lockmgr

import (
	"bytes"
	"fmt"
	"github.com/ValentinKolb/dKV/lib/store"
)

type logMgmImpl struct {
	store store.IStore
}

func NewLockManager(store store.IStore) ILockManager {
	return &logMgmImpl{
		store: store,
	}
}

func (lp *logMgmImpl) AcquireLock(key string, timeout uint64) (bool, []byte, error) {
	// Generate storage key (256 bit random value)
	ownerID, err := generateOwnerID()
	if err != nil {
		return false, nil, err
	}

	// Try to acquire the lock (by setting the value only if it doesn't exist - atomic CAS operation)
	err = lp.store.SetEIfUnset(key, ownerID, 0, timeout)
	if err != nil {
		fmt.Println("Error setting lock:", err)
		return false, nil, err
	}

	// Check if the lock was acquired
	value, found, err := lp.store.Get(key)
	if err != nil {
		return false, nil, err
	}

	// Return true if lock was acquired BY US
	if found && bytes.Equal(value, ownerID) {
		return true, ownerID, nil
	}
	// Return false if lock was acquired BY SOMEONE ELSE in the meantime
	return false, nil, nil
}

func (lp *logMgmImpl) ReleaseLock(key string, ownerID []byte) (bool, error) {
	// Check if the lock exists
	value, ok, err := lp.store.Get(key)
	if err != nil || !ok {
		return err == nil, err
	}

	// Check if the lock is owned by us
	if !bytes.Equal(ownerID, value) {
		return false, nil
	}

	// Release the lock
	err = lp.store.Delete(key)
	return err == nil, err
}
