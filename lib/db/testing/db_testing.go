package testing

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ValentinKolb/dKV/lib/db"
)

// DBFactory is a function that creates a new instance of a KVDB implementation
type DBFactory func() db.KVDB

// RunKVDBTests runs a comprehensive test suite for a KVDB implementation.
func RunKVDBTests(t *testing.T, name string, factory DBFactory) {
	t.Run(name, func(t *testing.T) {
		t.Run("Set&Get", func(t *testing.T) {
			testSetGet(t, factory())
		})

		t.Run("Expire", func(t *testing.T) {
			testExpire(t, factory())
		})

		t.Run("Delete", func(t *testing.T) {
			testDelete(t, factory())
		})

		t.Run("Has", func(t *testing.T) {
			testHas(t, factory())
		})

		t.Run("SetEIfUnset", func(t *testing.T) {
			testSetEIfUnset(t, factory())
		})

		t.Run("KeyExpiry", func(t *testing.T) {
			testKeyExpiry(t, factory())
		})

		t.Run("TestManyExpiringKeys", func(t *testing.T) {
			testManyExpiringKeys(t, factory())
		})

		t.Run("SaveLoad", func(t *testing.T) {
			testSaveLoad(t, factory)
		})

		t.Run("EdgeCases", func(t *testing.T) {
			testEdgeCases(t, factory())
		})

		t.Run("CollisionHandling", func(t *testing.T) {
			testCollisionHandling(t, factory())
		})

		t.Run("RealisticUsage", func(t *testing.T) {
			testRealisticUsage(t, factory())
		})
	})
}

// --------------------------------------------------------------------------
// Helper functions
// --------------------------------------------------------------------------

// Checks if the database supports the specified feature
// Skip the test if it is not supported
func requireFeature(t testing.TB, database db.KVDB, feature db.Feature) {
	if !database.SupportsFeature(feature) {
		t.Skip()
	}
}

// --------------------------------------------------------------------------
// Test functions
// --------------------------------------------------------------------------

func testSetGet(t *testing.T, database db.KVDB) {
	defer database.Close()

	requireFeature(t, database, db.FeatureSet)
	requireFeature(t, database, db.FeatureGet)

	testKey := "test-key"
	testValue1 := []byte("test-value1")
	testValue2 := []byte("test-value2")

	database.Set(testKey, testValue1, 0)

	result, exists := database.Get(testKey)
	if !exists {
		t.Errorf("Expected key %s to exist after Set", testKey)
	}

	if !bytes.Equal(result, testValue1) {
		t.Errorf("Expected value %s, got %s", testValue1, result)
	}

	database.Set(testKey, testValue2, 0)

	result, exists = database.Get(testKey)
	if !exists {
		t.Errorf("Expected key %s to exist after Set", testKey)
	}

	if !bytes.Equal(result, testValue2) {
		t.Errorf("Expected value %s, got %s", testValue2, result)
	}

	_, exists = database.Get("nonexistent-key")
	if exists {
		t.Errorf("Expected nonexistent key to return exists=false")
	}

	retrievedValue, _ := database.Get(testKey)
	retrievedValue[0] = 'X'

	originalValue, _ := database.Get(testKey)
	if bytes.Equal(retrievedValue, originalValue) {
		t.Errorf("Get should return a copy, not a reference to the stored value")
	}

	updatedValue := []byte("updated-value")
	database.Set(testKey, updatedValue, 0)

	result, exists = database.Get(testKey)
	if !exists {
		t.Errorf("Expected key %s to exist after update", testKey)
	}

	if !bytes.Equal(result, updatedValue) {
		t.Errorf("Expected updated value %s, got %s", updatedValue, result)
	}
}

func testKeyExpiry(t *testing.T, database db.KVDB) {
	defer database.Close()

	requireFeature(t, database, db.FeatureSet)
	requireFeature(t, database, db.FeatureGet)
	requireFeature(t, database, db.FeatureHas)

	testKey := "expiring-key"
	testValue := []byte("expiring-value")

	database.SetE(testKey, testValue, 100, 10, 20)

	database.SetWriteIdx(109)

	result, exists := database.Get(testKey)
	if !exists {
		t.Errorf("Key should still exist at index 109 (get)")
	}
	if !bytes.Equal(result, testValue) {
		t.Errorf("Expected value %s, got %s", testValue, result)
	}
	has := database.Has(testKey)
	if !has {
		t.Errorf("Key should still exist at index 109 (has)")
	}

	database.SetWriteIdx(110)

	result, exists = database.Get(testKey)
	if exists {
		t.Errorf("Key should have expired at index 110 (get)")
	}
	has = database.Has(testKey)
	if !has {
		t.Errorf("Key should still exist at index 110 (has)")
	}

	database.SetWriteIdx(120)

	result, exists = database.Get(testKey)
	if exists {
		t.Errorf("Key should have been deleted at index 120 (get)")
	}
	has = database.Has(testKey)
	if has {
		t.Errorf("Key should not exist at index 120 (has)")
	}

	testKey2 := "test-key2"
	testValue2 := []byte("test-value2")

	database.SetE(testKey2, testValue2, 200, 0, 10)

	database.SetWriteIdx(209)

	result, exists = database.Get(testKey2)
	if !exists {
		t.Errorf("Key should still exist at index 209")
	}
	if !bytes.Equal(result, testValue2) {
		t.Errorf("Expected value %s, got %s", testValue2, result)
	}
	has = database.Has(testKey2)
	if !has {
		t.Errorf("Key should still exist at index 209")
	}

	database.SetWriteIdx(210)

	result, exists = database.Get(testKey2)
	if exists {
		t.Errorf("Key should have been deleted at index 210")
	}
	has = database.Has(testKey2)
	if has {
		t.Errorf("Key should not exist at index 210")
	}

	testKey3 := "not-expiring-key"
	testValue3 := []byte("not-expiring-value")

	database.SetE(testKey3, testValue3, 300, 0, 0)

	database.SetWriteIdx(1000)
	result, exists = database.Get(testKey3)
	if !exists {
		t.Errorf("Key with TTL=0 should never expire")
	}
	if !bytes.Equal(result, testValue3) {
		t.Errorf("Expected value %s, got %s", testValue2, result)
	}
	has = database.Has(testKey3)
	if !has {
		t.Errorf("Key with TTL=0 should never expire")
	}
}

func testManyExpiringKeys(t *testing.T, database db.KVDB) {
	defer database.Close()

	requireFeature(t, database, db.FeatureSet)
	requireFeature(t, database, db.FeatureGet)

	numKeys := 1000
	baseIndex := uint64(1000)

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("expire-key-%d", i)
		value := []byte(fmt.Sprintf("expire-value-%d", i))
		ttl := uint64(i % 100)
		database.SetE(key, value, baseIndex, ttl, 0)

		if !database.Has(key) {
			t.Errorf("Key %s not found after Set", key)
		}
	}

	for offset := uint64(0); offset <= 100; offset += 10 {
		currentIndex := baseIndex + offset

		expiredCount := 0
		for i := 0; i < numKeys; i++ {
			key := fmt.Sprintf("expire-key-%d", i)
			ttl := uint64(i % 100)

			if ttl > 0 && ttl <= offset {

				database.SetWriteIdx(currentIndex)
				_, exists := database.Get(key)
				if exists {
					t.Errorf("Key %s should have expired at index %d (TTL=%d)",
						key, currentIndex, ttl)
				} else {
					expiredCount++
				}
			}
		}
	}
}

func testExpire(t *testing.T, database db.KVDB) {
	defer database.Close()

	requireFeature(t, database, db.FeatureSet)
	requireFeature(t, database, db.FeatureGet)

	testKey := "expire-test-key"
	testValue := []byte("expire-test-value")

	database.Set(testKey, testValue, 0)

	_, exists := database.Get(testKey)
	if !exists {
		t.Errorf("Expected key %s to exist after Set", testKey)
	}

	database.Expire(testKey, 10)

	_, exists = database.Get(testKey)
	if exists {
		t.Errorf("Expected key %s to not exist after Expire", testKey)
	}

	if !database.Has(testKey) {
		t.Errorf("Expected key %s to exist after Expire", testKey)
	}

	database.Expire("nonexistent-key", 0)
}

func testDelete(t *testing.T, database db.KVDB) {
	defer database.Close()

	requireFeature(t, database, db.FeatureSet)
	requireFeature(t, database, db.FeatureGet)
	requireFeature(t, database, db.FeatureDelete)

	testKey := "delete-test-key"
	testValue := []byte("delete-test-value")

	database.Set(testKey, testValue, 0)

	_, exists := database.Get(testKey)
	if !exists {
		t.Errorf("Expected key %s to exist after Set", testKey)
	}

	database.Delete(testKey, 10)

	_, exists = database.Get(testKey)
	if exists {
		t.Errorf("Expected key %s to not exist after Delete", testKey)
	}

	if database.Has(testKey) {
		t.Errorf("Expected key %s to not exist after Delete", testKey)
	}

	database.Delete("nonexistent-key", 0)
}

func testHas(t *testing.T, database db.KVDB) {
	defer database.Close()

	requireFeature(t, database, db.FeatureSet)
	requireFeature(t, database, db.FeatureDelete)
	requireFeature(t, database, db.FeatureHas)

	testKey := "has-exists-test-key"
	testValue := []byte("has-exists-test-value")

	if database.Has(testKey) {
		t.Errorf("Expected Has to return false for nonexistent key")
	}

	database.Set(testKey, testValue, 0)

	if !database.Has(testKey) {
		t.Errorf("Expected Has to return true after Set")
	}

	database.Expire(testKey, 0)

	if !database.Has(testKey) {
		t.Errorf("Expected Has to return true after Expire (ledger retention)")
	}
}

func testSetEIfUnset(t *testing.T, database db.KVDB) {
	defer database.Close()

	requireFeature(t, database, db.FeatureSet)
	requireFeature(t, database, db.FeatureGet)

	testKey := "test-key"
	testValue1 := []byte("test-value")
	testValue2 := []byte("test-value2")

	database.SetEIfUnset(testKey, testValue1, 0, 10, 0)

	result, exists := database.Get(testKey)
	if !exists {
		t.Errorf("Expected key %s to exist after Set", testKey)
	}

	if !bytes.Equal(result, testValue1) {
		t.Errorf("Expected value %s, got %s", testValue1, result)
	}

	database.SetEIfUnset(testKey, testValue2, 5, 20, 0)

	result, exists = database.Get(testKey)
	if !exists {
		t.Errorf("Expected key %s to exist after Set", testKey)
	}

	if !bytes.Equal(result, testValue1) {
		t.Errorf("Expected value %s, got %s", testValue1, result)
	}

	database.SetWriteIdx(11)
	_, exists = database.Get(testKey)
	if exists {
		t.Errorf("Expected key %s to not exist after ttl expired", testKey)
	}
}

func testSaveLoad(t *testing.T, factory DBFactory) {
	database := factory()
	database2 := factory()

	// close the databases after the test
	defer database.Close()
	defer database2.Close()

	requireFeature(t, database, db.FeatureSet)
	requireFeature(t, database, db.FeatureGet)
	requireFeature(t, database, db.FeatureSave)
	requireFeature(t, database, db.FeatureLoad)

	numEntries := 1000
	originalKeys := make([]string, numEntries)
	originalValues := make([][]byte, numEntries)

	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("save-load-test-key-%d", i)
		value := []byte(fmt.Sprintf("save-load-test-value-%d", i))
		originalKeys[i] = key
		originalValues[i] = value

		database.Set(key, value, 0)
	}

	var buf bytes.Buffer
	err := database.Save(&buf)
	if err != nil {
		t.Errorf("Unexpected error during Save: %v", err)
	}

	err = database2.Load(&buf)
	if err != nil {
		t.Errorf("Unexpected error during Load: %v", err)
	}

	for i := 0; i < numEntries; i++ {
		key := originalKeys[i]
		expectedValue := originalValues[i]

		actualValue, exists := database2.Get(key)
		if !exists {
			t.Errorf("Key %s not found after Load", key)
			continue
		}

		if !bytes.Equal(actualValue, expectedValue) {
			t.Errorf("Priority mismatch for key %s: expected %s, got %s", key, expectedValue, actualValue)
		}
	}

	for i := 0; i < numEntries; i++ {
		key := originalKeys[i]
		expectedValue := originalValues[i]

		actualValue, exists := database.Get(key)
		if !exists {
			t.Errorf("Key %s not found in original database", key)
			continue
		}

		if !bytes.Equal(actualValue, expectedValue) {
			t.Errorf("Priority mismatch in original database for key %s", key)
		}
	}
}

func testEdgeCases(t *testing.T, database db.KVDB) {
	defer database.Close()

	requireFeature(t, database, db.FeatureSet)
	requireFeature(t, database, db.FeatureGet)

	emptyKey := ""
	emptyKeyValue := []byte("value for empty key")

	database.Set(emptyKey, emptyKeyValue, 0)

	result, exists := database.Get(emptyKey)
	if !exists {
		t.Errorf("Empty key not found after Set")
	} else if !bytes.Equal(result, emptyKeyValue) {
		t.Errorf("Priority mismatch for empty key")
	}

	emptyValueKey := "empty-value-key"
	var emptyValue []byte

	database.Set(emptyValueKey, emptyValue, 0)

	result, exists = database.Get(emptyValueKey)
	if !exists {
		t.Errorf("Key for empty value not found after Set")
	} else if !bytes.Equal(result, emptyValue) {
		t.Errorf("Empty value mismatch")
	}

	nilValueKey := "nil-value-key"
	var nilValue []byte = nil

	database.Set(nilValueKey, nilValue, 0)

	result, exists = database.Get(nilValueKey)
	if !exists {
		t.Errorf("Key for nil value not found after Set")
	} else if len(result) != 0 {
		t.Errorf("Nil value resulted in non-empty value: %v", result)
	}

	if !t.Failed() {

		largeKey := string(make([]byte, 1000))
		largeKeyValue := []byte("value for large key")

		database.Set(largeKey, largeKeyValue, 0)

		result, exists = database.Get(largeKey)
		if !exists {
			t.Errorf("Large key not found after Set")
		} else if !bytes.Equal(result, largeKeyValue) {
			t.Errorf("Priority mismatch for large key")
		}

		largeValueKey := "large-value-key"
		largeValue := make([]byte, 100*1024*1024)

		for i := range largeValue {
			largeValue[i] = byte(i % 256)
		}

		database.Set(largeValueKey, largeValue, 0)

		result, exists = database.Get(largeValueKey)
		if !exists {
			t.Errorf("Key for large value not found after Set")
		} else if !bytes.Equal(result, largeValue) {

			headMismatch := !bytes.Equal(result[:10], largeValue[:10])
			tailMismatch := !bytes.Equal(result[len(result)-10:], largeValue[len(largeValue)-10:])
			if headMismatch || tailMismatch || len(result) != len(largeValue) {
				t.Errorf("Large value mismatch: Head mismatch=%v, Tail mismatch=%v, Size mismatch=%v",
					headMismatch, tailMismatch, len(result) != len(largeValue))
			}
		}
	}
}

func testCollisionHandling(t *testing.T, database db.KVDB) {
	defer database.Close()

	requireFeature(t, database, db.FeatureSet)
	requireFeature(t, database, db.FeatureGet)
	requireFeature(t, database, db.FeatureDelete)

	prefix := "collision-test-"
	numKeys := 1000

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("%s%d", prefix, i)
		value := []byte(fmt.Sprintf("value-%d", i))

		database.Set(key, value, 0)
	}

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("%s%d", prefix, i)
		expectedValue := []byte(fmt.Sprintf("value-%d", i))

		actualValue, exists := database.Get(key)
		if !exists {
			t.Errorf("Key %s not found", key)
			continue
		}

		if !bytes.Equal(actualValue, expectedValue) {
			t.Errorf("Priority for key %s does not match: expected %s, got %s",
				key, expectedValue, actualValue)
		}
	}

	for i := 0; i < numKeys; i += 2 {
		key := fmt.Sprintf("%s%d", prefix, i)
		database.Delete(key, 10)
	}

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("%s%d", prefix, i)
		_, exists := database.Get(key)

		if i%2 == 0 {
			if exists {
				t.Errorf("Key %s should be deleted", key)
			}
		} else {
			if !exists {
				t.Errorf("Key %s should still exist", key)
			}
		}
	}
}

func testRealisticUsage(t *testing.T, database db.KVDB) {
	defer database.Close()

	requireFeature(t, database, db.FeatureSet)
	requireFeature(t, database, db.FeatureGet)
	requireFeature(t, database, db.FeatureDelete)

	type operation struct {
		op    string
		key   string
		value []byte
	}

	numOperations := 10_000
	operations := make([]operation, numOperations)

	for i := 0; i < numOperations; i++ {
		var op string
		switch i % 10 {
		case 0, 1, 2, 3, 4, 5, 6:
			op = "set"
		case 7, 8:
			op = "get"
		case 9:
			op = "delete"
		}

		var key string
		if i%5 == 0 {

			key = fmt.Sprintf("hot-key-%d", i%50)
		} else {

			key = fmt.Sprintf("key-%d", i)
		}

		var value []byte
		if op == "set" {
			valueSize := 64
			if i%10 == 0 {

				valueSize = 1024
			}
			value = make([]byte, valueSize)

			for j := 0; j < valueSize; j++ {
				value[j] = byte((i + j) % 256)
			}
		}

		operations[i] = operation{op, key, value}
	}

	allKeys := make(map[string]bool)
	for _, op := range operations {
		allKeys[op.key] = true
	}

	numWorkers := 8
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	var errorCount int32

	opsPerWorker := numOperations / numWorkers

	for w := 0; w < numWorkers; w++ {
		go func(workerId int) {
			defer wg.Done()

			start := workerId * opsPerWorker
			end := start + opsPerWorker

			for i := start; i < end; i++ {
				op := operations[i]

				switch op.op {
				case "set":
					database.Set(op.key, op.value, 0)
				case "get":
					database.Get(op.key)
				case "delete":
					database.Delete(op.key, 0)
				}
			}
		}(w)
	}

	wg.Wait()

	if atomic.LoadInt32(&errorCount) > 0 {
		t.Fatalf("Test had %d errors during parallel operations", errorCount)
		return
	}

	var (
		dbMutex   sync.Mutex
		keyStatus = make(map[string]bool)
		keyValues = make(map[string][]byte)
		errorKeys = make(map[string]string)
	)

	var verifyWg sync.WaitGroup
	verifyWg.Add(len(allKeys))

	for key := range allKeys {
		go func(k string) {
			defer verifyWg.Done()

			_, exists := database.Get(k)

			dbMutex.Lock()
			defer dbMutex.Unlock()

			keyStatus[k] = exists

			if exists {

				value, ok := database.Get(k)
				if !ok {

					errorKeys[k] = "Key exists but Get returned false"
					return
				}

				keyValues[k] = value
			}
		}(key)
	}

	verifyWg.Wait()

	for key := range allKeys {
		_, exists := database.Get(key)

		if exists != keyStatus[key] {
			t.Errorf("Consistency error: Key %s existence changed during verification", key)
			continue
		}

		if exists {
			value, ok := database.Get(key)
			if !ok {
				t.Errorf("Consistency error: Key %s exists but could not be retrieved", key)
				continue
			}

			if !bytes.Equal(value, keyValues[key]) {
				t.Errorf("Priority mismatch for key %s between verification passes", key)
			}
		}
	}

	for key, errMsg := range errorKeys {
		t.Errorf("Error for key %s: %s", key, errMsg)
	}
}
