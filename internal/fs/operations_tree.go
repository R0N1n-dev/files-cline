package fs

import (
	"os"
	"path/filepath"
)

// copyTree recursively copies a file, symlink, or directory tree with abort support.
func (m *OperationsManager) copyTree(op *Operation, srcPath, dstPath string) error {
	if err := op.checkAbort(); err != nil {
		return err
	}

	info, err := os.Lstat(srcPath)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(srcPath)
		if err != nil {
			return err
		}
		return os.Symlink(target, dstPath)
	}

	if info.IsDir() {
		if err := os.MkdirAll(dstPath, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(srcPath)
		if err != nil {
			return err
		}
		for _, de := range entries {
			if err := op.checkAbort(); err != nil {
				return err
			}
			if err := m.copyTree(op, filepath.Join(srcPath, de.Name()), filepath.Join(dstPath, de.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	if err := m.copyFileChunked(op, srcPath, dstPath, info.Mode()); err != nil {
		return err
	}
	op.mu.Lock()
	op.Undo = append(op.Undo, UndoEntry{CreatedPath: dstPath})
	op.mu.Unlock()
	return nil
}
