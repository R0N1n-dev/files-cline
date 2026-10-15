package fs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func CreateFolder(parentDir, name string) (FileEntry, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "/") {
		return FileEntry{}, errors.New("invalid folder name")
	}
	full := filepath.Join(ExpandPath(parentDir), name)
	if _, err := os.Stat(full); err == nil {
		return FileEntry{}, fmt.Errorf("%q already exists", name)
	}
	if err := os.MkdirAll(full, 0755); err != nil {
		return FileEntry{}, err
	}
	return BuildFileEntryFromPath(full)
}

func CreateFile(parentDir, name, initialContent string) (FileEntry, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "/") {
		return FileEntry{}, errors.New("invalid file name")
	}
	full := filepath.Join(ExpandPath(parentDir), name)
	if _, err := os.Stat(full); err == nil {
		return FileEntry{}, fmt.Errorf("%q already exists", name)
	}
	if err := os.WriteFile(full, []byte(initialContent), 0644); err != nil {
		return FileEntry{}, err
	}
	return BuildFileEntryFromPath(full)
}

func RenameItem(oldPath, newName string) (FileEntry, error) {
	newName = strings.TrimSpace(newName)
	if newName == "" || strings.Contains(newName, "/") {
		return FileEntry{}, errors.New("invalid new name")
	}
	absOld, err := filepath.Abs(ExpandPath(oldPath))
	if err != nil {
		return FileEntry{}, err
	}
	newPath := filepath.Join(filepath.Dir(absOld), newName)
	if newPath == absOld {
		return BuildFileEntryFromPath(absOld)
	}
	if _, err := os.Lstat(newPath); err == nil {
		return FileEntry{}, fmt.Errorf("destination %q already exists", newName)
	}
	if err := os.Rename(absOld, newPath); err != nil {
		return FileEntry{}, err
	}
	return BuildFileEntryFromPath(newPath)
}

func BatchRename(pairs []RenamePair) ([]FileEntry, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	type resolvedMove struct {
		oldAbs string
		tmpAbs string
		newAbs string
	}
	moves := make([]resolvedMove, 0, len(pairs))
	targetSet := make(map[string]bool, len(pairs))

	for i, p := range pairs {
		newName := strings.TrimSpace(p.NewName)
		if newName == "" || strings.Contains(newName, "/") {
			return nil, fmt.Errorf("invalid target name %q", p.NewName)
		}
		oldAbs, err := filepath.Abs(ExpandPath(p.OldPath))
		if err != nil {
			return nil, err
		}
		newAbs := filepath.Join(filepath.Dir(oldAbs), newName)
		if targetSet[newAbs] {
			return nil, fmt.Errorf("duplicate target filename %q in batch", newName)
		}
		targetSet[newAbs] = true
		if oldAbs == newAbs {
			continue
		}
		tmpAbs := filepath.Join(filepath.Dir(oldAbs), fmt.Sprintf(".vela-renametmp-%d-%d", time.Now().UnixNano(), i))
		moves = append(moves, resolvedMove{oldAbs: oldAbs, tmpAbs: tmpAbs, newAbs: newAbs})
	}

	for i, m := range moves {
		if err := os.Rename(m.oldAbs, m.tmpAbs); err != nil {
			for j := i - 1; j >= 0; j-- {
				_ = os.Rename(moves[j].tmpAbs, moves[j].oldAbs)
			}
			return nil, fmt.Errorf("batch rename stage 1 failed on %q: %w", filepath.Base(m.oldAbs), err)
		}
	}

	results := make([]FileEntry, 0, len(moves))
	for i, m := range moves {
		if err := os.Rename(m.tmpAbs, m.newAbs); err != nil {
			for j := i; j < len(moves); j++ {
				_ = os.Rename(moves[j].tmpAbs, moves[j].oldAbs)
			}
			for j := i - 1; j >= 0; j-- {
				_ = os.Rename(moves[j].newAbs, moves[j].oldAbs)
			}
			return nil, fmt.Errorf("batch rename stage 2 failed on %q: %w", filepath.Base(m.newAbs), err)
		}
		if fe, err := BuildFileEntryFromPath(m.newAbs); err == nil {
			results = append(results, fe)
		}
	}

	runtime.KeepAlive(moves)
	return results, nil
}

func SearchDirectory(ctx context.Context, rootPath, query string, showHidden bool, maxResults int) ([]FileEntry, error) {
	absRoot, err := filepath.Abs(ExpandPath(rootPath))
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, nil
	}
	if maxResults <= 0 {
		maxResults = 300
	}

	var results []FileEntry
	walkErr := filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil || path == absRoot {
			return nil
		}
		name := d.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(strings.ToLower(name), q) {
			results = append(results, buildFileEntry(path, name, d))
			if len(results) >= maxResults {
				return io.EOF
			}
		}
		return nil
	})

	if walkErr != nil && !errors.Is(walkErr, io.EOF) && !errors.Is(walkErr, context.Canceled) {
		return results, walkErr
	}
	SortEntries(results, "name", false)
	return results, nil
}
