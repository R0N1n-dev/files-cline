package fs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// OpenInNativeFileManager reveals the given paths in the desktop's native file
// manager (Nautilus, Dolphin, Nemo, Thunar, ...), selecting the items when the
// file manager supports the org.freedesktop.FileManager1 interface.
func OpenInNativeFileManager(paths []string) error {
	if len(paths) == 0 {
		return os.ErrInvalid
	}

	absPaths := make([]string, 0, len(paths))
	dir := ""
	for _, p := range paths {
		abs, err := filepath.Abs(ExpandPath(p))
		if err != nil {
			continue
		}
		absPaths = append(absPaths, abs)
		if dir == "" {
			if info, err := os.Stat(abs); err == nil && info.IsDir() {
				dir = abs
			} else {
				dir = filepath.Dir(abs)
			}
		}
	}
	if len(absPaths) == 0 {
		return os.ErrNotExist
	}

	uris := make([]string, 0, len(absPaths))
	for _, p := range absPaths {
		uris = append(uris, FileURI(p))
	}

	// Preferred: D-Bus ShowItems (highlights the exact files).
	if err := dbusShowItems(uris); err == nil {
		return nil
	}

	// Fallback: gio open on the containing directory, then xdg-open.
	if gioBin, err := exec.LookPath("gio"); err == nil && dir != "" {
		cmd := exec.Command(gioBin, "open", "--", dir)
		cmd.Env = os.Environ()
		if _, err := cmd.CombinedOutput(); err == nil {
			return nil
		}
	}
	if xdg, err := exec.LookPath("xdg-open"); err == nil && dir != "" {
		cmd := exec.Command(xdg, dir)
		cmd.Env = os.Environ()
		if err := cmd.Start(); err == nil {
			go func() { _ = cmd.Wait() }()
			return nil
		}
	}
	return &ExecError{Tool: "file manager", Message: "no file manager integration available"}
}

// dbusShowItems calls org.freedesktop.FileManager1.ShowItems over the session bus.
func dbusShowItems(uris []string) error {
	if len(uris) == 0 {
		return os.ErrInvalid
	}
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		return &ExecError{Tool: "dbus", Message: "no session bus"}
	}
	dbusSend, err := exec.LookPath("dbus-send")
	if err != nil {
		return err
	}

	uriArray := "array:string:" + strings.Join(uris, ",")
	cmd := exec.Command(dbusSend,
		"--session",
		"--dest=org.freedesktop.FileManager1",
		"--type=method_call",
		"/org/freedesktop/FileManager1",
		"org.freedesktop.FileManager1.ShowItems",
		uriArray,
		"string:",
	)
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		return execError("dbus-send ShowItems", out, err)
	}
	return nil
}

// CopyPathsToClipboard places file URIs on the clipboard in the formats used by
// GNOME/KDE file managers (x-special/gnome-copied-files and text/uri-list).
func CopyPathsToClipboard(paths []string, cut bool) error {
	uris := make([]string, 0, len(paths))
	for _, p := range paths {
		abs, err := filepath.Abs(ExpandPath(p))
		if err != nil {
			continue
		}
		uris = append(uris, FileURI(abs))
	}
	if len(uris) == 0 {
		return os.ErrNotExist
	}
	action := "copy"
	if cut {
		action = "cut"
	}
	gnomePayload := action + "\n" + strings.Join(uris, "\n") + "\n"
	uriList := strings.Join(uris, "\r\n") + "\r\n"
	plain := strings.Join(paths, "\n")

	return writeClipboard(map[string]string{
		"x-special/gnome-copied-files": gnomePayload,
		"x-special/nautilus-clipboard": gnomePayload,
		"text/uri-list":                uriList,
		"UTF8_STRING":                  plain,
		"text/plain":                   plain,
	})
}
