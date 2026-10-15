# Vela Files

A Nautilus-inspired Linux file manager built with **Wails v2 (stable) + Vue 3 + Go (standard library)**.
No heavyweight dependencies: the filesystem engine is pure Go stdlib, and `gio` / `dbus-send` /
`wl-clipboard` / `xclip` / GTK are shelled out to only where Linux genuinely requires them
(trash, MIME, clipboard, native drag sources).

```
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ [<] [>] [↑] │ 🏠 Home ▸ Photos ▸ 2026-Vacation      [💾 183 GB free · 64% used] │ 🔍 Search    │
│                              [👁 hidden] (◐ 64%) [⊞|≡] [⧉ split] [ⓘ] [⌘] [🌙] [＋]           │
├────────────────────┬─────────────────────────────────────────────────────────────────────────┤
│ user@workstation   │ 23 items · 1.9 GB     4 selected · 18.4 MB        [drop to add files]  │
│ 🏠 Home            ├─────────────────────────────────────────────────────────────────────────┤
│ 🖥️ Desktop         │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│ 📄 Documents       │  │ 📁 Docs  │ │ 📁 Pics  │ │ 🖼️ DSC_  │ │ 🖼️ DSC_  │ │ 🎬 video │       │
│ ⬇️ Downloads       │  │ 42 items │ │1284 items│ │  [↻90°]  │ │ [↻270°]  │ │ 1.3 GB   │       │
│ 🎵 Music           │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘       │
│ 🖼️ Pictures        │                                                                        │
│ 🎬 Videos          │        ╔═════════════════════════════════════════════════╗              │
│ 🗑️ Trash     (7)   │        ║  Open With…              ▸  Image Viewer (default)║             │
│                    │        ║  Quick Look              ▸  GIMP                 ║              │
│ Drives & Mounts    │        ║  ───────────────────────  ▸  Darktable           ║              │
│ 💽 System Root     │        ║  Cut / Copy / Duplicate   ▸  Other Application…   ║             │
│    ███████░ 64%    │        ║  Rename            F2                             ║             │
│ 🔌 SanDisk USB     │        ║  Batch Rename…      (multi-select)                ║             │
│    ███░░░░░ 38%    │        ║  Extract Here       (.zip/.tar.gz)                ║             │
│    (3 in .Trash)   │        ║  Compress…                                        ║             │
│                    │        ║  Move to Trash     Delete                         ║             │
│ Bookmarks          │        ║  Drag Out to App                                  ║             │
│ ⭐ workspace       │        ║  Open in Native File Manager                      ║             │
│ ⭐ config          │        ║  Permissions       ▸  644 / 755 / 600 / 700       ║             │
│                    │        ╚═════════════════════════════════════════════════╝              │
│                    │   ┌── Operations ────────────────────────────────────────────┐         │
│                    │   │ COPY  Copying to SanDisk/Backup  68 MB/s  11s left       │         │
│                    │   │ [██████████████░░░░░░░] 64%  1.2 GB / 1.9 GB             │         │
│                    │   │  ↳ /demo/DSC_0142.ARW     [Pause] [Cancel] [Undo]        │         │
│                    │   └──────────────────────────────────────────────────────────┘         │
├────────────────────┴─────────────────────────────────────────────────────────────────────────┤
│ 🗑️ 3 items moved to Trash (home Trash + /media/user/SanDisk)                      [Undo]     │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

Light and dark themes are Libadwaita-derived (same layout, `--bg`/`--accent`/`--border` token sets),
follow the desktop preference by default, and cycle System → Light → Dark from the header bar.

## Known gaps — all fixed

| Gap | Solution |
| --- | --- |
| **No rubber-band selection** | Pointer-driven marquee in grid *and* list view. Hit-testing runs in scroll-space against virtual item rectangles, so items scrolled past during the drag are still selected; edge auto-scroll; `Ctrl` toggles, `Shift` adds. See `VirtualFileView.vue` (`onMouseDown` → `onMarqueeMove` → `itemRect`). |
| **No “Open with” menu** | `internal/fs/desktop*.go` parses XDG `.desktop` entries (Name/Icon/Exec/MimeType/NoDisplay/Terminal, Flatpak + Snap exports), resolves icons, expands Exec field codes (`%f %F %u %U %i %c %k`), merges `gio mime` registered/recommended handlers, launches detached with `setsid`, and can set the default handler (`gio mime` → `xdg-mime` fallback). UI: cascading submenu plus `OpenWithModal` with recommended/other apps, “set as default”, and a custom command runner. |
| **No EXIF rotation for photos** | Pure-Go EXIF parser (`exif.go`, `exif_tiff.go`, `exif_dims.go`) reads IFD0/Exif-IFD tags from JPEG (APP1), TIFF and WebP (RIFF `EXIF` chunk), plus PNG/JPEG dimensions. Orientation 1–8 is applied as a real pixel transform *before* scaling inside the goroutine thumbnail pool, so previews are always upright. EXIF (orientation, camera, lens, ISO, f/stop, exposure, focal length, date, pixel size) is also shown in the Inspector and Quick Look. |
| **Trash undo only covers the home trash** | Full FreeDesktop trash implementation (`trash.go`, `trash_info.go`, `trash_scan.go`, `trash_actions.go`, `trash_restore.go`): home trash **plus** per-mount topdir trash (`.Trash/<uid>` when sticky, else `.Trash-<uid>`), spec-compliant `.trashinfo` (`Path=` relative to the mount root for topdir trash, absolute for home), a unified `trash://` view aggregating every mount, restore that resolves the right original path per mount with collision auto-rename, multi-mount undo transactions, permanent delete, empty-all, and a `gio trash` fallback. Filesystem identity is decided by mount point rather than `st_dev`, because overlayfs/xino, btrfs subvolumes and fuse report inconsistent device numbers. |
| **Operations can’t be cancelled mid-copy** | `OperationsManager` runs every copy/move/extract in a goroutine with its own `context`. I/O is 64 KiB chunked with a `ctx.Err()` + pause-gate check on every chunk, throttled `op:progress` events (~12/s) carrying bytes, speed, ETA and current file, `Pause`/`Resume`, and `Cancel` which **deletes the partial destination file**. A cancel that lands exactly at the finish line still wins: created files are rolled back, and the `cancelling` state means the terminal state is only published after cleanup. Conflict policies: `ask` (blocking prompt), `rename`, `overwrite`, `skip`. Undo reverses completed copies/moves. |

## Beyond the brief (senior-engineer additions)

* **Virtual scrolling** for grid and list with overscan — 100k-entry folders keep ~40 DOM nodes.
* **Goroutine thumbnail pool** — bounded workers, in-flight job dedup, in-memory LRU + on-disk cache
  keyed by path/size/mtime, results streamed to the UI as `thumbs:ready`.
* **Inbound drop only, plus an escape hatch** — Wails `OnFileDrop` + HTML5 drop targets for files
  dropped from other apps (copy, or move with `Shift`), internal drag-and-drop between panes and onto
  folder tiles, and “Open in Native File Manager” via D-Bus
  `org.freedesktop.FileManager1.ShowItems` (highlights the exact selection in Nautilus/Dolphin/Nemo/
  Thunar) with `gio open`/`xdg-open` fallbacks.
* **Native outbound clipboard + drag helper** — `cmd/vela-dnd-helper` (also usable standalone) plus
  `internal/fs/clipboard*.go`: publishes `x-special/gnome-copied-files`, `x-special/nautilus-clipboard`,
  `text/uri-list`, `UTF8_STRING`/`text/plain` in one shot through GTK3 (python3-gi, multi-target like
  Nautilus) or `xclip` multi-`-t` / `wl-copy`, reads file selections back from the clipboard (so
  `Ctrl+V` of files copied in another file manager works), and starts a real GTK drag source
  (`dragon-drop` or inline PyGObject) because WebKit2GTK cannot originate OS drags from a webview.
* **Cancellable archive support** — create `.zip`/`.tar`/`.tar.gz` and extract with progress,
  zip-slip protection (`safeJoin`) and partial-file cleanup on cancel.
* **Quick Look (`Space`)** — image (EXIF-corrected, with metadata badges), text/code preview,
  archive listing without extracting, folder breakdown.
* **Command palette (`Ctrl+K`)** — every action plus path jumping and recursive search.
* **Split view (`F3`)**, **Inspector (`Ctrl+I`)**, batch-rename studio with live preview and
  collision detection, permission editor, duplicate, compress, terminal-here, bookmarks, hidden-file
  toggle, natural sorting, per-mount disk-usage meters, trash badge counts.
* **Headless API mode** — the same `App` API over HTTP/SSE (`main_headless.go`) so the UI can be
  developed, tested and demoed in a browser without GTK/WebKit, with a built-in demo dataset when no
  backend is reachable.

## Architecture

```
main.go                 Wails v2 entry (wails.Run, options, OnFileDrop, single-instance lock)
main_headless.go        Same App over HTTP JSON + SSE (dev/demo mode, no GTK needed)
app.go                  App struct: listing, sidebar, config, thumbs, inspection
app_actions.go          create/rename/batch-rename/chmod/duplicate/delete + copy/move/extract
app_integrations.go     trash, open-with, clipboard, native FM, terminal, search
internal/fs/
  types.go listing.go   FileEntry/DirectoryListing, listing, natural sort, breadcrumbs
  places.go mounts.go   XDG places, /proc/mounts discovery, disk usage (statfs)
  inspect.go mutations.go  inspection/preview, archive peek, create/rename/batch rename/search
  exif*.go image_ops.go thumbs.go  EXIF parser, orientation transform, scaling, worker pool
  operations*.go        cancellable copy/move/extract engine, progress, conflicts, undo
  trash*.go             FreeDesktop multi-mount trash: trash, list, restore, undo, empty
  mime.go desktop*.go   gio/xdg MIME detection, .desktop parsing, Exec field codes, launch
  clipboard*.go uri.go native_fm.go  native clipboard, file URIs, D-Bus ShowItems, drag-out
  archive.go config.go linux_extras.go  zip/tar create+extract, config store, terminal, chmod
  *_test.go             41 unit tests covering every gap above
cmd/vela-dnd-helper/    standalone native drag/clipboard helper CLI
frontend/
  src/store*.js         reactive state, navigation, selection, file ops, ops/trash/open-with
  src/api.js            Wails bindings ⇄ HTTP fallback, event bus, demo fallback
  src/components/       HeaderBar, SidebarPlaces, VirtualFileView, ContextMenu, OpenWithModal,
                        InspectorDrawer, OperationsPopover, BatchRenameModal, ConflictDialog,
                        CommandPalette, QuickLook, DragShelf, PromptModal, ToastHost, FileIcon
  tests/                vitest + jsdom component/logic tests
```

## Build & run (Linux)

```bash
# Desktop app (needs webkit2gtk dev headers: libwebkit2gtk-4.0-dev / webkit2gtk-4.1-dev)
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails doctor                 # verifies GTK/WebKit toolchain
wails dev                    # hot-reloading development
wails build -platform linux/amd64

# Native drag/clipboard helper (shipped next to the app binary)
go build -o build/bin/vela-dnd-helper ./cmd/vela-dnd-helper

# Headless API + built frontend, for browser dev/demo without GTK
go build -tags headless -o build/bin/vela-headless .
(cd frontend && npm install && npm run build)
./build/bin/vela-headless            # http://127.0.0.1:34115

# Tests
go test ./...                        # Go: fs engine, EXIF, ops, trash, open-with
(cd frontend && npm test)            # Vue: formatters + shell render smoke tests
```

Optional runtime helpers (each degrades gracefully when absent):

| Tool | Used for |
| --- | --- |
| `gio` | MIME detection, default handlers, `gio open`, `gio trash` fallback |
| `dbus-send` | `FileManager1.ShowItems` (reveal + select in the native file manager) |
| `python3-gi` (GTK3) | multi-target clipboard ownership, native outbound drag source |
| `wl-clipboard` / `xclip` / `xsel` | clipboard fallback on Wayland / X11 |
| `dragon-drop` | alternative native drag source |
| `xdg-mime`, `xdg-open`, `xdg-terminal-exec` | handler defaults, opening, terminal launch |

## Keyboard map

| Keys | Action |
| --- | --- |
| `Enter` / double-click | Open · `Backspace` up · `Alt+←/→` history |
| Arrows / `Home` / `End` | Move selection (grid-aware) |
| `Ctrl+A`, click, `Ctrl`-click, `Shift`-click, drag on empty space | Selection + rubber band |
| `F2` / `Ctrl+D` | Inline rename / duplicate |
| `Space` | Quick Look · `Ctrl+I` inspector · `F3` split view |
| `Ctrl+C/X/V` | Copy/cut/paste (works with other Linux apps both ways) |
| `Delete` | Move to Trash (Undo toast) |
| `Ctrl+H` | Show hidden · `Ctrl+Shift+N` new folder |
| `Ctrl+K` | Command palette |

## Notes & limitations

* Outbound OS drags from a WebKit2GTK webview are impossible by design; the app routes them through
  the native helper / clipboard shelf instead.
* Trashing a file on a filesystem whose root is not writable (and that has no `.Trash-<uid>` yet)
  fails, exactly as the FreeDesktop spec and `gio trash` prescribe; the error explains why.
* `st_dev` is never trusted for filesystem identity (overlayfs/xino, btrfs subvolumes, fuse).
* Wails bindings and the headless HTTP API expose the identical method set, so the frontend runs
  unchanged in both modes; with no backend at all it renders a demo dataset.
