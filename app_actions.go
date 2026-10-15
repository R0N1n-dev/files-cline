package main

import (
	"os"
	"path/filepath"

	"vela-files/internal/fs"
)

func fsHome() (string, error) { return os.UserHomeDir() }

func fsAbs(path string) (string, error) { return filepath.Abs(fs.ExpandPath(path)) }

// CreateFolder creates a new folder inside parentDir.
func (a *App) CreateFolder(parentDir, name string) (fs.FileEntry, error) {
	entry, err := fs.CreateFolder(parentDir, name)
	if err == nil {
		a.emit("fs:changed", map[string]any{"dir": parentDir, "action": "create", "path": entry.Path})
	}
	return entry, err
}

// CreateFile creates a new file inside parentDir, optionally with template content.
func (a *App) CreateFile(parentDir, name, content string) (fs.FileEntry, error) {
	entry, err := fs.CreateFile(parentDir, name, content)
	if err == nil {
		a.emit("fs:changed", map[string]any{"dir": parentDir, "action": "create", "path": entry.Path})
	}
	return entry, err
}

// Rename renames a single item.
func (a *App) Rename(oldPath, newName string) (fs.FileEntry, error) {
	entry, err := fs.RenameItem(oldPath, newName)
	if err == nil {
		a.emit("fs:changed", map[string]any{"dir": filepath.Dir(oldPath), "action": "rename", "path": entry.Path})
	}
	return entry, err
}

// BatchRename renames many items atomically with rollback on failure.
func (a *App) BatchRename(pairs []fs.RenamePair) ([]fs.FileEntry, error) {
	entries, err := fs.BatchRename(pairs)
	if err == nil {
		a.emit("fs:changed", map[string]any{"action": "batch-rename", "count": len(entries)})
	}
	return entries, err
}

// DuplicateItems copies items in place using the "name copy.ext" convention.
func (a *App) DuplicateItems(paths []string) ([]string, error) {
	created, err := fs.DuplicateItems(paths)
	if err == nil {
		a.emit("fs:changed", map[string]any{"action": "duplicate", "count": len(created)})
	}
	return created, err
}

// ChmodPaths applies an octal permission mode to the selection.
func (a *App) ChmodPaths(paths []string, mode string) (int, error) {
	n, err := fs.ChmodPaths(paths, mode)
	if err == nil {
		a.emit("fs:changed", map[string]any{"action": "chmod", "count": n, "mode": mode})
	}
	return n, err
}

// DeletePermanently removes items from disk, bypassing the trash.
func (a *App) DeletePermanently(paths []string) (int, error) {
	n, err := fs.DeletePermanentlyPaths(paths)
	if err == nil {
		a.emit("fs:changed", map[string]any{"action": "delete", "count": n})
	}
	return n, err
}

// ---------- Cancellable operations (copy / move / extract) ----------

// Copy starts a cancellable copy of sources into destDir.
func (a *App) Copy(sources []string, destDir, conflictMode string) (string, error) {
	return a.ops.StartCopy(sources, destDir, a.resolveConflictMode(conflictMode))
}

// Move starts a cancellable move of sources into destDir.
func (a *App) Move(sources []string, destDir, conflictMode string) (string, error) {
	return a.ops.StartMove(sources, destDir, a.resolveConflictMode(conflictMode))
}

// ExtractArchive starts a cancellable extraction of archives into destDir.
func (a *App) ExtractArchive(archives []string, destDir string) (string, error) {
	return a.ops.StartExtract(archives, destDir)
}

// CreateArchive compresses paths into destArchive (.zip, .tar, .tar.gz).
func (a *App) CreateArchive(paths []string, destArchive string) (string, error) {
	return fs.CreateArchive(paths, destArchive, a.emit)
}

// ListOperations returns active and recent operations for the operations popover.
func (a *App) ListOperations() []fs.Operation { return a.ops.List() }

// CancelOperation cancels a running operation mid-copy and removes partial output.
func (a *App) CancelOperation(id string) error { return a.ops.Cancel(id) }

// PauseOperation pauses a running operation between chunks.
func (a *App) PauseOperation(id string) error { return a.ops.Pause(id) }

// ResumeOperation resumes a paused operation.
func (a *App) ResumeOperation(id string) error { return a.ops.Resume(id) }

// CancelAllOperations cancels every running operation.
func (a *App) CancelAllOperations() { a.ops.CancelAll() }

// ClearFinishedOperations removes finished jobs from the popover list.
func (a *App) ClearFinishedOperations() { a.ops.ClearFinished() }

// UndoOperation reverts a finished copy or move.
func (a *App) UndoOperation(id string) error { return a.ops.Undo(id) }

// ResolveConflict supplies a user decision for a blocked conflict prompt.
func (a *App) ResolveConflict(opID, srcPath, decision string) {
	a.ops.ResolveConflict(opID, srcPath, decision)
}

// HandleDrop is the inbound drag-and-drop entry point (files dropped from the OS).
func (a *App) HandleDrop(sources []string, destDir string, move bool) (string, error) {
	if move {
		return a.Move(sources, destDir, fs.ConflictRename)
	}
	return a.Copy(sources, destDir, fs.ConflictRename)
}

func (a *App) resolveConflictMode(requested string) string {
	if requested != "" {
		return requested
	}
	return a.config.Get().ConflictMode
}
