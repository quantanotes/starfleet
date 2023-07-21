package main

import (
	"context"
	"fmt"
	"testing"
)

func TestWorkerPool(t *testing.T) {
	ctx := context.Background()

	simWorker1 := simulateWorker()
	simWorker2 := simulateWorker()
	defer simWorker1.Close()
	defer simWorker2.Close()

	fmt.Printf("host 1: %s host 2: %s\n", simWorker1.URL, simWorker2.URL)

	workerPool := NewWorkerPool(WorkerPoolConfig{
		WorkerConfig{Host: simWorker1.URL, Capacity: 2},
		WorkerConfig{Host: simWorker2.URL, Capacity: 1},
	})
	go workerPool.Run()

	job1 := NewJob(ctx, "1", nil)
	job2 := NewJob(ctx, "2", nil)
	job3 := NewJob(ctx, "3", nil)

	workerPool.Enlist(job1)
	workerPool.Enlist(job2)
	workerPool.Enlist(job3)

	for {
		select {
		case token := <-job1.Output:
			fmt.Printf("job 1: %s\n", token)
		case token := <-job2.Output:
			fmt.Printf("job 2: %s\n", token)
		case token := <-job3.Output:
			fmt.Printf("job 3: %s\n", token)
		case <-job1.Done:
			fmt.Println("job 1 done")
		case <-job2.Done:
			fmt.Println("job 2 done")
		case <-job3.Done:
			fmt.Println("job 3 done")
			return
		}
	}
}
