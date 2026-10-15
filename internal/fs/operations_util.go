package fs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// measureSources computes total bytes and item counts for progress/ETA accuracy.
func measureSources(sources []string) (totalBytes int64, items int) {
	for _, src := range sources {
		abs := ExpandPath(src)
		info, err := os.Stat(abs)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			totalBytes += info.Size()
			items++
			continue
		}
		_ = filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if fi, err := d.Info(); err == nil {
				totalBytes += fi.Size()
				items++
			}
			return nil
		})
	}
	return totalBytes, items
}

// uniqueDestination returns a non-colliding destination path using "name (2).ext".
func uniqueDestination(destDir, baseName string) string {
	target := filepath.Join(destDir, baseName)
	if _, err := os.Lstat(target); err != nil {
		return target
	}
	ext := filepath.Ext(baseName)
	stem := strings.TrimSuffix(baseName, ext)
	for i := 2; i < 10000; i++ {
		candidate := filepath.Join(destDir, fmt.Sprintf("%s (%d)%s", stem, i, ext))
		if _, err := os.Lstat(candidate); err != nil {
			return candidate
		}
	}
	return filepath.Join(destDir, fmt.Sprintf("%s-copy%s", stem, ext))
}

// revertEntries undoes recorded filesystem actions in reverse order.
func revertEntries(entries []UndoEntry) error {
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		switch {
		case e.MovedFrom != "":
			if _, err := os.Lstat(e.MovedTo); err == nil {
				_ = os.MkdirAll(filepath.Dir(e.MovedFrom), 0755)
				if err := os.Rename(e.MovedTo, e.MovedFrom); err != nil {
					return fmt.Errorf("undo move failed: %w", err)
				}
			}
		case e.CreatedPath != "":
			if err := os.RemoveAll(e.CreatedPath); err != nil {
				return fmt.Errorf("undo copy failed: %w", err)
			}
		}
	}
	return nil
}

func fileSizeOf(path string) int64 {
	if info, err := os.Stat(path); err == nil {
		return info.Size()
	}
	return 0
}

func fileModTime(path string) time.Time {
	if info, err := os.Stat(path); err == nil {
		return info.ModTime()
	}
	return time.Now()
}

func isCrossDeviceError(err error) bool {
	if err == nil {
		return false
	}
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		return strings.Contains(linkErr.Err.Error(), "cross-device") ||
			strings.Contains(linkErr.Err.Error(), "EXDEV")
	}
	return strings.Contains(err.Error(), "cross-device") || strings.Contains(err.Error(), "EXDEV")
}
