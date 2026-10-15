package main

import (
	"context"
	"sync"

	"vela-files/internal/fs"
)

// App is the Wails-bound backend. It deliberately avoids importing the Wails
// runtime so the whole API stays unit-testable and reusable by the headless
// dev server; event emission is injected as a callback.
type App struct {
	emit         fs.EventEmitter
	thumbs       *fs.ThumbnailPool
	trash        *fs.TrashManager
	ops          *fs.OperationsManager
	mime         *fs.MimeHelper
	config       *fs.ConfigStore
	searchCtx    context.Context
	searchMu     sync.Mutex
	searchCancel context.CancelFunc
}

// NewApp wires up the backend services with the supplied event emitter.
func NewApp(emit fs.EventEmitter) *App {
	if emit == nil {
		emit = func(string, any) {}
	}
	app := &App{
		emit:   emit,
		trash:  fs.NewTrashManager(emit),
		ops:    fs.NewOperationsManager(emit),
		mime:   fs.NewMimeHelper(),
		config: fs.NewConfigStore(),
		thumbs: fs.NewThumbnailPool(4, ""),
	}
	app.thumbs.SetResultHandler(func(res fs.ThumbnailResult) {
		emit("thumbs:ready", res)
	})
	return app
}

// Close releases background resources (called on shutdown).
func (a *App) Close() {
	a.thumbs.Close()
	a.ops.CancelAll()
}

// ---------- Navigation & listing ----------

// ListDirectory returns a sorted, filtered directory listing ready for virtual scrolling.
func (a *App) ListDirectory(path string, showHidden bool, sortBy string, sortDesc bool, filter string) (*fs.DirectoryListing, error) {
	return fs.ListDirectory(path, showHidden, sortBy, sortDesc, filter, a.trash)
}

// GetSidebar returns XDG places, mounted volumes with free space, and bookmarks.
func (a *App) GetSidebar() fs.SidebarData {
	cfg := a.config.Get()
	return fs.GetSidebarData(a.trash, cfg.Bookmarks)
}

// GetHomeDir returns the current user's home directory.
func (a *App) GetHomeDir() string {
	home, err := fsHome()
	if err != nil {
		return "/"
	}
	return home
}

// GetConfig returns the persisted configuration.
func (a *App) GetConfig() fs.AppConfig { return a.config.Get() }

// SaveConfig merges and persists a configuration update.
func (a *App) SaveConfig(cfg fs.AppConfig) (fs.AppConfig, error) { return a.config.Save(cfg) }

// AddBookmark pins a folder to the sidebar.
func (a *App) AddBookmark(path string) ([]string, error) { return a.config.AddBookmark(path) }

// RemoveBookmark unpins a folder from the sidebar.
func (a *App) RemoveBookmark(path string) ([]string, error) { return a.config.RemoveBookmark(path) }

// GetMounts lists mounted filesystems with capacity information.
func (a *App) GetMounts() []fs.MountVolume { return fs.DiscoverMountedVolumes(a.trash) }

// GetDiskUsage reports total/free bytes for the filesystem containing path.
func (a *App) GetDiskUsage(path string) map[string]any {
	total, free, pct := fs.GetDiskUsage(fs.ExpandPath(path))
	return map[string]any{"total": total, "free": free, "usedPercent": pct}
}

// ExpandPath resolves ~ and relative paths to an absolute path.
func (a *App) ExpandPath(path string) string {
	abs, err := fsAbs(path)
	if err != nil {
		return fs.ExpandPath(path)
	}
	return abs
}

// ---------- Thumbnails ----------

// GetThumbnail returns a cached or freshly generated EXIF-corrected thumbnail data URI.
func (a *App) GetThumbnail(path string, size int) (string, error) {
	return a.thumbs.GetOrGenerateThumbnail(path, size)
}

// QueueThumbnails schedules background thumbnail generation for a list of images.
func (a *App) QueueThumbnails(paths []string, size int) {
	a.thumbs.QueueBatch(paths, size)
}

// ---------- Inspection & preview ----------

// InspectFile returns deep metadata: EXIF, text preview, archive contents, folder breakdown.
func (a *App) InspectFile(path string) (*fs.FileInspection, error) {
	return fs.InspectFile(path, a.thumbs, a.mime)
}
