package fs

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"os/user"
	"strings"
	"unicode/utf8"
)

func FastClassifyMime(name, ext string) (mimeType string, category string) {
	lowerName := strings.ToLower(name)
	if lowerName == "dockerfile" || lowerName == "makefile" || strings.HasSuffix(lowerName, ".mk") {
		return "text/x-makefile", "code"
	}
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg", "image"
	case "png":
		return "image/png", "image"
	case "gif":
		return "image/gif", "image"
	case "webp":
		return "image/webp", "image"
	case "svg":
		return "image/svg+xml", "image"
	case "bmp", "ico", "tiff", "tif", "heic", "avif":
		return "image/" + ext, "image"
	case "mp4", "mkv", "webm", "mov", "avi", "flv", "m4v":
		return "video/" + ext, "video"
	case "mp3", "wav", "flac", "ogg", "m4a", "aac", "opus":
		return "audio/" + ext, "audio"
	case "pdf":
		return "application/pdf", "pdf"
	case "zip", "tar", "gz", "tgz", "bz2", "xz", "7z", "rar", "zst":
		return "application/x-archive", "archive"
	case "go", "js", "ts", "vue", "py", "rs", "c", "cpp", "h", "hpp", "java", "rb", "php", "sh", "bash", "zsh", "json", "yaml", "yml", "toml", "xml", "html", "css", "scss", "sql", "lua":
		return "text/plain", "code"
	case "md", "txt", "rst", "log", "ini", "conf", "cfg", "env", "csv":
		return "text/plain", "document"
	case "doc", "docx", "odt", "rtf", "xls", "xlsx", "ods", "ppt", "pptx", "odp":
		return "application/vnd.oasis.opendocument", "document"
	default:
		return "application/octet-stream", "file"
	}
}

func InspectFile(rawPath string, thumbPool *ThumbnailPool, mimeHelper *MimeHelper) (*FileInspection, error) {
	entry, err := BuildFileEntryFromPath(rawPath)
	if err != nil {
		return nil, err
	}

	res := &FileInspection{
		Entry:     entry,
		OwnerName: "user",
		GroupName: "user",
	}
	if u, err := user.Current(); err == nil {
		res.OwnerName = u.Username
	}
	if mimeHelper != nil {
		if realMime := mimeHelper.DetectMimeType(entry.Path, entry.IsDir); realMime != "" {
			res.Entry.MimeType = realMime
		}
		res.DefaultApp = mimeHelper.GetDefaultApp(res.Entry.MimeType)
	}

	if entry.IsDir {
		if dirEntries, err := os.ReadDir(entry.Path); err == nil {
			var sizeSum int64
			for _, de := range dirEntries {
				if de.IsDir() {
					res.SubfolderCount++
				} else {
					res.SubfileCount++
					if info, err := de.Info(); err == nil {
						sizeSum += info.Size()
					}
				}
			}
			res.CalculatedSize = sizeSum
		}
		return res, nil
	}

	if entry.Category == "image" {
		if exifMeta, err := ReadEXIFMetadata(entry.Path); err == nil && exifMeta != nil {
			res.EXIF = exifMeta
		}
		if thumbPool != nil {
			if uri, err := thumbPool.GetOrGenerateThumbnail(entry.Path, 360); err == nil {
				res.ThumbnailURI = uri
			}
		}
		return res, nil
	}

	if entry.Category == "archive" {
		if entries, total, err := peekArchiveEntries(entry.Path, 100); err == nil {
			res.ArchiveEntries = entries
			res.ArchiveTotal = total
		}
		return res, nil
	}

	if entry.Category == "code" || entry.Category == "document" || entry.Size < 512*1024 {
		if f, err := os.Open(entry.Path); err == nil {
			defer f.Close()
			buf := make([]byte, 32*1024)
			n, _ := f.Read(buf)
			if n > 0 {
				sample := buf[:n]
				if isLikelyUTF8Text(sample) {
					res.TextPreview = string(sample)
					res.TextLineCount = strings.Count(res.TextPreview, "\n") + 1
				} else {
					res.IsBinary = true
				}
			}
		}
	}
	return res, nil
}

func isLikelyUTF8Text(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	checkLen := len(data)
	if checkLen > 4096 {
		checkLen = 4096
	}
	for i := 0; i < checkLen; i++ {
		if data[i] == 0 {
			return false
		}
	}
	return utf8.Valid(data[:checkLen])
}

func peekArchiveEntries(archivePath string, limit int) ([]ArchiveEntrySummary, int, error) {
	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".zip") {
		zr, err := zip.OpenReader(archivePath)
		if err != nil {
			return nil, 0, err
		}
		defer zr.Close()
		total := len(zr.File)
		out := make([]ArchiveEntrySummary, 0, minInt(total, limit))
		for i, f := range zr.File {
			if i >= limit {
				break
			}
			out = append(out, ArchiveEntrySummary{
				Name:    f.Name,
				Size:    int64(f.UncompressedSize64),
				IsDir:   f.FileInfo().IsDir(),
				ModTime: f.Modified.Format("2006-01-02 15:04"),
			})
		}
		return out, total, nil
	}

	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") || strings.HasSuffix(lower, ".tar") {
		f, err := os.Open(archivePath)
		if err != nil {
			return nil, 0, err
		}
		defer f.Close()
		var r io.Reader = f
		if strings.HasSuffix(lower, ".gz") || strings.HasSuffix(lower, ".tgz") {
			gz, err := gzip.NewReader(f)
			if err != nil {
				return nil, 0, err
			}
			defer gz.Close()
			r = gz
		}
		tr := tar.NewReader(r)
		var out []ArchiveEntrySummary
		total := 0
		for {
			hdr, err := tr.Next()
			if err != nil {
				break
			}
			total++
			if len(out) < limit {
				out = append(out, ArchiveEntrySummary{
					Name:    hdr.Name,
					Size:    hdr.Size,
					IsDir:   hdr.Typeflag == tar.TypeDir,
					ModTime: hdr.ModTime.Format("2006-01-02 15:04"),
				})
			}
		}
		return out, total, nil
	}
	return nil, 0, errors.New("unsupported archive format")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
