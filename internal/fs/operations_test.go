package fs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeSizedFile(t *testing.T, path string, size int) {
	t.Helper()
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 251)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func waitForOp(t *testing.T, m *OperationsManager, id string, timeout time.Duration) Operation {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		op, ok := m.Get(id)
		if !ok {
			t.Fatalf("operation %s vanished", id)
		}
		if op.State != StateRunning && op.State != StatePaused && op.State != StateCancelling {
			return op
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("operation %s did not finish within %s", id, timeout)
	return Operation{}
}

func TestCopyCompletesAndUndoRestoresState(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.bin")
	dest := filepath.Join(dir, "dest")
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	writeSizedFile(t, src, 512*1024)

	m := NewOperationsManager(nil)
	id, err := m.StartCopy([]string{src}, dest, ConflictRename)
	if err != nil {
		t.Fatalf("StartCopy: %v", err)
	}
	op := waitForOp(t, m, id, 10*time.Second)
	if op.State != StateCompleted {
		t.Fatalf("state = %s, error = %s", op.State, op.Error)
	}
	copied := filepath.Join(dest, "source.bin")
	if info, err := os.Stat(copied); err != nil || info.Size() != 512*1024 {
		t.Fatalf("copied file missing or wrong size: %v", err)
	}
	if op.Progress != 100 {
		t.Fatalf("progress = %v, want 100", op.Progress)
	}

	if err := m.Undo(id); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if _, err := os.Stat(copied); !os.IsNotExist(err) {
		t.Fatalf("undo should have removed the copy, stat err = %v", err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source must remain after undo: %v", err)
	}
}

func TestCancelMidCopyRemovesPartialFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "big.bin")
	dest := filepath.Join(dir, "dest")
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	writeSizedFile(t, src, 48*1024*1024)

	m := NewOperationsManager(nil)
	id, err := m.StartCopy([]string{src}, dest, ConflictRename)
	if err != nil {
		t.Fatalf("StartCopy: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		op, _ := m.Get(id)
		if op.DoneBytes > 0 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if err := m.Cancel(id); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	op := waitForOp(t, m, id, 10*time.Second)
	if op.State != StateCancelled {
		t.Fatalf("state = %s (error=%s), want cancelled", op.State, op.Error)
	}
	if _, err := os.Stat(filepath.Join(dest, "big.bin")); !os.IsNotExist(err) {
		t.Fatalf("partial destination file must be removed on cancel (stat err = %v)", err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source must be untouched: %v", err)
	}
}

func TestPauseResumeOperation(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "pause.bin")
	dest := filepath.Join(dir, "dest")
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	writeSizedFile(t, src, 24*1024*1024)

	m := NewOperationsManager(nil)
	id, err := m.StartCopy([]string{src}, dest, ConflictRename)
	if err != nil {
		t.Fatalf("StartCopy: %v", err)
	}
	if err := m.Pause(id); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	if paused, _ := m.Get(id); paused.State != StatePaused {
		t.Fatalf("state = %s, want paused", paused.State)
	}
	if err := m.Resume(id); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	op := waitForOp(t, m, id, 15*time.Second)
	if op.State != StateCompleted {
		t.Fatalf("state = %s (error=%s), want completed", op.State, op.Error)
	}
}

func TestCopyRefusesDirectoryIntoItself(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "self")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatal(err)
	}
	writeSizedFile(t, filepath.Join(src, "f.txt"), 16)

	m := NewOperationsManager(nil)
	id, err := m.StartCopy([]string{src}, src, ConflictRename)
	if err != nil {
		t.Fatalf("StartCopy: %v", err)
	}
	op := waitForOp(t, m, id, 5*time.Second)
	if op.State != StateError {
		t.Fatalf("state = %s, want error for self-copy", op.State)
	}
}
