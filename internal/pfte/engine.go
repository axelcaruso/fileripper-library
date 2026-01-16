/*
 * Copyright 2026 The FileRipper Team
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package pfte

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"fileripper/internal/network"
	"github.com/pkg/sftp"
)

const (
	BatchSizeBoost        = 128
	BatchSizeConservative = 4
	DirCreationWorkers    = 8
)

type TransferMode int

const (
	ModeBoost TransferMode = iota
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

// StartTransfer now accepts sourcePath (local/remote) and destPath (remote dest for upload)
func (e *Engine) StartTransfer(sessions []*network.SftpSession, operation string, sourcePath string, destPath string) error {
	if len(sessions) == 0 || sessions[0].SftpClient == nil {
		return fmt.Errorf("no_active_sessions")
	}
	mainSession := sessions[0]

	concurrency := BatchSizeConservative
	if e.Mode == ModeBoost {
		concurrency = BatchSizeBoost
	}

	// --- UPLOAD LOGIC ---
	if operation == "UPLOAD" {
		absSource, err := filepath.Abs(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %v", err)
		}

		fmt.Printf(">> PFTE: Source '%s' -> Remote '%s'\n", absSource, destPath)

		// Base dir is the parent of the source folder (e.g., C:\Users\...)
		baseDir := filepath.Dir(absSource)

		var foldersToCreate []string
		var filesToTransfer []*TransferJob
		totalBytes := int64(0)

		err = filepath.Walk(absSource, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf(">> Warning: Error accessing %s: %v\n", p, err)
				return nil
			}

			// Handle Symlinks: We follow them to keep it simple and functional across OS
			if info.Mode()&os.ModeSymlink != 0 {
				resolvedPath, err := filepath.EvalSymlinks(p)
				if err != nil {
					fmt.Printf(">> Warning: Could not resolve symlink %s: %v\n", p, err)
					return nil
				}
				info, err = os.Stat(resolvedPath)
				if err != nil {
					fmt.Printf(">> Warning: Could not stat resolved symlink %s: %v\n", p, err)
					return nil
				}
				// If it's a directory symlink, we'd need a recursive walk here, 
				// but for now we handle it as a file or empty dir to avoid loops.
			}

			// Calculate relative path from local base
			relPath, err := filepath.Rel(baseDir, p)
			if err != nil {
				return err
			}

			// Cross-platform path normalization: SFTP always wants forward slashes
			remoteRel := filepath.ToSlash(relPath)
			finalRemotePath := path.Join(destPath, remoteRel)

			if info.IsDir() {
				if remoteRel != "." && remoteRel != "" {
					foldersToCreate = append(foldersToCreate, finalRemotePath)
				}
			} else {
				filesToTransfer = append(filesToTransfer, &TransferJob{
					LocalPath:  p,
					RemotePath: finalRemotePath,
					Operation:  "UPLOAD",
				})
				totalBytes += info.Size()
			}
			return nil
		})
		if err != nil {
			return err
		}

		sort.Slice(foldersToCreate, func(i, j int) bool {
			return len(foldersToCreate[i]) < len(foldersToCreate[j])
		})

		dirCount := len(foldersToCreate)
		if dirCount > 0 {
			fmt.Printf(">> PFTE: Phase 1 - Creating %d directories...\n", dirCount)
			dirChan := make(chan string, dirCount)
			var wg sync.WaitGroup
			var doneCount int32
			var printMu sync.Mutex
			for _, d := range foldersToCreate {
				dirChan <- d
			}
			close(dirChan)

			for i := 0; i < DirCreationWorkers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for dir := range dirChan {
						_ = mainSession.SftpClient.MkdirAll(dir)
						current := atomic.AddInt32(&doneCount, 1)
						printMu.Lock()
						progress := float64(current) / float64(dirCount) * 100
						fmt.Printf("\r>> Structure: [%3.0f%%] Processed...                 ", progress)
						printMu.Unlock()
					}
				}()
			}
			wg.Wait()
			fmt.Println("\n>> PFTE: Structure verified.")
		}

		fileCount := int64(len(filesToTransfer))
		if fileCount == 0 {
			return nil
		}

		fmt.Printf(">> PFTE: Phase 2 - Queuing %d files...\n", fileCount)
		for _, job := range filesToTransfer {
			e.Queue.Add(job)
		}
		GlobalMonitor.Reset(fileCount, totalBytes)

		workerPool := NewWorkerPool(concurrency, e.Queue)
		workerPool.StartUnleash(sessions)
		return nil

		// --- DOWNLOAD LOGIC (Unchanged but using sourcePath) ---
	} else {
		// Just mapping arguments: sourcePath is what we want to download
		return e.startDownload(sessions, mainSession, concurrency, sourcePath)
	}
}

// Helper to keep the file clean
func (e *Engine) startDownload(sessions []*network.SftpSession, mainSession *network.SftpSession, concurrency int, targetPath string) error {
	localBase := "dump"
	if _, err := os.Stat(localBase); os.IsNotExist(err) {
		os.Mkdir(localBase, 0755)
	}

	pwd, _ := mainSession.SftpClient.Getwd()
	fmt.Printf(">> Remote PWD: %s\n", pwd)

	targetName := path.Base(targetPath)
	if targetPath == "" || targetPath == "." {
		targetName = ""
	}
	remoteSource := targetPath
	if remoteSource == "" {
		remoteSource = "."
	}

	fmt.Printf(">> PFTE: Verifying '%s'...\n", remoteSource)

	var info os.FileInfo
	var err error
	info, err = mainSession.SftpClient.Stat(remoteSource) // Use Stat to follow symlink if the target itself is one

	if err != nil && targetName != "" {
		fmt.Println(">> Status: Path not found. Initiating Deep Search (Depth: 3)...")
		foundPath := findRemotePath(mainSession.SftpClient, ".", targetName, 3)
		if foundPath != "" {
			fmt.Printf(">> THE HOUND: Found '%s' at '%s'! Auto-correcting...\n", targetName, foundPath)
			remoteSource = foundPath
			info, err = mainSession.SftpClient.Stat(remoteSource)
		} else {
			return fmt.Errorf("target not found")
		}
	}
	if err != nil {
		return err
	}

	queuedCount := int64(0)
	totalBytes := int64(0)

	fmt.Printf(">> PFTE: Scanning '%s' recursively...\n", remoteSource)

	walker := mainSession.SftpClient.Walk(remoteSource)
	for walker.Step() {
		if walker.Err() != nil {
			continue
		}
		remotePath := walker.Path()
		stat := walker.Stat()

		relPath, err := filepath.Rel(remoteSource, remotePath)
		if err != nil {
			relPath = filepath.Base(remotePath)
		}
		rootDirName := filepath.Base(remoteSource)
		if remoteSource == "." || remoteSource == "/" {
			rootDirName = "root_dump"
		}
		localPath := filepath.Join(localBase, rootDirName, relPath)

		if !info.IsDir() && remotePath == remoteSource {
			localPath = filepath.Join(localBase, rootDirName)
		}

		if stat.Mode()&os.ModeSymlink != 0 {
			// If it's a symlink, we try to see what's on the other side
			realStat, err := mainSession.SftpClient.Stat(remotePath)
			if err != nil {
				fmt.Printf(">> Warning: Remote broken symlink %s\n", remotePath)
				continue
			}
			stat = realStat
		}

		if stat.IsDir() {
			os.MkdirAll(localPath, 0755)
			continue
		}

		e.Queue.Add(&TransferJob{
			LocalPath:  localPath,
			RemotePath: remotePath,
			Operation:  "DOWNLOAD",
		})
		queuedCount++
		totalBytes += stat.Size()
	}

	fmt.Printf(">> PFTE: Job ready. Files: %d, Total Size: %d bytes.\n", queuedCount, totalBytes)
	GlobalMonitor.Reset(queuedCount, totalBytes)

	if queuedCount > 0 {
		workerPool := NewWorkerPool(concurrency, e.Queue)
		workerPool.StartUnleash(sessions)
	} else {
		fmt.Println(">> Warning: Folder is empty.")
	}
	return nil
}

func findRemotePath(client *sftp.Client, root, targetName string, maxDepth int) string {
	if maxDepth < 0 {
		return ""
	}
	files, err := client.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, f := range files {
		if strings.EqualFold(f.Name(), targetName) {
			return path.Join(root, f.Name())
		}
	}
	for _, f := range files {
		if f.IsDir() {
			found := findRemotePath(client, path.Join(root, f.Name()), targetName, maxDepth-1)
			if found != "" {
				return found
			}
		}
	}
	return ""
}
