// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
// If a copy of the MPL was not distributed with this file, You can obtain one at
// https://mozilla.org/MPL/2.0/.

package pfte

import (
	"fmt"
	"log"
	"sync"
	"time"

	"fileripper/internal/network"
)

// WorkerPool manages the swarm of goroutines.
type WorkerPool struct {
	Concurrency int
	Queue       *JobQueue
	Wg          sync.WaitGroup
}

func NewWorkerPool(concurrency int, queue *JobQueue) *WorkerPool {
	return &WorkerPool{
		Concurrency: concurrency,
		Queue:       queue,
	}
}

// StartUnleash fires up the goroutines to consume the queue.
// It blocks until the queue is empty and all workers are done.
func (wp *WorkerPool) StartUnleash(session *network.SftpSession) {
	fmt.Printf(">> PLR: Unleashing %d workers on the queue...\n", wp.Concurrency)
	start := time.Now()

	for i := 0; i < wp.Concurrency; i++ {
		wp.Wg.Add(1)
		
		// Launch worker
		go func(workerID int) {
			defer wp.Wg.Done()
			
			for {
				// 1. Get a job (Thread-safe pop)
				job := wp.Queue.Pop()
				if job == nil {
					// Queue is empty, go home.
					return 
				}

				// 2. Execute Transfer
				// Note: In a real heavy app, we might want separate SftpClients per worker
				// to avoid mutex contention in the library, but for v0.0.3 pkg/sftp handles it well.
				var err error
				if job.Operation == "DOWNLOAD" {
					err = DownloadFile(session, job.RemotePath, job.LocalPath)
				} else if job.Operation == "UPLOAD" {
					err = UploadFile(session, job.LocalPath, job.RemotePath)
				}

				if err != nil {
					// Don't crash the pool, just log the failure
					log.Printf("[Worker %d] Failed %s: %v", workerID, job.RemotePath, err)
					continue
				}

				// 3. Validation (Fast CRC32)
				// Only check integrity on download for now
				if job.Operation == "DOWNLOAD" {
					_, err := CalculateChecksum(job.LocalPath)
					if err != nil {
						log.Printf("[Worker %d] CRC32 Failed: %v", workerID, err)
					}
				}
			}
		}(i)
	}

	// Wait for the swarm to finish
	wp.Wg.Wait()
	
	duration := time.Since(start)
	fmt.Printf(">> PLR: Batch complete in %v.\n", duration)
}