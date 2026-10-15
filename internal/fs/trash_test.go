package fs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempXDG(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(base, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(base, "config"))
	return base
}

func TestTrashAndUndoRestoresFile(t *testing.T) {
	base := withTempXDG(t)
	work := filepath.Join(base, "work")
	if err := os.MkdirAll(work, 0755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(work, "notes.txt")
	writeSizedFile(t, target, 2048)

	tm := NewTrashManager(nil)
	txn, err := tm.TrashItems([]string{target})
	if err != nil {
		t.Fatalf("TrashItems: %v", err)
	}
	if len(txn.Items) != 1 {
		t.Fatalf("expected 1 trashed item, got %d", len(txn.Items))
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("original file should be gone, stat err = %v", err)
	}

	item := txn.Items[0]
	if item.OriginalPath != target {
		t.Fatalf("original path = %q, want %q", item.OriginalPath, target)
	}
	if _, err := os.Stat(item.PayloadPath); err != nil {
		t.Fatalf("payload missing from trash: %v", err)
	}
	infoBytes, err := os.ReadFile(item.InfoPath)
	if err != nil {
		t.Fatalf("read .trashinfo: %v", err)
	}
	info := string(infoBytes)
	if !strings.Contains(info, "[Trash Info]") || !strings.Contains(info, "Path=") || !strings.Contains(info, "DeletionDate=") {
		t.Fatalf(".trashinfo is not spec-compliant:\n%s", info)
	}

	// Unified trash view lists it.
	items := tm.ListAllItems()
	if len(items) != 1 || items[0].Name != "notes.txt" {
		t.Fatalf("ListAllItems = %+v", items)
	}

	// Undo restores the file to its exact original location.
	n, err := tm.UndoLastTrash()
	if err != nil {
		t.Fatalf("UndoLastTrash: %v", err)
	}
	if n != 1 {
		t.Fatalf("restored %d items, want 1", n)
	}
	if info, err := os.Stat(target); err != nil || info.Size() != 2048 {
		t.Fatalf("file not restored: %v", err)
	}
	if len(tm.ListAllItems()) != 0 {
		t.Fatal("trash should be empty after undo")
	}
}

func TestTrashMultipleMountsRecordsEveryMount(t *testing.T) {
	base := withTempXDG(t)
	work := filepath.Join(base, "multi")
	if err := os.MkdirAll(filepath.Join(work, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(work, "a.txt")
	b := filepath.Join(work, "sub", "b.txt")
	writeSizedFile(t, a, 128)
	writeSizedFile(t, b, 256)

	tm := NewTrashManager(nil)
	txn, err := tm.TrashItems([]string{a, b})
	if err != nil {
		t.Fatalf("TrashItems: %v", err)
	}
	if len(txn.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(txn.Items))
	}
	if got := tm.TotalTrashItemCount(); got != 2 {
		t.Fatalf("TotalTrashItemCount = %d, want 2", got)
	}

	restored, err := tm.RestoreByIDs([]string{txn.Items[0].ID})
	if err != nil {
		t.Fatalf("RestoreByIDs: %v", err)
	}
	if restored != 1 {
		t.Fatalf("restored = %d, want 1", restored)
	}
	if _, err := os.Stat(txn.Items[0].OriginalPath); err != nil {
		t.Fatalf("restored file missing: %v", err)
	}

	emptied, err := tm.EmptyTrash()
	if err != nil {
		t.Fatalf("EmptyTrash: %v", err)
	}
	if emptied != 1 {
		t.Fatalf("emptied = %d, want 1 remaining item", emptied)
	}
	if tm.TotalTrashItemCount() != 0 {
		t.Fatal("trash should be empty after EmptyTrash")
	}
}

func TestRestoreHandlesNameCollision(t *testing.T) {
	base := withTempXDG(t)
	work := filepath.Join(base, "collide")
	if err := os.MkdirAll(work, 0755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(work, "report.txt")
	writeSizedFile(t, target, 64)

	tm := NewTrashManager(nil)
	txn, err := tm.TrashItems([]string{target})
	if err != nil {
		t.Fatalf("TrashItems: %v", err)
	}
	// Recreate a file at the original path before restoring.
	writeSizedFile(t, target, 32)

	if _, err := tm.RestoreItems(txn.Items); err != nil {
		t.Fatalf("RestoreItems: %v", err)
	}
	if _, err := os.Stat(filepath.Join(work, "report (2).txt")); err != nil {
		t.Fatalf("collision restore should auto-rename: %v", err)
	}
	if info, err := os.Stat(target); err != nil || info.Size() != 32 {
		t.Fatalf("existing file must be preserved: %v", err)
	}
}

func TestTopdirTrashInfoResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	trashDir := filepath.Join(dir, ".Trash-1000")
	if err := os.MkdirAll(filepath.Join(trashDir, "files"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(trashDir, "info"), 0700); err != nil {
		t.Fatal(err)
	}
	payload := filepath.Join(trashDir, "files", "usb.txt")
	writeSizedFile(t, payload, 10)

	infoPath := filepath.Join(trashDir, "info", "usb.txt.trashinfo")
	if err := writeTrashInfo(infoPath, "media/backup/usb.txt", nowFunc()); err != nil {
		t.Fatalf("writeTrashInfo: %v", err)
	}

	item, err := parseTrashInfo(infoPath, trashDir, "/mnt/usb", true)
	if err != nil {
		t.Fatalf("parseTrashInfo: %v", err)
	}
	if item.OriginalPath != "/mnt/usb/media/backup/usb.txt" {
		t.Fatalf("original path = %q, want /mnt/usb/media/backup/usb.txt", item.OriginalPath)
	}
	if !item.IsTopdir || item.MountPoint != "/mnt/usb" {
		t.Fatalf("topdir metadata wrong: %+v", item)
	}
}

func TestPercentEncodeRoundTrip(t *testing.T) {
	for _, p := range []string{"/home/user/My Files/résumé (1).pdf", "/mnt/usb/naïve dir/a#b?.txt"} {
		encoded := percentEncodePath(p)
		if strings.ContainsAny(encoded, "#?") {
			t.Fatalf("encoded path still contains reserved chars: %q", encoded)
		}
		if got := percentDecode(encoded); got != p {
			t.Fatalf("round trip = %q, want %q", got, p)
		}
	}
}

func TestDeletePermanentlyRefusesRootAndHome(t *testing.T) {
	if _, err := DeletePermanentlyPaths([]string{"/"}); err == nil {
		t.Fatal("must refuse deleting /")
	}
}
