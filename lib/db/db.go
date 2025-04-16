package db

import "io"

// --------------------------------------------------------------------------
// Helper Types
// --------------------------------------------------------------------------

type Implementation string

const (
	ImplMaple Implementation = "maple"
)

// Feature represents database features as bit flags
type Feature uint64

const (
	FeatureSet            Feature = 1 << iota // Support for Set operations
	FeatureSetE                               // Support for SetE operations
	FeatureSetEIfUnset                        // Support for SetEIfUnset operations
	FeatureGet                                // Support for Get operations
	FeatureExpire                             // Support for Expire operations
	FeatureDelete                             // Support for Delete operations
	FeatureHas                                // Support for Has operations
	FeatureSave                               // Support for Save operations
	FeatureLoad                               // Support for Load operations
	FeatureGarbageCollect                     // Support for GarbageCollect operations
)

func (f Feature) String() string {
	switch f {
	case FeatureSet:
		return "Set"
	case FeatureGet:
		return "Get"
	case FeatureSetE:
		return "SetE"
	case FeatureSetEIfUnset:
		return "SetEIfUnset"
	case FeatureDelete:
		return "Delete"
	case FeatureHas:
		return "Has"
	case FeatureSave:
		return "Save"
	case FeatureLoad:
		return "Load"
	case FeatureGarbageCollect:
		return "GarbageCollect"
	default:
		return "Unknown"
	}
}

type DatabaseInfo struct {
	SizeBytes         int            `json:"size_bytes"`
	DbType            Implementation `json:"db_type"`
	SupportedFeatures []Feature      `json:"supported_features"`
	Metadata          interface{}    `json:"metadata"`
}

// --------------------------------------------------------------------------
// Database Interface
// --------------------------------------------------------------------------

// KVDB defines an interface for key-value database implementations.
// It provides methods for basic operations like Set, Get, Delete, and various utility functions.
// Any implementation of this interface must manage keys in a consistent way.
// Implementations can vary in their feature support, which can be queried with SupportsFeature.
type KVDB interface {

	// --------------------------------------------------------------------------
	// Write Operations
	// --------------------------------------------------------------------------

	// Set inserts or updates an entry with the given key, value, and currentIndex.
	// If the key already exists, the old value should be overwritten.
	// The writeIndex parameter is used as a logical timestamp for the entry.
	Set(key string, value []byte, writeIndex uint64)

	// SetEIfUnset inserts an entry with the given key, value, and currentIndex.
	// If the key already exists, the old value is not updated.
	// The writeIndex parameter is used as a logical timestamp for the entry.
	// The expireIn parameter is used to set an expiration time for the entry, the entry is still findable after expiration with the Has() method.
	// The deleteIn parameter is used to set a deletion time for the entry, the entry is not findable after deletion.
	// Note: expireIn=0 and deleteIn=0 means no expiration or deletion. Setting expireIn=0 and deleteIn=N is equivalent to expireIn=N and deleteIn=N.
	SetEIfUnset(key string, value []byte, writeIndex uint64, expireIn, deleteIn uint64)

	// SetE inserts or updates an entry with the given key, value, timestamp and a ttl (time to live).
	// If the key already exists, the old value should be overwritten.
	// The writeIndex parameter is used as a logical timestamp for the entry.
	// The expireIn parameter is used to set an expiration time for the entry, the entry is still findable after expiration with the Has() method.
	// The deleteIn parameter is used to set a deletion time for the entry, the entry is not findable after deletion.
	// Note: expireIn=0 and deleteIn=0 means no expiration or deletion. Setting expireIn=0 and deleteIn=N is equivalent to expireIn=N and deleteIn=N.
	SetE(key string, value []byte, writeIndex uint64, expireIn, deleteIn uint64)

	// Expire marks the entry with the specified key as expired.
	// The entry is key findable with the Has() method.
	Expire(key string, writeIndex uint64)

	// Delete removes an entry with the specified key.
	// The key should be removed from the database and not be findable anymore.
	Delete(key string, writeIndex uint64)

	// --------------------------------------------------------------------------
	// Query Operations
	// --------------------------------------------------------------------------

	// Get retrieves the value for an exact key.
	// The boolean return value indicates whether a value for the key was found.
	Get(key string) (value []byte, loaded bool)

	// Has checks whether a key exists in the database.
	// This method should return true even if the value for the key is expired.
	Has(key string) (loaded bool)

	// --------------------------------------------------------------------------
	// Persistence Operations
	// --------------------------------------------------------------------------

	// Save persists the current state of the database to the provided io.Writer.
	Save(w io.Writer) (err error)

	// Load restores the database state data provided by an io.Reader.
	Load(r io.Reader) (err error)

	// --------------------------------------------------------------------------
	// Feature Support
	// --------------------------------------------------------------------------

	// SupportsFeature checks if the database implementation supports the specified feature.
	// Returns true if the feature is supported, false otherwise.
	// Multiple features can be checked at once using bitwise OR (|) operator.
	SupportsFeature(feature Feature) (ok bool)

	// GetInfo returns information about the database.
	GetInfo() (info DatabaseInfo)

	// --------------------------------------------------------------------------
	// Write Index Operations
	// --------------------------------------------------------------------------

	// SetWriteIdx sets the current index of the database only if the provided index is greater than the current index.
	SetWriteIdx(index uint64)

	// WriteIdx returns the current index of the database .
	WriteIdx() (index uint64)

	// Close closes the database.
	Close() (err error)
}
