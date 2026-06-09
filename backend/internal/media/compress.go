package media

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	"golang.org/x/image/draw"
)

const maxImageDimension = 1920
const jpegQuality = 85

// CompressImageIfNeeded resizes large images and re-encodes JPEG/PNG/WebP-as-JPEG for storage savings.
// GIFs are passed through unchanged to preserve animation.
func CompressImageIfNeeded(mime string, data []byte) ([]byte, string, error) {
	clean := strings.Split(mime, ";")[0]
	if clean == "image/gif" {
		return data, clean, nil
	}
	if !strings.HasPrefix(clean, "image/") {
		return data, clean, nil
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, clean, nil
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= maxImageDimension && h <= maxImageDimension && len(data) < 400*1024 {
		return data, clean, nil
	}

	nw, nh := w, h
	if w > maxImageDimension || h > maxImageDimension {
		if w >= h {
			nw = maxImageDimension
			nh = h * maxImageDimension / w
		} else {
			nh = maxImageDimension
			nw = w * maxImageDimension / h
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	var out bytes.Buffer
	outMime := clean
	switch format {
	case "png":
		if err := png.Encode(&out, dst); err != nil {
			return data, clean, err
		}
		outMime = "image/png"
	default:
		if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
			return data, clean, err
		}
		outMime = "image/jpeg"
	}
	if out.Len() >= len(data) {
		return data, clean, nil
	}
	return out.Bytes(), outMime, nil
}

func ReadAll(r io.Reader, max int64) ([]byte, error) {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, io.LimitReader(r, max))
	return buf.Bytes(), err
}
