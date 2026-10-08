package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// fastConfig はテスト用の短いリトライ間隔。実運用の設定値とは別に、
// バックオフの倍率だけを担保対象にする。
func fastConfig(maxRetries int) RetryConfig {
	return RetryConfig{
		MaxRetries:    maxRetries,
		RetryInterval: time.Millisecond,
		BackoffFactor: 1.5,
	}
}

func TestRetryableQueue_SuccessOnFirstAttempt(t *testing.T) {
	attempts := make(chan int, 4)
	handler := func(msg Message) error {
		attempts <- msg.Attempts
		return nil
	}

	queue := NewRetryableQueue(10, fastConfig(3), handler)
	queue.Start(1)

	queue.Publish(Message{ID: "msg-ok", Payload: "hello"})

	select {
	case got := <-attempts:
		if got != 1 {
			t.Errorf("handler should see Attempts=1 on first try, got %d", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("message was not processed")
	}

	// Stop 後に DLQ が空であることを確認する。
	queue.Stop()
	if dlq := queue.GetDLQ().GetAll(); len(dlq) != 0 {
		t.Errorf("expected empty DLQ, got %v", dlq)
	}
}

func TestRetryableQueue_ExhaustedRetriesGoToDLQ(t *testing.T) {
	const maxRetries = 3
	attempts := make(chan int, maxRetries)
	handler := func(msg Message) error {
		attempts <- msg.Attempts
		return fmt.Errorf("permanent failure")
	}

	queue := NewRetryableQueue(10, fastConfig(maxRetries), handler)
	queue.Start(1)

	queue.Publish(Message{ID: "msg-bad"})

	// MaxRetries 回呼ばれるのを待つ（1 回目 + リトライ 2 回）。
	for i := 0; i < maxRetries; i++ {
		select {
		case <-attempts:
		case <-time.After(2 * time.Second):
			t.Fatalf("handler was not called for attempt %d", i+1)
		}
	}
	queue.Stop()

	dlq := queue.GetDLQ().GetAll()
	if len(dlq) != 1 {
		t.Fatalf("expected 1 message in DLQ, got %d", len(dlq))
	}
	if dlq[0].ID != "msg-bad" {
		t.Errorf("unexpected DLQ message: %+v", dlq[0])
	}
	if dlq[0].Attempts != maxRetries {
		t.Errorf("expected Attempts=%d in DLQ, got %d", maxRetries, dlq[0].Attempts)
	}
}

func TestRetryableQueue_RecoversOnRetry(t *testing.T) {
	const maxRetries = 3
	attempts := make(chan int, maxRetries)
	handler := func(msg Message) error {
		attempts <- msg.Attempts
		if msg.Attempts == 1 {
			return fmt.Errorf("temporary failure")
		}
		return nil // 2 回目で成功
	}

	queue := NewRetryableQueue(10, fastConfig(maxRetries), handler)
	queue.Start(1)

	queue.Publish(Message{ID: "msg-flaky"})

	seen := 0
	deadline := time.After(2 * time.Second)
	for seen < 2 {
		select {
		case got := <-attempts:
			seen++
			if got != seen {
				t.Errorf("attempt counter should increment, got %d at call %d", got, seen)
			}
		case <-deadline:
			t.Fatalf("handler was called only %d times", seen)
		}
	}

	// リトライで成功したメッセージは DLQ に入らない。
	queue.Stop()
	if dlq := queue.GetDLQ().GetAll(); len(dlq) != 0 {
		t.Errorf("recovered message should not be in DLQ, got %v", dlq)
	}
}

func TestRetryableQueue_ConcurrentWorkersNoRace(t *testing.T) {
	// -race 下で、複数ワーカーが並行処理しても競合しないことを担保する。
	// 半分のメッセージは常に失敗させ、DLQ 行きの経路も並行で通す。
	const numMessages = 20
	const maxRetries = 2

	// 偶数番号のメッセージは常に失敗させ、DLQ 行きの経路も並行で通す。
	alwaysFail := make(map[string]bool, numMessages/2)
	ids := make([]string, numMessages)
	for i := range ids {
		ids[i] = fmt.Sprintf("msg-%d", i)
		if i%2 == 0 {
			alwaysFail[ids[i]] = true
		}
	}

	var mu sync.Mutex
	seen := make(map[string]int)
	handler := func(msg Message) error {
		mu.Lock()
		seen[msg.ID]++
		mu.Unlock()
		if alwaysFail[msg.ID] {
			return fmt.Errorf("permanent failure")
		}
		return nil
	}

	queue := NewRetryableQueue(numMessages, fastConfig(maxRetries), handler)
	queue.Start(3)

	for _, id := range ids {
		queue.Publish(Message{ID: id})
	}

	// 全メッセージが MaxRetries 回処理されるのを待つ。
	deadline := time.Now().Add(5 * time.Second)
	for {
		mu.Lock()
		total := 0
		for _, id := range ids {
			total += seen[id]
		}
		mu.Unlock()
		if total >= numMessages+numMessages/2 || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	queue.Stop()

	mu.Lock()
	defer mu.Unlock()
	for i, id := range ids {
		wantCalls := 1
		if i%2 == 0 {
			wantCalls = maxRetries
		}
		if seen[id] != wantCalls {
			t.Errorf("message %s: expected %d handler calls, got %d", id, wantCalls, seen[id])
		}
	}

	dlq := queue.GetDLQ().GetAll()
	if len(dlq) != numMessages/2 {
		t.Errorf("expected %d messages in DLQ, got %d", numMessages/2, len(dlq))
	}
}

func TestDeadLetterQueue_GetAllReturnsCopy(t *testing.T) {
	dlq := NewDeadLetterQueue()
	dlq.Add(Message{ID: "a"})
	dlq.Add(Message{ID: "b"})

	// GetAll は内部スライスのコピーを返すので、
	// 呼び出し側での破壊的操作が DLQ 本体に影響しない。
	got := dlq.GetAll()
	got[0].ID = "mutated"

	if again := dlq.GetAll(); again[0].ID != "a" {
		t.Errorf("GetAll must return a copy, got %q after mutation", again[0].ID)
	}
}
