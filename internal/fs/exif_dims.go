package fs

import (
	"encoding/binary"
	"io"
	"os"
)

// extractWebPExifPayload reads the EXIF chunk from a RIFF/WEBP container.
func extractWebPExifPayload(f *os.File) ([]byte, error) {
	if _, err := f.Seek(12, io.SeekStart); err != nil {
		return nil, err
	}
	for {
		var hdr [8]byte
		if _, err := io.ReadFull(f, hdr[:]); err != nil {
			return nil, err
		}
		fourCC := string(hdr[0:4])
		chunkSize := int(binary.LittleEndian.Uint32(hdr[4:8]))
		if chunkSize < 0 || chunkSize > 1<<24 {
			return nil, nil
		}
		if fourCC == "EXIF" {
			buf := make([]byte, chunkSize)
			if _, err := io.ReadFull(f, buf); err != nil {
				return nil, err
			}
			if len(buf) > 6 && string(buf[0:6]) == "Exif\x00\x00" {
				return buf[6:], nil
			}
			return buf, nil
		}
		skip := int64(chunkSize + (chunkSize & 1))
		if _, err := f.Seek(skip, io.SeekCurrent); err != nil {
			return nil, err
		}
	}
}

// readJPEGDimensions scans SOF markers for image dimensions.
func readJPEGDimensions(f *os.File) (int, int) {
	if _, err := f.Seek(2, io.SeekStart); err != nil {
		return 0, 0
	}
	for {
		var marker [2]byte
		if _, err := io.ReadFull(f, marker[:]); err != nil {
			return 0, 0
		}
		if marker[0] != 0xFF {
			return 0, 0
		}
		m := marker[1]
		if m == 0xD8 || m == 0x01 || (m >= 0xD0 && m <= 0xD7) {
			continue
		}
		if m == 0xDA || m == 0xD9 {
			return 0, 0
		}
		var lenBuf [2]byte
		if _, err := io.ReadFull(f, lenBuf[:]); err != nil {
			return 0, 0
		}
		segLen := int(binary.BigEndian.Uint16(lenBuf[:]))
		// SOF0..SOF15 (excluding DHT 0xC4, JPG 0xC8, DAC 0xCC)
		if m >= 0xC0 && m <= 0xCF && m != 0xC4 && m != 0xC8 && m != 0xCC {
			buf := make([]byte, segLen-2)
			if _, err := io.ReadFull(f, buf); err != nil {
				return 0, 0
			}
			if len(buf) >= 5 {
				return int(binary.BigEndian.Uint16(buf[3:5])), int(binary.BigEndian.Uint16(buf[1:3]))
			}
			return 0, 0
		}
		if _, err := f.Seek(int64(segLen-2), io.SeekCurrent); err != nil {
			return 0, 0
		}
	}
}

// readPNGDimensions reads width/height from the IHDR chunk.
func readPNGDimensions(f *os.File) (int, int) {
	if _, err := f.Seek(16, io.SeekStart); err != nil {
		return 0, 0
	}
	buf := make([]byte, 8)
	if _, err := io.ReadFull(f, buf); err != nil {
		return 0, 0
	}
	return int(binary.BigEndian.Uint32(buf[0:4])), int(binary.BigEndian.Uint32(buf[4:8]))
}
