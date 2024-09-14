package queue

import (
	"fmt"
	"sync"
	"testing"
)

func TestQueue_IsEmpty(t *testing.T) {
	q := NewQueue()
	if !q.IsEmpty() {
		t.Errorf("Expected queue to be empty")
	}
}

func TestQueue_Clear(t *testing.T) {
	q := NewQueue()
	q.Enqueue(Item{Name: "test1"})
	q.Enqueue(Item{Name: "test2"})
	q.Clear()
	if !q.IsEmpty() {
		t.Errorf("Expected queue to be empty after Clear is called")
	}
}

func TestQueue_ConcurrentEnqueueDequeue(t *testing.T) {
	q := NewQueue()
	var wg sync.WaitGroup
	numItems := 100

	// Enqueue items concurrently
	for i := 0; i < numItems; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			q.Enqueue(Item{Name: fmt.Sprintf("item%d", i)})
		}(i)
	}

	// Dequeue items concurrently
	for i := 0; i < numItems; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.Dequeue()
		}()
	}

	wg.Wait()

	if !q.IsEmpty() {
		t.Errorf("Expected queue to be empty after concurrent Enqueue and Dequeue operations")
	}
}
