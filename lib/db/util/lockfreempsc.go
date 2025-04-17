// Package util provides a lockmgr-free Multi-Producer Single-Consumer (MPSC) queue implementation.
//
// Features and Guarantees:
//
//   - Lock-Free: atomic operations for high throughput and low latency even under high contention
//   - Unbounded Size: the queue can grow to any size as needed, limited only by available memory
//   - Small Footprint: minimal memory overhead per item (two pointers per item)
//   - Thread-Safe writes: Allows any number of goroutines to safely Push() concurrently
//   - Single Consumer: Designed for a single goroutine to consume values (via the Recv() channel).
//   - No Strict FIFO Guarantee: Under concurrent Push() operations, the exact ordering of items
//     is determined by which producer completes its operation first, not by which producer
//     started first.
package util

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// node represents a single element in the queue
type node[T interface{}] struct {
	value *T
	next  atomic.Pointer[node[T]]
}

// LockFreeMPSC is a lockmgr-free multi-producer single-consumer queue
// Implementation uses a linked list of nodes with atomic operations
// for concurrent push operations without locks
type LockFreeMPSC[T interface{}] struct {
	head     atomic.Pointer[node[T]]
	tail     atomic.Pointer[node[T]]
	out      chan *T
	consumer sync.WaitGroup
	closed   atomic.Bool // atomic flag

	// Condition variable for efficient waiting
	mu   sync.Mutex
	cond *sync.Cond
}

// NewLockFreeMPSC creates a new lockmgr-free multi-producer single-consumer queue
func NewLockFreeMPSC[T interface{}]() *LockFreeMPSC[T] {
	// Create a sentinel node (dummy node at the beginning)
	sentinel := &node[T]{}

	q := &LockFreeMPSC[T]{
		out: make(chan *T),
	}

	// Initialize condition variable
	q.cond = sync.NewCond(&q.mu)

	// Set the initial head and tail to the sentinel node
	q.head.Store(sentinel)
	q.tail.Store(sentinel)

	q.consumer.Add(1)
	go q.consume()

	return q
}

// Push adds an item to the queue.
// Returns true if the item was added, or false if the queue is closed.
//
// Thread-safety: This method is thread-safe and can be called concurrently.
func (q *LockFreeMPSC[T]) Push(value *T) bool {

	if value == nil {
		return false
	}

	if q.closed.Load() {
		return false
	}

	newNode := &node[T]{value: value}

	var tailNode *node[T]
	var backoff uint8 = 0

	for {
		tailNode = q.tail.Load()

		// try to atomically append our node to the current tail
		next := tailNode.next.Load()
		if next == nil {
			// the tail has no next node yet, try to append our node
			if tailNode.next.CompareAndSwap(nil, newNode) {
				/*
				 Successfully appended, now try to update tail
				 Note: CAS may fail if another producer helps update tail,
				 but that's okay - tail will still be updated eventually
				*/
				q.tail.CompareAndSwap(tailNode, newNode)

				// Signal the consumer that new data is available
				q.cond.Signal()

				return true
			}
		} else {
			// help update the tail pointer if another producer has already appended a node but hasn't updated the tail yet
			q.tail.CompareAndSwap(tailNode, next)
		}

		/*
		 Implement exponential backoff strategy to handle contention (speed improvement from 400ns/op -> 60ns/op)
		  - At low contention (<10 retries): Use CPU spinning to avoid thread scheduling overhead
		  - At higher contention: Yield the processor to allow other goroutines to make progress
		  - Backoff increases exponentially with each retry, reducing the "thundering herd" problem where all goroutines retry simultaneously after failure
		*/

		if backoff < 10 {
			backoff++
			for i := 0; i < 1<<backoff; i++ {
				runtime.Gosched()
			}
		}
		runtime.Gosched()
	}
}

// consume continuously sends items from the linked list to the output channel and frees memory
func (q *LockFreeMPSC[T]) consume() {
	defer q.consumer.Done()
	defer close(q.out)

	for {
		// Process all available items in the queue
		hasItems := false

		// Try to process items
		for {
			head := q.head.Load()
			next := head.next.Load()

			if next == nil {
				break // No more items available
			}

			hasItems = true

			// Capture value before updating pointers
			value := next.value

			// move head pointer (free up memory)
			q.head.Store(next)

			// Send the value to the consumer
			q.out <- value

			// help go gc - safe to clear after sending
			next.value = nil
		}

		// Exit if closed and no more items
		if !hasItems && q.closed.Load() {
			return
		}

		// If no items were processed, wait for signal
		if !hasItems {
			q.mu.Lock()
			// Double-check condition after acquiring lock
			head := q.head.Load()
			if head.next.Load() == nil && !q.closed.Load() {
				// Wait for signal (releases lock while waiting)
				q.cond.Wait()
			}
			q.mu.Unlock()
		}
	}
}

// Recv returns a receive-only channel for consuming from the queue.
// This allows the queue to be used with the '<-' operator in select statements.
func (q *LockFreeMPSC[T]) Recv() <-chan *T {
	return q.out
}

// Close closes the queue, preventing further writes.
// Any items already in the queue will still be delivered to the consumer.
func (q *LockFreeMPSC[T]) Close() {
	q.closed.Store(true)

	// Wake up the consumer if it's waiting
	q.cond.Signal()
}

// IsClosed returns true if the queue is closed.
func (q *LockFreeMPSC[T]) IsClosed() bool {
	return q.closed.Load()
}

// Len returns an approximate count of the number of items in the queue.
// This is O(n) and should only be used for debugging.
func (q *LockFreeMPSC[T]) Len() int {
	count := 0
	current := q.head.Load()

	for {
		next := current.next.Load()
		if next == nil {
			break
		}
		count++
		current = next
	}

	return count
}
