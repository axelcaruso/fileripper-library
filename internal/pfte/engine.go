// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
// If a copy of the MPL was not distributed with this file, You can obtain one at
// https://mozilla.org/MPL/2.0/.

package pfte

import (
	"fmt"
	"os"
	"path/filepath"

	"fileripper/internal/network"
)

const (
	BatchSizeBoost        = 64
	BatchSizeConservative = 2
)

type TransferMode int

const (
	ModeBoost        TransferMode = iota 
	ModeConservative                     
)

type Engine struct {
	Mode  TransferMode
	Queue *JobQueue 
}

func NewEngine() *Engine {
	return &Engine{
		Mode:  ModeBoost, 
		Queue: NewQueue(),
	}
}

func (e *Engine) StartTransfer(session *network.SftpSession, downloadAll bool) error {
	if session.SftpClient == nil {
		return fmt.Errorf("sftp_client_not_initialized")
	}

	concurrency := BatchSizeConservative
	if e.Mode == ModeBoost {
		concurrency = BatchSizeBoost
	}
	
	if !downloadAll {
		return nil
	}

	localDir := "dump"
	if _, err := os.Stat(localDir); os.IsNotExist(err) {
		os.Mkdir(localDir, 0755)
	}

	fmt.Println(">> PFTE: Scanning remote root...")
	
	files, err := session.SftpClient.ReadDir(".")
	if err != nil {
		return err
	}

	queuedCount := int64(0)
	totalBytes := int64(0)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		remotePath := file.Name()
		localPath := filepath.Join(localDir, file.Name())

		e.Queue.Add(&TransferJob{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Operation:  "DOWNLOAD",
		})
		
		queuedCount++
		totalBytes += file.Size()
	}

	fmt.Printf(">> PFTE: Queue filled. Files: %d, Total Size: %d bytes.\n", queuedCount, totalBytes)
	
	GlobalMonitor.Reset(queuedCount, totalBytes)

	if queuedCount == 0 {
		return nil
	}

	// This call was failing because plr.go had errors. 
	// Now it should work.
	workerPool := NewWorkerPool(concurrency, e.Queue)
	workerPool.StartUnleash(session)

	return nil
}