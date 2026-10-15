package fs

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseDesktopFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "org.example.Viewer.desktop")
	content := `[Desktop Entry]
Type=Application
Name=Example Viewer
GenericName=Image Viewer
Comment=Views your images
Icon=example-viewer
Exec=example-viewer %U
MimeType=image/jpeg;image/png;inode/directory;
Categories=Graphics;Viewer;
Terminal=false

[Desktop Action Edit]
Name=Edit
Exec=example-editor %f
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	app, err := parseDesktopFile(path)
	if err != nil {
		t.Fatalf("parseDesktopFile: %v", err)
	}
	if app.Name != "Example Viewer" || app.Exec != "example-viewer %U" {
		t.Fatalf("app = %+v", app)
	}
	if len(app.MimeTypes) != 3 || app.MimeTypes[0] != "image/jpeg" {
		t.Fatalf("mimetypes = %v", app.MimeTypes)
	}
	if app.Terminal {
		t.Fatal("Terminal should be false")
	}
}

func TestParseDesktopFileSkipsHiddenAndNoDisplay(t *testing.T) {
	dir := t.TempDir()
	noDisplay := filepath.Join(dir, "hidden.desktop")
	body := "[Desktop Entry]\nName=Hidden\nExec=hidden\nNoDisplay=true\n"
	if err := os.WriteFile(noDisplay, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseDesktopFile(noDisplay); err == nil {
		t.Fatal("NoDisplay entries must be skipped")
	}

	notAnApp := filepath.Join(dir, "link.desktop")
	if err := os.WriteFile(notAnApp, []byte("[Desktop Entry]\nType=Link\nURL=https://example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseDesktopFile(notAnApp); err == nil {
		t.Fatal("entries without Exec must be skipped")
	}
}

func TestBuildExecArgsFieldCodes(t *testing.T) {
	paths := []string{"/tmp/a b.txt", "/tmp/c.txt"}

	if got := BuildExecArgs("app %f", paths); !reflect.DeepEqual(got, []string{"app", "/tmp/a b.txt"}) {
		t.Fatalf("%%f = %v", got)
	}
	if got := BuildExecArgs("app %F", paths); !reflect.DeepEqual(got, []string{"app", "/tmp/a b.txt", "/tmp/c.txt"}) {
		t.Fatalf("%%F = %v", got)
	}
	got := BuildExecArgs("app %U", paths)
	if len(got) != 3 || got[1] != "file:///tmp/a%20b.txt" || got[2] != "file:///tmp/c.txt" {
		t.Fatalf("%%U = %v", got)
	}
	// Quoted arguments with spaces must stay a single argv element.
	if got := BuildExecArgs(`"/opt/my app/bin" --open %f`, paths); got[0] != "/opt/my app/bin" {
		t.Fatalf("quoted exec = %v", got)
	}
	// Apps without field codes get the selection appended.
	if got := BuildExecArgs("app", paths); len(got) != 3 {
		t.Fatalf("no field code = %v", got)
	}
}

func TestSplitExecLineQuoting(t *testing.T) {
	got := splitExecLine(`env FOO=bar "my tool" --flag 'single arg' plain`)
	want := []string{"env", "FOO=bar", "my tool", "--flag", "single arg", "plain"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("split = %v, want %v", got, want)
	}
}

func TestAppsForMimeIncludesDeclaredHandlers(t *testing.T) {
	base := withTempXDG(t)
	appsDir := filepath.Join(base, "data", "applications")
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		t.Fatal(err)
	}
	body := "[Desktop Entry]\nName=Vela Test Viewer\nExec=vela-test %U\nMimeType=image/png;\n"
	if err := os.WriteFile(filepath.Join(appsDir, "vela-test.desktop"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	apps := AppsForMime("image/png")
	found := false
	for _, a := range apps {
		if a.ID == "vela-test.desktop" {
			found = true
		}
	}
	if !found {
		t.Fatalf("declared handler missing from Open With list: %+v", apps)
	}
}

func TestMimeHelperFallsBackToExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shot.png")
	writeSizedFile(t, path, 16)

	h := NewMimeHelper()
	got := h.DetectMimeType(path, false)
	if got != "image/png" && got != "application/octet-stream" {
		t.Fatalf("unexpected mime for png: %q", got)
	}
	if h.DetectMimeType(dir, true) != "inode/directory" {
		t.Fatal("directories must report inode/directory")
	}
}

func TestParseURIListAndClipboardPayload(t *testing.T) {
	payload := parseURIList("copy\nfile:///home/u/a.txt\nfile:///home/u/My%20File.txt\n")
	if payload.Cut {
		t.Fatal("copy payload must not be marked cut")
	}
	if len(payload.Paths) != 2 || payload.Paths[1] != "/home/u/My File.txt" {
		t.Fatalf("payload = %+v", payload)
	}

	cutPayload := parseURIList("cut\nfile:///tmp/x.bin\n")
	if !cutPayload.Cut || len(cutPayload.Paths) != 1 {
		t.Fatalf("cut payload = %+v", cutPayload)
	}
}

func TestCopyPathsToClipboardRequiresSelection(t *testing.T) {
	if err := CopyPathsToClipboard(nil, false); err == nil {
		t.Fatal("empty selection must be rejected")
	}
}
