package main

import (
	"sync"
	"testing"
	"time"
)

func TestChannelQueue_PublishAndSubscribe(t *testing.T) {
	queue := NewChannelQueue(10)

	// バッファに空きがある間は Publish が成功することを確認する。
	for i := 0; i < 3; i++ {
		if err := queue.Publish(Message{ID: "msg", Payload: "hello"}); err != nil {
			t.Fatalf("unexpected publish error: %v", err)
		}
	}

	// Subscribe したチャネルから Publish した順に取り出せることを確認する。
	ch := queue.Subscribe()
	for i := 0; i < 3; i++ {
		select {
		case msg := <-ch:
			if msg.ID != "msg" || msg.Payload != "hello" {
				t.Errorf("unexpected message: %+v", msg)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for message %d", i)
		}
	}

	queue.Close()
}

func TestChannelQueue_PublishDropsWhenFull(t *testing.T) {
	queue := NewChannelQueue(1)

	if err := queue.Publish(Message{ID: "first"}); err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}

	// バッファが一杯のときはブロックせずエラーを返す（ドロップ）。
	if err := queue.Publish(Message{ID: "second"}); err == nil {
		t.Fatal("expected an error when the queue is full")
	} else if err.Error() != "queue is full" {
		t.Errorf("unexpected error message: %q", err.Error())
	}

	queue.Close()
}

func TestChannelQueue_PublishAfterClose(t *testing.T) {
	queue := NewChannelQueue(1)
	queue.Close()

	if err := queue.Publish(Message{ID: "msg"}); err == nil {
		t.Fatal("expected an error after Close")
	} else if err.Error() != "queue is closed" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestChannelQueue_CloseIsIdempotent(t *testing.T) {
	queue := NewChannelQueue(1)

	// 2 回 Close しても二重 close パニックを起こさないことを確認する。
	queue.Close()
	queue.Close()

	// チャネルが閉じられていることも確認する（range が即座に終わる）。
	for range queue.Subscribe() {
		t.Fatal("expected the channel to be closed and empty")
	}
}

func TestChannelQueue_CloseUnblocksSubscriber(t *testing.T) {
	queue := NewChannelQueue(1)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range queue.Subscribe() {
			// ドレインするだけ。
		}
	}()

	queue.Close()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("subscriber was not unblocked by Close")
	}
}

func TestChannelQueue_ConcurrentPublishAndClose(t *testing.T) {
	// -race 下で競合がないことを担保する: 複数 goroutine が Publish しながら
	// Close しても、エラーか成功のどちらかが整然と返る。
	queue := NewChannelQueue(8)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = queue.Publish(Message{ID: "msg"}) // エラーはドロップ戦略なので無視
			}
		}()
	}

	time.Sleep(10 * time.Millisecond)
	queue.Close()
	wg.Wait()
}
