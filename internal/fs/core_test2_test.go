package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchDirectoryRespectsHidden(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	writeSizedFile(t, filepath.Join(dir, "sub", "needle.txt"), 8)
	writeSizedFile(t, filepath.Join(dir, ".needle-hidden"), 8)

	results, err := SearchDirectory(context.Background(), dir, "needle", false, 50)
	if err != nil {
		t.Fatalf("SearchDirectory: %v", err)
	}
	if len(results) != 1 || results[0].Name != "needle.txt" {
		t.Fatalf("results = %+v", results)
	}

	all, err := SearchDirectory(context.Background(), dir, "needle", true, 50)
	if err != nil || len(all) != 2 {
		t.Fatalf("hidden search results = %+v (err=%v)", all, err)
	}
}

func TestSearchDirectoryHonoursCancellation(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 20; i++ {
		writeSizedFile(t, filepath.Join(dir, "f"+itoa(uint32(i))+".txt"), 4)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := SearchDirectory(ctx, dir, "f", false, 50); err == nil {
		t.Fatal("cancelled search should report an error")
	}
}

func TestGetDiskUsageAndMountPoint(t *testing.T) {
	total, free, pct := GetDiskUsage(t.TempDir())
	if total == 0 {
		t.Fatal("expected non-zero total disk size")
	}
	if free > total || pct < 0 || pct > 100 {
		t.Fatalf("implausible usage: total=%d free=%d pct=%f", total, free, pct)
	}
	mp, fstype := FindMountPoint("/workspace")
	if mp == "" || fstype == "" {
		t.Fatalf("mount point = %q fs = %q", mp, fstype)
	}
}

func TestConfigStoreRoundTrip(t *testing.T) {
	withTempXDG(t)
	store := NewConfigStore()
	cfg := store.Get()
	if cfg.SortBy == "" || cfg.IconSize == 0 {
		t.Fatalf("defaults missing: %+v", cfg)
	}

	cfg.Theme = "dark"
	cfg.ViewMode = "list"
	cfg.SortBy = "modified"
	cfg.SortDesc = true
	cfg.ShowHidden = true
	if _, err := store.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded := NewConfigStore().Get()
	if reloaded.Theme != "dark" || reloaded.ViewMode != "list" || !reloaded.SortDesc || !reloaded.ShowHidden {
		t.Fatalf("reloaded config = %+v", reloaded)
	}

	bookmarkDir := t.TempDir()
	if _, err := store.AddBookmark(bookmarkDir); err != nil {
		t.Fatalf("AddBookmark: %v", err)
	}
	marks := NewConfigStore().Get().Bookmarks
	if len(marks) != 1 || marks[0] != bookmarkDir {
		t.Fatalf("bookmarks = %v", marks)
	}
	if after, err := store.RemoveBookmark(bookmarkDir); err != nil || len(after) != 0 {
		t.Fatalf("RemoveBookmark = %v, %v", after, err)
	}
}

func TestDuplicateItemsCreatesCopy(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "doc.txt")
	writeSizedFile(t, original, 64)

	created, err := DuplicateItems([]string{original})
	if err != nil {
		t.Fatalf("DuplicateItems: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("created = %v", created)
	}
	if info, err := os.Stat(created[0]); err != nil || info.Size() != 64 {
		t.Fatalf("duplicate missing: %v", err)
	}
	if info, err := os.Stat(original); err != nil || info.Size() != 64 {
		t.Fatalf("original must remain: %v", err)
	}
}

func TestChmodPathsAppliesMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "script.sh")
	writeSizedFile(t, target, 8)

	if _, err := ChmodPaths([]string{target}, "755"); err != nil {
		t.Fatalf("ChmodPaths: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Fatalf("mode = %v, want 0755", info.Mode().Perm())
	}
	if _, err := ChmodPaths([]string{target}, "999"); err == nil {
		t.Fatal("invalid octal mode must be rejected")
	}
}
