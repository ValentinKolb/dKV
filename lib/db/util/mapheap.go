// Package util
//
// This file provides a specialized priority queue for garbage collection purposes.
//
// This implementation combines a binary heap with a hash map to provide both
// efficient priority-based operations and key-based access. It's particularly
// useful for garbage collection scenarios where items need to be prioritized
// by age or other metrics, while still allowing direct access to specific items.
//
// Key advantages of this implementation:
//
// 1. Time Complexity:
//   - O(log n) for priority operations (Push, Pop, Update)
//   - O(1) for key-based lookups and existence checks
//   - O(log n) for key-based removal
//
// 2. Garbage Collection Benefits:
//   - Efficiently identifies the oldest/lowest-priority items for collection
//   - Supports direct removal when items are manually freed
//   - Allows checking if specific items are scheduled for collection
//   - Can update priorities when items are accessed (for LRU-like behaviors)
//
// 3. Memory Efficiency:
//   - Avoids memory leaks by properly cleaning up references
//   - Minimizes overhead with a compact representation
//   - Scales efficiently for large numbers of objects
//
// 4. Concurrency Considerations:
//   - Note: This implementation is not thread-safe by default
//   - For concurrent use, external synchronization should be applied
//
// Use cases include memory management systems, caches with expiration,
// connection pools, and any scenario requiring both prioritization and
// direct access to elements.
//
// Example usage:
//
//	// Create a new queue
//	gcQueue := NewMapHeap()
//	heap.Init(gcQueue)
//
//	// Add items with object IDs and timestamps
//	gcQueue.AddItem(1001, timestamp1)
//	gcQueue.AddItem(1002, timestamp2)
//
//	// Get the oldest item
//	oldest, exists := gcQueue.Peek()
//
//	// Remove a specific item (e.g., when manually freed)
//	gcQueue.RemoveByKey(1001)
//
//	// Process items in priority order
//	for gcQueue.Len() > 0 {
//	    item := heap.Pop(gcQueue).(*item)
//	    // Process item for garbage collection
//	}
package util

import (
	"container/heap"
	"strconv"
)

// item represents an item in our garbage collection queue
// with a uint64 key for identification and uint64 value for priority
type item struct {
	Key      uint64 // Unique identifier for the item
	Priority uint64 // Priority used for priority in the heap
	index    int    // Index in the heap, maintained by heap package
}

func (i *item) String() string {
	return "{Key: " + strconv.FormatUint(i.Key, 10) + ", Priority: " + strconv.FormatUint(i.Priority, 10) + "}"
}

// MapHeap implements a priority queue for garbage collection
// with both heap operations and key-based access
type MapHeap struct {
	items    []*item          // The actual heap slice
	itemsMap map[uint64]*item // Map for O(1) access by key
}

// NewMapHeap creates a new garbage collection queue
func NewMapHeap() *MapHeap {
	return &MapHeap{
		items:    make([]*item, 0),
		itemsMap: make(map[uint64]*item),
	}
}

// Len returns the number of items in the queue (part of heap.Interface)
func (gcq *MapHeap) Len() int { return len(gcq.items) }

// Less compares items by value (part of heap.Interface)
// For GC, typically we want oldest items first (min-heap by timestamp)
func (gcq *MapHeap) Less(i, j int) bool {
	return gcq.items[i].Priority < gcq.items[j].Priority
}

// Swap exchanges items at positions i and j (part of heap.Interface)
func (gcq *MapHeap) Swap(i, j int) {
	gcq.items[i], gcq.items[j] = gcq.items[j], gcq.items[i]
	gcq.items[i].index = i
	gcq.items[j].index = j
}

// Push adds an item to the heap (part of heap.Interface)
func (gcq *MapHeap) Push(x interface{}) {
	n := len(gcq.items)
	item := x.(*item)
	item.index = n
	gcq.items = append(gcq.items, item)
	gcq.itemsMap[item.Key] = item
}

// Pop removes and returns the minimum item (part of heap.Interface)
func (gcq *MapHeap) Pop() interface{} {
	old := gcq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // Avoid memory leak
	item.index = -1 // For safety
	gcq.items = old[:n-1]
	delete(gcq.itemsMap, item.Key)
	return item
}

// AddItem adds a new item to the queue or updates existing one
func (gcq *MapHeap) AddItem(key, priority uint64) {
	// Check if item already exists
	if item, exists := gcq.itemsMap[key]; exists {
		// Update priority and fix heap
		item.Priority = priority
		heap.Fix(gcq, item.index)
		return
	}

	// Create and add new item
	item := &item{
		Key:      key,
		Priority: priority,
	}
	heap.Push(gcq, item)
}

// RemoveByKey removes an item by its key
func (gcq *MapHeap) RemoveByKey(key uint64) (uint64, bool) {
	item, exists := gcq.itemsMap[key]
	if !exists {
		return 0, false
	}

	// Remove from heap
	heap.Remove(gcq, item.index)
	return item.Priority, true
}

// Peek returns the minimum value item without removing it
func (gcq *MapHeap) Peek() (*item, bool) {
	if len(gcq.items) == 0 {
		return nil, false
	}
	return gcq.items[0], true
}

// Contains checks if a key exists in the queue
func (gcq *MapHeap) Contains(key uint64) bool {
	_, exists := gcq.itemsMap[key]
	return exists
}

// GetByKey retrieves an item by its key without removing it
func (gcq *MapHeap) GetByKey(key uint64) (*item, bool) {
	item, exists := gcq.itemsMap[key]
	return item, exists
}
