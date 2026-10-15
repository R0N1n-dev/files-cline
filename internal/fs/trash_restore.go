package fs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// copyPathRecursive copies a file or tree without progress reporting (used for
// cross-device trash/restore moves).
func copyPathRecursive(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	case info.IsDir():
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, de := range entries {
			if err := copyPathRecursive(filepath.Join(src, de.Name()), filepath.Join(dst, de.Name())); err != nil {
				return err
			}
		}
		return nil
	default:
		return copyFileSimple(src, dst, info.Mode().Perm())
	}
}

func copyFileSimple(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}

// RestoreItems restores trashed items to their original locations, handling
// collisions by auto-renaming and cross-mount moves transparently.
func (t *TrashManager) RestoreItems(items []TrashItem) (int, error) {
	restored := 0
	var firstErr error
	for _, it := range items {
		if err := t.restoreOne(it); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		restored++
	}
	t.emitEvent("trash:changed", map[string]any{"restored": restored})
	if restored == 0 && firstErr != nil {
		return 0, firstErr
	}
	return restored, nil
}

func (t *TrashManager) restoreOne(it TrashItem) error {
	if _, err := os.Lstat(it.PayloadPath); err != nil {
		return fmt.Errorf("trash payload missing for %q", it.Name)
	}
	dest := it.OriginalPath
	if dest == "" {
		return fmt.Errorf("original path unknown for %q", it.Name)
	}
	if _, err := os.Lstat(dest); err == nil {
		dest = uniqueDestination(filepath.Dir(dest), filepath.Base(dest))
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	if err := os.Rename(it.PayloadPath, dest); err != nil {
		if !isCrossDeviceError(err) {
			return err
		}
		if err := copyPathRecursive(it.PayloadPath, dest); err != nil {
			return err
		}
		if err := os.RemoveAll(it.PayloadPath); err != nil {
			return err
		}
	}
	_ = os.Remove(it.InfoPath)
	return nil
}

// RestoreByIDs restores items identified by their .trashinfo paths.
func (t *TrashManager) RestoreByIDs(ids []string) (int, error) {
	idSet := map[string]bool{}
	for _, id := range ids {
		idSet[id] = true
	}
	var targets []TrashItem
	for _, it := range t.ListAllItems() {
		if idSet[it.ID] || idSet[it.PayloadPath] {
			targets = append(targets, it)
		}
	}
	if len(targets) == 0 {
		return 0, errors.New("no matching trash entries found")
	}
	return t.RestoreItems(targets)
}

// UndoLastTrash restores the most recent trash transaction across all mounts.
func (t *TrashManager) UndoLastTrash() (int, error) {
	t.mu.undo = trimUndoStack(t.mu.undo)
	for i := len(t.mu.undo) - 1; i >= 0; i-- {
		if t.mu.undo[i].Undone {
			continue
		}
		txn := t.mu.undo[i]
		restored, err := t.RestoreItems(txn.Items)
		if err != nil && restored == 0 {
			return 0, err
		}
		t.mu.undo[i].Undone = true
		return restored, nil
	}
	return 0, errors.New("nothing to undo")
}

func trimUndoStack(stack []TrashTransaction) []TrashTransaction {
	const maxUndo = 25
	if len(stack) > maxUndo {
		return stack[len(stack)-maxUndo:]
	}
	return stack
}

// DeletePermanently removes items from trash without restoring them.
func (t *TrashManager) DeletePermanently(ids []string) (int, error) {
	idSet := map[string]bool{}
	for _, id := range ids {
		idSet[id] = true
	}
	removed := 0
	for _, it := range t.ListAllItems() {
		if !idSet[it.ID] && !idSet[it.PayloadPath] {
			continue
		}
		if err := os.RemoveAll(it.PayloadPath); err != nil {
			return removed, err
		}
		_ = os.Remove(it.InfoPath)
		removed++
	}
	if removed == 0 {
		return 0, errors.New("no matching trash entries found")
	}
	t.emitEvent("trash:changed", map[string]any{"deleted": removed})
	return removed, nil
}

// EmptyTrash permanently deletes everything in every trash directory.
func (t *TrashManager) EmptyTrash() (int, error) {
	removed := 0
	for _, dir := range t.ListAllTrashDirs() {
		for _, sub := range []string{"files", "info"} {
			target := filepath.Join(dir, sub)
			entries, err := os.ReadDir(target)
			if err != nil {
				continue
			}
			for _, de := range entries {
				if err := os.RemoveAll(filepath.Join(target, de.Name())); err == nil {
					if sub == "files" {
						removed++
					}
				}
			}
		}
	}
	t.emitEvent("trash:changed", map[string]any{"emptied": removed})
	return removed, nil
}

// DeletePermanentlyPaths removes files from disk bypassing trash entirely.
func DeletePermanentlyPaths(paths []string) (int, error) {
	removed := 0
	for _, p := range paths {
		abs, err := filepath.Abs(ExpandPath(p))
		if err != nil {
			continue
		}
		if abs == "/" || abs == mustHomeDir() {
			return removed, fmt.Errorf("refusing to delete %q", abs)
		}
		if err := os.RemoveAll(abs); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

var _ = time.Now
