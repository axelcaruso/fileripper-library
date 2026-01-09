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
func (wp *WorkerPool) StartUnleash(session *network.SftpSession) {
	// Uses fmt (Fixes import error)
	fmt.Printf(">> PLR: Unleashing %d workers...\n", wp.Concurrency)
	
	GlobalMonitor.SetRunning(true)
	
	// Uses time (Fixes import error)
	start := time.Now()

	for i := 0; i < wp.Concurrency; i++ {
		wp.Wg.Add(1)
		
		go func(workerID int) {
			defer wp.Wg.Done()
			
			for {
				job := wp.Queue.Pop()
				if job == nil {
					return 
				}

				GlobalMonitor.SetCurrentFile(job.RemotePath)

				var err error
				if job.Operation == "DOWNLOAD" {
					err = DownloadFileWithProgress(session, job.RemotePath, job.LocalPath)
				} else if job.Operation == "UPLOAD" {
					err = UploadFileWithProgress(session, job.LocalPath, job.RemotePath)
				}

				if err != nil {
					// Uses log
					log.Printf("[Worker %d] Failed %s: %v", workerID, job.RemotePath, err)
					continue
				}

				GlobalMonitor.IncFileDone()
			}
		}(i)
	}

	wp.Wg.Wait()
	GlobalMonitor.SetRunning(false)
	
	duration := time.Since(start)
	fmt.Printf(">> PLR: Batch complete in %v.\n", duration)
}