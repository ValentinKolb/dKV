package util

import (
	"container/heap"
	"sort"
	"testing"
)

// TestNewMapHeap tests the creation of a new MapHeap
func TestNewMapHeap(t *testing.T) {
	mh := NewMapHeap()

	if mh == nil {
		t.Fatal("NewMapHeap() returned nil")
	}

	if mh.Len() != 0 {
		t.Errorf("New heap should be empty, but has length %d", mh.Len())
	}

	if len(mh.itemsMap) != 0 {
		t.Errorf("New heap's map should be empty, but has %d items", len(mh.itemsMap))
	}
}

// TestAddItem tests adding items to the heap
func TestAddItem(t *testing.T) {
	mh := NewMapHeap()
	heap.Init(mh)

	// Add a few items
	mh.AddItem(1, 100)
	mh.AddItem(2, 200)
	mh.AddItem(3, 50)

	if mh.Len() != 3 {
		t.Errorf("Heap should have 3 items, but has %d", mh.Len())
	}

	// Check if items exist
	if !mh.Contains(1) {
		t.Error("Heap should contain key 1")
	}

	if !mh.Contains(2) {
		t.Error("Heap should contain key 2")
	}

	if !mh.Contains(3) {
		t.Error("Heap should contain key 3")
	}

	// Check the order (min heap, so the lowest value should be first)
	item, exists := mh.Peek()
	if !exists {
		t.Fatal("Peek() should return an item")
	}

	if item.Key != 3 || item.Priority != 50 {
		t.Errorf("Expected min item to be (3,50), got (%d,%d)", item.Key, item.Priority)
	}
}

// TestUpdateItem tests updating existing items
func TestUpdateItem(t *testing.T) {
	mh := NewMapHeap()
	heap.Init(mh)

	// Add items
	mh.AddItem(1, 100)
	mh.AddItem(2, 200)

	// Update an item
	mh.AddItem(1, 300) // Increase priority of item 1

	// Check if update worked
	item, exists := mh.GetByKey(1)
	if !exists {
		t.Fatal("Item with key 1 should exist")
	}

	if item.Priority != 300 {
		t.Errorf("Item with key 1 should have value 300, got %d", item.Priority)
	}

	// Check if heap property is maintained
	min, _ := mh.Peek()
	if min.Key != 2 {
		t.Errorf("Min item should now be key 2, got %d", min.Key)
	}

	// Update to lower value
	mh.AddItem(2, 50)

	min, _ = mh.Peek()
	if min.Key != 2 || min.Priority != 50 {
		t.Errorf("Min item should now be (2,50), got (%d,%d)", min.Key, min.Priority)
	}
}

// TestRemoveByKey tests removing items by key
func TestRemoveByKey(t *testing.T) {
	mh := NewMapHeap()
	heap.Init(mh)

	mh.AddItem(1, 100)
	mh.AddItem(2, 200)
	mh.AddItem(3, 300)

	// Remove item with key 2
	value, exists := mh.RemoveByKey(2)

	if !exists {
		t.Fatal("RemoveByKey should return true for existing key")
	}

	if value != 200 {
		t.Errorf("RemoveByKey should return value 200, got %d", value)
	}

	if mh.Len() != 2 {
		t.Errorf("Heap should have 2 items after removal, has %d", mh.Len())
	}

	if mh.Contains(2) {
		t.Error("Heap should not contain key 2 after removal")
	}

	// Try to remove non-existent key
	_, exists = mh.RemoveByKey(99)
	if exists {
		t.Error("RemoveByKey should return false for non-existent key")
	}
}

// TestPopOrder tests if items are popped in correct order
func TestPopOrder(t *testing.T) {
	mh := NewMapHeap()
	heap.Init(mh)

	// Add items in random order
	items := []struct {
		key   uint64
		value uint64
	}{
		{5, 50},
		{3, 30},
		{1, 10},
		{4, 40},
		{2, 20},
	}

	for _, item := range items {
		mh.AddItem(item.key, item.value)
	}

	// Sort the items for comparison
	sort.Slice(items, func(i, j int) bool {
		return items[i].value < items[j].value
	})

	// Pop all items and verify order
	for i, expected := range items {
		if mh.Len() == 0 {
			t.Fatalf("Heap empty after %d items, expected %d items", i, len(items))
		}

		item := heap.Pop(mh).(*item)
		if item.Key != expected.key || item.Priority != expected.value {
			t.Errorf("Pop %d: expected (%d,%d), got (%d,%d)",
				i, expected.key, expected.value, item.Key, item.Priority)
		}
	}

	if mh.Len() != 0 {
		t.Errorf("Heap should be empty after popping all items, has %d items", mh.Len())
	}
}

// TestPeekEmptyHeap tests behavior when peeking an empty heap
func TestPeekEmptyHeap(t *testing.T) {
	mh := NewMapHeap()
	heap.Init(mh)

	_, exists := mh.Peek()
	if exists {
		t.Error("Peek on empty heap should return exists=false")
	}
}

// TestGetByKey tests retrieving items by key
func TestGetByKey(t *testing.T) {
	mh := NewMapHeap()
	heap.Init(mh)

	mh.AddItem(1, 100)
	mh.AddItem(2, 200)

	// Get existing item
	item, exists := mh.GetByKey(1)
	if !exists {
		t.Fatal("GetByKey should find existing key")
	}

	if item.Key != 1 || item.Priority != 100 {
		t.Errorf("GetByKey returned incorrect item: expected (1,100), got (%d,%d)",
			item.Key, item.Priority)
	}

	// Get non-existent item
	_, exists = mh.GetByKey(99)
	if exists {
		t.Error("GetByKey should return exists=false for non-existent key")
	}
}

// TestLargeNumberOfIt
