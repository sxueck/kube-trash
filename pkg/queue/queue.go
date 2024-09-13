package queue

import (
	"container/list"
	"sync"
)

// Item represents an item in the queue
type Item struct {
	Data      []byte
	Kind      string
	Name      string
	Namespace string
}

// Queue represents a thread safe queue of QueueItem
type Queue struct {
	lock sync.Mutex
	list *list.List
}

// NewQueue Creates and returns a new Queue
func NewQueue() *Queue {
	return &Queue{
		list: list.New(),
	}
}

// Enqueue Adds a QueueItem to the end of the queue
func (q *Queue) Enqueue(item Item) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.list.PushBack(item)
}

// Dequeue removes and returns the QueueItem at the front of the queue
func (q *Queue) Dequeue() (Item, bool) {
	q.lock.Lock()
	defer q.lock.Unlock()
	if q.list.Len() == 0 {
		return Item{}, false
	}
	element := q.list.Front()
	q.list.Remove(element)
	return element.Value.(Item), true
}

// Peek returns the QueueItem at the front of the queue without removing it
func (q *Queue) Peek() (Item, bool) {
	q.lock.Lock()
	defer q.lock.Unlock()
	if q.list.Len() == 0 {
		return Item{}, false
	}
	return q.list.Front().Value.(Item), true
}

// Size returns the number of items in the queue
func (q *Queue) Size() int {
	q.lock.Lock()
	defer q.lock.Unlock()
	return q.list.Len()
}

// IsEmpty returns true if the queue is empty
func (q *Queue) IsEmpty() bool {
	q.lock.Lock()
	defer q.lock.Unlock()
	return q.list.Len() == 0
}

// Clear removes all items from the queue
func (q *Queue) Clear() {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.list.Init()
}
