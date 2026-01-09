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

// StartTransfer executes the mass download logic.
func (e *Engine) StartTransfer(session *network.SftpSession, downloadAll bool) error {
	if session.SftpClient == nil {
		return fmt.Errorf("sftp_client_not_initialized")
	}

	// 1. Determine Concurrency
	concurrency := BatchSizeConservative
	if e.Mode == ModeBoost {
		concurrency = BatchSizeBoost
	}
	
	if !downloadAll {
		fmt.Println(">> PFTE: No operation specified. Use --all to download everything.")
		return nil
	}

	// 2. Setup Local Dump Directory
	localDir := "dump"
	if _, err := os.Stat(localDir); os.IsNotExist(err) {
		os.Mkdir(localDir, 0755)
	}

	// 3. Discovery Phase (Fill the Queue)
	fmt.Println(">> PFTE: Scanning remote root for mass download...")
	
	files, err := session.SftpClient.ReadDir(".")
	if err != nil {
		return err
	}

	queuedCount := 0
	for _, file := range files {
		// Skip directories for v0.0.3 (recursion comes later)
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
	}

	fmt.Printf(">> PFTE: Queue filled with %d files.\n", queuedCount)
	if queuedCount == 0 {
		fmt.Println(">> PFTE: Nothing to download.")
		return nil
	}

	// 4. Start the Parallelizer (PLR)
	workerPool := NewWorkerPool(concurrency, e.Queue)
	workerPool.StartUnleash(session)

	return nil
}