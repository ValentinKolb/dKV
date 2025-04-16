package lockmgr

// ILockManager defines the interface for a lockmgr provider.
type ILockManager interface {
	// AcquireLock acquires a lockmgr for the given key with an optional timeout.
	// Return a boolean indicating whether the lockmgr was acquired, an owner ID, and an error if any.
	AcquireLock(key string, timeout uint64) (ok bool, ownerID []byte, err error)

	// ReleaseLock releases the lockmgr for the given key.
	// Return a boolean indicating whether the lockmgr was released, and an error if any.
	// The method will also return True is the lockmgr did not exist.
	ReleaseLock(key string, ownerID []byte) (ok bool, err error)
}
