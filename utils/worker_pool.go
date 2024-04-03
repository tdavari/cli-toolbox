package utils

import (
	"sync"
)

// Task definition
type Task interface {
	Process()
}

// Worker pool definition
type WorkerPool[T Task] struct {
	Tasks       []T
	Concurrency int
	tasksChan   chan T
	wg          sync.WaitGroup
}

// Functions to execute the worker pool

func (wp *WorkerPool[T]) worker() {
	for task := range wp.tasksChan {
		task.Process()
		wp.wg.Done()
	}
}

func (wp *WorkerPool[T]) Run() {
	// Initialize the tasks channel
	wp.tasksChan = make(chan T, len(wp.Tasks))

	// Start workers
	for i := 0; i < wp.Concurrency; i++ {
		go wp.worker()
	}

	// Send tasks to the tasks channel
	wp.wg.Add(len(wp.Tasks))
	for _, task := range wp.Tasks {
		wp.tasksChan <- task
	}
	close(wp.tasksChan)

	// Wait for all tasks to finish
	wp.wg.Wait()
}
