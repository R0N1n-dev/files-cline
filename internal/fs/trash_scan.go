package fs

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// parseTrashInfo reads a .trashinfo file and resolves the original absolute path.
func parseTrashInfo(infoPath, trashDir, mountPoint string, isTopdir bool) (TrashItem, error) {
	f, err := os.Open(infoPath)
	if err != nil {
		return TrashItem{}, err
	}
	defer f.Close()

	item := TrashItem{
		InfoPath:   infoPath,
		TrashDir:   trashDir,
		MountPoint: mountPoint,
		IsTopdir:   isTopdir,
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "Path="):
			decoded := percentDecode(strings.TrimPrefix(line, "Path="))
			if strings.HasPrefix(decoded, "/") || mountPoint == "" {
				item.OriginalPath = decoded
			} else {
				item.OriginalPath = filepath.Join(mountPoint, decoded)
			}
		case strings.HasPrefix(line, "DeletionDate="):
			val := strings.TrimPrefix(line, "DeletionDate=")
			if ts, err := time.ParseInLocation("2006-01-02T15:04:05", val, time.Local); err == nil {
				item.DeletionDate = ts
			}
		}
	}

	base := strings.TrimSuffix(filepath.Base(infoPath), ".trashinfo")
	item.Name = base
	item.ID = infoPath
	item.PayloadPath = filepath.Join(trashDir, "files", base)
	if info, err := os.Lstat(item.PayloadPath); err == nil {
		item.IsDir = info.IsDir()
		if !info.IsDir() {
			item.Size = info.Size()
		}
	}
	if _, err := os.Lstat(item.OriginalPath); err != nil {
		item.OriginalMissing = true
	}
	return item, nil
}

// scanTrashDir lists all items inside a single trash directory.
func scanTrashDir(trashDir string) []TrashItem {
	infoDir := filepath.Join(trashDir, "info")
	entries, err := os.ReadDir(infoDir)
	if err != nil {
		return nil
	}
	mountPoint := ""
	isTopdir := false
	if trashDir != HomeTrashDir() {
		isTopdir = true
		mountPoint, _ = FindMountPoint(trashDir)
	}
	items := make([]TrashItem, 0, len(entries))
	for _, de := range entries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".trashinfo") {
			continue
		}
		if item, err := parseTrashInfo(filepath.Join(infoDir, de.Name()), trashDir, mountPoint, isTopdir); err == nil {
			items = append(items, item)
		}
	}
	return items
}

// ListAllItems returns every trashed item across all mounts, newest first.
func (t *TrashManager) ListAllItems() []TrashItem {
	var all []TrashItem
	for _, dir := range t.ListAllTrashDirs() {
		all = append(all, scanTrashDir(dir)...)
	}
	sortTrashItemsByDate(all)
	return all
}

// ListTrashDirectory renders the unified trash view as a DirectoryListing.
func (t *TrashManager) ListTrashDirectory(showHidden bool, sortBy string, sortDesc bool, filterQuery string) (*DirectoryListing, error) {
	items := t.ListAllItems()
	filterLower := strings.ToLower(strings.TrimSpace(filterQuery))

	entries := make([]FileEntry, 0, len(items))
	var totalSize int64
	hiddenCount := 0
	mountSet := map[string]bool{}

	for _, it := range items {
		isHidden := strings.HasPrefix(it.Name, ".")
		if isHidden {
			hiddenCount++
			if !showHidden {
				continue
			}
		}
		if filterLower != "" && !strings.Contains(strings.ToLower(it.Name), filterLower) {
			continue
		}
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(it.Name), "."))
		mimeType, category := FastClassifyMime(it.Name, ext)
		if it.IsDir {
			mimeType, category = "inode/directory", "folder"
		}
		entries = append(entries, FileEntry{
			Name:             it.Name,
			Path:             it.PayloadPath,
			Size:             it.Size,
			ModTime:          it.DeletionDate,
			ModTimeUnix:      it.DeletionDate.UnixMilli(),
			IsDir:            it.IsDir,
			IsHidden:         isHidden,
			Extension:        ext,
			MimeType:         mimeType,
			Category:         category,
			TrashID:          it.ID,
			OriginalPath:     it.OriginalPath,
			DeletionDate:     it.DeletionDate,
			TrashMountPoint:  it.MountPoint,
			TrashDirLocation: it.TrashDir,
		})
		totalSize += it.Size
		if it.MountPoint != "" {
			mountSet[it.MountPoint] = true
		}
	}

	SortEntries(entries, sortBy, sortDesc)
	mounts := make([]string, 0, len(mountSet))
	for m := range mountSet {
		mounts = append(mounts, m)
	}

	home := HomeTrashDir()
	diskTotal, diskFree, usedPct := GetDiskUsage(home)

	return &DirectoryListing{
		Path:            "trash://",
		ParentPath:      "",
		Breadcrumbs:     []BreadcrumbSegment{{Name: "Trash", Path: "trash://"}},
		Entries:         entries,
		TotalCount:      len(entries),
		HiddenCount:     hiddenCount,
		TotalSize:       totalSize,
		DiskTotal:       diskTotal,
		DiskFree:        diskFree,
		DiskUsedPercent: usedPct,
		MountPoint:      strings.Join(mounts, ", "),
		FileSystemType:  "trash",
		Writable:        true,
		IsTrashView:     true,
	}, nil
}
