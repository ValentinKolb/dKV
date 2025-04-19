package testing

import (
	"bytes"
	"fmt"
	"github.com/ValentinKolb/dKV/lib/db"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"
)

// RunKVDBBenchmarks runs all benchmarks for a key-value database implementations
func RunKVDBBenchmarks(b *testing.B, name string, factory DBFactory) {

	b.Run("Set", func(b *testing.B) {
		benchmarkSet(b, factory())
	})

	b.Run("SetExisting", func(b *testing.B) {
		benchmarkSetExisting(b, factory())
	})

	b.Run("SetLargeValue", func(b *testing.B) {
		benchmarkSetLargeValue(b, factory())
	})

	b.Run("SetWithExpiry", func(b *testing.B) {
		benchmarkSetWithExpiry(b, factory())
	})

	b.Run("Get", func(b *testing.B) {
		benchmarkGet(b, factory())
	})

	b.Run("GetWithExpiry", func(b *testing.B) {
		benchmarkGetWithExpiry(b, factory())
	})

	b.Run("Delete", func(b *testing.B) {
		benchmarkDelete(b, factory())
	})

	b.Run("Has", func(b *testing.B) {
		benchmarkHas(b, factory())
	})

	b.Run("Has(not)", func(b *testing.B) {
		benchmarkHasNot(b, factory())
	})

	b.Run("SaveLoad", func(b *testing.B) {
		benchmarkSaveLoad(b, factory)
	})

	b.Run("MixedUsage", func(b *testing.B) {
		benchmarkMixedUsage(b, factory())
	})

	b.Run("MixedUsageWithExpiry", func(b *testing.B) {
		benchmarkMixedOperationsWithExpiry(b, factory())
	})
}

// --------------------------------------------------------------------------
// Benchmark functions
// --------------------------------------------------------------------------

// Benchmark for Set operation
func benchmarkSet(b *testing.B, database db.KVDB) {

	b.Cleanup(func() {
		database.Close()
	})

	requireFeature(b, database, db.FeatureSet)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			key := fmt.Sprintf("test-key-%d", counter)
			value := []byte(fmt.Sprintf("test-value-%d", counter))
			database.Set(key, value, 0)
			counter++
		}
	})
}

// Benchmark for Set operation with existing keys
func benchmarkSetExisting(b *testing.B, database db.KVDB) {

	b.Cleanup(func() {
		database.Close()
	})

	requireFeature(b, database, db.FeatureSet)

	// Prepare data
	numKeys := b.N
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		value := []byte(fmt.Sprintf("test-value-%d", i))
		database.Set(key, value, 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			key := fmt.Sprintf("test-key-%d", counter%numKeys)
			value := []byte(fmt.Sprintf("test-value-%d", counter))
			database.Set(key, value, 0)
			counter++
		}
	})
}

// Benchmark for Set operation with large values
func benchmarkSetLargeValue(b *testing.B, database db.KVDB) {

	b.Cleanup(func() {
		database.Close()
	})

	requireFeature(b, database, db.FeatureSet)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			key := fmt.Sprintf("test-key-%d", counter)
			largeValue := make([]byte, 1*1024*1024) // 1MB
			database.Set(key, largeValue, 0)
			counter++
		}
	})
}

// Parallel benchmarking for Get operation
func benchmarkGet(b *testing.B, database db.KVDB) {

	b.Cleanup(func() {
		database.Close()
	})

	requireFeature(b, database, db.FeatureSet)
	requireFeature(b, database, db.FeatureGet)

	// Prepare data
	numKeys := 10000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		value := []byte(fmt.Sprintf("test-value-%d", i))
		database.Set(key, value, 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			key := fmt.Sprintf("test-key-%d", counter%numKeys)
			database.Get(key)
			counter++
		}
	})
}

// Parallel benchmarking for Delete operation
func benchmarkDelete(b *testing.B, database db.KVDB) {

	b.Cleanup(func() {
		database.Close()
	})

	requireFeature(b, database, db.FeatureSet)
	requireFeature(b, database, db.FeatureDelete)

	numKeys := 100000
	if b.N < numKeys {
		numKeys = b.N
	}

	// Prepare data
	keys := make([]string, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = fmt.Sprintf("test-key-%d", i)
		value := []byte(fmt.Sprintf("test-value-%d", i))
		database.Set(keys[i], value, 0)
	}

	// Counter for atomic access
	var counter int64

	// Reset timer since we were doing setup
	b.ResetTimer()

	// Run parallel delete operations
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := int(atomic.AddInt64(&counter, 1)-1) % numKeys
			database.Delete(keys[idx], 0)
		}
	})
}

// Parallel benchmarking for Has operation (with key miss)
func benchmarkHasNot(b *testing.B, database db.KVDB) {

	b.Cleanup(func() {
		database.Close()
	})

	// Prepare data
	requireFeature(b, database, db.FeatureHas)
	const key = "test-key"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			database.Has(key)
		}
	})
}

// Parallel benchmarking for Has operation
func benchmarkHas(b *testing.B, database db.KVDB) {

	b.Cleanup(func() {
		database.Close()
	})

	requireFeature(b, database, db.FeatureSet)
	requireFeature(b, database, db.FeatureHas)

	// Prepare data
	numKeys := 10000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		value := []byte(fmt.Sprintf("test-value-%d", i))
		database.Set(key, value, 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			key := fmt.Sprintf("test-key-%d", counter%numKeys)
			database.Has(key)
			counter++
		}
	})
}

// Benchmark for Save and Load operations
// For these operations, parallelization is not meaningful as they typically
// lockmgr the entire database
func benchmarkSaveLoad(b *testing.B, factory DBFactory) {

	database := factory()

	b.Cleanup(func() {
		database.Close()
	})

	requireFeature(b, database, db.FeatureSet)
	requireFeature(b, database, db.FeatureSave)
	requireFeature(b, database, db.FeatureLoad)

	// Create a database with some data
	numEntries := 10000
	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		value := []byte(fmt.Sprintf("test-value-%d", i))
		database.Set(key, value, 0)
	}

	b.Run("Save", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			database.Save(&buf)
		}
	})

	// Prepare a data buffer for Load benchmark
	var loadBuf bytes.Buffer
	database.Save(&loadBuf)
	data := loadBuf.Bytes()

	b.Run("Load", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			loadDB := factory()
			defer loadDB.Close()
			loadDB.Load(bytes.NewReader(data))
		}
	})
}

// Benchmark for mixed usage patterns
func benchmarkMixedUsage(b *testing.B, database db.KVDB) {
	b.Cleanup(func() {
		database.Close()
	})

	requireFeature(b, database, db.FeatureSet)
	requireFeature(b, database, db.FeatureGet)
	requireFeature(b, database, db.FeatureDelete)
	requireFeature(b, database, db.FeatureHas)

	// Number of pre-populated keys
	numKeys := 100000
	if b.N < numKeys {
		numKeys = b.N
	}

	// Prepare initial data
	keys := make([]string, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = fmt.Sprintf("test-key-%d", i)
		value := []byte(fmt.Sprintf("test-value-%d", i))
		database.Set(keys[i], value, 0)
	}

	// Counter for atomic access
	var counter int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// Local counter for each goroutine
		localCounter := 0

		for pb.Next() {
			// Get a somewhat random index
			idx := int(atomic.AddInt64(&counter, 1)-1) % numKeys

			// Select operation (0-4: get, set, delete, has, exists)
			op := localCounter % 5

			// For every 10th operation, use a completely new key
			var key string
			if localCounter%10 == 0 {
				key = fmt.Sprintf("new-key-%d", localCounter)
			} else {
				key = keys[idx]
			}

			// Perform the selected operation
			switch op {
			case 0: // Get
				database.Get(key)
			case 1: // Set
				value := []byte(fmt.Sprintf("mixed-value-%d", localCounter))
				database.Set(key, value, 0)
			case 2: // Delete
				database.Delete(key, 0)
			case 3: // Has
				database.Has(key)
			}

			localCounter++
		}
	})
}

// benchmarkSetWithExpiry tests the performance of SetE with TTL
func benchmarkSetWithExpiry(b *testing.B, database db.KVDB) {
	b.Cleanup(func() {
		database.Close()
	})

	requireFeature(b, database, db.FeatureSet)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		currentIndex := uint64(0)
		//for i := 0; i < 1_0_000; i++ {
		for pb.Next() {
			key := fmt.Sprintf("test-expiry-key-%d", currentIndex)
			value := []byte(fmt.Sprintf("test-expiry-value-%d", currentIndex))
			database.SetE(key, value, currentIndex, currentIndex+1, currentIndex+2)
			currentIndex++
		}
	})
}

// benchmarkGetWithExpiry tests the performance of Get with expired keys
func benchmarkGetWithExpiry(b *testing.B, database db.KVDB) {
	b.Cleanup(func() {
		database.Close()
	})

	requireFeature(b, database, db.FeatureSet)
	requireFeature(b, database, db.FeatureGet)

	// Prepare data with various expiration times
	numKeys := 10000
	baseIndex := uint64(1000)

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("test-expiry-key-%d", i)
		value := []byte(fmt.Sprintf("test-expiry-value-%d", i))
		ttl := uint64(0)

		// 50% of keys get a TTL
		if i%2 == 0 {
			ttl = uint64(i % 1000) // Various TTLs
		}

		database.SetE(key, value, baseIndex, 0, ttl)
	}

	// Benchmark with an index where about 25% of keys should have expired
	currentIndex := baseIndex + 500

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			key := fmt.Sprintf("test-expiry-key-%d", counter%numKeys)
			database.SetWriteIdx(currentIndex)
			database.Get(key)
			counter++
		}
	})
}

// benchmarkMixedOperationsWithExpiry tests mixed operations with expiration
func benchmarkMixedOperationsWithExpiry(b *testing.B, database db.KVDB) {
	b.Cleanup(func() {
		database.Close()
	})

	requireFeature(b, database, db.FeatureSet)
	requireFeature(b, database, db.FeatureGet)

	// Prepare some initial data
	numKeys := 50_000
	baseIndex := uint64(1000)

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("test-mixed-key-%d", i)
		value := []byte(fmt.Sprintf("test-mixed-value-%d", i))
		ttl := uint64(i % 2000) // Various TTLs
		database.SetE(key, value, baseIndex, 0, ttl)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

		for pb.Next() {
			// Random operation: 70% Get, 30% Set
			key := fmt.Sprintf("test-mixed-key-%d", counter%numKeys)
			currentIndex := baseIndex + uint64(counter)

			if rnd.Float32() < .7 {
				// Get operation
				database.Get(key)
			} else {
				// Set operation with TTL
				value := []byte(fmt.Sprintf("test-mixed-updated-value-%d", counter))
				ttl := uint64(rnd.Intn(1000))
				database.SetE(key, value, currentIndex, 0, ttl)
			}

			counter++
		}
	})
}
