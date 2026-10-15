package fs

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CreateArchive compresses the given paths into a .zip, .tar.gz, or .tgz archive.
// Existing destination files are replaced.
func CreateArchive(paths []string, destArchive string, emit EventEmitter) (string, error) {
	if len(paths) == 0 {
		return "", os.ErrInvalid
	}
	destAbs, err := filepath.Abs(ExpandPath(destArchive))
	if err != nil {
		return "", err
	}
	lower := strings.ToLower(destAbs)

	parent := filepath.Dir(destAbs)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return "", err
	}

	switch {
	case strings.HasSuffix(lower, ".zip"):
		return destAbs, createZipArchive(paths, destAbs, emit)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return destAbs, createTarGzArchive(paths, destAbs, emit)
	case strings.HasSuffix(lower, ".tar"):
		return destAbs, createTarArchive(paths, destAbs, emit)
	default:
		return "", fmt.Errorf("unsupported archive format for %q (use .zip, .tar, or .tar.gz)", filepath.Base(destAbs))
	}
}

func createZipArchive(paths []string, dest string, emit EventEmitter) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()

	total, done := 0, 0
	_ = total
	for _, p := range paths {
		abs, err := filepath.Abs(ExpandPath(p))
		if err != nil {
			continue
		}
		rootName := filepath.Base(abs)
		info, err := os.Lstat(abs)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			if err := zipAddFile(zw, abs, rootName); err != nil {
				return err
			}
			done++
			if emit != nil {
				emit("archive:progress", map[string]any{"done": done, "file": rootName})
			}
			continue
		}
		err = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, relErr := filepath.Rel(filepath.Dir(abs), path)
			if relErr != nil {
				return nil
			}
			if d.IsDir() {
				_, err := zw.Create(rel + "/")
				return err
			}
			return zipAddFile(zw, path, rel)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func zipAddFile(zw *zip.Writer, src, name string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = filepath.ToSlash(name)
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, in)
	return err
}

func createTarGzArchive(paths []string, dest string, emit EventEmitter) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	return writeTar(paths, tar.NewWriter(gz), emit)
}

func createTarArchive(paths []string, dest string, emit EventEmitter) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return writeTar(paths, tar.NewWriter(f), emit)
}

func writeTar(paths []string, tw *tar.Writer, emit EventEmitter) error {
	defer tw.Close()
	count := 0
	for _, p := range paths {
		abs, err := filepath.Abs(ExpandPath(p))
		if err != nil {
			continue
		}
		info, err := os.Lstat(abs)
		if err != nil {
			continue
		}
		baseDir := filepath.Dir(abs)
		if !info.IsDir() {
			if err := tarAddFile(tw, abs, filepath.Base(abs)); err != nil {
				return err
			}
			count++
			if emit != nil {
				emit("archive:progress", map[string]any{"done": count, "file": filepath.Base(abs)})
			}
			continue
		}
		err = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, relErr := filepath.Rel(baseDir, path)
			if relErr != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			return tarAddFile(tw, path, rel)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func tarAddFile(tw *tar.Writer, src, name string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	hdr.Name = filepath.ToSlash(name)
	hdr.ModTime = time.Now()
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	_, err = io.Copy(tw, in)
	return err
}
