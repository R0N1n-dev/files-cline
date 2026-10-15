// Command vela-dnd-helper is the native Linux outbound drag-and-drop and
// clipboard helper used by Vela Files (and usable standalone from scripts).
//
// WebKit2GTK cannot originate OS-level drags from a webview, so the app hands
// its selection to this helper, which owns a real GTK drag source or clipboard
// selection in the desktop session.
//
// Usage:
//
//	vela-dnd-helper clipboard [--cut] <path>...
//	vela-dnd-helper drag <path>...
//	vela-dnd-helper open-fm <path>...
//	vela-dnd-helper uri-list <path>...
//	vela-dnd-helper version
package main

import (
	"fmt"
	"os"

	"vela-files/internal/fs"
)

const version = "2.4.0"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	switch args[0] {
	case "clipboard":
		cut := false
		rest := args[1:]
		if len(rest) > 0 && (rest[0] == "--cut" || rest[0] == "-x") {
			cut = true
			rest = rest[1:]
		}
		if len(rest) == 0 {
			fail("clipboard: no paths given")
		}
		if err := fs.CopyPathsToClipboard(rest, cut); err != nil {
			fail("clipboard: %v", err)
		}
		fmt.Fprintf(os.Stderr, "clipboard: %d item(s) placed (%s)\n", len(rest), cutOrCopy(cut))

	case "drag":
		if len(args) < 2 {
			fail("drag: no paths given")
		}
		if err := fs.StartNativeDrag(args[1:]); err != nil {
			fail("drag: %v", err)
		}
		fmt.Fprintln(os.Stderr, "drag: native drag source started — drop onto the target app")

	case "open-fm":
		if len(args) < 2 {
			fail("open-fm: no paths given")
		}
		if err := fs.OpenInNativeFileManager(args[1:]); err != nil {
			fail("open-fm: %v", err)
		}

	case "uri-list":
		if len(args) < 2 {
			fail("uri-list: no paths given")
		}
		for _, p := range args[1:] {
			fmt.Println(fs.FileURI(p))
		}

	case "version", "--version", "-v":
		fmt.Printf("vela-dnd-helper %s\n", version)

	case "help", "--help", "-h":
		usage()

	default:
		usage()
		os.Exit(2)
	}
}

func cutOrCopy(cut bool) string {
	if cut {
		return "cut"
	}
	return "copy"
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "vela-dnd-helper: "+format+"\n", a...)
	os.Exit(1)
}

func usage() {
	fmt.Fprint(os.Stderr, `vela-dnd-helper `+version+` — native drag/clipboard helper for Vela Files

Commands:
  clipboard [--cut] <path>...   Put files on the system clipboard (GNOME/KDE formats)
  drag <path>...                Start a native GTK drag source for outbound drag-and-drop
  open-fm <path>...             Reveal files in the native file manager (D-Bus ShowItems)
  uri-list <path>...            Print text/uri-list URIs to stdout
  version                       Print version
`)
}
