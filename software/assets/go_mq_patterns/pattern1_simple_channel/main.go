package main

import (
	"fmt"
	"sync"
	"time"
)

// Pattern 1: Simple Channel-based Message Queue
// Uses Go channels as an in-memory message queue for basic pub/sub

type Message struct {
	ID        string
	Payload   string
	Timestamp time.Time
}

type ChannelQueue struct {
	ch     chan Message
	closed bool
	mu     sync.RWMutex
}

func NewChannelQueue(bufferSize int) *ChannelQueue {
	return &ChannelQueue{
		ch: make(chan Message, bufferSize),
	}
}

func (q *ChannelQueue) Publish(msg Message) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	select {
	case q.ch <- msg:
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}

func (q *ChannelQueue) Subscribe() <-chan Message {
	return q.ch
}

func (q *ChannelQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		q.closed = true
		close(q.ch)
	}
}

func main() {
	queue := NewChannelQueue(100)

	var wg sync.WaitGroup

	// Consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for msg := range queue.Subscribe() {
			fmt.Printf("Received: ID=%s, Payload=%s\n", msg.ID, msg.Payload)
		}
		fmt.Println("Consumer stopped")
	}()

	// Producer
	for i := 0; i < 5; i++ {
		msg := Message{
			ID:        fmt.Sprintf("msg-%d", i),
			Payload:   fmt.Sprintf("Hello %d", i),
			Timestamp: time.Now(),
		}
		if err := queue.Publish(msg); err != nil {
			fmt.Printf("Publish error: %v\n", err)
		}
	}

	time.Sleep(100 * time.Millisecond)
	queue.Close()
	wg.Wait()
}
