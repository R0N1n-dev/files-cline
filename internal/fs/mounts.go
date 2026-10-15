package fs

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func DiscoverMountedVolumes(trashMgr *TrashManager) []MountVolume {
	var volumes []MountVolume
	seenMounts := make(map[string]bool)

	addVolume := func(name, mountPoint, device, fsType string, removable bool) {
		if seenMounts[mountPoint] {
			return
		}
		total, free, usedPct := GetDiskUsage(mountPoint)
		if total == 0 {
			return
		}
		seenMounts[mountPoint] = true
		trashCount := 0
		hasTrash := false
		if trashMgr != nil {
			if tDir := trashMgr.FindExistingTopdirTrash(mountPoint); tDir != "" {
				hasTrash = true
				trashCount = trashMgr.CountTrashInDir(tDir)
			}
		}
		volumes = append(volumes, MountVolume{
			Name: name, MountPoint: mountPoint, Device: device, FSType: fsType,
			TotalBytes: total, FreeBytes: free, UsedBytes: total - free, UsedPercent: usedPct,
			IsRemovable: removable, HasTrash: hasTrash, TrashCount: trashCount,
		})
	}

	addVolume("System Root", "/", "rootfs", "ext4/overlay", false)
	if info, err := os.Stat("/workspace"); err == nil && info.IsDir() {
		addVolume("Workspace Volume", "/workspace", "/dev/workspace", "ext4", false)
	}

	f, err := os.Open("/proc/mounts")
	if err != nil {
		return volumes
	}
	defer f.Close()

	ignoredFS := map[string]bool{
		"proc": true, "sysfs": true, "devtmpfs": true, "devpts": true, "tmpfs": true,
		"cgroup": true, "cgroup2": true, "pstore": true, "bpf": true, "tracefs": true,
		"debugfs": true, "securityfs": true, "hugetlbfs": true, "mqueue": true, "fusectl": true,
		"configfs": true, "autofs": true, "efivarfs": true, "squashfs": true, "nsfs": true,
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		device, mountPt, fsType := fields[0], unescapeOctalMountPath(fields[1]), fields[2]
		if ignoredFS[fsType] ||
			strings.HasPrefix(mountPt, "/proc") ||
			strings.HasPrefix(mountPt, "/sys") ||
			strings.HasPrefix(mountPt, "/dev") ||
			strings.HasPrefix(mountPt, "/run/credentials") ||
			strings.HasPrefix(mountPt, "/etc/") {
			continue
		}
		isRemovable := strings.HasPrefix(mountPt, "/media/") ||
			strings.HasPrefix(mountPt, "/run/media/") ||
			strings.HasPrefix(mountPt, "/mnt/")
		label := filepath.Base(mountPt)
		if mountPt == "/" {
			label = "System Root"
		}
		addVolume(label, mountPt, device, fsType, isRemovable)
	}
	return volumes
}

func FindMountPoint(absPath string) (mountPoint string, fsType string) {
	clean := filepath.Clean(absPath)
	bestMount, bestFS := "/", "ext4"
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return bestMount, bestFS
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		mp, ft := unescapeOctalMountPath(fields[1]), fields[2]
		if mp == "/" || clean == mp || strings.HasPrefix(clean, mp+"/") {
			if len(mp) >= len(bestMount) {
				bestMount, bestFS = mp, ft
			}
		}
	}
	return bestMount, bestFS
}

func unescapeOctalMountPath(s string) string {
	r := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return r.Replace(s)
}
