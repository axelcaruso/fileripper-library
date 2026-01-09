// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
// If a copy of the MPL was not distributed with this file, You can obtain one at
// https://mozilla.org/MPL/2.0/.

package pfte

import (
	"io"
	"os"

	"fileripper/internal/network"
)

// BufferSize is defined ONLY HERE to avoid redeclaration errors.
// 64KB buffer for optimal TCP throughput.
const BufferSize = 64 * 1024 

// ProgressWriter is a wrapper to update the monitor as we write
type ProgressWriter struct {
	Writer io.Writer
	OnWrite func(int64)
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	if n > 0 {
		pw.OnWrite(int64(n))
	}
	return n, err
}

// DownloadFileWithProgress pulls a remote file and updates GlobalMonitor
func DownloadFileWithProgress(session *network.SftpSession, remotePath, localPath string) error {
	src, err := session.SftpClient.Open(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	buf := make([]byte, BufferSize)
	
	// Wrap the destination writer to intercept bytes
	writer := &ProgressWriter{
		Writer: dst,
		OnWrite: func(n int64) {
			GlobalMonitor.AddBytes(n)
		},
	}

	_, err = io.CopyBuffer(writer, src, buf)
	if err != nil {
		return err
	}

	dst.Sync()
	return nil
}

// UploadFileWithProgress sends a local file and updates GlobalMonitor
func UploadFileWithProgress(session *network.SftpSession, localPath, remotePath string) error {
	src, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := session.SftpClient.Create(remotePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	buf := make([]byte, BufferSize)

	writer := &ProgressWriter{
		Writer: dst,
		OnWrite: func(n int64) {
			GlobalMonitor.AddBytes(n)
		},
	}

	_, err = io.CopyBuffer(writer, src, buf)
	return err
}

// Legacy functions needed if referenced elsewhere, updated to use the single BufferSize
func UploadFile(session *network.SftpSession, localPath, remotePath string) error {
	// Simple wrapper around the progress version or standard copy
	return UploadFileWithProgress(session, localPath, remotePath)
}

func DownloadFile(session *network.SftpSession, remotePath, localPath string) error {
	return DownloadFileWithProgress(session, remotePath, localPath)
}