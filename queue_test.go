package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestQueue(t *testing.T) {
	q := NewQueue(2)
	var wg sync.WaitGroup
	wg.Add(4)

	simulateQueueJob("job1", &q, &wg)
	simulateQueueJob("job2", &q, &wg)
	simulateQueueJobWithCancel("job3", &q, &wg)
	simulateQueueJob("job4", &q, &wg)

	wg.Wait()
}

func simulateQueueJob(id string, q *Queue, wg *sync.WaitGroup) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer wg.Done()
		defer cancel()

		fmt.Printf("Starting job %s\n", id)

		t := time.Now()
		q.Wait(ctx, id)

		fmt.Printf("%s finished queuing after %v\n", id, time.Since(t))

		time.Sleep(1 * time.Second)

		fmt.Printf("Finishing job %s\n", id)
	}()
	return ctx, cancel
}

func simulateQueueJobWithCancel(id string, q *Queue, wg *sync.WaitGroup) {
	_, cancel := simulateQueueJob(id, q, wg)
	time.Sleep(100 * time.Millisecond)
	cancel()
}
