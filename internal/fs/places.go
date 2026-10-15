package fs

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
)

func BuildBreadcrumbs(absPath string) []BreadcrumbSegment {
	clean := filepath.Clean(absPath)
	if clean == "/" {
		return []BreadcrumbSegment{{Name: "/", Path: "/"}}
	}
	parts := strings.Split(strings.TrimPrefix(clean, "/"), "/")
	crumbs := make([]BreadcrumbSegment, 0, len(parts)+1)
	crumbs = append(crumbs, BreadcrumbSegment{Name: "/", Path: "/"})
	curr := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		curr = curr + "/" + p
		crumbs = append(crumbs, BreadcrumbSegment{Name: p, Path: curr})
	}
	return crumbs
}

func ExpandPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "~" {
		if h, err := os.UserHomeDir(); err == nil {
			return h
		}
		return "/workspace"
	}
	if strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

func GetDiskUsage(path string) (total uint64, free uint64, usedPercent float64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0
	}
	total = stat.Blocks * uint64(stat.Bsize)
	free = stat.Bavail * uint64(stat.Bsize)
	if total > 0 && total >= free {
		used := total - free
		usedPercent = (float64(used) / float64(total)) * 100.0
	}
	return total, free, usedPercent
}

func isDirWritable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return (info.Mode().Perm() & 0222) != 0
}

func GetSidebarData(trashMgr *TrashManager, customBookmarks []string) SidebarData {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/workspace"
	}
	userName := "user"
	if u, err := user.Current(); err == nil && u.Username != "" {
		userName = u.Username
	}
	hostName, _ := os.Hostname()

	places := []PlaceItem{
		{ID: "home", Name: "Home", Path: homeDir, Icon: "home", Kind: "xdg"},
	}

	xdgCandidates := []struct {
		id, name, fallback, icon string
	}{
		{"workspace", "Workspace", "/workspace", "code"},
		{"desktop", "Desktop", filepath.Join(homeDir, "Desktop"), "desktop"},
		{"documents", "Documents", filepath.Join(homeDir, "Documents"), "documents"},
		{"downloads", "Downloads", filepath.Join(homeDir, "Downloads"), "downloads"},
		{"music", "Music", filepath.Join(homeDir, "Music"), "music"},
		{"pictures", "Pictures", filepath.Join(homeDir, "Pictures"), "pictures"},
		{"videos", "Videos", filepath.Join(homeDir, "Videos"), "videos"},
	}

	seenPaths := map[string]bool{homeDir: true}
	for _, c := range xdgCandidates {
		if info, err := os.Stat(c.fallback); err == nil && info.IsDir() && !seenPaths[c.fallback] {
			seenPaths[c.fallback] = true
			places = append(places, PlaceItem{ID: c.id, Name: c.name, Path: c.fallback, Icon: c.icon, Kind: "xdg"})
		}
	}

	trashCount := 0
	if trashMgr != nil {
		trashCount = trashMgr.TotalTrashItemCount()
	}
	places = append(places, PlaceItem{
		ID: "trash", Name: "Trash", Path: "trash://", Icon: "trash", Kind: "trash", Badge: trashCount,
	})

	bookmarks := make([]PlaceItem, 0, len(customBookmarks))
	for i, bp := range customBookmarks {
		clean := filepath.Clean(ExpandPath(bp))
		bookmarks = append(bookmarks, PlaceItem{
			ID: fmt.Sprintf("bm-%d", i), Name: filepath.Base(clean), Path: clean, Icon: "bookmark", Kind: "bookmark",
		})
	}

	return SidebarData{
		HomeDir:   homeDir,
		UserName:  userName,
		HostName:  hostName,
		Places:    places,
		Volumes:   DiscoverMountedVolumes(trashMgr),
		Bookmarks: bookmarks,
	}
}
