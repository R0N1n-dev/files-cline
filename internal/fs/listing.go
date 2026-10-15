package fs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

func ListDirectory(rawPath string, showHidden bool, sortBy string, sortDesc bool, filterQuery string, trashMgr *TrashManager) (*DirectoryListing, error) {
	if rawPath == "trash://" || rawPath == "trash:///" {
		if trashMgr == nil {
			return nil, errors.New("trash manager not initialized")
		}
		return trashMgr.ListTrashDirectory(showHidden, sortBy, sortDesc, filterQuery)
	}

	targetPath := ExpandPath(rawPath)
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %q: %w", rawPath, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access %q: %w", absPath, err)
	}
	if !info.IsDir() {
		absPath = filepath.Dir(absPath)
	}

	dirEntries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory %q: %w", absPath, err)
	}

	filterLower := strings.ToLower(strings.TrimSpace(filterQuery))
	entries := make([]FileEntry, 0, len(dirEntries))
	hiddenCount := 0
	var totalSize int64

	for _, de := range dirEntries {
		name := de.Name()
		isHidden := strings.HasPrefix(name, ".")
		if isHidden {
			hiddenCount++
			if !showHidden {
				continue
			}
		}
		if filterLower != "" && !strings.Contains(strings.ToLower(name), filterLower) {
			continue
		}
		fullPath := filepath.Join(absPath, name)
		fe := buildFileEntry(fullPath, name, de)
		if !fe.IsDir {
			totalSize += fe.Size
		}
		entries = append(entries, fe)
	}

	SortEntries(entries, sortBy, sortDesc)

	parentPath := filepath.Dir(absPath)
	if parentPath == absPath {
		parentPath = ""
	}

	diskTotal, diskFree, diskUsedPct := GetDiskUsage(absPath)
	mountPoint, fsType := FindMountPoint(absPath)
	writable := isDirWritable(absPath)

	return &DirectoryListing{
		Path:            absPath,
		ParentPath:      parentPath,
		Breadcrumbs:     BuildBreadcrumbs(absPath),
		Entries:         entries,
		TotalCount:      len(entries),
		HiddenCount:     hiddenCount,
		TotalSize:       totalSize,
		DiskTotal:       diskTotal,
		DiskFree:        diskFree,
		DiskUsedPercent: diskUsedPct,
		MountPoint:      mountPoint,
		FileSystemType:  fsType,
		Writable:        writable,
		IsTrashView:     false,
	}, nil
}

func buildFileEntry(fullPath, name string, de fs.DirEntry) FileEntry {
	isHidden := strings.HasPrefix(name, ".")
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))

	var lstatInfo fs.FileInfo
	var err error
	if de != nil {
		lstatInfo, err = de.Info()
	}
	if lstatInfo == nil || err != nil {
		lstatInfo, err = os.Lstat(fullPath)
	}

	fe := FileEntry{
		Name:      name,
		Path:      fullPath,
		IsHidden:  isHidden,
		Extension: ext,
	}

	if err != nil || lstatInfo == nil {
		fe.MimeType = "application/octet-stream"
		fe.Category = "file"
		return fe
	}

	mode := lstatInfo.Mode()
	fe.Size = lstatInfo.Size()
	fe.ModTime = lstatInfo.ModTime()
	fe.ModTimeUnix = lstatInfo.ModTime().UnixMilli()
	fe.Permissions = mode.String()
	fe.IsDir = lstatInfo.IsDir()
	fe.IsSymlink = (mode & os.ModeSymlink) != 0
	fe.IsExecutable = !fe.IsDir && (mode.Perm()&0111) != 0

	if statT, ok := lstatInfo.Sys().(*syscall.Stat_t); ok {
		fe.DeviceID = uint64(statT.Dev)
	}

	if fe.IsSymlink {
		if target, linkErr := os.Readlink(fullPath); linkErr == nil {
			fe.SymlinkTarget = target
		}
		if resolvedInfo, statErr := os.Stat(fullPath); statErr == nil {
			fe.IsDir = resolvedInfo.IsDir()
			fe.Size = resolvedInfo.Size()
		} else {
			fe.IsBrokenLink = true
		}
	}

	if fe.IsDir {
		fe.MimeType = "inode/directory"
		fe.Category = "folder"
	} else {
		fe.MimeType, fe.Category = FastClassifyMime(name, ext)
	}
	return fe
}

func BuildFileEntryFromPath(fullPath string) (FileEntry, error) {
	abs, err := filepath.Abs(ExpandPath(fullPath))
	if err != nil {
		return FileEntry{}, err
	}
	if _, err := os.Lstat(abs); err != nil {
		return FileEntry{}, err
	}
	return buildFileEntry(abs, filepath.Base(abs), nil), nil
}

func SortEntries(entries []FileEntry, sortBy string, sortDesc bool) {
	sortBy = strings.ToLower(strings.TrimSpace(sortBy))
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		var cmp int
		switch sortBy {
		case "size":
			if a.Size < b.Size {
				cmp = -1
			} else if a.Size > b.Size {
				cmp = 1
			}
		case "modified", "date", "time":
			if a.ModTimeUnix < b.ModTimeUnix {
				cmp = -1
			} else if a.ModTimeUnix > b.ModTimeUnix {
				cmp = 1
			}
		case "type", "extension", "category":
			cmp = strings.Compare(a.Category+":"+a.Extension, b.Category+":"+b.Extension)
		default:
			cmp = naturalCompare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		}
		if cmp == 0 {
			cmp = naturalCompare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		}
		if sortDesc {
			return cmp > 0
		}
		return cmp < 0
	})
}

func naturalCompare(a, b string) int {
	ia, ib := 0, 0
	for ia < len(a) && ib < len(b) {
		ca, cb := a[ia], b[ib]
		if ca >= '0' && ca <= '9' && cb >= '0' && cb <= '9' {
			ja, jb := ia, ib
			for ja < len(a) && a[ja] >= '0' && a[ja] <= '9' {
				ja++
			}
			for jb < len(b) && b[jb] >= '0' && b[jb] <= '9' {
				jb++
			}
			na := strings.TrimLeft(a[ia:ja], "0")
			nb := strings.TrimLeft(b[ib:jb], "0")
			if len(na) != len(nb) {
				if len(na) < len(nb) {
					return -1
				}
				return 1
			}
			if na != nb {
				if na < nb {
					return -1
				}
				return 1
			}
			ia, ib = ja, jb
			continue
		}
		if ca != cb {
			if ca < cb {
				return -1
			}
			return 1
		}
		ia++
		ib++
	}
	return len(a) - len(b)
}
