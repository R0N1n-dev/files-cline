package fs

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// percentEncodePath escapes a path for .trashinfo while keeping "/" literal.
func percentEncodePath(p string) string {
	const hexDigits = "0123456789ABCDEF"
	var sb strings.Builder
	for i := 0; i < len(p); i++ {
		c := p[i]
		safe := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '/' || c == '-' || c == '_' ||
			c == '.' || c == '~' || c == ' '
		if safe {
			sb.WriteByte(c)
			continue
		}
		sb.WriteByte('%')
		sb.WriteByte(hexDigits[c>>4])
		sb.WriteByte(hexDigits[c&0x0f])
	}
	return sb.String()
}

// percentDecode reverses percentEncodePath.
func percentDecode(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			hi := hexVal(s[i+1])
			lo := hexVal(s[i+2])
			if hi >= 0 && lo >= 0 {
				sb.WriteByte(byte(hi<<4 | lo))
				i += 2
				continue
			}
		}
		sb.WriteByte(s[i])
	}
	return sb.String()
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// writeTrashInfo writes a spec-compliant .trashinfo file for an item.
func writeTrashInfo(infoPath, originalPath string, deletionDate time.Time) error {
	pathValue := originalPath
	body := "[Trash Info]\n" +
		"Path=" + percentEncodePath(pathValue) + "\n" +
		"DeletionDate=" + deletionDate.Format("2006-01-02T15:04:05") + "\n"
	return os.WriteFile(infoPath, []byte(body), 0600)
}

// trashPayloadName returns a collision-free name inside the trash files/ dir.
func trashPayloadName(trashDir, baseName string) string {
	target := filepath.Join(trashDir, "files", baseName)
	if _, err := os.Lstat(target); err != nil {
		return baseName
	}
	ext := filepath.Ext(baseName)
	stem := strings.TrimSuffix(baseName, ext)
	for i := 2; i < 100000; i++ {
		candidate := stem + "." + itoa(uint32(i)) + ext
		if _, err := os.Lstat(filepath.Join(trashDir, "files", candidate)); err != nil {
			return candidate
		}
	}
	return baseName + "." + itoa(uint32(time.Now().UnixNano()%100000))
}

func sortTrashItemsByDate(items []TrashItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].DeletionDate.After(items[j].DeletionDate)
	})
}

// findTrashItemByOriginal locates a freshly trashed item by its original path.
func (t *TrashManager) findTrashItemByOriginal(originalPath string) *TrashItem {
	for _, dir := range t.ListAllTrashDirs() {
		for _, item := range scanTrashDir(dir) {
			if item.OriginalPath == originalPath {
				cp := item
				return &cp
			}
		}
	}
	return nil
}
