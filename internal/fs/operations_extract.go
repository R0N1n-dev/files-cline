package fs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StartExtract begins a cancellable extraction of archives into destDir.
func (m *OperationsManager) StartExtract(archives []string, destDir string) (string, error) {
	if len(archives) == 0 {
		return "", errors.New("no archives provided")
	}
	destAbs, err := filepath.Abs(ExpandPath(destDir))
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(destAbs); err != nil || !info.IsDir() {
		return "", fmt.Errorf("destination %q is not a directory", destAbs)
	}

	total := int64(0)
	for _, a := range archives {
		if size, err := archiveUncompressedSize(ExpandPath(a)); err == nil {
			total += size
		}
	}

	op := newOperation(OpExtract, archives, destAbs, ConflictRename, total, len(archives))
	op.ID = m.nextID(OpExtract)
	m.register(op)
	m.emitEvent("op:state", snapshot(op))

	go func() {
		runErr := m.runExtract(op)
		op.mu.Lock()
		cancelled := op.State == StateCancelling || op.State == StateCancelled || op.ctx.Err() != nil
		entries := append([]UndoEntry{}, op.Undo...)
		op.FinishedAt = nowFunc()
		op.SpeedBps, op.ETASeconds = 0, 0
		switch {
		case cancelled:
			op.State = StateCancelled
			op.Message = "Extraction cancelled — partial files removed"
			op.mu.Unlock()
			_ = revertEntries(entries)
			m.emitEvent("op:done", snapshot(op))
			return
		case runErr != nil:
			op.State = StateError
			op.Error = runErr.Error()
		default:
			op.State = StateCompleted
			op.Progress = 100
			op.Message = fmt.Sprintf("Extracted %d archive(s)", len(op.Sources))
		}
		final := snapshotLocked(op)
		op.mu.Unlock()
		m.emitEvent("op:done", final)
	}()

	return op.ID, nil
}

func (m *OperationsManager) runExtract(op *Operation) error {
	for _, archive := range op.Sources {
		if err := op.checkAbort(); err != nil {
			return err
		}
		if err := m.extractOne(op, ExpandPath(archive), op.DestDir); err != nil {
			return err
		}
	}
	return nil
}

func (m *OperationsManager) extractOne(op *Operation, archivePath, destDir string) error {
	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".zip") {
		return m.extractZip(op, archivePath, destDir)
	}
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") || strings.HasSuffix(lower, ".tar") {
		compressed := strings.HasSuffix(lower, ".gz") || strings.HasSuffix(lower, ".tgz")
		return m.extractTar(op, archivePath, destDir, compressed)
	}
	return fmt.Errorf("unsupported archive format: %s", filepath.Base(archivePath))
}

// writeStream copies an archive entry to disk in chunks, honouring cancellation
// and removing the partial file when the user aborts.
func (m *OperationsManager) writeStream(op *Operation, src io.Reader, dstPath string, perm os.FileMode) error {
	out, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	buf := make([]byte, copyBufferSize)
	for {
		if err := op.checkAbort(); err != nil {
			out.Close()
			_ = os.Remove(dstPath)
			return err
		}
		n, readErr := src.Read(buf)
		if n > 0 {
			if _, wErr := out.Write(buf[:n]); wErr != nil {
				out.Close()
				_ = os.Remove(dstPath)
				return wErr
			}
			op.reportProgress(m, dstPath, int64(n))
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			out.Close()
			_ = os.Remove(dstPath)
			return readErr
		}
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dstPath)
		return err
	}
	op.mu.Lock()
	op.Undo = append(op.Undo, UndoEntry{CreatedPath: dstPath})
	op.mu.Unlock()
	return nil
}

// safeJoin resolves an archive entry path inside rootDir, blocking path traversal.
func safeJoin(rootDir, entryName string) (string, error) {
	clean := filepath.Clean(strings.TrimPrefix(filepath.ToSlash(entryName), "/"))
	if clean == "." || clean == ".." {
		return "", errors.New("invalid archive entry")
	}
	target := filepath.Join(rootDir, clean)
	rel, err := filepath.Rel(rootDir, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("archive entry escapes destination")
	}
	return target, nil
}

var _ = time.Now
