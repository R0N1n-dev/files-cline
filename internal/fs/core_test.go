package fs

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNaturalSortAndDirectoryGrouping(t *testing.T) {
	entries := []FileEntry{
		{Name: "file10.txt", IsDir: false},
		{Name: "file2.txt", IsDir: false},
		{Name: "zeta", IsDir: true},
		{Name: "alpha", IsDir: true},
	}
	SortEntries(entries, "name", false)

	got := []string{entries[0].Name, entries[1].Name, entries[2].Name, entries[3].Name}
	want := []string{"alpha", "zeta", "file2.txt", "file10.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sort = %v, want %v", got, want)
	}
}

func TestSortBySizeDescending(t *testing.T) {
	entries := []FileEntry{
		{Name: "small", Size: 10},
		{Name: "large", Size: 900},
		{Name: "medium", Size: 100},
	}
	SortEntries(entries, "size", true)
	if entries[0].Name != "large" || entries[2].Name != "small" {
		t.Fatalf("size desc sort wrong: %v", entries)
	}
}

func TestBatchRenameHandlesSwaps(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	writeSizedFile(t, a, 11)
	writeSizedFile(t, b, 22)

	entries, err := BatchRename([]RenamePair{
		{OldPath: a, NewName: "b.txt"},
		{OldPath: b, NewName: "a.txt"},
	})
	if err != nil {
		t.Fatalf("BatchRename: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("renamed %d entries, want 2", len(entries))
	}
	if info, err := os.Stat(a); err != nil || info.Size() != 22 {
		t.Fatalf("a.txt should now hold the original b content: %v", err)
	}
	if info, err := os.Stat(b); err != nil || info.Size() != 11 {
		t.Fatalf("b.txt should now hold the original a content: %v", err)
	}
}

func TestBatchRenameRejectsDuplicates(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	writeSizedFile(t, a, 1)
	writeSizedFile(t, b, 1)
	if _, err := BatchRename([]RenamePair{
		{OldPath: a, NewName: "same.txt"},
		{OldPath: b, NewName: "same.txt"},
	}); err == nil {
		t.Fatal("duplicate targets must be rejected")
	}
}

func TestCreateFolderAndFileValidation(t *testing.T) {
	dir := t.TempDir()
	if _, err := CreateFolder(dir, "New Folder"); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if _, err := CreateFolder(dir, "New Folder"); err == nil {
		t.Fatal("duplicate folder must fail")
	}
	if _, err := CreateFolder(dir, "bad/name"); err == nil {
		t.Fatal("slashes must be rejected")
	}
	if _, err := CreateFile(dir, "notes.md", "# hi"); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "notes.md"))
	if err != nil || string(data) != "# hi" {
		t.Fatalf("file content wrong: %q err=%v", data, err)
	}
}

func TestRenameItemAndExtensionClassify(t *testing.T) {
	dir := t.TempDir()
	writeSizedFile(t, filepath.Join(dir, "old.txt"), 5)
	entry, err := RenameItem(filepath.Join(dir, "old.txt"), "new.md")
	if err != nil {
		t.Fatalf("RenameItem: %v", err)
	}
	if entry.Name != "new.md" || entry.Extension != "md" {
		t.Fatalf("entry = %+v", entry)
	}

	mime, category := FastClassifyMime("photo.JPEG", "jpeg")
	if mime != "image/jpeg" || category != "image" {
		t.Fatalf("classify = %s/%s", mime, category)
	}
	if _, cat := FastClassifyMime("main.go", "go"); cat != "code" {
		t.Fatalf("go files should be code, got %s", cat)
	}
}

func TestListDirectoryFiltersHiddenAndSorts(t *testing.T) {
	dir := t.TempDir()
	writeSizedFile(t, filepath.Join(dir, "visible.txt"), 10)
	writeSizedFile(t, filepath.Join(dir, ".hidden"), 10)
	if err := os.MkdirAll(filepath.Join(dir, "folder"), 0755); err != nil {
		t.Fatal(err)
	}

	listing, err := ListDirectory(dir, false, "name", false, "", nil)
	if err != nil {
		t.Fatalf("ListDirectory: %v", err)
	}
	if listing.TotalCount != 2 {
		t.Fatalf("total = %d, want 2 (hidden excluded)", listing.TotalCount)
	}
	if listing.HiddenCount != 1 {
		t.Fatalf("hidden count = %d, want 1", listing.HiddenCount)
	}
	if !listing.Entries[0].IsDir {
		t.Fatal("folders must sort first")
	}
	if len(listing.Breadcrumbs) == 0 || listing.Breadcrumbs[len(listing.Breadcrumbs)-1].Path != dir {
		t.Fatalf("breadcrumbs = %+v", listing.Breadcrumbs)
	}

	filtered, err := ListDirectory(dir, true, "name", false, "vis", nil)
	if err != nil {
		t.Fatalf("ListDirectory filtered: %v", err)
	}
	if filtered.TotalCount != 1 || filtered.Entries[0].Name != "visible.txt" {
		t.Fatalf("filter result = %+v", filtered.Entries)
	}
}

func TestExpandPathResolvesHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	if got := ExpandPath("~"); got != home {
		t.Fatalf("ExpandPath(~) = %q, want %q", got, home)
	}
	if got := ExpandPath("~/docs"); got != filepath.Join(home, "docs") {
		t.Fatalf("ExpandPath(~/docs) = %q", got)
	}
}
