package fs

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// buildExifJPEG constructs a minimal JPEG carrying an EXIF orientation tag.
func buildExifJPEG(t *testing.T, orientation uint16) []byte {
	t.Helper()

	tiff := new(bytes.Buffer)
	tiff.WriteString("II")
	_ = binary.Write(tiff, binary.LittleEndian, uint16(42))
	_ = binary.Write(tiff, binary.LittleEndian, uint32(8)) // IFD0 offset
	_ = binary.Write(tiff, binary.LittleEndian, uint16(1)) // one entry
	_ = binary.Write(tiff, binary.LittleEndian, uint16(0x0112))
	_ = binary.Write(tiff, binary.LittleEndian, uint16(3)) // SHORT
	_ = binary.Write(tiff, binary.LittleEndian, uint32(1))
	_ = binary.Write(tiff, binary.LittleEndian, orientation)
	_ = binary.Write(tiff, binary.LittleEndian, uint16(0)) // padding of value field
	_ = binary.Write(tiff, binary.LittleEndian, uint32(0)) // next IFD

	app1 := new(bytes.Buffer)
	app1.WriteString("Exif\x00\x00")
	app1.Write(tiff.Bytes())

	out := new(bytes.Buffer)
	out.Write([]byte{0xFF, 0xD8}) // SOI
	out.Write([]byte{0xFF, 0xE1}) // APP1
	_ = binary.Write(out, binary.BigEndian, uint16(app1.Len()+2))
	out.Write(app1.Bytes())
	out.Write([]byte{0xFF, 0xD9}) // EOI
	return out.Bytes()
}

func TestReadEXIFOrientationFromJPEG(t *testing.T) {
	for _, want := range []int{1, 3, 6, 8} {
		dir := t.TempDir()
		path := filepath.Join(dir, "photo.jpg")
		if err := os.WriteFile(path, buildExifJPEG(t, uint16(want)), 0644); err != nil {
			t.Fatalf("write jpeg: %v", err)
		}

		meta, err := ReadEXIFMetadata(path)
		if err != nil {
			t.Fatalf("ReadEXIFMetadata: %v", err)
		}
		if meta == nil {
			t.Fatalf("expected EXIF metadata for orientation %d", want)
		}
		if meta.Orientation != want {
			t.Fatalf("orientation = %d, want %d", meta.Orientation, want)
		}
		if meta.OrientationLabel == "" {
			t.Fatal("orientation label must be populated")
		}
	}
}

func TestApplyOrientationRotatesPixels(t *testing.T) {
	// 2x1 image: left pixel red, right pixel blue.
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	src.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 255})

	// Orientation 6 = rotate 90° CW -> becomes 1x2, red on top.
	rot := ApplyOrientation(src, 6)
	if rot.Bounds().Dx() != 1 || rot.Bounds().Dy() != 2 {
		t.Fatalf("bounds = %dx%d, want 1x2", rot.Bounds().Dx(), rot.Bounds().Dy())
	}
	if c := rot.NRGBAAt(0, 0); c.R != 255 || c.B != 0 {
		t.Fatalf("top pixel = %v, want red", c)
	}
	if c := rot.NRGBAAt(0, 1); c.B != 255 || c.R != 0 {
		t.Fatalf("bottom pixel = %v, want blue", c)
	}

	// Orientation 3 = rotate 180° -> order reversed, still 2x1.
	half := ApplyOrientation(src, 3)
	if c := half.NRGBAAt(0, 0); c.B != 255 {
		t.Fatalf("rotated-180 first pixel = %v, want blue", c)
	}
}

func TestThumbnailPoolAppliesEXIFRotation(t *testing.T) {
	dir := t.TempDir()
	pool := NewThumbnailPool(2, filepath.Join(dir, "cache"))
	defer pool.Close()

	// Real JPEG (8x4) with EXIF orientation 6.
	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 20), G: uint8(y * 30), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode: %v", err)
	}
	exifBytes := buildExifJPEG(t, 6)

	// Splice the APP1 segment into the encoded JPEG right after SOI.
	payload := append([]byte{}, buf.Bytes()[:2]...)
	payload = append(payload, exifBytes[2:len(exifBytes)-2]...)
	payload = append(payload, buf.Bytes()[2:]...)

	path := filepath.Join(dir, "rotated.jpg")
	if err := os.WriteFile(path, payload, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	meta, err := ReadEXIFMetadata(path)
	if err != nil || meta == nil || meta.Orientation != 6 {
		t.Fatalf("expected orientation 6, got %+v (err=%v)", meta, err)
	}

	uri, err := pool.GetOrGenerateThumbnail(path, 64)
	if err != nil {
		t.Fatalf("thumbnail: %v", err)
	}
	if len(uri) < 32 || uri[:5] != "data:" {
		t.Fatalf("expected data URI thumbnail, got %q", uri[:minInt(len(uri), 24)])
	}

	// Cached second call must return the identical URI.
	again, err := pool.GetOrGenerateThumbnail(path, 64)
	if err != nil || again != uri {
		t.Fatalf("cache miss on second call: %v", err)
	}
}

func TestScaleToFitPreservesAspect(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 400, 200))
	scaled := ScaleToFit(src, 100, 100)
	if scaled.Bounds().Dx() != 100 || scaled.Bounds().Dy() != 50 {
		t.Fatalf("bounds = %dx%d, want 100x50", scaled.Bounds().Dx(), scaled.Bounds().Dy())
	}
}
