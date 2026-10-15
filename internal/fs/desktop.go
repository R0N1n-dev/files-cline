package fs

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// DesktopApp represents an installed XDG application that can open files.
type DesktopApp struct {
	ID          string   `json:"id"` // e.g. org.gimp.GIMP.desktop
	Name        string   `json:"name"`
	GenericName string   `json:"genericName,omitempty"`
	Comment     string   `json:"comment,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	IconPath    string   `json:"iconPath,omitempty"`
	Exec        string   `json:"exec"`
	MimeTypes   []string `json:"mimeTypes"`
	Categories  []string `json:"categories,omitempty"`
	IsDefault   bool     `json:"isDefault"`
	NoDisplay   bool     `json:"noDisplay"`
	Terminal    bool     `json:"terminal"`
}

// desktopSearchDirs returns the XDG application directories in precedence order.
func desktopSearchDirs() []string {
	var dirs []string
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		dirs = append(dirs, filepath.Join(dataHome, "applications"))
	} else if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "share", "applications"))
	}
	if dataDirs := os.Getenv("XDG_DATA_DIRS"); dataDirs != "" {
		for _, d := range strings.Split(dataDirs, ":") {
			if d = strings.TrimSpace(d); d != "" {
				dirs = append(dirs, filepath.Join(d, "applications"))
			}
		}
	} else {
		dirs = append(dirs,
			"/usr/local/share/applications",
			"/usr/share/applications",
		)
	}
	// Flatpak and Snap exports.
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "share", "flatpak", "exports", "share", "applications"))
	}
	dirs = append(dirs, "/var/lib/flatpak/exports/share/applications", "/var/lib/snapd/desktop/applications")
	return dirs
}

// parseDesktopFile reads a .desktop entry, extracting the [Desktop Entry] group.
func parseDesktopFile(path string) (*DesktopApp, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	app := &DesktopApp{ID: filepath.Base(path)}
	inDesktopEntry := false
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			inDesktopEntry = trimmed == "[Desktop Entry]"
			continue
		}
		if !inDesktopEntry {
			continue
		}
		eq := strings.Index(trimmed, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:eq])
		val := strings.TrimSpace(trimmed[eq+1:])

		switch key {
		case "Name":
			if app.Name == "" {
				app.Name = val
			}
		case "GenericName":
			if app.GenericName == "" {
				app.GenericName = val
			}
		case "Comment":
			if app.Comment == "" {
				app.Comment = val
			}
		case "Icon":
			if app.Icon == "" {
				app.Icon = val
			}
		case "Exec":
			if app.Exec == "" {
				app.Exec = val
			}
		case "MimeType":
			for _, mt := range strings.Split(val, ";") {
				if mt = strings.TrimSpace(mt); mt != "" {
					app.MimeTypes = append(app.MimeTypes, mt)
				}
			}
		case "Categories":
			for _, c := range strings.Split(val, ";") {
				if c = strings.TrimSpace(c); c != "" {
					app.Categories = append(app.Categories, c)
				}
			}
		case "NoDisplay":
			app.NoDisplay = strings.EqualFold(val, "true")
		case "Terminal":
			app.Terminal = strings.EqualFold(val, "true")
		case "Hidden":
			if strings.EqualFold(val, "true") {
				return nil, os.ErrNotExist
			}
		}
	}

	if app.Name == "" || app.Exec == "" || app.NoDisplay {
		return nil, os.ErrNotExist
	}
	app.IconPath = resolveIconPath(app.Icon)
	return app, nil
}

// resolveIconPath searches standard icon theme directories for a named icon.
func resolveIconPath(icon string) string {
	if icon == "" {
		return ""
	}
	if filepath.IsAbs(icon) {
		if _, err := os.Stat(icon); err == nil {
			return icon
		}
		return ""
	}
	exts := []string{".svg", ".png", ".xpm"}
	baseDirs := []string{
		"/usr/share/icons/hicolor", "/usr/local/share/icons/hicolor",
		"/usr/share/pixmaps", "/usr/local/share/pixmaps",
	}
	if home, err := os.UserHomeDir(); err == nil {
		baseDirs = append([]string{filepath.Join(home, ".local", "share", "icons", "hicolor")}, baseDirs...)
	}
	sizes := []string{"256x256", "128x128", "64x64", "48x48", "scalable", ""}
	subdirs := []string{"apps", "mimetypes", ""}

	for _, base := range baseDirs {
		for _, size := range sizes {
			for _, sub := range subdirs {
				for _, ext := range exts {
					parts := []string{base}
					if size != "" {
						parts = append(parts, size)
					}
					if sub != "" {
						parts = append(parts, sub)
					}
					parts = append(parts, icon+ext)
					candidate := filepath.Join(parts...)
					if _, err := os.Stat(candidate); err == nil {
						return candidate
					}
				}
			}
		}
	}
	return ""
}
