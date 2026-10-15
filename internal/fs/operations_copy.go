package fs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const copyBufferSize = 64 * 1024

// copyFileChunked streams one file in 64 KiB chunks with cancellation and pause support.
// A cancelled copy removes the partially written destination file.
func (m *OperationsManager) copyFileChunked(op *Operation, srcPath, dstPath string, mode fs.FileMode) error {
	if err := op.checkAbort(); err != nil {
		return err
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}

	buf := make([]byte, copyBufferSize)
	aborted := false
	for {
		if err := op.checkAbort(); err != nil {
			aborted = true
			break
		}
		n, readErr := src.Read(buf)
		if n > 0 {
			if _, wErr := dst.Write(buf[:n]); wErr != nil {
				dst.Close()
				os.Remove(dstPath)
				return wErr
			}
			op.reportProgress(m, srcPath, int64(n))
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			dst.Close()
			os.Remove(dstPath)
			return readErr
		}
	}

	if aborted {
		dst.Close()
		// Remove the incomplete file so no corrupt half-copy is left behind.
		_ = os.Remove(dstPath)
		return ErrCancelled
	}

	if err := dst.Sync(); err != nil {
		dst.Close()
		_ = os.Remove(dstPath)
		return err
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(dstPath)
		return err
	}

	_ = os.Chtimes(dstPath, time.Now(), fileModTime(srcPath))
	op.mu.Lock()
	op.ItemsDone++
	op.mu.Unlock()
	return nil
}

// resolveConflict applies the conflict policy for an existing destination.
// It returns the final destination path, or "" to skip the file.
func (m *OperationsManager) resolveConflict(op *Operation, srcPath, dstPath string) (string, error) {
	if _, err := os.Lstat(dstPath); err != nil {
		return dstPath, nil
	}

	mode := op.ConflictMode
	if mode == "" {
		mode = ConflictRename
	}

	if mode == ConflictAsk {
		key := op.ID + "|" + srcPath
		ch := make(chan string, 1)
		m.mu.Lock()
		m.resolvers[key] = ch
		m.mu.Unlock()
		defer func() {
			m.mu.Lock()
			delete(m.resolvers, key)
			m.mu.Unlock()
		}()

		m.emitEvent("op:conflict", map[string]any{
			"id":      op.ID,
			"srcPath": srcPath,
			"dstPath": dstPath,
			"srcName": filepath.Base(srcPath),
			"srcSize": fileSizeOf(srcPath),
			"dstSize": fileSizeOf(dstPath),
		})

		select {
		case decision := <-ch:
			mode = decision
		case <-time.After(5 * time.Minute):
			mode = ConflictRename
		case <-op.ctx.Done():
			return "", ErrCancelled
		}
	}

	switch mode {
	case ConflictSkip:
		op.mu.Lock()
		op.Skipped++
		op.mu.Unlock()
		return "", nil
	case ConflictOverwrite:
		op.mu.Lock()
		op.Overwritten++
		op.mu.Unlock()
		if err := os.RemoveAll(dstPath); err != nil {
			return "", err
		}
		return dstPath, nil
	default:
		op.mu.Lock()
		op.Renamed++
		op.mu.Unlock()
		return uniqueDestination(filepath.Dir(dstPath), filepath.Base(dstPath)), nil
	}
}
