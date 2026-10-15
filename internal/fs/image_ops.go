package fs

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
)

// decodeImage decodes common image formats using only the Go standard library.
func decodeImage(data []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// ApplyOrientation rewrites an image according to its EXIF orientation tag (1..8),
// so thumbnails and previews always appear upright.
func ApplyOrientation(src image.Image, orientation int) *image.NRGBA {
	if orientation <= 1 || orientation > 8 {
		return swapToNRGBA(src)
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	// Orientations 5..8 swap width/height.
	swap := orientation >= 5
	outW, outH := w, h
	if swap {
		outW, outH = h, w
	}

	dst := image.NewNRGBA(image.Rect(0, 0, outW, outH))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			var dx, dy int
			switch orientation {
			case 2: // mirror horizontal
				dx, dy = w-1-x, y
			case 3: // rotate 180
				dx, dy = w-1-x, h-1-y
			case 4: // mirror vertical
				dx, dy = x, h-1-y
			case 5: // transpose: mirror horizontal then rotate 270 CW
				dx, dy = y, x
			case 6: // rotate 90 CW
				dx, dy = h-1-y, x
			case 7: // transverse: mirror horizontal then rotate 90 CW
				dx, dy = h-1-y, w-1-x
			case 8: // rotate 270 CW (= 90 CCW)
				dx, dy = y, w-1-x
			default:
				dx, dy = x, y
			}
			dst.SetNRGBA(dx, dy, c)
		}
	}
	return dst
}

func swapToNRGBA(src image.Image) *image.NRGBA {
	if n, ok := src.(*image.NRGBA); ok {
		b := n.Bounds()
		if b.Min.X == 0 && b.Min.Y == 0 {
			return n
		}
	}
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.SetNRGBA(x, y, color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA))
		}
	}
	return dst
}

// ScaleToFit downscales an image with bilinear sampling, preserving aspect ratio.
// Images already smaller than the target are returned unchanged.
func ScaleToFit(src *image.NRGBA, maxW, maxH int) *image.NRGBA {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= 0 || sh <= 0 {
		return src
	}
	if (maxW <= 0 || sw <= maxW) && (maxH <= 0 || sh <= maxH) {
		return src
	}

	scale := 1.0
	if maxW > 0 {
		scale = float64(maxW) / float64(sw)
	}
	if maxH > 0 {
		if s := float64(maxH) / float64(sh); s < scale {
			scale = s
		}
	}
	if scale >= 1.0 {
		return src
	}

	dw := int(float64(sw)*scale + 0.5)
	dh := int(float64(sh)*scale + 0.5)
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}

	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	xRatio := float64(sw) / float64(dw)
	yRatio := float64(sh) / float64(dh)

	for dy := 0; dy < dh; dy++ {
		sy := int(float64(dy) * yRatio)
		if sy >= sh {
			sy = sh - 1
		}
		for dx := 0; dx < dw; dx++ {
			sx := int(float64(dx) * xRatio)
			if sx >= sw {
				sx = sw - 1
			}
			dst.SetNRGBA(dx, dy, src.NRGBAAt(b.Min.X+sx, b.Min.Y+sy))
		}
	}
	return dst
}

// EncodeThumbnail encodes a scaled image compactly and returns it as a data URI.
func EncodeThumbnail(img *image.NRGBA) (string, error) {
	var buf bytes.Buffer
	hasAlpha := false
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y && !hasAlpha; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.NRGBAAt(x, y).A < 255 {
				hasAlpha = true
				break
			}
		}
	}

	ext := "jpg"
	mime := "image/jpeg"
	if hasAlpha {
		if err := png.Encode(&buf, img); err != nil {
			return "", err
		}
		ext, mime = "png", "image/png"
	} else {
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 82}); err != nil {
			return "", err
		}
	}
	_ = ext
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// decodeGIF is referenced so gif decoding stays registered for animated previews.
var _ = gif.Decode
