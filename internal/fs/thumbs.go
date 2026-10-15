package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// ThumbnailJob is one unit of work for the bounded goroutine thumbnail pool.
type ThumbnailJob struct {
	Path string
	Size int
}

// ThumbnailResult is emitted for each generated (or failed) thumbnail.
type ThumbnailResult struct {
	Path string `json:"path"`
	URI  string `json:"uri"`
	Err  string `json:"err,omitempty"`
}

// ThumbnailPool runs a bounded worker pool that decodes images, applies EXIF
// orientation, downscales, and caches results on disk + in memory.
type ThumbnailPool struct {
	mu        sync.Mutex
	memCache  map[string]string // key -> data URI
	inflight  map[string]bool
	cacheDir  string
	jobs      chan ThumbnailJob
	wg        sync.WaitGroup
	closed    bool
	onResult  func(ThumbnailResult)
	workers   int
	maxMemory int
	order     []string
}

// NewThumbnailPool creates a pool with a fixed worker count. A worker count of
// 0 or less is clamped to 4 (safe on constrained machines).
func NewThumbnailPool(workers int, cacheDir string) *ThumbnailPool {
	if workers <= 0 {
		workers = 4
	}
	if cacheDir == "" {
		if base, err := os.UserCacheDir(); err == nil {
			cacheDir = filepath.Join(base, "vela-files", "thumbnails")
		} else {
			cacheDir = filepath.Join(os.TempDir(), "vela-files-thumbs")
		}
	}
	_ = os.MkdirAll(cacheDir, 0755)

	p := &ThumbnailPool{
		memCache:  make(map[string]string),
		inflight:  make(map[string]bool),
		cacheDir:  cacheDir,
		jobs:      make(chan ThumbnailJob, 512),
		workers:   workers,
		maxMemory: 800,
	}

	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	return p
}

// SetResultHandler registers a callback invoked with each finished thumbnail.
// It is wired to a Wails event emitter in the desktop build.
func (p *ThumbnailPool) SetResultHandler(fn func(ThumbnailResult)) {
	p.mu.Lock()
	p.onResult = fn
	p.mu.Unlock()
}

func (p *ThumbnailPool) worker() {
	defer p.wg.Done()
	for job := range p.jobs {
		uri, err := p.generate(job.Path, job.Size)
		res := ThumbnailResult{Path: job.Path}
		if err != nil {
			res.Err = err.Error()
		} else {
			res.URI = uri
		}

		p.mu.Lock()
		delete(p.inflight, cacheKey(job.Path, job.Size))
		handler := p.onResult
		p.mu.Unlock()

		if handler != nil {
			handler(res)
		}
	}
}

func cacheKey(path string, size int) string {
	sum := sha256.Sum256([]byte(path + "|" + strconv.Itoa(size)))
	return hex.EncodeToString(sum[:16])
}

func (p *ThumbnailPool) diskPath(path string, size int, modTime int64) string {
	sum := sha256.Sum256([]byte(path + "|" + strconv.Itoa(size) + "|" + strconv.FormatInt(modTime, 10)))
	return filepath.Join(p.cacheDir, hex.EncodeToString(sum[:20])+".thumb")
}

// GetOrGenerateThumbnail synchronously returns a cached thumbnail or generates one.
func (p *ThumbnailPool) GetOrGenerateThumbnail(path string, size int) (string, error) {
	key := cacheKey(path, size)

	p.mu.Lock()
	if uri, ok := p.memCache[key]; ok {
		p.mu.Unlock()
		return uri, nil
	}
	p.mu.Unlock()

	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if dPath := p.diskPath(path, size, info.ModTime().Unix()); dPath != "" {
		if data, err := os.ReadFile(dPath); err == nil && len(data) > 0 {
			uri := string(data)
			p.store(key, uri)
			return uri, nil
		}
	}

	uri, err := p.generate(path, size)
	if err != nil {
		return "", err
	}
	if dPath := p.diskPath(path, size, info.ModTime().Unix()); dPath != "" {
		_ = os.WriteFile(dPath, []byte(uri), 0644)
	}
	return uri, nil
}

func (p *ThumbnailPool) store(key, uri string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.memCache[key]; !exists {
		p.order = append(p.order, key)
	}
	p.memCache[key] = uri
	for len(p.order) > p.maxMemory {
		oldest := p.order[0]
		p.order = p.order[1:]
		delete(p.memCache, oldest)
	}
}

func (p *ThumbnailPool) generate(path string, size int) (string, error) {
	if size <= 0 {
		size = 256
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	img, err := decodeImage(data)
	if err != nil {
		return "", err
	}

	orientation := 1
	if meta, exErr := ReadEXIFMetadata(path); exErr == nil && meta != nil && meta.Orientation > 1 {
		orientation = meta.Orientation
	}

	oriented := ApplyOrientation(img, orientation)
	scaled := ScaleToFit(oriented, size, size)
	return EncodeThumbnail(scaled)
}

// Queue schedules an asynchronous thumbnail job. Duplicate requests are deduplicated.
func (p *ThumbnailPool) Queue(path string, size int) {
	key := cacheKey(path, size)

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	if _, ok := p.memCache[key]; ok {
		p.mu.Unlock()
		return
	}
	if p.inflight[key] {
		p.mu.Unlock()
		return
	}
	p.inflight[key] = true
	p.mu.Unlock()

	select {
	case p.jobs <- ThumbnailJob{Path: path, Size: size}:
	case <-time.After(2 * time.Second):
		p.mu.Lock()
		delete(p.inflight, key)
		p.mu.Unlock()
	}
}

// QueueBatch schedules many thumbnails at once.
func (p *ThumbnailPool) QueueBatch(paths []string, size int) {
	for _, path := range paths {
		p.Queue(path, size)
	}
}

// Close drains and stops the pool.
func (p *ThumbnailPool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()
	close(p.jobs)
	p.wg.Wait()
}

// ErrNoThumbnail is returned when a file cannot be previewed as an image.
var ErrNoThumbnail = errors.New("no thumbnail available")
