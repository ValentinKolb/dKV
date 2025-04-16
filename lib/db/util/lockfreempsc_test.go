package util

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestBasicOperations tests basic push and consume functionality
func TestBasicOperations(t *testing.T) {
	q := NewLockFreeMPSC[int]()
	defer q.Close()

	// Push 10 items
	for i := 0; i < 10; i++ {
		if !q.Push(&i) {
			t.Fatalf("Failed to push item %d", i)
		}
	}

	// Consume 10 items
	for i := 0; i < 10; i++ {
		select {
		case val := <-q.Recv():
			if *val != i {
				t.Errorf("Expected %d, got %v", i, val)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Timeout waiting for item %d", i)
		}
	}

	// Make sure queue is empty
	select {
	case val := <-q.Recv():
		t.Errorf("Queue should be empty, but got %v", val)
	case <-time.After(10 * time.Millisecond):
		// Expected timeout, queue is empty
	}
}

// TestConcurrentProducers verifies the queue works correctly with multiple producers
func TestConcurrentProducers(t *testing.T) {
	q := NewLockFreeMPSC[int]()
	defer q.Close()

	const numProducers = 10
	const itemsPerProducer = 1000
	totalItems := numProducers * itemsPerProducer

	// Use a map to track received items
	var mu sync.Mutex
	received := make(map[string]bool)

	// Start a consumer goroutine
	done := make(chan struct{})
	receivedCount := 0

	go func() {
		defer close(done)

		for receivedCount < totalItems {
			select {
			case val := <-q.Recv():

				if val == nil {
					t.Errorf("Received nil item")
					return
				}

				mu.Lock()
				key := fmt.Sprintf("%v", val)
				if received[key] {
					t.Errorf("Duplicate item received: %v", val)
				}
				received[key] = true
				receivedCount++
				mu.Unlock()
			case <-time.After(2 * time.Second):
				t.Errorf("Timeout waiting for items, received %d of %d", receivedCount, totalItems)
				return
			}
		}
	}()

	// Start producers
	var wg sync.WaitGroup
	wg.Add(numProducers)

	for p := 0; p < numProducers; p++ {
		go func(producerID int) {
			defer wg.Done()

			base := producerID * itemsPerProducer
			for i := 0; i < itemsPerProducer; i++ {
				val := base + i
				if !q.Push(&val) {
					t.Errorf("Producer %d failed to push item %d", producerID, i)
				}

				// Add some randomness to producer timing
				if i%100 == 0 {
					runtime.Gosched()
				}
			}
		}(p)
	}

	// Wait for all producers to finish
	wg.Wait()

	// Wait for consumer to process all items
	select {
	case <-done:
		// Consumer finished
	case <-time.After(5 * time.Second):
		t.Fatalf("Timeout waiting for consumer to finish")
	}

	// Verify we got all expected items
	if receivedCount != totalItems {
		t.Errorf("Expected %d items, got %d", totalItems, receivedCount)
	}
}

// TestCloseQueue verifies closing behavior
func TestCloseQueue(t *testing.T) {
	q := NewLockFreeMPSC[int]()

	// Push some items
	for i := 0; i < 5; i++ {
		q.Push(&i)
	}

	// Close the queue
	q.Close()

	// Verify we can't push after closing
	val := 100
	if q.Push(&val) {
		t.Error("Should not be able to push after queue is closed")
	}

	// Verify we can still read existing items
	for i := 0; i < 5; i++ {
		select {
		case val := <-q.Recv():
			if *val != i {
				t.Errorf("Expected %d, got %v", i, val)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Timeout waiting for item %d after close", i)
		}
	}

	// Verify the channel is closed after reading all items
	if _, ok := <-q.Recv(); ok {
		t.Error("Channel should be closed but is still open")
	}
}

// TestSelectStatement tests the queue in a select statement
func TestSelectStatement(t *testing.T) {
	q := NewLockFreeMPSC[string]()
	defer q.Close()

	// Test select with empty queue should not block when other case is ready
	otherChan := make(chan int, 1)
	otherChan <- 42

	select {
	case val := <-q.Recv():
		t.Errorf("Should not receive from empty queue, got %v", val)
	case <-otherChan:
		// Expected path
	default:
		t.Error("sel defaulted, should have received from otherChan")
	}

	// Create a fresh otherChan to ensure it's empty
	otherChan = make(chan int, 1)

	// Test select with non-empty queue
	val := "test"
	q.Push(&val)

	// The queue may need a small amount of time to process the item internally
	// Wait for the item to become available on the channel
	timeout := time.After(100 * time.Millisecond)
	var received bool

	for !received {
		select {
		case val := <-q.Recv():
			// Got the value, check it and we're done
			if *val != "test" {
				t.Errorf("Expected 'test', got %v", val)
			}
			received = true
		case <-otherChan:
			t.Error("Should have received from queue, not otherChan")
			return
		case <-timeout:
			t.Error("Timeout waiting for item from queue")
			return
		default:
			// Nothing ready yet, give the consumer goroutine a chance to run
			runtime.Gosched()
		}
	}
}

// TestOrderingUnderLoad tests that items are received in a reasonable order
// (though not guaranteed to be exactly FIFO under concurrent producers)
func TestOrderingUnderLoad(t *testing.T) {
	q := NewLockFreeMPSC[int]()
	defer q.Close()

	// Start a single producer pushing sequential numbers
	const itemCount = 10000
	go func() {
		for i := 0; i < itemCount; i++ {
			q.Push(&i)
		}
	}()

	// Consume items and verify they're roughly in order
	var prev int = -1
	outOfOrderCount := 0

	for i := 0; i < itemCount; i++ {
		select {
		case val := <-q.Recv():
			current := *val
			if current < prev {
				outOfOrderCount++
			}
			prev = current
		case <-time.After(1 * time.Second):
			t.Fatalf("Timeout waiting for item %d", i)
		}
	}

	// With a single producer, items should be in order
	if outOfOrderCount > 0 {
		t.Errorf("Found %d items out of order with single producer", outOfOrderCount)
	}
}

// TestMemoryLeak tests for memory leaks in the queue
func TestMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	q := NewLockFreeMPSC[int]()
	defer q.Close()

	// Process a large number of items
	const iterations = 1000000
	const batchSize = 1000

	// Single consumer
	var consumedCount int32
	go func() {
		for atomic.LoadInt32(&consumedCount) < iterations {
			<-q.Recv()
			atomic.AddInt32(&consumedCount, 1)
		}
	}()

	// Record memory stats before
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Push and consume many items in batches
	for i := 0; i < iterations; i += batchSize {
		// Push a batch
		for j := 0; j < batchSize; j++ {
			q.Push(&j)
		}

		// Wait for consumer to catch up
		for atomic.LoadInt32(&consumedCount) < int32(i+batchSize) {
			time.Sleep(10 * time.Millisecond)
		}

		// Force GC to clean up processed nodes
		if i%50000 == 0 {
			runtime.GC()
		}
	}

	// Get memory stats after
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Calculate the memory increase
	increase := m2.Alloc - m1.Alloc
	increasePerItem := float64(increase) / float64(iterations)

	// Allow a small amount of overhead per item processed
	const maxAllowedBytesPerItem = 2.0 // Extremely conservative
	if increasePerItem > maxAllowedBytesPerItem {
		t.Errorf("Possible memory leak: %v bytes increase per item processed", increasePerItem)
	}
}

// BenchmarkSingleProducer benchmarks the queue with a single producer
func BenchmarkSingleProducer(b *testing.B) {
	q := NewLockFreeMPSC[int]()
	defer q.Close()

	// Start consumer
	go func() {
		for range q.Recv() {
			// Just consume
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Push(&i)
	}
}

// BenchmarkMultiProducer benchmarks the queue with multiple producers
func BenchmarkMultiProducer(b *testing.B) {
	q := NewLockFreeMPSC[int]()
	defer q.Close()

	// Start consumer
	go func() {
		for range q.Recv() {
			// Just consume
		}
	}()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			q.Push(&i)
			i++
		}
	})
}
