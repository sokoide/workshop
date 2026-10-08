package main

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestWorkerPool_CompletesAllJobs(t *testing.T) {
	const numJobs = 20
	pool := NewWorkerPool(3, numJobs, numJobs)

	processor := func(job Job) Result {
		return Result{JobID: job.ID, Output: fmt.Sprintf("Processed: %s", job.Payload)}
	}

	pool.Start(processor)

	// 複数 worker で並列処理しても、ジョブが 1 つも失われないことを確認する。
	var wg sync.WaitGroup
	wg.Add(1)
	results := make(map[string]Result)
	go func() {
		defer wg.Done()
		for r := range pool.Results() {
			results[r.JobID] = r
		}
	}()

	for i := 0; i < numJobs; i++ {
		pool.Submit(Job{ID: fmt.Sprintf("job-%d", i), Payload: fmt.Sprintf("Task %d", i)})
	}

	// Close は全ジョブ完了後に results チャネルを閉じる（graceful shutdown）。
	pool.Close()
	wg.Wait()

	if len(results) != numJobs {
		t.Fatalf("expected %d results, got %d", numJobs, len(results))
	}
	for i := 0; i < numJobs; i++ {
		id := fmt.Sprintf("job-%d", i)
		r, ok := results[id]
		if !ok {
			t.Errorf("job %s produced no result", id)
			continue
		}
		if r.Error != nil {
			t.Errorf("job %s: unexpected error: %v", id, r.Error)
		}
		if r.Output != fmt.Sprintf("Processed: Task %d", i) {
			t.Errorf("job %s: unexpected output: %q", id, r.Output)
		}
	}
}

func TestWorkerPool_PropagatesProcessorErrors(t *testing.T) {
	pool := NewWorkerPool(2, 5, 5)

	sentinel := errors.New("boom")
	processor := func(job Job) Result {
		return Result{JobID: job.ID, Error: sentinel}
	}

	pool.Start(processor)
	pool.Submit(Job{ID: "bad"})
	pool.Close()

	select {
	case r := <-pool.Results():
		if !errors.Is(r.Error, sentinel) {
			t.Errorf("expected the processor error, got %v", r.Error)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the result")
	}
}

func TestWorkerPool_CloseWaitsForInFlightJobs(t *testing.T) {
	const numJobs = 6
	pool := NewWorkerPool(2, numJobs, numJobs)

	started := make(chan struct{}, numJobs)
	processor := func(job Job) Result {
		started <- struct{}{}
		time.Sleep(20 * time.Millisecond) // 処理中に Close が呼ばれる状況を作る
		return Result{JobID: job.ID}
	}

	pool.Start(processor)
	for i := 0; i < numJobs; i++ {
		pool.Submit(Job{ID: fmt.Sprintf("job-%d", i)})
	}

	<-started // 最低 1 ジョブが始まった時点でシャットダウンする。
	pool.Close()

	// Close の戻り時点で全結果が読み取れること = 実行中ジョブの完了を待つこと。
	received := 0
	for range pool.Results() {
		received++
	}
	if received != numJobs {
		t.Errorf("expected %d results after Close, got %d", numJobs, received)
	}
}
