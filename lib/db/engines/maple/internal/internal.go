package internal

import (
	"fmt"
	"github.com/ValentinKolb/dKV/lib/db/util"
	"github.com/puzpuzpuz/xsync/v3"
)

// --------------------------------------------------------------------------
// Event Types are used to signal changes in the database state
// --------------------------------------------------------------------------

type EventType int

const (
	EventTWrite EventType = iota
	EventTDelete
)

func (e EventType) String() string {
	switch e {
	case EventTWrite:
		return "Write"
	case EventTDelete:
		return "Delete"
	default:
		return "Unknown"
	}
}

type Event struct {
	Type EventType
	Key  util.UintKey
}

func (e Event) String() string {
	return fmt.Sprintf("Event{Type: %s, Key: %d}", e.Type, e.Key)
}

// --------------------------------------------------------------------------
// Entry Type (key-value pair with metadata)
// --------------------------------------------------------------------------

// Entry stores a key-value pair with metadata
type Entry struct {
	Value    []byte // Priority data
	ExpireAt uint64 // Priority expiration timestamp
	DeleteAt uint64 // Deletion timestamp
	Index    uint64 // Current Index when this entry was created/updated
}

// TTLInfo returns whether the entry is expired and whether the entry is deleted (at the given write index)
func (e Entry) TTLInfo(writeIdx uint64) (bool, bool) {
	var (
		isExpired = e.ExpireAt != 0 && writeIdx >= e.ExpireAt
		isDeleted = e.DeleteAt != 0 && writeIdx >= e.DeleteAt
	)

	return isExpired, isDeleted
}

// --------------------------------------------------------------------------
// Shard Type (partition of the database)
// --------------------------------------------------------------------------

// Shard represents a partition of the database
// Each shard has its own independent lockmgr and maps
type Shard struct {
	Data       *xsync.MapOf[util.UintKey, Entry] // Map of active key-value entries
	ExpireHeap *util.MapHeap
	DeleteHeap *util.MapHeap
	Events     *util.LockFreeMPSC[Event]
}

// NewShard creates a new shard with the provided hash function
func NewShard(hasher func(util.UintKey, uint64) uint64) *Shard {
	return &Shard{
		Data:       xsync.NewMapOfWithHasher[util.UintKey, Entry](hasher),
		ExpireHeap: util.NewMapHeap(),
		DeleteHeap: util.NewMapHeap(),
		Events:     util.NewLockFreeMPSC[Event](), // this channel is closed to stop the gc per shard
	}
}

// GetShard returns the appropriate shard for a given key
//
// Thread-safety: This method is thread-safe and can be called concurrently.
func GetShard[T any](key util.UintKey, shards []*T) *T {
	// Shift right by 7 bits to use higher-quality bits for distribution
	shiftedKey := uint64(key) >> 7
	shardPos := shiftedKey % uint64(len(shards))
	return shards[shardPos]
}
