// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
// If a copy of the MPL was not distributed with this file, You can obtain one at
// https://mozilla.org/MPL/2.0/.

package pfte

import (
	"sync"
	"sync/atomic"
	"time"
)

// Global instance to be accessed by API and Engine
var GlobalMonitor *TransferMonitor

func init() {
	GlobalMonitor = NewMonitor()
}

type TransferStats struct {
	TotalFiles      int64   `json:"total_files"`
	FilesDone       int64   `json:"files_done"`
	TotalBytes      int64   `json:"total_bytes"`
	BytesDone       int64   `json:"bytes_done"`
	ProgressPercent float64 `json:"progress_percent"`
	SpeedMBs        float64 `json:"speed_mb_s"`
	CurrentFile     string  `json:"current_file"` // Last file started
	IsRunning       bool    `json:"is_running"`
}

type TransferMonitor struct {
	totalFiles      int64
	filesDone       int64
	totalBytes      int64
	bytesDone       int64 // Atomic
	
	currentFile     string
	mu              sync.Mutex // Protects string and bools
	isRunning       bool

	// Speed calculation helpers
	lastBytes       int64
	lastCheck       time.Time
	currentSpeed    float64
}

func NewMonitor() *TransferMonitor {
	return &TransferMonitor{
		lastCheck: time.Now(),
	}
}

// Reset clears stats for a new transfer batch
func (m *TransferMonitor) Reset(totalFiles, totalBytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	atomic.StoreInt64(&m.totalFiles, totalFiles)
	atomic.StoreInt64(&m.totalBytes, totalBytes)
	atomic.StoreInt64(&m.filesDone, 0)
	atomic.StoreInt64(&m.bytesDone, 0)
	
	m.currentFile = "Initializing..."
	m.isRunning = true
	m.lastBytes = 0
	m.lastCheck = time.Now()
	m.currentSpeed = 0
}

// UpdateBytes is called by workers atomically
func (m *TransferMonitor) AddBytes(n int64) {
	atomic.AddInt64(&m.bytesDone, n)
}

func (m *TransferMonitor) IncFileDone() {
	atomic.AddInt64(&m.filesDone, 1)
}

func (m *TransferMonitor) SetCurrentFile(name string) {
	m.mu.Lock()
	m.currentFile = name
	m.mu.Unlock()
}

func (m *TransferMonitor) SetRunning(running bool) {
	m.mu.Lock()
	m.isRunning = running
	m.mu.Unlock()
}

// GetStats calculates live speed and returns the snapshot
func (m *TransferMonitor) GetStats() TransferStats {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	bytesNow := atomic.LoadInt64(&m.bytesDone)
	totalBytes := atomic.LoadInt64(&m.totalBytes)
	
	// Calculate Speed (Moving average could be better, but instant is fine for now)
	duration := now.Sub(m.lastCheck).Seconds()
	if duration >= 0.5 { // Update speed every 500ms approx
		diff := bytesNow - m.lastBytes
		m.currentSpeed = (float64(diff) / 1024 / 1024) / duration // MB/s
		
		m.lastBytes = bytesNow
		m.lastCheck = now
	}

	percent := 0.0
	if totalBytes > 0 {
		percent = (float64(bytesNow) / float64(totalBytes)) * 100
	}

	return TransferStats{
		TotalFiles:      atomic.LoadInt64(&m.totalFiles),
		FilesDone:       atomic.LoadInt64(&m.filesDone),
		TotalBytes:      totalBytes,
		BytesDone:       bytesNow,
		ProgressPercent: percent,
		SpeedMBs:        m.currentSpeed,
		CurrentFile:     m.currentFile,
		IsRunning:       m.isRunning,
	}
}