package fs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// MimeHelper detects MIME types and resolves default applications using
// `gio` (preferred) with `xdg-mime` and extension-based fallbacks.
type MimeHelper struct {
	mu    sync.Mutex
	cache map[string]string
}

// NewMimeHelper creates a MIME helper with a small detection cache.
func NewMimeHelper() *MimeHelper {
	return &MimeHelper{cache: make(map[string]string)}
}

// DetectMimeType returns the best-known MIME type for a path.
func (h *MimeHelper) DetectMimeType(path string, isDir bool) string {
	if isDir {
		return "inode/directory"
	}
	abs, err := filepath.Abs(ExpandPath(path))
	if err != nil {
		abs = path
	}

	h.mu.Lock()
	if cached, ok := h.cache[abs]; ok {
		h.mu.Unlock()
		return cached
	}
	h.mu.Unlock()

	mimeType := ""
	if gioBin, err := exec.LookPath("gio"); err == nil {
		cmd := exec.Command(gioBin, "info", "-a", "standard::content-type", "--", abs)
		cmd.Env = os.Environ()
		if out, err := cmd.Output(); err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if idx := strings.Index(line, "standard::content-type:"); idx >= 0 {
					mimeType = strings.TrimSpace(line[idx+len("standard::content-type:"):])
					break
				}
			}
		}
	}
	if mimeType == "" {
		if xdg, err := exec.LookPath("xdg-mime"); err == nil {
			cmd := exec.Command(xdg, "query", "filetype", abs)
			if out, err := cmd.Output(); err == nil {
				mimeType = strings.TrimSpace(string(out))
			}
		}
	}
	if mimeType == "" {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(abs), "."))
		mimeType, _ = FastClassifyMime(filepath.Base(abs), ext)
	}

	h.mu.Lock()
	if len(h.cache) > 4000 {
		h.cache = make(map[string]string)
	}
	h.cache[abs] = mimeType
	h.mu.Unlock()
	return mimeType
}

// queryXDGMimeDefault asks xdg-mime for the default desktop entry of a MIME type.
func queryXDGMimeDefault(mimeType string) string {
	xdg, err := exec.LookPath("xdg-mime")
	if err != nil {
		return ""
	}
	cmd := exec.Command(xdg, "query", "default", mimeType)
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

// OpenDefault opens paths with the system default application via gio open.
func OpenDefault(paths []string) error {
	if len(paths) == 0 {
		return os.ErrInvalid
	}
	if gioBin, err := exec.LookPath("gio"); err == nil {
		args := append([]string{"open", "--"}, paths...)
		cmd := exec.Command(gioBin, args...)
		cmd.Env = os.Environ()
		if out, err := cmd.CombinedOutput(); err == nil {
			return nil
		} else if msg := strings.TrimSpace(string(out)); msg != "" {
			return &ExecError{Tool: "gio open", Message: msg}
		}
	}
	if xdg, err := exec.LookPath("xdg-open"); err == nil {
		for _, p := range paths {
			cmd := exec.Command(xdg, p)
			cmd.Env = os.Environ()
			if err := cmd.Start(); err != nil {
				return err
			}
			go func() { _ = cmd.Wait() }()
		}
		return nil
	}
	return &ExecError{Tool: "open", Message: "no opener available (install gio or xdg-utils)"}
}

// OpenWithApplication opens paths with a desktop app chosen by the user.
func (h *MimeHelper) OpenWithApplication(desktopID string, paths []string) error {
	app, err := findDesktopFileByID(desktopID)
	if err != nil {
		return err
	}
	return LaunchApp(*app, paths)
}

// OpenWithDialog asks the desktop environment to show its native "Open With" chooser.
func (h *MimeHelper) OpenWithDialog(paths []string) error {
	if len(paths) == 0 {
		return os.ErrInvalid
	}
	// gio open on multiple files triggers the DE chooser when no default exists.
	return OpenDefault(paths)
}
