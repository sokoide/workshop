package main

import (
	"fmt"
	"sync"
	"time"
)

// Pattern 2: Fan-out Message Queue
// Broadcasts messages to multiple subscribers

type Message struct {
	ID      string
	Payload string
}

type Subscriber struct {
	id string
	ch chan Message
}

type FanoutQueue struct {
	mu          sync.RWMutex
	subscribers map[string]*Subscriber
	bufferSize  int
}

func NewFanoutQueue(bufferSize int) *FanoutQueue {
	return &FanoutQueue{
		subscribers: make(map[string]*Subscriber),
		bufferSize:  bufferSize,
	}
}

func (q *FanoutQueue) Subscribe(id string) <-chan Message {
	q.mu.Lock()
	defer q.mu.Unlock()

	sub := &Subscriber{
		id: id,
		ch: make(chan Message, q.bufferSize),
	}
	q.subscribers[id] = sub
	return sub.ch
}

func (q *FanoutQueue) Unsubscribe(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if sub, ok := q.subscribers[id]; ok {
		delete(q.subscribers, id)
		close(sub.ch)
	}
}

func (q *FanoutQueue) Publish(msg Message) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, sub := range q.subscribers {
		select {
		case sub.ch <- msg:
		default:
			fmt.Printf("Subscriber %s buffer full, dropping message\n", sub.id)
		}
	}
}

func (q *FanoutQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	for id, sub := range q.subscribers {
		delete(q.subscribers, id)
		close(sub.ch)
	}
}

func main() {
	queue := NewFanoutQueue(10)
	var wg sync.WaitGroup

	// Multiple consumers
	for i := 0; i < 3; i++ {
		subID := fmt.Sprintf("consumer-%d", i)
		ch := queue.Subscribe(subID)
		wg.Add(1)
		go func(id string, msgCh <-chan Message) {
			defer wg.Done()
			for msg := range msgCh {
				fmt.Printf("[%s] Received: %s - %s\n", id, msg.ID, msg.Payload)
			}
		}(subID, ch)
	}

	// Producer
	for i := 0; i < 3; i++ {
		msg := Message{
			ID:      fmt.Sprintf("msg-%d", i),
			Payload: fmt.Sprintf("Broadcast message %d", i),
		}
		queue.Publish(msg)
	}

	time.Sleep(100 * time.Millisecond)
	queue.Close()
	wg.Wait()
}
