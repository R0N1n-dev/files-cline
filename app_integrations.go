package main

import (
	"context"
	"path/filepath"

	"vela-files/internal/fs"
)

// ---------- Trash (multi-mount, undoable) ----------

// MoveToTrash sends paths to trash, honouring per-mount topdir trashes.
func (a *App) MoveToTrash(paths []string) (fs.TrashTransaction, error) {
	txn, err := a.trash.TrashItems(paths)
	if err == nil {
		a.emit("trash:changed", map[string]any{"count": len(txn.Items), "mounts": txn.Mounts})
		a.emit("fs:changed", map[string]any{"action": "trash", "count": len(txn.Items)})
	}
	return txn, err
}

// UndoTrash restores the most recent trash transaction across all mounts.
func (a *App) UndoTrash() (int, error) {
	n, err := a.trash.UndoLastTrash()
	if err == nil {
		a.emit("fs:changed", map[string]any{"action": "trash-undo", "count": n})
	}
	return n, err
}

// RestoreTrashItems restores specific trashed items by their trashinfo IDs.
func (a *App) RestoreTrashItems(ids []string) (int, error) {
	n, err := a.trash.RestoreByIDs(ids)
	if err == nil {
		a.emit("fs:changed", map[string]any{"action": "trash-restore", "count": n})
	}
	return n, err
}

// DeleteTrashItems permanently removes specific items from trash.
func (a *App) DeleteTrashItems(ids []string) (int, error) {
	n, err := a.trash.DeletePermanently(ids)
	if err == nil {
		a.emit("trash:changed", map[string]any{"deleted": n})
	}
	return n, err
}

// EmptyTrash permanently deletes everything across all trash directories.
func (a *App) EmptyTrash() (int, error) {
	n, err := a.trash.EmptyTrash()
	if err == nil {
		a.emit("fs:changed", map[string]any{"action": "empty-trash", "count": n})
	}
	return n, err
}

// GetTrashInfo describes the last trash transaction so the UI can offer Undo.
func (a *App) GetTrashInfo() map[string]any {
	txn, ok := a.trash.LastTransactionInfo()
	if !ok {
		return map[string]any{"available": false}
	}
	return map[string]any{
		"available": true,
		"id":        txn.ID,
		"count":     len(txn.Items),
		"mounts":    txn.Mounts,
		"usedGio":   txn.UsedGio,
	}
}

// ---------- Open With / default apps ----------

// GetAppsForPath returns the detected MIME type, the default app, and all apps
// that can open the given file (the "Open With" menu data).
func (a *App) GetAppsForPath(path string) (map[string]any, error) {
	abs, err := fsAbs(path)
	if err != nil {
		return nil, err
	}
	info, err := fs.BuildFileEntryFromPath(abs)
	if err != nil {
		return nil, err
	}
	mimeType := a.mime.DetectMimeType(abs, info.IsDir)
	apps := fs.AppsForMime(mimeType)
	var defaultApp *fs.DesktopApp
	for i := range apps {
		if apps[i].IsDefault {
			defaultApp = &apps[i]
			break
		}
	}
	return map[string]any{
		"mimeType":   mimeType,
		"apps":       apps,
		"defaultApp": defaultApp,
	}, nil
}

// OpenWith opens paths with a specific desktop application ID.
func (a *App) OpenWith(desktopID string, paths []string) error {
	return a.mime.OpenWithApplication(desktopID, paths)
}

// OpenDefault opens paths with the system default handler.
func (a *App) OpenDefault(paths []string) error { return fs.OpenDefault(paths) }

// OpenWithDialog opens the desktop environment's native application chooser.
func (a *App) OpenWithDialog(paths []string) error { return a.mime.OpenWithDialog(paths) }

// SetDefaultApp makes a desktop app the default handler for a MIME type.
func (a *App) SetDefaultApp(mimeType, desktopID string) error {
	return a.mime.SetDefaultApp(mimeType, desktopID)
}

// RunCustomCommand executes a user-supplied command with XDG field-code expansion.
func (a *App) RunCustomCommand(commandLine string, paths []string) error {
	return fs.LaunchCommand(commandLine, paths)
}

// ListInstalledApps returns every launchable desktop application.
func (a *App) ListInstalledApps() []fs.DesktopApp { return fs.DiscoverInstalledApps() }

// ---------- Native integration (file manager, terminal, clipboard, drag-out) ----------

// ShowInNativeFileManager reveals the selection in Nautilus/Dolphin/Nemo/Thunar.
func (a *App) ShowInNativeFileManager(paths []string) error {
	return fs.OpenInNativeFileManager(paths)
}

// OpenTerminal opens a terminal emulator in the given directory.
func (a *App) OpenTerminal(dir string) error { return fs.OpenTerminalAt(dir) }

// CopyToClipboard puts the selection on the system clipboard in GNOME/KDE
// file-list formats so it can be pasted into other Linux apps.
func (a *App) CopyToClipboard(paths []string, cut bool) error {
	return fs.CopyPathsToClipboard(paths, cut)
}

// ReadClipboard reads a file list (or text) from the system clipboard.
func (a *App) ReadClipboard() (fs.ClipboardPayload, error) { return fs.ReadClipboard() }

// StartOutboundDrag launches a native GTK drag source so files can be dragged
// out of the webview into any Linux application.
func (a *App) StartOutboundDrag(paths []string) error {
	return fs.StartNativeDrag(paths)
}

// ---------- Search ----------

// Search starts a recursive search and returns matching entries.
func (a *App) Search(rootPath, query string, showHidden bool) ([]fs.FileEntry, error) {
	a.searchMu.Lock()
	if a.searchCancel != nil {
		a.searchCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.searchCtx, a.searchCancel = ctx, cancel
	a.searchMu.Unlock()

	return fs.SearchDirectory(ctx, rootPath, query, showHidden, 400)
}

// CancelSearch aborts the in-flight recursive search.
func (a *App) CancelSearch() {
	a.searchMu.Lock()
	defer a.searchMu.Unlock()
	if a.searchCancel != nil {
		a.searchCancel()
		a.searchCancel = nil
	}
}

// ResolveDropTarget decides the destination directory for an inbound drop:
// the hovered folder when dropping onto one, otherwise the active directory.
func (a *App) ResolveDropTarget(activeDir, hoveredPath string) string {
	if hoveredPath != "" {
		if info, err := fs.BuildFileEntryFromPath(hoveredPath); err == nil && info.IsDir {
			return info.Path
		}
	}
	return filepath.Clean(activeDir)
}
