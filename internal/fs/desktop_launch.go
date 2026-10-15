package fs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DiscoverInstalledApps scans XDG application directories for launchable apps.
func DiscoverInstalledApps() []DesktopApp {
	seen := map[string]bool{}
	var apps []DesktopApp
	for _, dir := range desktopSearchDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, de := range entries {
			if de.IsDir() || !strings.HasSuffix(de.Name(), ".desktop") || seen[de.Name()] {
				continue
			}
			app, err := parseDesktopFile(filepath.Join(dir, de.Name()))
			if err != nil {
				continue
			}
			seen[de.Name()] = true
			apps = append(apps, *app)
		}
	}
	return apps
}

// AppsForMime returns apps that can open a MIME type: the default first, then
// gio-registered handlers, then every app declaring the type in MimeType=.
func AppsForMime(mimeType string) []DesktopApp {
	installed := DiscoverInstalledApps()
	byID := make(map[string]DesktopApp, len(installed))
	for _, a := range installed {
		byID[a.ID] = a
	}

	registered, defaultID := registeredHandlers(mimeType)

	var result []DesktopApp
	used := map[string]bool{}
	appendApp := func(id string, isDefault bool) {
		if id == "" || used[id] {
			return
		}
		app, ok := byID[id]
		if !ok {
			parsed, err := findDesktopFileByID(id)
			if err != nil {
				return
			}
			app = *parsed
		}
		used[id] = true
		app.IsDefault = isDefault
		result = append(result, app)
	}

	if defaultID != "" {
		appendApp(defaultID, true)
	}
	for _, id := range registered {
		appendApp(id, id == defaultID)
	}
	for _, a := range installed {
		if used[a.ID] {
			continue
		}
		for _, mt := range a.MimeTypes {
			if mt == mimeType {
				appendApp(a.ID, false)
				break
			}
		}
	}
	return result
}

func findDesktopFileByID(id string) (*DesktopApp, error) {
	if !strings.HasSuffix(id, ".desktop") {
		id += ".desktop"
	}
	for _, dir := range desktopSearchDirs() {
		if app, err := parseDesktopFile(filepath.Join(dir, id)); err == nil {
			return app, nil
		}
	}
	return nil, os.ErrNotExist
}

// registeredHandlers parses `gio mime <type>` for registered and default apps.
func registeredHandlers(mimeType string) (registered []string, defaultID string) {
	gioBin, err := exec.LookPath("gio")
	if err != nil {
		return nil, ""
	}
	cmd := exec.Command(gioBin, "mime", mimeType)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		return nil, ""
	}

	inRegistered, inRecommended := false, false
	for _, raw := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		switch {
		case strings.HasPrefix(trimmed, "Default application for"):
			if idx := strings.LastIndex(trimmed, ":"); idx >= 0 {
				defaultID = strings.TrimSpace(trimmed[idx+1:])
			}
		case strings.HasPrefix(trimmed, "Registered applications:"):
			inRegistered, inRecommended = true, false
		case strings.HasPrefix(trimmed, "Recommended applications:"):
			inRecommended, inRegistered = true, false
		default:
			if (inRegistered || inRecommended) && strings.HasSuffix(trimmed, ".desktop") {
				registered = append(registered, trimmed)
			}
		}
	}
	return registered, defaultID
}

// GetDefaultApp returns the default handler for a MIME type, if known.
func (h *MimeHelper) GetDefaultApp(mimeType string) *DesktopApp {
	if mimeType == "" {
		return nil
	}
	_, defaultID := registeredHandlers(mimeType)
	if defaultID == "" {
		defaultID = queryXDGMimeDefault(mimeType)
	}
	if defaultID == "" {
		return nil
	}
	if app, err := findDesktopFileByID(defaultID); err == nil {
		app.IsDefault = true
		return app
	}
	return nil
}

// SetDefaultApp makes an app the default handler for a MIME type.
func (h *MimeHelper) SetDefaultApp(mimeType, desktopID string) error {
	if !strings.HasSuffix(desktopID, ".desktop") {
		desktopID += ".desktop"
	}
	if gioBin, err := exec.LookPath("gio"); err == nil {
		cmd := exec.Command(gioBin, "mime", mimeType, desktopID)
		if _, err := cmd.CombinedOutput(); err == nil {
			return nil
		}
	}
	if xdg, err := exec.LookPath("xdg-mime"); err == nil {
		cmd := exec.Command(xdg, "default", desktopID, mimeType)
		if out, err := cmd.CombinedOutput(); err != nil {
			return execError("xdg-mime", out, err)
		}
		return nil
	}
	return &ExecError{Tool: "gio mime", Message: "no handler-setting tool available"}
}
