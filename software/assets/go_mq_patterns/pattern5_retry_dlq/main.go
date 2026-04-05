package main

import (
	"fmt"
	"sync"
	"time"
)

// Pattern 5: Retry with Dead Letter Queue (DLQ)
// Handles failed messages with retry logic and dead letter queue

type Message struct {
	ID        string
	Payload   string
	Attempts  int
	CreatedAt time.Time
}

type RetryConfig struct {
	MaxRetries    int
	RetryInterval time.Duration
	BackoffFactor float64
}

type DeadLetterQueue struct {
	mu       sync.Mutex
	messages []Message
}

func NewDeadLetterQueue() *DeadLetterQueue {
	return &DeadLetterQueue{
		messages: make([]Message, 0),
	}
}

func (d *DeadLetterQueue) Add(msg Message) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, msg)
}

func (d *DeadLetterQueue) GetAll() []Message {
	d.mu.Lock()
	defer d.mu.Unlock()
	result := make([]Message, len(d.messages))
	copy(result, d.messages)
	return result
}

type RetryableQueue struct {
	mainQueue  chan Message
	retryQueue chan Message
	dlq        *DeadLetterQueue
	config     RetryConfig
	handler    func(Message) error
	wg         sync.WaitGroup
	stopCh     chan struct{}
}

func NewRetryableQueue(bufferSize int, config RetryConfig, handler func(Message) error) *RetryableQueue {
	return &RetryableQueue{
		mainQueue:  make(chan Message, bufferSize),
		retryQueue: make(chan Message, bufferSize),
		dlq:        NewDeadLetterQueue(),
		config:     config,
		handler:    handler,
		stopCh:     make(chan struct{}),
	}
}

func (q *RetryableQueue) Start(numWorkers int) {
	// Main queue workers
	for i := 0; i < numWorkers; i++ {
		q.wg.Add(1)
		go q.processMain(i)
	}

	// Retry queue worker
	q.wg.Add(1)
	go q.processRetry()
}

func (q *RetryableQueue) processMain(workerID int) {
	defer q.wg.Done()
	for {
		select {
		case msg, ok := <-q.mainQueue:
			if !ok {
				return
			}
			q.handleMessage(workerID, msg)
		case <-q.stopCh:
			return
		}
	}
}

func (q *RetryableQueue) processRetry() {
	defer q.wg.Done()
	for {
		select {
		case msg, ok := <-q.retryQueue:
			if !ok {
				return
			}
			// Calculate backoff delay
			delay := time.Duration(float64(q.config.RetryInterval) *
				float64(msg.Attempts) * q.config.BackoffFactor)

			select {
			case <-time.After(delay):
				q.handleMessage(-1, msg)
			case <-q.stopCh:
				return
			}
		case <-q.stopCh:
			return
		}
	}
}

func (q *RetryableQueue) handleMessage(workerID int, msg Message) {
	msg.Attempts++

	if err := q.handler(msg); err != nil {
		if msg.Attempts >= q.config.MaxRetries {
			fmt.Printf("Message %s exceeded max retries, moving to DLQ\n", msg.ID)
			q.dlq.Add(msg)
		} else {
			fmt.Printf("Message %s failed (attempt %d/%d), scheduling retry\n",
				msg.ID, msg.Attempts, q.config.MaxRetries)
			q.retryQueue <- msg
		}
	} else {
		fmt.Printf("Message %s processed successfully\n", msg.ID)
	}
}

func (q *RetryableQueue) Publish(msg Message) {
	q.mainQueue <- msg
}

func (q *RetryableQueue) GetDLQ() *DeadLetterQueue {
	return q.dlq
}

func (q *RetryableQueue) Stop() {
	close(q.stopCh)
	close(q.mainQueue)
	close(q.retryQueue)
	q.wg.Wait()
}

func main() {
	// Simulate a handler that fails for certain messages
	failCount := 0
	handler := func(msg Message) error {
		if msg.ID == "msg-2" {
			failCount++
			if failCount <= 4 { // Fail 4 times, exceed max retries
				return fmt.Errorf("simulated failure")
			}
		}
		if msg.ID == "msg-3" && msg.Attempts == 1 {
			return fmt.Errorf("temporary failure")
		}
		return nil
	}

	config := RetryConfig{
		MaxRetries:    3,
		RetryInterval: 100 * time.Millisecond,
		BackoffFactor: 1.5,
	}

	queue := NewRetryableQueue(10, config, handler)
	queue.Start(2)

	// Publish messages
	for i := 0; i < 5; i++ {
		queue.Publish(Message{
			ID:        fmt.Sprintf("msg-%d", i),
			Payload:   fmt.Sprintf("Payload %d", i),
			CreatedAt: time.Now(),
		})
	}

	// Wait for processing
	time.Sleep(2 * time.Second)
	queue.Stop()

	// Check DLQ
	dlqMessages := queue.GetDLQ().GetAll()
	fmt.Printf("\nDead Letter Queue contains %d messages:\n", len(dlqMessages))
	for _, msg := range dlqMessages {
		fmt.Printf("  - %s (attempts: %d)\n", msg.ID, msg.Attempts)
	}
}
