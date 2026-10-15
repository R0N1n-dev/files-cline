package fs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var nowFunc = time.Now

// OpenTerminalAt launches the user's preferred terminal emulator in a directory.
func OpenTerminalAt(dir string) error {
	abs, err := filepath.Abs(ExpandPath(dir))
	if err != nil {
		return err
	}
	if info, err := os.Stat(abs); err != nil {
		return err
	} else if !info.IsDir() {
		abs = filepath.Dir(abs)
	}

	candidates := [][]string{
		{"xdg-terminal-exec", "--working-directory=" + abs},
		{"kgx", "--working-directory", abs},
		{"gnome-terminal", "--working-directory", abs},
		{"konsole", "--workdir", abs},
		{"xfce4-terminal", "--working-directory", abs},
		{"alacritty", "--working-directory", abs},
		{"kitty", "--directory", abs},
		{"wezterm", "start", "--cwd", abs},
		{"foot", "--working-directory", abs},
		{"xterm", "-e", "cd " + abs + " && $SHELL"},
	}
	for _, c := range candidates {
		bin, err := exec.LookPath(c[0])
		if err != nil {
			continue
		}
		cmd := exec.Command(bin, c[1:]...)
		cmd.Env = os.Environ()
		cmd.Dir = abs
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err != nil {
			continue
		}
		go func() { _ = cmd.Wait() }()
		return nil
	}
	return &ExecError{Tool: "terminal", Message: "no supported terminal emulator found"}
}

// ChmodPaths applies an octal permission mode (e.g. "644", "0755") to paths.
func ChmodPaths(paths []string, mode string) (int, error) {
	mode = strings.TrimPrefix(strings.TrimSpace(mode), "0")
	if mode == "" {
		mode = "644"
	}
	parsed, err := parseOctal(mode)
	if err != nil {
		return 0, err
	}
	applied := 0
	for _, p := range paths {
		abs, err := filepath.Abs(ExpandPath(p))
		if err != nil {
			continue
		}
		if err := os.Chmod(abs, os.FileMode(parsed)); err != nil {
			return applied, err
		}
		applied++
	}
	return applied, nil
}

func parseOctal(s string) (uint32, error) {
	var v uint32
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '7' {
			return 0, &ExecError{Tool: "chmod", Message: "invalid octal mode " + s}
		}
		v = v*8 + uint32(c-'0')
	}
	if v > 07777 {
		return 0, &ExecError{Tool: "chmod", Message: "mode out of range: " + s}
	}
	return v, nil
}

// DuplicateItems copies paths next to themselves using the "name (copy).ext"
// convention and returns the created paths.
func DuplicateItems(paths []string) ([]string, error) {
	created := make([]string, 0, len(paths))
	for _, p := range paths {
		abs, err := filepath.Abs(ExpandPath(p))
		if err != nil {
			continue
		}
		ext := filepath.Ext(abs)
		stem := strings.TrimSuffix(filepath.Base(abs), ext)
		target := uniqueDestination(filepath.Dir(abs), stem+" copy"+ext)
		if err := copyPathRecursive(abs, target); err != nil {
			return created, err
		}
		created = append(created, target)
	}
	return created, nil
}

// TrashRestoreAll is a convenience helper used by the UI's "Undo" toast.
func (t *TrashManager) LastTransactionInfo() (TrashTransaction, bool) {
	t.mu.undo = trimUndoStack(t.mu.undo)
	for i := len(t.mu.undo) - 1; i >= 0; i-- {
		if !t.mu.undo[i].Undone {
			return t.mu.undo[i], true
		}
	}
	return TrashTransaction{}, false
}
