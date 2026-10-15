package fs

import (
	"os"
	"os/exec"
	"strings"
)

// sessionType reports the active display server session.
func sessionType() string {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "wayland"
	}
	if os.Getenv("DISPLAY") != "" {
		return "x11"
	}
	return "none"
}

// writeClipboard publishes multiple MIME targets to the system clipboard.
// It prefers a GTK3 helper (which can own several targets at once, exactly like
// Nautilus), then xclip on X11, then wl-copy on Wayland.
func writeClipboard(formats map[string]string) error {
	primary := choosePrimaryFormat(formats)
	if primary == "" {
		return os.ErrInvalid
	}

	// 1) GTK3 helper (multi-target, best fidelity for file lists).
	if err := gtkClipboardWrite(formats); err == nil {
		return nil
	}

	// 2) X11: xclip can serve several targets from one invocation.
	if sessionType() == "x11" {
		if xclip, err := exec.LookPath("xclip"); err == nil {
			args := []string{"-selection", "clipboard"}
			for mime := range formats {
				args = append(args, "-t", mime)
			}
			args = append(args, "-i")
			cmd := exec.Command(xclip, args...)
			cmd.Stdin = strings.NewReader(formats[primary])
			if out, err := cmd.CombinedOutput(); err == nil {
				return nil
			} else if msg := strings.TrimSpace(string(out)); msg != "" {
				return &ExecError{Tool: "xclip", Message: msg}
			}
		}
		if xsel, err := exec.LookPath("xsel"); err == nil {
			cmd := exec.Command(xsel, "--clipboard", "--input")
			cmd.Stdin = strings.NewReader(formats[primary])
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}

	// 3) Wayland (and last-resort fallback).
	if wlCopy, err := exec.LookPath("wl-copy"); err == nil {
		cmd := exec.Command(wlCopy, "--type", primary)
		cmd.Stdin = strings.NewReader(formats[primary])
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return &ExecError{Tool: "clipboard", Message: "install wl-clipboard, xclip, or python3-gi"}
}

// choosePrimaryFormat picks the clipboard MIME type that best fits the desktop.
func choosePrimaryFormat(formats map[string]string) string {
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	preferred := []string{"text/uri-list", "UTF8_STRING", "text/plain"}
	if strings.Contains(desktop, "gnome") || strings.Contains(desktop, "cinnamon") ||
		strings.Contains(desktop, "mate") || strings.Contains(desktop, "budgie") ||
		strings.Contains(desktop, "unity") {
		preferred = []string{"x-special/gnome-copied-files", "text/uri-list", "UTF8_STRING", "text/plain"}
	}
	for _, mime := range preferred {
		if payload, ok := formats[mime]; ok && payload != "" {
			return mime
		}
	}
	for mime := range formats {
		return mime
	}
	return ""
}

// ClipboardPayload is a parsed clipboard file selection.
type ClipboardPayload struct {
	Paths []string `json:"paths"`
	Cut   bool     `json:"cut"`
	Text  string   `json:"text,omitempty"`
}

// ReadClipboard reads a file selection (or plain text) from the system clipboard.
// This makes "paste files copied in Nautilus/Dolphin" work inside Vela Files.
func ReadClipboard() (ClipboardPayload, error) {
	var out ClipboardPayload

	// GNOME/KDE file list first.
	for _, mime := range []string{"x-special/gnome-copied-files", "text/uri-list"} {
		data, err := readClipboardTarget(mime)
		if err != nil || strings.TrimSpace(data) == "" {
			continue
		}
		payload := parseURIList(data)
		if len(payload.Paths) > 0 {
			return payload, nil
		}
	}

	data, err := readClipboardTarget("UTF8_STRING")
	if err != nil {
		return out, err
	}
	out.Text = data
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "file://") || strings.HasPrefix(line, "/") {
			p := strings.TrimPrefix(line, "file://")
			out.Paths = append(out.Paths, percentDecode(p))
		}
	}
	return out, nil
}

// parseURIList decodes text/uri-list and gnome-copied-files payloads.
func parseURIList(data string) ClipboardPayload {
	payload := ClipboardPayload{}
	for _, raw := range strings.Split(data, "\n") {
		line := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "copy" || line == "cut" {
			payload.Cut = line == "cut"
			continue
		}
		if strings.HasPrefix(line, "file://") {
			payload.Paths = append(payload.Paths, percentDecode(strings.TrimPrefix(line, "file://")))
		}
	}
	return payload
}

func readClipboardTarget(mime string) (string, error) {
	if sessionType() == "x11" {
		if xclip, err := exec.LookPath("xclip"); err == nil {
			cmd := exec.Command(xclip, "-selection", "clipboard", "-t", mime, "-o")
			if out, err := cmd.Output(); err == nil {
				return string(out), nil
			}
		}
	}
	if wlPaste, err := exec.LookPath("wl-paste"); err == nil {
		cmd := exec.Command(wlPaste, "--no-newline", "--type", mime)
		if out, err := cmd.Output(); err == nil {
			return string(out), nil
		}
		cmd = exec.Command(wlPaste, "--no-newline")
		if out, err := cmd.Output(); err == nil && mime == "UTF8_STRING" {
			return string(out), nil
		}
	}
	return "", os.ErrNotExist
}
