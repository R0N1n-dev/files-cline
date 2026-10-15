package fs

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func (m *OperationsManager) extractZip(op *Operation, archivePath, destDir string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, zf := range zr.File {
		if err := op.checkAbort(); err != nil {
			return err
		}
		target, err := safeJoin(destDir, zf.Name)
		if err != nil {
			continue
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(target, zf.Mode().Perm()|0700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		err = m.writeStream(op, rc, target, zf.Mode().Perm())
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *OperationsManager) extractTar(op *Operation, archivePath, destDir string, compressed bool) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var reader io.Reader = f
	if compressed {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		reader = gz
	}

	tr := tar.NewReader(reader)
	for {
		if err := op.checkAbort(); err != nil {
			return err
		}
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := safeJoin(destDir, hdr.Name)
		if err != nil {
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)|0700); err != nil {
				return err
			}
		case tar.TypeSymlink:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			_ = os.Remove(target)
			_ = os.Symlink(hdr.Linkname, target)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := m.writeStream(op, tr, target, os.FileMode(hdr.Mode).Perm()); err != nil {
				return err
			}
		}
	}
}

func archiveUncompressedSize(archivePath string) (int64, error) {
	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".zip") {
		zr, err := zip.OpenReader(archivePath)
		if err != nil {
			return 0, err
		}
		defer zr.Close()
		var total int64
		for _, f := range zr.File {
			total += int64(f.UncompressedSize64)
		}
		return total, nil
	}
	if strings.HasSuffix(lower, ".tar") || strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		f, err := os.Open(archivePath)
		if err != nil {
			return 0, err
		}
		defer f.Close()
		var reader io.Reader = f
		if strings.HasSuffix(lower, ".gz") || strings.HasSuffix(lower, ".tgz") {
			gz, err := gzip.NewReader(f)
			if err != nil {
				return 0, err
			}
			defer gz.Close()
			reader = gz
		}
		tr := tar.NewReader(reader)
		var total int64
		for {
			hdr, err := tr.Next()
			if err != nil {
				break
			}
			if hdr.Typeflag == tar.TypeReg {
				total += hdr.Size
			}
		}
		return total, nil
	}
	return 0, errors.New("unsupported archive format")
}
