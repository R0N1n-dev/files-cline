package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StartCopy begins a cancellable copy of sources into destDir.
func (m *OperationsManager) StartCopy(sources []string, destDir, conflictMode string) (string, error) {
	return m.startTransfer(OpCopy, sources, destDir, conflictMode)
}

// StartMove begins a cancellable move of sources into destDir.
func (m *OperationsManager) StartMove(sources []string, destDir, conflictMode string) (string, error) {
	return m.startTransfer(OpMove, sources, destDir, conflictMode)
}

func (m *OperationsManager) startTransfer(kind string, sources []string, destDir, conflictMode string) (string, error) {
	if len(sources) == 0 {
		return "", errors.New("no source items provided")
	}
	destAbs, err := filepath.Abs(ExpandPath(destDir))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(destAbs)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("destination %q is not a directory", destAbs)
	}
	if conflictMode == "" {
		conflictMode = ConflictRename
	}

	totalBytes, items := measureSources(sources)
	op := newOperation(kind, sources, destAbs, conflictMode, totalBytes, items)
	op.ID = m.nextID(kind)
	m.register(op)
	m.emitEvent("op:state", snapshot(op))

	go func() {
		runErr := m.runTransfer(op)
		op.mu.Lock()
		cancelled := op.State == StateCancelling || op.State == StateCancelled || op.ctx.Err() != nil
		entries := append([]UndoEntry{}, op.Undo...)
		op.FinishedAt = time.Now()
		op.SpeedBps, op.ETASeconds = 0, 0
		switch {
		case cancelled:
			// A cancel that landed at the finish line still wins: roll back output.
			op.State = StateCancelled
			op.Message = "Cancelled — partial files removed"
			op.CanUndo = false
			op.Undo = nil
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
			if op.TotalBytes > 0 {
				op.DoneBytes = op.TotalBytes
			}
			op.CanUndo = true
			if op.Kind == OpMove {
				op.Message = fmt.Sprintf("Moved %d item(s)", op.ItemsDone)
			} else {
				op.Message = fmt.Sprintf("Copied %d item(s)", op.ItemsDone)
			}
		}
		final := snapshotLocked(op)
		op.mu.Unlock()
		m.emitEvent("op:done", final)
	}()

	return op.ID, nil
}

func (m *OperationsManager) runTransfer(op *Operation) error {
	for _, src := range op.Sources {
		if err := op.checkAbort(); err != nil {
			return err
		}
		srcAbs, err := filepath.Abs(ExpandPath(src))
		if err != nil {
			return err
		}
		srcInfo, err := os.Lstat(srcAbs)
		if err != nil {
			return fmt.Errorf("source unavailable: %w", err)
		}

		dstPath := filepath.Join(op.DestDir, filepath.Base(srcAbs))
		if srcInfo.IsDir() && (dstPath == srcAbs || strings.HasPrefix(dstPath+string(os.PathSeparator), srcAbs+string(os.PathSeparator))) {
			return fmt.Errorf("cannot copy %q into itself", filepath.Base(srcAbs))
		}

		resolved, err := m.resolveConflict(op, srcAbs, dstPath)
		if err != nil {
			return err
		}
		if resolved == "" {
			continue
		}
		dstPath = resolved

		if op.Kind == OpMove {
			if err := os.Rename(srcAbs, dstPath); err == nil {
				op.mu.Lock()
				op.ItemsDone++
				op.Undo = append(op.Undo, UndoEntry{MovedFrom: srcAbs, MovedTo: dstPath})
				op.mu.Unlock()
				op.reportProgress(m, srcAbs, 0)
				continue
			} else if !isCrossDeviceError(err) {
				return err
			}
			if err := m.copyTree(op, srcAbs, dstPath); err != nil {
				return err
			}
			op.mu.Lock()
			op.Undo = append(op.Undo, UndoEntry{MovedFrom: srcAbs, MovedTo: dstPath})
			op.mu.Unlock()
			if err := op.checkAbort(); err != nil {
				return err
			}
			if err := os.RemoveAll(srcAbs); err != nil {
				return err
			}
			continue
		}

		if err := m.copyTree(op, srcAbs, dstPath); err != nil {
			return err
		}
	}
	return nil
}

// Undo reverts a finished copy or move operation.
func (m *OperationsManager) Undo(id string) error {
	m.mu.Lock()
	op, ok := m.ops[id]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("operation %q not found", id)
	}
	op.mu.Lock()
	entries := append([]UndoEntry{}, op.Undo...)
	canUndo := op.CanUndo
	op.mu.Unlock()
	if !canUndo {
		return errors.New("this operation cannot be undone")
	}

	if err := revertEntries(entries); err != nil {
		return err
	}

	op.mu.Lock()
	op.CanUndo = false
	op.Undo = nil
	op.Message = "Reverted"
	op.mu.Unlock()
	m.emitEvent("op:undo", snapshot(op))
	return nil
}
