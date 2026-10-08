package main

import (
	"testing"
	"time"
)

func TestFanoutQueue_DeliversToAllSubscribers(t *testing.T) {
	queue := NewFanoutQueue(10)

	chans := make([]<-chan Message, 3)
	for i := range chans {
		chans[i] = queue.Subscribe(string(rune('a' + i)))
	}

	// 同じメッセージが全サブスクライバに配信されることを確認する。
	msg := Message{ID: "msg-1", Payload: "broadcast"}
	queue.Publish(msg)

	for i, ch := range chans {
		select {
		case got := <-ch:
			if got.ID != msg.ID || got.Payload != msg.Payload {
				t.Errorf("subscriber %d got unexpected message: %+v", i, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d did not receive the message", i)
		}
	}

	queue.Close()
}

func TestFanoutQueue_DropsWhenSubscriberBufferFull(t *testing.T) {
	const bufferSize = 2
	queue := NewFanoutQueue(bufferSize)
	ch := queue.Subscribe("slow")

	// 誰も読み取らない状態でバッファ以上に Publish しても、
	// ブロックせずバッファ分だけ届けば残りはドロップされる。
	const published = bufferSize + 3
	for i := 0; i < published; i++ {
		queue.Publish(Message{ID: "msg"})
	}

	// Unsubscribe するとチャネルが閉じるので、残ったメッセージを数えられる。
	queue.Unsubscribe("slow")
	received := 0
	for range ch {
		received++
	}
	if received != bufferSize {
		t.Errorf("expected exactly %d buffered messages, got %d", bufferSize, received)
	}
}

func TestFanoutQueue_UnsubscribeRemovesSubscriber(t *testing.T) {
	queue := NewFanoutQueue(10)
	ch := queue.Subscribe("gone")
	queue.Unsubscribe("gone")

	// 二重 Unsubscribe してもパニックしない。
	queue.Unsubscribe("gone")

	// 解除済みサブスクライバには何も届かない（Publish が成功すればよい）。
	queue.Publish(Message{ID: "msg"})

	select {
	case msg, ok := <-ch:
		if ok {
			t.Errorf("expected a closed channel, got message %+v", msg)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("expected the channel to be closed by Unsubscribe")
	}

	queue.Close()
}

func TestFanoutQueue_CloseClosesAllSubscriberChannels(t *testing.T) {
	queue := NewFanoutQueue(10)

	chans := make([]<-chan Message, 3)
	for i := range chans {
		chans[i] = queue.Subscribe(string(rune('a' + i)))
	}

	queue.Close()

	for i, ch := range chans {
		select {
		case _, ok := <-ch:
			if ok {
				t.Errorf("subscriber %d: expected a closed channel", i)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: channel was not closed by Close", i)
		}
	}
}
