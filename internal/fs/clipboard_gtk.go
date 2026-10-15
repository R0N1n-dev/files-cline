package fs

import (
	"os"
	"os/exec"
	"strings"
)

// gtkClipboardWrite uses GTK3 (via python3-gi) to own the clipboard with several
// MIME targets at once — the same mechanism Nautilus uses, so pasting into
// Nautilus, Dolphin, Telegram, browsers, and IDEs all work.
func gtkClipboardWrite(formats map[string]string) error {
	python, err := exec.LookPath("python3")
	if err != nil {
		return os.ErrNotExist
	}
	// Verify PyGObject is importable before paying for a process spawn.
	check := exec.Command(python, "-c", "import gi; gi.require_version('Gtk','3.0')")
	if err := check.Run(); err != nil {
		return os.ErrNotExist
	}

	script := gtkClipboardScript()
	args := []string{"-c", script}
	for mime, payload := range formats {
		args = append(args, mime, payload)
	}
	cmd := exec.Command(python, args...)
	cmd.Env = os.Environ()
	// The helper forks into the background and owns the selection.
	out, err := cmd.CombinedOutput()
	if err != nil {
		return execError("gtk clipboard", out, err)
	}
	return nil
}

// gtkClipboardScript returns the inline PyGObject program that serves the clipboard.
func gtkClipboardScript() string {
	return strings.TrimSpace(`
import sys, os
import gi
gi.require_version('Gtk', '3.0')
gi.require_version('Gdk', '3.0')
from gi.repository import Gtk, Gdk, GLib

pairs = sys.argv[1:]
targets = []
for i in range(0, len(pairs), 2):
    targets.append((pairs[i], pairs[i+1]))
if not targets:
    sys.exit(1)

clipboard = Gtk.Clipboard.get(Gdk.SELECTION_CLIPBOARD)

def on_get(selection, info, data):
    data.set_text(targets[info][1], -1)
    return True

def on_clear(selection, data):
    Gtk.main_quit()

def on_targets(selection, targets_out, data):
    atoms = [Gdk.Atom.intern(name, False) for name, _ in targets]
    targets_out[0] = atoms
    return True

clipboard.set_with_data(
    [Gdk.Atom.intern(name, False) for name, _ in targets],
    on_get, on_clear, on_targets, None
)
clipboard.store()
GLib.timeout_add_seconds(180, Gtk.main_quit)
Gtk.main()
`)
}

// gtkDragScript returns the inline PyGObject program that starts a real GTK drag
// source, letting users drag files out of Vela Files into any Linux application.
func gtkDragScript() string {
	return strings.TrimSpace(`
import sys
import gi
gi.require_version('Gtk', '3.0')
gi.require_version('Gdk', '3.0')
from gi.repository import Gtk, Gdk, GLib

paths = sys.argv[1:]
if not paths:
    sys.exit(1)

uris = ['file://' + p for p in paths]

def on_drag_data_get(widget, ctx, selection, info, time):
    selection.set_uris(uris)

def on_drag_begin(widget, ctx):
    pass

window = Gtk.Window(type=Gtk.WindowType.TOPLEVEL)
window.set_title('Drag files out')
window.set_default_size(320, 120)
label = Gtk.Label(label='Drag the card below into another app\n(%d item(s))' % len(paths))
window.add(label)
window.set_keep_above(True)
window.show_all()

label.drag_source_set(Gdk.ModifierType.BUTTON1_MASK, [], Gdk.DragAction.COPY)
label.drag_source_add_uri_targets()
label.connect('drag-data-get', on_drag_data_get)
label.connect('drag-begin', on_drag_begin)
window.connect('destroy', Gtk.main_quit)
GLib.timeout_add_seconds(300, Gtk.main_quit)
Gtk.main()
`)
}

// StartNativeDrag opens a native drag source window for outbound drag-and-drop.
// WebKit2GTK cannot start OS-level drags from a webview, so Vela Files hands the
// selection to a native GTK drag source (or dragon-drop when available).
func StartNativeDrag(paths []string) error {
	if len(paths) == 0 {
		return os.ErrInvalid
	}
	abs := make([]string, 0, len(paths))
	for _, p := range paths {
		abs = append(abs, ExpandPath(p))
	}

	if dragon, err := exec.LookPath("dragon-drop"); err == nil {
		args := append([]string{"-x", "-a"}, abs...)
		cmd := exec.Command(dragon, args...)
		cmd.Env = os.Environ()
		if err := cmd.Start(); err == nil {
			go func() { _ = cmd.Wait() }()
			return nil
		}
	}

	python, err := exec.LookPath("python3")
	if err != nil {
		return os.ErrNotExist
	}
	check := exec.Command(python, "-c", "import gi; gi.require_version('Gtk','3.0')")
	if err := check.Run(); err != nil {
		return os.ErrNotExist
	}
	args := append([]string{"-c", gtkDragScript()}, abs...)
	cmd := exec.Command(python, args...)
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// HelperBinaryPath locates the optional vela-dnd-helper executable.
func HelperBinaryPath() string {
	if explicit := os.Getenv("VELA_DND_HELPER"); explicit != "" {
		if _, err := os.Stat(explicit); err == nil {
			return explicit
		}
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepathJoinDir(exe, "vela-dnd-helper")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if path, err := exec.LookPath("vela-dnd-helper"); err == nil {
		return path
	}
	return ""
}

func filepathJoinDir(exePath, name string) string {
	idx := strings.LastIndex(exePath, "/")
	if idx < 0 {
		return name
	}
	return exePath[:idx+1] + name
}
