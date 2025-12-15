package types

import (
	"io"
	"sync"
)

// SyncWriter is a thread-safe writer for concurrent output.
type SyncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func NewSyncWriter(w io.Writer) *SyncWriter {
	return &SyncWriter{w: w}
}

func (sw *SyncWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}
