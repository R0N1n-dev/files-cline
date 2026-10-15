package fs

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// EXIFMetadata holds the photo metadata surfaced in the Inspector and used for rotation.
type EXIFMetadata struct {
	Orientation      int    `json:"orientation"` // 1..8 per EXIF spec; 1 = normal
	OrientationLabel string `json:"orientationLabel"`
	Make             string `json:"make,omitempty"`
	Model            string `json:"model,omitempty"`
	DateTime         string `json:"dateTime,omitempty"`
	ExposureTime     string `json:"exposureTime,omitempty"`
	FNumber          string `json:"fNumber,omitempty"`
	ISO              int    `json:"iso,omitempty"`
	FocalLength      string `json:"focalLength,omitempty"`
	LensModel        string `json:"lensModel,omitempty"`
	PixelWidth       int    `json:"pixelWidth,omitempty"`
	PixelHeight      int    `json:"pixelHeight,omitempty"`
	HasEXIF          bool   `json:"hasExif"`
}

// OrientationLabel converts an EXIF orientation code into a human-readable transform name.
func OrientationLabel(o int) string {
	switch o {
	case 2:
		return "Mirrored horizontal"
	case 3:
		return "Rotated 180°"
	case 4:
		return "Mirrored vertical"
	case 5:
		return "Transposed (mirror + 90° CW)"
	case 6:
		return "Rotated 90° CW"
	case 7:
		return "Transverse (mirror + 270° CW)"
	case 8:
		return "Rotated 270° CW"
	default:
		return "Normal"
	}
}

// ReadEXIFMetadata parses EXIF orientation and camera tags from JPEG, TIFF, and WebP files.
// It returns (nil, nil) when the file simply has no EXIF block, and an error only on I/O failure.
func ReadEXIFMetadata(path string) (*EXIFMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	head := make([]byte, 12)
	n, err := io.ReadFull(f, head)
	if err != nil && n < 4 {
		return nil, err
	}
	head = head[:n]

	var payload []byte

	switch {
	case n >= 4 && head[0] == 0xFF && head[1] == 0xD8:
		payload, err = extractJPEGExifPayload(f)
		if err != nil || payload == nil {
			meta := &EXIFMetadata{Orientation: 1, OrientationLabel: OrientationLabel(1)}
			meta.PixelWidth, meta.PixelHeight = readJPEGDimensions(f)
			return meta, nil
		}
	case n >= 4 && (string(head[0:4]) == "II*\x00" || string(head[0:4]) == "MM\x00*"):
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		payload, _ = io.ReadAll(io.LimitReader(f, 1<<20))
	case n >= 12 && string(head[0:4]) == "RIFF" && string(head[8:12]) == "WEBP":
		payload, _ = extractWebPExifPayload(f)
	case n >= 8 && string(head[1:4]) == "PNG":
		meta := &EXIFMetadata{Orientation: 1, OrientationLabel: OrientationLabel(1)}
		meta.PixelWidth, meta.PixelHeight = readPNGDimensions(f)
		return meta, nil
	default:
		return nil, nil
	}

	if len(payload) == 0 {
		return nil, nil
	}

	meta := parseTIFFOrientation(payload)
	if meta == nil {
		return nil, nil
	}
	meta.OrientationLabel = OrientationLabel(meta.Orientation)
	if meta.PixelWidth == 0 || meta.PixelHeight == 0 {
		if w, h := readJPEGDimensions(f); w > 0 {
			meta.PixelWidth, meta.PixelHeight = w, h
		}
	}
	return meta, nil
}

// extractJPEGExifPayload walks JPEG markers to find the APP1 "Exif\0\0" segment.
func extractJPEGExifPayload(f *os.File) ([]byte, error) {
	if _, err := f.Seek(2, io.SeekStart); err != nil {
		return nil, err
	}
	for {
		var marker [2]byte
		if _, err := io.ReadFull(f, marker[:]); err != nil {
			return nil, err
		}
		if marker[0] != 0xFF {
			return nil, errors.New("invalid JPEG marker stream")
		}
		m := marker[1]
		if m == 0xD8 || m == 0x01 || (m >= 0xD0 && m <= 0xD7) {
			continue
		}
		if m == 0xDA || m == 0xD9 {
			return nil, nil
		}
		var lenBuf [2]byte
		if _, err := io.ReadFull(f, lenBuf[:]); err != nil {
			return nil, err
		}
		segLen := int(binary.BigEndian.Uint16(lenBuf[:]))
		if segLen < 2 {
			return nil, errors.New("invalid JPEG segment length")
		}
		body := segLen - 2
		if m == 0xE1 {
			buf := make([]byte, body)
			if _, err := io.ReadFull(f, buf); err != nil {
				return nil, err
			}
			if len(buf) > 6 && string(buf[0:6]) == "Exif\x00\x00" {
				return buf[6:], nil
			}
			continue
		}
		if _, err := f.Seek(int64(body), io.SeekCurrent); err != nil {
			return nil, err
		}
	}
}
