package service

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

// Job represents a background task
type Job func(ctx context.Context)

type WorkerPool interface {
	Submit(job Job)
	Start(ctx context.Context)
}

type workerPool struct {
	maxWorkers int
	jobQueue   chan Job
	logger     *logrus.Logger
}

func NewWorkerPool(maxWorkers int, logger *logrus.Logger) WorkerPool {
	return &workerPool{
		maxWorkers: maxWorkers,
		jobQueue:   make(chan Job, 1000), // Large buffer for bulk playlist syncs
		logger:     logger,
	}
}

func (p *workerPool) Start(ctx context.Context) {
	p.logger.Infof("WorkerPool started with %d workers", p.maxWorkers)
	var wg sync.WaitGroup
	for i := 0; i < p.maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case job := <-p.jobQueue:
					job(ctx)
				case <-ctx.Done():
					p.logger.Debugf("Worker %d shutting down", workerID)
					return
				}
			}
		}(i)
	}
}

func (p *workerPool) Submit(job Job) {
	p.jobQueue <- job
}
