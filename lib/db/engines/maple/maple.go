package maple

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/ValentinKolb/dKV/lib/db"
	"github.com/ValentinKolb/dKV/lib/db/engines/maple/internal"
	"github.com/ValentinKolb/dKV/lib/db/util"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// --------------------------------------------------------------------------
// Constants
// --------------------------------------------------------------------------

// Constants for database behavior and structure
const (
	magicNum          = "MAPLEDB\x00"          // File format identifier
	mapleVersion      = 3                      // Database version
	defaultGCInterval = 100 * time.Millisecond // Default interval between GC runs
)

// --------------------------------------------------------------------------
// Core Maple database structure
// --------------------------------------------------------------------------

// mapleImpl implements a high-performance database with sharded data
type mapleImpl struct {
	numShards int               // Number of shards
	seed      uint64            // Seed for hash function
	shards    []*internal.Shard // Array of shards
	currIndex atomic.Uint64     // Current logical timestamp (for TTLInfo)

	// garbage collection
	gcInterval  time.Duration
	gcIsRunning atomic.Bool
}

// DBOptions configures the mapleImpl behavior during initialization
type DBOptions struct {
	NumShards  int           // Number of shards (nil = auto)
	GCInterval time.Duration // Time between GC runs (0 = use default: 1 sec)
}

// DefaultOptions returns the default mapleImpl options
func DefaultOptions() *DBOptions {
	return &DBOptions{
		NumShards:  runtime.NumCPU(),  // Auto-determine based on CPU count
		GCInterval: defaultGCInterval, // Default GC interval
	}
}

// --------------------------------------------------------------------------
// Initialization and Setup
// --------------------------------------------------------------------------

// NewMapleDB creates a new MapleDB instance with the specified options (optional)
//
// Thread-safety: This function is not thread-safe and should only be called once
// during initialization.
func NewMapleDB(opts *DBOptions) db.KVDB {

	// Generate default options if not provided
	if opts == nil {
		opts = DefaultOptions()
	}

	// Generate a seed for this mapleImpl instance
	seed := util.GenerateSeed()
	hasher := createIdentityHasher()

	// Create shards
	shards := make([]*internal.Shard, opts.NumShards)
	for i := 0; i < opts.NumShards; i++ {
		shards[i] = internal.NewShard(hasher)
	}

	newDB := &mapleImpl{
		numShards:  opts.NumShards,
		seed:       seed,
		shards:     shards,
		gcInterval: opts.GCInterval,
	}

	// Initialize atomic values
	newDB.currIndex.Store(0)
	newDB.gcIsRunning.Store(false)

	// start garbage collection
	newDB.startGC()

	return newDB
}

// --------------------------------------------------------------------------
// Hash Helper Functions
// --------------------------------------------------------------------------

// StringToUint64 converts a string to a util.UintKey with hashing
// and applies the mapleImpl seed to ensure uniqueness between mapleImpl instances
//
// Thread-safety: This method is thread-safe and can be called concurrently.
func (maple *mapleImpl) StringToUint64(s string) util.UintKey {
	return util.HashString(s, maple.seed)
}

// createIdentityHasher creates a hash function that combines a key with a seed
func createIdentityHasher() func(util.UintKey, uint64) uint64 {
	return func(key util.UintKey, mapSeed uint64) uint64 {
		return uint64(key) ^ mapSeed
	}
}

// --------------------------------------------------------------------------
// Core KVDB Interface Methods - Write Operations
// --------------------------------------------------------------------------

// Set inserts or updates an entry with the given key, value, and currentIndex.
// If the key already exists, the old value is overwritten.
// The writeIndex parameter is used as a logical timestamp for the entry.
//
// Thread-safety: This method is thread-safe and can be called concurrently.
func (maple *mapleImpl) Set(key string, value []byte, writeIdx uint64) {
	maple.compute(key, value, writeIdx, 0, 0, func(new, _ internal.Entry, _ bool) (internal.Entry, bool) {
		return new, false
	})
}

// SetE stores a value for a key with an expiration time.
// If the key already exists, the old value, old expireIn and old deleteIn are overwritten.
//
//   - key: the key to store
//   - value: the value to store
//   - writeIndex: the current logical index
//   - expireIn: when the value should expire (relative to writeIndex) (0 = no expiration, the key can still be found with the Has() method)
//   - deleteIn: when the key and value should be deleted (relative to writeIndex) (0 = no expiration)
//
// Note: expireIn=0 or deleteIn=0 means no expiration or deletion. Setting expireIn=0 and deleteIn=N is equivalent to expireIn=N and deleteAt=N because any deleted entry is also expired.
//
// Thread-safety: This method is thread-safe and can be called concurrently.
func (maple *mapleImpl) SetE(key string, value []byte, writeIndex uint64, expireIn, deleteIn uint64) {
	maple.compute(key, value, writeIndex, expireIn, deleteIn, func(new, _ internal.Entry, _ bool) (internal.Entry, bool) {
		return new, false
	})
}

// SetEIfUnset inserts an entry with the given key, value, and currentIndex.
// If the key already exists, the old value is not updated.
//
//   - key: the key to store
//   - value: the value to store
//   - writeIndex: the current logical index
//   - expireIn: when the value should expire (relative to writeIndex) (0 = no expiration, the key can still be found with the Has() method)
//   - deleteIn: when the key and value should be deleted (relative to writeIndex) (0 = no expiration)
//
// Note: expireIn=0 or deleteIn=0 means no expiration or deletion. Setting expireIn=0 and deleteIn=N is equivalent to expireAt=N and deleteAt=N because any deleted entry is also expired.
//
// Thread-safety: This method is thread-safe and can be called concurrently.
func (maple *mapleImpl) SetEIfUnset(key string, value []byte, writeIndex uint64, expireIn, deleteIn uint64) {
	maple.compute(key, value, writeIndex, expireIn, deleteIn, func(new, old internal.Entry, loaded bool) (internal.Entry, bool) {
		if loaded {
			return old, false
		} else {
			return new, false
		}
	})
}

// compute is a helper method for shared implementation between Set, SetE, SetEIfUnset, Extend and ExtendIf.
// It handles the actual storage of the key-value pair, GC registration and ignoring stale writes.
//
// To use this function, you need to provide a function that takes the new and old entry (and whether old was loaded i.e. existed) as parameters.
// This function can modify the new entry and return it. The bool return value indicates whether the entry should be deleted (if it exists).
//
// Thread-safety: This function uses linearizability control to ensure thread-safety.
func (maple *mapleImpl) compute(key string, value []byte, writeIndex uint64, expireIn, deleteIn uint64, fn func(new, old internal.Entry, loaded bool) (entry internal.Entry, delete bool)) {

	// update the current index
	maple.SetWriteIdx(writeIndex)

	intKey := maple.StringToUint64(key)
	shard := internal.GetShard(intKey, maple.shards)

	// Copy value to prevent memory corruption
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	// Calculate expiration time
	var (
		expireAt uint64
		deleteAt uint64
	)
	if expireIn > 0 {
		expireAt = writeIndex + expireIn
	}
	if deleteIn > 0 {
		deleteAt = writeIndex + deleteIn
	}

	// add entry to gc after the entry is created
	var event *internal.Event

	// Use Compute for atomic conditional update
	shard.Data.Compute(intKey, func(oldEntry internal.Entry, oldEntryExists bool) (internal.Entry, bool) {
		// If the key doesn't exist or the current index is newer or equal, update
		if !oldEntryExists || writeIndex >= oldEntry.Index {

			// test if entry is logically deleted or expired
			// we do this so that the fn only ever sees a consistent view of the entry
			var loaded = oldEntryExists
			if oldEntryExists {
				isExpired, isDeleted := oldEntry.TTLInfo(writeIndex)

				// if the entry is logically deleted, we need to set the loaded flag to false
				loaded = !isDeleted

				// if the entry is expired, we need to set the value to nil
				if isExpired {
					oldEntry.Value = nil
					oldEntry.ExpireAt = writeIndex
				}
			}

			// compute the new entry
			entry, del := fn(internal.Entry{
				Value:    valueCopy,
				ExpireAt: expireAt,
				DeleteAt: deleteAt,
				Index:    writeIndex,
			}, oldEntry, loaded)

			// CASE DELETE

			// delete the entry if the function returns nil
			if del {
				// an old entry existed -> gc event
				if oldEntryExists {

					event = &internal.Event{
						Type: internal.EventTDelete,
						Key:  intKey,
					}
				}
				return oldEntry, true
			}

			// CASE WRITE

			// add item to gc if a ttl is set
			if expireIn > 0 || deleteIn > 0 {

				event = &internal.Event{
					Type: internal.EventTWrite,
					Key:  intKey,
				}
			}

			// Create new entry
			return entry, false // false means don't delete
		}

		// Otherwise, keep the old entry (stale writes are ignored)
		return oldEntry, false
	})

	// add event to gc events queue
	if event != nil {
		shard.Events.Push(event)
	}
}

// Expire marks the entry with the specified key as expired. This change is immediate.
// The key is still findable with the Has() method.
//
// Thread-safety: This method is thread-safe and can be called concurrently.
func (maple *mapleImpl) Expire(key string, writeIndex uint64) {
	maple.compute(key, nil, writeIndex, 0, 0, func(_, old internal.Entry, loaded bool) (internal.Entry, bool) {
		if !loaded {
			return old, true // set delete to true because else the value will be created
		}

		// case expire
		old.ExpireAt = writeIndex
		old.Value = nil

		return old, false
	})
}

// Delete removes an entry with the specified key.
// The key and value are be removed from the database. The key is not findable anymore. This change is immediate.
//
// Thread-safety: This method is thread-safe and can be called concurrently.
func (maple *mapleImpl) Delete(key string, writeIndex uint64) {
	maple.compute(key, nil, writeIndex, 0, 0, func(_, old internal.Entry, loaded bool) (entry internal.Entry, delete bool) {
		if !loaded {
			return old, true // set delete to true because else the value will be created
		}
		old.DeleteAt = writeIndex

		return old, false
	})
}

// --------------------------------------------------------------------------
// Core KVDB Interface Methods - Read Operations
// --------------------------------------------------------------------------

// Get retrieves a value for a key.
// The boolean indicates whether a (not expired) value for the key was found.
// The returned value is a copy of the stored data and therefore safe to use and modify.
//
// Thread-safety: This method is thread-safe and can be called concurrently.
func (maple *mapleImpl) Get(key string) ([]byte, bool) {

	// Convert string to integer key
	intKey := maple.StringToUint64(key)
	shard := internal.GetShard(intKey, maple.shards)

	// Init variables
	var (
		data []byte
		ok   bool
	)

	// Atomic lookup
	shard.Data.Compute(intKey, func(e internal.Entry, loaded bool) (internal.Entry, bool) {
		// case the key doesn't exist
		if !loaded {

			ok = false
			return e, true // set delete to true because else the value will be created
		}

		// case deleted or expired
		if isExpired, isDeleted := e.TTLInfo(maple.currIndex.Load()); isDeleted || isExpired {

			return e, false
		}

		// case valid data -> copy data
		ok = true
		data = make([]byte, len(e.Value))
		copy(data, e.Value)

		return e, false
	})

	return data, ok
}

// Has checks if a key exists in the database.
// This method does not check if the value for the key is expired. Use Get() for that.
//
// Thread-safety: This method is thread-safe and can be called concurrently.
func (maple *mapleImpl) Has(key string) bool {

	// Convert string to integer key
	intKey := maple.StringToUint64(key)
	shard := internal.GetShard(intKey, maple.shards)

	var ok bool
	shard.Data.Compute(intKey, func(e internal.Entry, loaded bool) (internal.Entry, bool) {
		// case the key doesn't exist
		if !loaded {
			return e, true // set delete to true because else the value will be created
		}

		// case deleted (we don't check for expired keys because they are still findable)
		if _, isDeleted := e.TTLInfo(maple.currIndex.Load()); !isDeleted {
			ok = true
		}

		return e, false
	})

	return ok
}

// --------------------------------------------------------------------------
// Garbage Collection
// --------------------------------------------------------------------------

// startGC starts the garbage collector
// if the GC is already running, this function does nothing
//
// Thread-safety: This method is thread-safe and can be called concurrently.
func (maple *mapleImpl) startGC() {
	if maple.gcIsRunning.CompareAndSwap(false, true) {
		go maple.garbageCollector()
	}
}

// stopGC stops the garbage collector.
// if the GC is not running, this function does nothing.
// the gc can't be started again after it has been stopped!
//
// Thread-safety: This method is thread-safe and can be called concurrently.
func (maple *mapleImpl) stopGC() {
	if maple.gcIsRunning.CompareAndSwap(true, false) {
		for _, shard := range maple.shards {
			shard.Events.Close()
		}
	}
}

// garbageCollector is the main garbage collection loop
// WARNING: this method should never be called! to enable GC, use startGC() and stopGC()
//
// Thread-safety: This function is not thread-safe!
func (maple *mapleImpl) garbageCollector() {

	// wait group for all shards
	var wg sync.WaitGroup
	wg.Add(len(maple.shards))

	// run gc for each shard in parallel
	for i := range maple.shards {
		go func(shardIndex int) { // start goroutine for each shard
			defer wg.Done()

			// the shard this goroutine is responsible for
			shard := maple.shards[shardIndex]

			// timeouts
			gcTimer := time.NewTimer(defaultGCInterval)
			defer gcTimer.Stop()

			for {
				// reset timeout
				gcTimer.Reset(defaultGCInterval)

				endLoop := false
				for !endLoop {
					select {
					// case add new entry to gc
					case event, ok := <-shard.Events.Recv():

						if !ok {
							return
						}
						key := event.Key
						shard := internal.GetShard(key, maple.shards)

						switch event.Type {
						case internal.EventTWrite:
							// get entry
							if entry, ok := shard.Data.Load(key); ok {
								// add entry to gc for expiration
								if entry.ExpireAt != 0 {
									shard.ExpireHeap.AddItem(uint64(key), entry.ExpireAt)
								}

								// add entry to gc for deletion
								if entry.DeleteAt != 0 {
									shard.DeleteHeap.AddItem(uint64(key), entry.DeleteAt)
								}
							}
						case internal.EventTDelete:
							shard.ExpireHeap.RemoveByKey(uint64(key))
							shard.DeleteHeap.RemoveByKey(uint64(key))
						default:
							panic(fmt.Sprintf("unknown event %s", event))
						}

					case <-gcTimer.C:
						endLoop = true
					}
				}

				// ACTUAL GC LOGIC BELOW

				/*
					Note: We only get this index once at the beginning of one gc cycle to ensure that
					we don't end up in an endless loop if the index is updated during the gc cycle.
				*/
				writeIndex := maple.currIndex.Load()

				// check if the expiry heap has expired entries
				for {

					item, exists := shard.ExpireHeap.Peek()
					if !exists || item.Priority > writeIndex {
						break
					}

					// expire entry of value
					shard.Data.Compute(util.UintKey(item.Key), func(e internal.Entry, loaded bool) (internal.Entry, bool) {
						if !loaded {
							return e, true
						}

						// double-check the entry is expired (see explanation in Note-1 below)
						if isExpired, _ := e.TTLInfo(writeIndex); !isExpired {
							return e, false
						}

						// help the go gc
						e.Value = nil

						// set value to nil
						return internal.Entry{
							Value:    nil,
							ExpireAt: e.ExpireAt,
							DeleteAt: e.DeleteAt,
							Index:    e.Index,
						}, false
					})

					// remove entry from expire gc (see explanation in Note-2 below)
					shard.ExpireHeap.RemoveByKey(item.Key)
				}

				// check if the delete heap has deleted entries
				for {
					item, exists := shard.DeleteHeap.Peek()
					if !exists || item.Priority > writeIndex {
						break
					}

					// delete entry
					shard.Data.Compute(util.UintKey(item.Key), func(e internal.Entry, loaded bool) (internal.Entry, bool) {
						if !loaded {
							return e, true
						}

						/*
							Note-1: We double-check this because the entry could have been updated in the meantime
						*/

						// double-check the entry is deleted
						if _, isDeleted := e.TTLInfo(writeIndex); !isDeleted {
							return e, false
						}

						// help the go gc
						e.Value = nil

						// delete value
						return internal.Entry{}, true
					})

					/*
						Note-2: why do we remove the item from the heaps even if the entry is not deleted?

						If we don't remove the item from the heap, the item will be reprocessed in the next iteration -> endless loop

						Ok but don't we then loose track of the item and never delete it? No! If the item was updated in the meantime
						and thus the deletion time was changed, the item was also added to the event queue and will be reprocessed in
						the next iteration of the most outer loop (in the select statement).
					*/

					// remove entry from delete and expire heap
					shard.ExpireHeap.RemoveByKey(item.Key)
					shard.DeleteHeap.RemoveByKey(item.Key)
				}
			}
		}(i)
	}

	// wait until gc is done for all shards
	wg.Wait()
}

// --------------------------------------------------------------------------
// Persistence Operations
// --------------------------------------------------------------------------

// Save persists the database to the writer
// Concurrent reading and writing is allowed during Save operation
//
// Thread-safety: This function allows concurrent operations with all other functions
// except Load. It takes snapshots of the data without blocking modifications.
func (maple *mapleImpl) Save(w io.Writer) error {
	// Use a buffered writer for better performance
	bw := bufio.NewWriterSize(w, 1024*1024) // 1 MB buffer

	// Create snapshots of data and ledger entries
	type EntryToSave struct {
		key   util.UintKey
		entry internal.Entry
	}

	var dataEntries []EntryToSave

	// Collect snapshots of all shards as a SAVE operation
	// Process each shard
	for _, shard := range maple.shards {

		// Collect data entries up to saveUntilIndex
		shard.Data.Range(func(key util.UintKey, entry internal.Entry) bool {

			// test if the entry is deleted and skip if yes
			if _, isDeleted := entry.TTLInfo(maple.currIndex.Load()); isDeleted {
				return true
			}

			// Create deep copy
			entryCopy := internal.Entry{
				ExpireAt: entry.ExpireAt,
				Index:    entry.Index,
				Value:    make([]byte, len(entry.Value)),
			}
			copy(entryCopy.Value, entry.Value)

			dataEntries = append(dataEntries, EntryToSave{key, entryCopy})
			return true
		})
	}

	// Write file header
	if _, err := bw.WriteString(magicNum); err != nil {
		return err
	}

	// Write maple version
	if err := binary.Write(bw, binary.LittleEndian, uint8(mapleVersion)); err != nil {
		return err
	}

	// Write seed
	if err := binary.Write(bw, binary.LittleEndian, maple.seed); err != nil {
		return err
	}

	// Write total data entries count
	if err := binary.Write(bw, binary.LittleEndian, uint64(len(dataEntries))); err != nil {
		return err
	}

	// Write data entries
	for _, item := range dataEntries {

		// Write key
		if err := binary.Write(bw, binary.LittleEndian, uint64(item.key)); err != nil {
			return err
		}

		// Write expiration timestamp
		if err := binary.Write(bw, binary.LittleEndian, item.entry.DeleteAt); err != nil {
			return err
		}

		// Write deletion timestamp
		if err := binary.Write(bw, binary.LittleEndian, item.entry.ExpireAt); err != nil {
			return err
		}

		// Write created index
		if err := binary.Write(bw, binary.LittleEndian, item.entry.Index); err != nil {
			return err
		}

		// Write value length
		valueLen := uint32(len(item.entry.Value))
		if err := binary.Write(bw, binary.LittleEndian, valueLen); err != nil {
			return err
		}

		// Write value bytes
		if _, err := bw.Write(item.entry.Value); err != nil {
			return err
		}
	}

	// Flush buffer to ensure all data is written
	return bw.Flush()
}

// Load restores a database from the reader
//
// Thread-safety: This function is not thread-safe and should not be called concurrently
func (maple *mapleImpl) Load(r io.Reader) error {

	// stop gc during load
	maple.stopGC()
	defer maple.startGC() // we only can re-enable the gc because all shards are recreated

	// Use a buffered reader for better performance
	br := bufio.NewReaderSize(r, 1024*1024) // 1 MB buffer

	// Read and verify magic number
	magicBytes := make([]byte, len(magicNum))
	if _, err := io.ReadFull(br, magicBytes); err != nil {
		return err
	}

	if string(magicBytes) != magicNum {
		return fmt.Errorf("invalid file format: magic number mismatch")
	}

	// Read and verify version
	var version uint8
	if err := binary.Read(br, binary.LittleEndian, &version); err != nil {
		return err
	}

	if int(version) != mapleVersion {
		return fmt.Errorf("unsupported version: %d (expected %d)", version, mapleVersion)
	}

	// Read seed
	var seed uint64
	if err := binary.Read(br, binary.LittleEndian, &seed); err != nil {
		return err
	}

	// Recreate empty shards with the loaded seed
	hasher := createIdentityHasher()
	shards := make([]*internal.Shard, maple.numShards)
	for i := 0; i < maple.numShards; i++ {
		shards[i] = internal.NewShard(hasher)
	}

	maple.shards = shards
	maple.seed = seed

	// Initialize atomic values
	maple.currIndex.Store(0)

	// Read data entries count
	var dataCount uint64
	if err := binary.Read(br, binary.LittleEndian, &dataCount); err != nil {
		return err
	}

	// Track the highest index seen during load
	var maxIndex uint64 = 0

	// Read data entries
	for i := uint64(0); i < dataCount; i++ {
		// Read key
		var keyUint uint64
		if err := binary.Read(br, binary.LittleEndian, &keyUint); err != nil {
			return err
		}
		key := util.UintKey(keyUint)

		// Read expiration timestamp
		var expireAt uint64
		if err := binary.Read(br, binary.LittleEndian, &expireAt); err != nil {
			return err
		}

		// Read deletion timestamp
		var deleteAt uint64
		if err := binary.Read(br, binary.LittleEndian, &deleteAt); err != nil {
			return err
		}

		// Read created index
		var createdIndex uint64
		if err := binary.Read(br, binary.LittleEndian, &createdIndex); err != nil {
			return err
		}

		// Track the highest index
		if createdIndex > maxIndex {
			maxIndex = createdIndex
		}

		// Read value length
		var valueLen uint32
		if err := binary.Read(br, binary.LittleEndian, &valueLen); err != nil {
			return err
		}

		// Read value bytes
		value := make([]byte, valueLen)
		if _, err := io.ReadFull(br, value); err != nil {
			return err
		}

		// TTLInfo the appropriate shard and store entry
		shard := internal.GetShard(key, maple.shards)
		shard.Data.Store(key, internal.Entry{
			Value:    value,
			ExpireAt: expireAt,
			DeleteAt: deleteAt,
			Index:    createdIndex,
		})

		// add entry directly to gc, we can do this here because this method is single threaded
		if expireAt != 0 {
			shard.ExpireHeap.AddItem(uint64(key), expireAt)
		}
		if deleteAt != 0 {
			shard.DeleteHeap.AddItem(uint64(key), deleteAt)
		}
	}

	// Update current index to the highest seen during load
	maple.SetWriteIdx(maxIndex)

	return nil
}

// --------------------------------------------------------------------------
// KVDB Interface Implementation - Features and Metadata
// --------------------------------------------------------------------------

// GetInfo returns statistics about the database
func (maple *mapleImpl) GetInfo() db.DatabaseInfo {

	// get current index only once to reduce contention
	currentWriteIndex := maple.currIndex.Load()

	// create a size histogram for the info
	histogram := util.NewSizeHistogram()
	samplesPerShard := 100
	wg := sync.WaitGroup{}
	wg.Add(len(maple.shards))

	// more stats
	mu := sync.Mutex{}
	samplesCount := 0
	expiredBacklog := 0
	deletedBacklog := 0
	shardSizes := make([]float64, len(maple.shards))

	// concurrently collect samples from all shards
	for shardIndex, shard := range maple.shards {
		go func(i int, s *internal.Shard) {
			defer wg.Done()
			count := 0
			expiredCount := 0
			deletedCount := 0
			s.Data.Range(func(key util.UintKey, entry internal.Entry) bool {
				// track size in histogram
				histogram.AddSample(len(entry.Value))

				// check if entry is expired or deleted but not yet processed by the gc
				isExpired, isDeleted := entry.TTLInfo(currentWriteIndex)
				if isExpired && entry.Value != nil {
					expiredCount++
				}
				if isDeleted {
					deletedCount++
				}

				// only sample a few entries per shard
				count++
				return count < samplesPerShard
			})

			// stats lockmgr
			mu.Lock()
			defer mu.Unlock()

			// track stats
			samplesCount += count
			expiredBacklog += expiredCount
			deletedBacklog += deletedCount
			shardSizes[i] = float64(s.Data.Size())
		}(shardIndex, shard)
	}

	// wait for all shards to finish
	wg.Wait()

	// calculate size
	entryOverhead := 32 // 8 bytes each for key, expireAt, deleteAt, index
	medianSize := histogram.MedianEstimate() + entryOverhead
	avgSize := histogram.AverageSize() + entryOverhead

	// weighted estimate (60% median, 40% average)
	sizeBytes := (medianSize*60 + avgSize*40) / 100

	// Metadata for this specific database implementation
	meta := &struct {
		CurrentWriteIndex uint64                 `json:"current_write_index"`
		ShardCount        int                    `json:"shard_count"`
		ShardDistribution util.DistributionStats `json:"shard_distribution"`
		ExpiredBacklog    float64                `json:"expired_backlog"`
		DeletedBacklog    float64                `json:"deleted_backlog"`
		Info              string                 `json:"info"`
	}{
		CurrentWriteIndex: currentWriteIndex,
		ShardCount:        len(maple.shards),
		ShardDistribution: util.NewDistributionStats(shardSizes),
		ExpiredBacklog:    float64(expiredBacklog) / float64(samplesCount), // how many entries are expired but not yet processed by the gc in percent
		DeletedBacklog:    float64(deletedBacklog) / float64(samplesCount), // how many entries are deleted but not yet processed by the gc in percent
		Info:              "All values (including SizeBytes) are estimates and may vary depending on the database state.",
	}

	// features
	supportedFeatures := []db.Feature{
		db.FeatureSet, db.FeatureSetE, db.FeatureSetEIfUnset,
		db.FeatureExpire | db.FeatureDelete,
		db.FeatureGet, db.FeatureHas,
		db.FeatureSave, db.FeatureLoad,
		db.FeatureGarbageCollect,
	}

	return db.DatabaseInfo{
		SizeBytes:         sizeBytes,
		DbType:            db.ImplMaple,
		SupportedFeatures: supportedFeatures,
		Metadata:          meta,
	}
}

// SupportsFeature checks if this implementation supports a specific KVDB feature
func (maple *mapleImpl) SupportsFeature(feature db.Feature) bool {
	supportedFeatures := db.FeatureSet |
		db.FeatureSetE |
		db.FeatureSetEIfUnset |
		db.FeatureGet |
		db.FeatureExpire |
		db.FeatureDelete |
		db.FeatureHas |
		db.FeatureSave |
		db.FeatureLoad |
		db.FeatureGarbageCollect
	return supportedFeatures&feature == feature
}

// Close stops the garbage collector
func (maple *mapleImpl) Close() error {
	maple.stopGC()
	return nil
}

// --------------------------------------------------------------------------
// Index and Timestamp Management
// --------------------------------------------------------------------------

// SetWriteIdx safely updates the current index
// It only updates if the new index is greater than the current one
//
// Thread-safety: This method is thread-safe and can be called concurrently.
// It uses atomic operations to ensure that the index only increases.
func (maple *mapleImpl) SetWriteIdx(newIdx uint64) {
	// Only update if the new index is greater
	for {
		currIdx := maple.currIndex.Load()
		if newIdx <= currIdx {
			return
		}
		if maple.currIndex.CompareAndSwap(currIdx, newIdx) {
			return
		}
	}
}

// WriteIdx returns the current index of the database
func (maple *mapleImpl) WriteIdx() uint64 {
	return maple.currIndex.Load()
}
