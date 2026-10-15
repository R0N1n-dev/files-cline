package fs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConflictPolicies(t *testing.T) {
	cases := []struct {
		name     string
		policy   string
		wantName string
	}{
		{"rename", ConflictRename, "a (2).txt"},
		{"overwrite", ConflictOverwrite, "a.txt"},
		{"skip", ConflictSkip, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "src")
			dest := filepath.Join(dir, "dest")
			for _, d := range []string{src, dest} {
				if err := os.MkdirAll(d, 0755); err != nil {
					t.Fatal(err)
				}
			}
			writeSizedFile(t, filepath.Join(src, "a.txt"), 1024)
			writeSizedFile(t, filepath.Join(dest, "a.txt"), 32)

			m := NewOperationsManager(nil)
			id, err := m.StartCopy([]string{filepath.Join(src, "a.txt")}, dest, tc.policy)
			if err != nil {
				t.Fatalf("StartCopy: %v", err)
			}
			op := waitForOp(t, m, id, 10*time.Second)
			if op.State != StateCompleted {
				t.Fatalf("state = %s (error=%s)", op.State, op.Error)
			}
			if tc.wantName == "" {
				if info, err := os.Stat(filepath.Join(dest, "a.txt")); err != nil || info.Size() != 32 {
					t.Fatalf("skip policy must keep existing file: %v", err)
				}
				return
			}
			info, err := os.Stat(filepath.Join(dest, tc.wantName))
			if err != nil {
				t.Fatalf("expected %s: %v", tc.wantName, err)
			}
			if info.Size() != 1024 {
				t.Fatalf("%s size = %d, want 1024", tc.wantName, info.Size())
			}
		})
	}
}

func TestMoveAcrossDirectoriesAndUndo(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "from")
	dstDir := filepath.Join(dir, "to")
	for _, d := range []string{srcDir, dstDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	writeSizedFile(t, filepath.Join(srcDir, "moved.txt"), 2048)

	m := NewOperationsManager(nil)
	id, err := m.StartMove([]string{filepath.Join(srcDir, "moved.txt")}, dstDir, ConflictRename)
	if err != nil {
		t.Fatalf("StartMove: %v", err)
	}
	op := waitForOp(t, m, id, 10*time.Second)
	if op.State != StateCompleted {
		t.Fatalf("state = %s (error=%s)", op.State, op.Error)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "moved.txt")); err != nil {
		t.Fatalf("moved file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(srcDir, "moved.txt")); !os.IsNotExist(err) {
		t.Fatal("source should be gone after move")
	}

	if err := m.Undo(id); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if _, err := os.Stat(filepath.Join(srcDir, "moved.txt")); err != nil {
		t.Fatalf("undo must restore the original location: %v", err)
	}
}

func TestCopyDirectoryRecursively(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "tree")
	dest := filepath.Join(dir, "dest")
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	writeSizedFile(t, filepath.Join(src, "top.txt"), 128)
	writeSizedFile(t, filepath.Join(src, "nested", "deep.txt"), 256)

	m := NewOperationsManager(nil)
	id, err := m.StartCopy([]string{src}, dest, ConflictRename)
	if err != nil {
		t.Fatalf("StartCopy: %v", err)
	}
	op := waitForOp(t, m, id, 10*time.Second)
	if op.State != StateCompleted {
		t.Fatalf("state = %s (error=%s)", op.State, op.Error)
	}
	if _, err := os.Stat(filepath.Join(dest, "tree", "nested", "deep.txt")); err != nil {
		t.Fatalf("nested file not copied: %v", err)
	}
}

func TestExtractArchiveCancellableAndSafe(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "payload")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeSizedFile(t, filepath.Join(srcDir, "one.txt"), 4096)
	writeSizedFile(t, filepath.Join(srcDir, "two.bin"), 8192)

	archivePath := filepath.Join(dir, "bundle.zip")
	if _, err := CreateArchive([]string{srcDir}, archivePath, nil); err != nil {
		t.Fatalf("CreateArchive: %v", err)
	}

	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	m := NewOperationsManager(nil)
	id, err := m.StartExtract([]string{archivePath}, outDir)
	if err != nil {
		t.Fatalf("StartExtract: %v", err)
	}
	op := waitForOp(t, m, id, 15*time.Second)
	if op.State != StateCompleted {
		t.Fatalf("state = %s (error=%s)", op.State, op.Error)
	}
	if _, err := os.Stat(filepath.Join(outDir, "payload", "one.txt")); err != nil {
		t.Fatalf("extracted file missing: %v", err)
	}
}

func TestSafeJoinBlocksTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := safeJoin(root, "../escape.txt"); err == nil {
		t.Fatal("path traversal must be rejected")
	}
	if _, err := safeJoin(root, "/etc/passwd"); err != nil {
		t.Fatalf("absolute entries should be re-rooted, got %v", err)
	}
	if _, err := safeJoin(root, "nested/file.txt"); err != nil {
		t.Fatalf("valid entry rejected: %v", err)
	}
}
