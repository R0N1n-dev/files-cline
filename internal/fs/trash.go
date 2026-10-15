package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TrashItem describes one trashed file or directory, from any mounted filesystem.
type TrashItem struct {
	ID              string    `json:"id"` // absolute path of the .trashinfo file
	Name            string    `json:"name"`
	OriginalPath    string    `json:"originalPath"`
	DeletionDate    time.Time `json:"deletionDate"`
	InfoPath        string    `json:"infoPath"`
	PayloadPath     string    `json:"payloadPath"`
	TrashDir        string    `json:"trashDir"`
	MountPoint      string    `json:"mountPoint"`
	IsTopdir        bool      `json:"isTopdir"`
	IsDir           bool      `json:"isDir"`
	Size            int64     `json:"size"`
	OriginalMissing bool      `json:"originalMissing"`
}

// TrashTransaction records one trash action so it can be undone across mounts.
type TrashTransaction struct {
	ID      string      `json:"id"`
	When    time.Time   `json:"when"`
	Items   []TrashItem `json:"items"`
	Undone  bool        `json:"undone"`
	UsedGio bool        `json:"usedGio"`
	Mounts  []string    `json:"mounts"`
}

// TrashManager implements the FreeDesktop.org Trash specification across all
// mounted filesystems, with gio fallback and full cross-mount undo.
type TrashManager struct {
	emit EventEmitter
	mu   struct {
		undo []TrashTransaction
	}
}

// NewTrashManager creates a trash manager.
func NewTrashManager(emit EventEmitter) *TrashManager {
	return &TrashManager{emit: emit}
}

func (t *TrashManager) emitEvent(name string, data any) {
	if t.emit != nil {
		t.emit(name, data)
	}
}

// HomeTrashDir returns $XDG_DATA_HOME/Trash (default ~/.local/share/Trash).
func HomeTrashDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "Trash")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}
	return filepath.Join(home, ".local", "share", "Trash")
}

// FindExistingTopdirTrash returns an existing topdir trash directory for a mount
// point, honouring the .Trash/<uid> then .Trash-<uid> preference order.
func (t *TrashManager) FindExistingTopdirTrash(mountPoint string) string {
	uid := itoa(uint32(os.Getuid()))
	shared := filepath.Join(mountPoint, ".Trash", uid)
	if info, err := os.Stat(shared); err == nil && info.IsDir() {
		return shared
	}
	private := filepath.Join(mountPoint, ".Trash-"+uid)
	if info, err := os.Stat(private); err == nil && info.IsDir() {
		return private
	}
	return ""
}

// topdirTrashFor determines the writable trash directory for a path on a mount,
// creating .Trash-<uid> when required by the spec.
func (t *TrashManager) topdirTrashFor(absPath string) (string, string, error) {
	mountPoint, _ := FindMountPoint(absPath)
	uid := itoa(uint32(os.Getuid()))

	sharedRoot := filepath.Join(mountPoint, ".Trash")
	if info, err := os.Stat(sharedRoot); err == nil && info.IsDir() && info.Mode()&os.ModeSticky != 0 {
		dir := filepath.Join(sharedRoot, uid)
		if err := os.MkdirAll(filepath.Join(dir, "files"), 0700); err != nil {
			return "", mountPoint, err
		}
		_ = os.MkdirAll(filepath.Join(dir, "info"), 0700)
		return dir, mountPoint, nil
	}

	dir := filepath.Join(mountPoint, ".Trash-"+uid)
	if err := os.MkdirAll(filepath.Join(dir, "files"), 0700); err != nil {
		return "", mountPoint, fmt.Errorf("no writable trash on %s: %w", mountPoint, err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "info"), 0700); err != nil {
		return "", mountPoint, fmt.Errorf("no writable trash on %s: %w", mountPoint, err)
	}
	return dir, mountPoint, nil
}

// ListAllTrashDirs returns every trash directory visible on this system:
// the home trash plus one per mount that has a topdir trash.
func (t *TrashManager) ListAllTrashDirs() []string {
	home := HomeTrashDir()
	dirs := []string{home}
	seen := map[string]bool{home: true}

	for _, v := range DiscoverMountedVolumes(nil) {
		if v.MountPoint == "/" {
			continue
		}
		if d := t.FindExistingTopdirTrash(v.MountPoint); d != "" && !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// CountTrashInDir counts payload entries in a trash directory.
func (t *TrashManager) CountTrashInDir(trashDir string) int {
	entries, err := os.ReadDir(filepath.Join(trashDir, "files"))
	if err != nil {
		return 0
	}
	return len(entries)
}

// TotalTrashItemCount aggregates item counts across every trash directory.
func (t *TrashManager) TotalTrashItemCount() int {
	total := 0
	for _, dir := range t.ListAllTrashDirs() {
		total += t.CountTrashInDir(dir)
	}
	return total
}
