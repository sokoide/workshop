package main

import (
	"fmt"
	"sync"
	"time"
)

// Pattern 3: Worker Pool with Message Queue
// Distributes work across multiple workers for parallel processing

type Job struct {
	ID      string
	Payload string
}

type Result struct {
	JobID  string
	Output string
	Error  error
}

type WorkerPool struct {
	jobs       chan Job
	results    chan Result
	numWorkers int
	wg         sync.WaitGroup
}

func NewWorkerPool(numWorkers, jobBuffer, resultBuffer int) *WorkerPool {
	return &WorkerPool{
		jobs:       make(chan Job, jobBuffer),
		results:    make(chan Result, resultBuffer),
		numWorkers: numWorkers,
	}
}

func (p *WorkerPool) Start(processor func(Job) Result) {
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go func(workerID int) {
			defer p.wg.Done()
			for job := range p.jobs {
				fmt.Printf("Worker %d processing job %s\n", workerID, job.ID)
				result := processor(job)
				p.results <- result
			}
		}(i)
	}
}

func (p *WorkerPool) Submit(job Job) {
	p.jobs <- job
}

func (p *WorkerPool) Results() <-chan Result {
	return p.results
}

func (p *WorkerPool) Close() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

func main() {
	pool := NewWorkerPool(3, 10, 10)

	// Define job processor
	processor := func(job Job) Result {
		// Simulate work
		time.Sleep(50 * time.Millisecond)
		return Result{
			JobID:  job.ID,
			Output: fmt.Sprintf("Processed: %s", job.Payload),
		}
	}

	pool.Start(processor)

	// Collect results in background
	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		for result := range pool.Results() {
			if result.Error != nil {
				fmt.Printf("Job %s failed: %v\n", result.JobID, result.Error)
			} else {
				fmt.Printf("Job %s completed: %s\n", result.JobID, result.Output)
			}
		}
	}()

	// Submit jobs
	for i := 0; i < 10; i++ {
		pool.Submit(Job{
			ID:      fmt.Sprintf("job-%d", i),
			Payload: fmt.Sprintf("Task %d", i),
		})
	}

	pool.Close()
	resultWg.Wait()
}
