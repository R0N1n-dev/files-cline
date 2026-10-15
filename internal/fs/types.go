package fs

import "time"

// EventEmitter abstracts Wails runtime.EventsEmit so internal/fs can emit
// real-time progress and notifications while remaining testable with pure Go stdlib.
type EventEmitter func(eventName string, data any)

// FileEntry represents a single file or directory item in a virtual-scrolled view.
type FileEntry struct {
	Name          string    `json:"name"`
	Path          string    `json:"path"`
	Size          int64     `json:"size"`
	ModTime       time.Time `json:"modTime"`
	ModTimeUnix   int64     `json:"modTimeUnix"`
	IsDir         bool      `json:"isDir"`
	IsSymlink     bool      `json:"isSymlink"`
	SymlinkTarget string    `json:"symlinkTarget,omitempty"`
	IsBrokenLink  bool      `json:"isBrokenLink,omitempty"`
	IsHidden      bool      `json:"isHidden"`
	IsExecutable  bool      `json:"isExecutable"`
	Extension     string    `json:"extension"`
	MimeType      string    `json:"mimeType"`
	Category      string    `json:"category"` // folder, image, video, audio, archive, code, document, pdf, file
	Permissions   string    `json:"permissions"`
	DeviceID      uint64    `json:"deviceId"`
	ItemCount     int       `json:"itemCount,omitempty"`
	// Trash-specific fields when viewing trash://
	TrashID          string    `json:"trashId,omitempty"`
	OriginalPath     string    `json:"originalPath,omitempty"`
	DeletionDate     time.Time `json:"deletionDate,omitempty"`
	TrashMountPoint  string    `json:"trashMountPoint,omitempty"`
	TrashDirLocation string    `json:"trashDirLocation,omitempty"`
}

// BreadcrumbSegment represents a clickable segment in the Nautilus path bar.
type BreadcrumbSegment struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// DirectoryListing contains full metadata and sorted entries for a directory.
type DirectoryListing struct {
	Path            string              `json:"path"`
	ParentPath      string              `json:"parentPath"`
	Breadcrumbs     []BreadcrumbSegment `json:"breadcrumbs"`
	Entries         []FileEntry         `json:"entries"`
	TotalCount      int                 `json:"totalCount"`
	HiddenCount     int                 `json:"hiddenCount"`
	TotalSize       int64               `json:"totalSize"`
	DiskTotal       uint64              `json:"diskTotal"`
	DiskFree        uint64              `json:"diskFree"`
	DiskUsedPercent float64             `json:"diskUsedPercent"`
	MountPoint      string              `json:"mountPoint"`
	FileSystemType  string              `json:"fileSystemType"`
	Writable        bool                `json:"writable"`
	IsTrashView     bool                `json:"isTrashView"`
}

// PlaceItem represents a sidebar shortcut (XDG user directory, bookmark, or trash).
type PlaceItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	Icon  string `json:"icon"`
	Kind  string `json:"kind"` // xdg, trash, bookmark
	Badge int    `json:"badge,omitempty"`
}

// MountVolume represents a mounted filesystem on Linux.
type MountVolume struct {
	Name        string  `json:"name"`
	MountPoint  string  `json:"mountPoint"`
	Device      string  `json:"device"`
	FSType      string  `json:"fsType"`
	TotalBytes  uint64  `json:"totalBytes"`
	FreeBytes   uint64  `json:"freeBytes"`
	UsedBytes   uint64  `json:"usedBytes"`
	UsedPercent float64 `json:"usedPercent"`
	IsRemovable bool    `json:"isRemovable"`
	HasTrash    bool    `json:"hasTrash"`
	TrashCount  int     `json:"trashCount"`
}

// SidebarData groups places, mounts, and bookmarks for the left navigation pane.
type SidebarData struct {
	HomeDir   string        `json:"homeDir"`
	UserName  string        `json:"userName"`
	HostName  string        `json:"hostName"`
	Places    []PlaceItem   `json:"places"`
	Volumes   []MountVolume `json:"volumes"`
	Bookmarks []PlaceItem   `json:"bookmarks"`
}

// ArchiveEntrySummary represents a file inside a zip/tar archive for Quick Peek.
type ArchiveEntrySummary struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"isDir"`
	ModTime string `json:"modTime"`
}

// FileInspection holds detailed metadata for the Inspector Drawer & Spacebar Quick Peek.
type FileInspection struct {
	Entry          FileEntry             `json:"entry"`
	OwnerName      string                `json:"ownerName"`
	GroupName      string                `json:"groupName"`
	TextPreview    string                `json:"textPreview,omitempty"`
	TextLineCount  int                   `json:"textLineCount,omitempty"`
	IsBinary       bool                  `json:"isBinary"`
	EXIF           *EXIFMetadata         `json:"exif,omitempty"`
	ThumbnailURI   string                `json:"thumbnailUri,omitempty"`
	ArchiveEntries []ArchiveEntrySummary `json:"archiveEntries,omitempty"`
	ArchiveTotal   int                   `json:"archiveTotal,omitempty"`
	SubfolderCount int                   `json:"subfolderCount,omitempty"`
	SubfileCount   int                   `json:"subfileCount,omitempty"`
	CalculatedSize int64                 `json:"calculatedSize,omitempty"`
	DefaultApp     *DesktopApp           `json:"defaultApp,omitempty"`
}

// RenamePair specifies a single rename instruction in a batch rename operation.
type RenamePair struct {
	OldPath string `json:"oldPath"`
	NewName string `json:"newName"`
}
