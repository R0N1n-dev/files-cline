package fs

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// TrashItems moves files/directories to trash using the FreeDesktop spec, and
// falls back to `gio trash` when the native path is unavailable. The returned
// transaction can be undone even when items landed on different mounts.
func (t *TrashManager) TrashItems(paths []string) (TrashTransaction, error) {
	txn := TrashTransaction{
		ID:   fmt.Sprintf("trash-%d", time.Now().UnixNano()),
		When: time.Now(),
	}

	var failed []string
	var nativeErr error
	for _, p := range paths {
		abs, err := filepath.Abs(ExpandPath(p))
		if err != nil {
			failed = append(failed, p)
			nativeErr = err
			continue
		}
		item, err := t.trashOne(abs)
		if err != nil {
			failed = append(failed, abs)
			if nativeErr == nil {
				nativeErr = err
			}
			continue
		}
		txn.Items = append(txn.Items, item)
		if item.IsTopdir && item.MountPoint != "" {
			txn.Mounts = append(txn.Mounts, item.MountPoint)
		}
	}

	if len(failed) > 0 {
		gioTxn, err := t.trashWithGio(failed)
		if err != nil {
			if len(txn.Items) == 0 {
				return txn, fmt.Errorf("cannot move to Trash: %w (per-mount trash unreachable, gio unavailable)", nativeErr)
			}
		} else {
			txn.UsedGio = true
			txn.Items = append(txn.Items, gioTxn.Items...)
			txn.Mounts = append(txn.Mounts, gioTxn.Mounts...)
		}
	}

	if len(txn.Items) == 0 {
		return txn, errors.New("nothing could be moved to trash")
	}

	t.mu.undo = append(t.mu.undo, txn)
	t.emitEvent("trash:changed", map[string]any{
		"transaction": txn.ID,
		"count":       len(txn.Items),
		"mounts":      txn.Mounts,
	})
	return txn, nil
}

// deviceOf returns the st_dev of a path (0 when unavailable).
func deviceOf(path string) uint64 {
	var st syscall.Stat_t
	if err := syscall.Stat(path, &st); err != nil {
		return 0
	}
	return uint64(st.Dev)
}

// sameFilesystem reports whether two paths live on the same mounted filesystem.
// Mount points are compared first because stacked filesystems (overlayfs with
// xino, btrfs subvolumes, fuse) can report different st_dev values for paths on
// the same mount, which would otherwise break FreeDesktop trash selection.
func sameFilesystem(a, b string) bool {
	mpA, _ := FindMountPoint(a)
	mpB, _ := FindMountPoint(b)
	if mpA != "" && mpB != "" {
		return mpA == mpB
	}
	devA, devB := deviceOf(a), deviceOf(b)
	if devA == 0 || devB == 0 {
		return true
	}
	return devA == devB
}

// trashOne performs a spec-compliant native trashing of a single absolute path.
// Per the FreeDesktop spec, the home trash is used when the file lives on the
// same filesystem as the home trash directory; otherwise a per-mount topdir
// trash (.Trash/<uid> or .Trash-<uid>) is used so external drives work too.
func (t *TrashManager) trashOne(absPath string) (TrashItem, error) {
	info, err := os.Lstat(absPath)
	if err != nil {
		return TrashItem{}, err
	}

	trashDir := HomeTrashDir()
	if err := os.MkdirAll(filepath.Join(trashDir, "files"), 0700); err != nil {
		return TrashItem{}, err
	}
	if err := os.MkdirAll(filepath.Join(trashDir, "info"), 0700); err != nil {
		return TrashItem{}, err
	}

	mountPoint, _ := FindMountPoint(absPath)
	isTopdir := false

	if !sameFilesystem(trashDir, absPath) {
		dir, mp, err := t.topdirTrashFor(absPath)
		if err != nil {
			return TrashItem{}, err
		}
		trashDir, mountPoint, isTopdir = dir, mp, true
	}

	base := filepath.Base(absPath)
	payloadName := trashPayloadName(trashDir, base)
	payloadPath := filepath.Join(trashDir, "files", payloadName)
	infoPath := filepath.Join(trashDir, "info", payloadName+".trashinfo")

	if err := os.Rename(absPath, payloadPath); err != nil {
		if !isCrossDeviceError(err) {
			return TrashItem{}, err
		}
		if err := copyPathRecursive(absPath, payloadPath); err != nil {
			return TrashItem{}, err
		}
		if err := os.RemoveAll(absPath); err != nil {
			_ = os.RemoveAll(payloadPath)
			return TrashItem{}, err
		}
	}

	// Home trash stores absolute paths; topdir trash stores paths relative to the mount root.
	storedPath := absPath
	if isTopdir {
		if rel, err := filepath.Rel(mountPoint, absPath); err == nil {
			storedPath = rel
		}
	}
	if err := writeTrashInfo(infoPath, storedPath, time.Now()); err != nil {
		_ = os.Rename(payloadPath, absPath)
		return TrashItem{}, err
	}

	return TrashItem{
		ID:           infoPath,
		Name:         payloadName,
		OriginalPath: absPath,
		DeletionDate: time.Now(),
		InfoPath:     infoPath,
		PayloadPath:  payloadPath,
		TrashDir:     trashDir,
		MountPoint:   mountPoint,
		IsTopdir:     isTopdir,
		IsDir:        info.IsDir(),
		Size:         info.Size(),
	}, nil
}

// trashWithGio delegates to the `gio trash` CLI for paths the native path rejected.
func (t *TrashManager) trashWithGio(paths []string) (TrashTransaction, error) {
	gioBin, err := exec.LookPath("gio")
	if err != nil {
		return TrashTransaction{}, errors.New("gio is not installed; cannot trash this location")
	}
	args := append([]string{"trash", "--"}, paths...)
	cmd := exec.Command(gioBin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return TrashTransaction{}, fmt.Errorf("gio trash failed: %s", msg)
	}

	txn := TrashTransaction{
		ID:      fmt.Sprintf("trash-gio-%d", time.Now().UnixNano()),
		When:    time.Now(),
		UsedGio: true,
	}
	for _, p := range paths {
		abs, err := filepath.Abs(ExpandPath(p))
		if err != nil {
			continue
		}
		if item := t.findTrashItemByOriginal(abs); item != nil {
			txn.Items = append(txn.Items, *item)
			if item.IsTopdir && item.MountPoint != "" {
				txn.Mounts = append(txn.Mounts, item.MountPoint)
			}
		}
	}
	if len(txn.Items) == 0 {
		return txn, errors.New("gio reported success but no trash entries were found")
	}
	return txn, nil
}

func mustHomeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "/"
}
