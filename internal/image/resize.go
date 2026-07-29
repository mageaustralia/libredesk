// resize.go provides image resizing utilities for multimodal AI support.
package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	stdimage "image"
	"image/color"
	stddraw "image/draw"
	"io"

	"github.com/disintegration/imaging"
)

const (
	// MaxAIDimension is the max width or height for images sent to AI APIs.
	MaxAIDimension = 500
	// JpegQuality is the quality setting for JPEG encoding.
	JpegQuality = 85
)

// ResizeForAI reads an image, resizes it to fit within MaxAIDimension, and returns bytes.
// Preserves aspect ratio. Returns original size encoding if already small enough.
// Uses the same imaging library as thumbnail generation for consistency.
func ResizeForAI(reader io.Reader, contentType string) ([]byte, string, error) {
	img, err := imaging.Decode(reader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Determine output format based on content type
	format := imaging.JPEG
	outputContentType := "image/jpeg"
	if contentType == "image/png" {
		format = imaging.PNG
		outputContentType = "image/png"
	} else if contentType == "image/gif" {
		format = imaging.GIF
		outputContentType = "image/gif"
	}

	// Check if resizing is needed
	if width > MaxAIDimension || height > MaxAIDimension {
		// Resize maintaining aspect ratio - imaging.Fit does exactly this
		img = imaging.Fit(img, MaxAIDimension, MaxAIDimension, imaging.Lanczos)
	}

	// Encode the image
	var buf bytes.Buffer
	opts := []imaging.EncodeOption{}
	if format == imaging.JPEG {
		opts = append(opts, imaging.JPEGQuality(JpegQuality))
	}
	if err := imaging.Encode(&buf, img, format, opts...); err != nil {
		return nil, "", fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), outputContentType, nil
}

const (
	// MaxEmailDimension is the max width or height for inline images embedded
	// in outgoing emails. Pasted screenshots are often full-desktop PNGs of
	// several MB; anything past ~1600px adds size without adding legibility.
	MaxEmailDimension = 1600
	// ResizeEmailMinBytes: inline images smaller than this are sent untouched —
	// re-encoding them saves little and costs quality.
	ResizeEmailMinBytes = 300 * 1024
)

// ResizeForEmail downscales an oversized inline image for email delivery.
// Mostly-opaque PNGs are flattened onto white and re-encoded as JPEG —
// screenshots saved as PNG are the classic multi-MB offender, and macOS window
// captures carry a transparent shadow border that would otherwise force them
// to stay PNG. Genuinely transparent images (logos etc.) stay PNG.
// Returns (data, contentType, true) when a smaller version was produced, or
// (original, original type, false) when the image is already reasonable, the
// format is unsupported (gif/webp/svg), or the "resized" bytes came out larger.
func ResizeForEmail(data []byte, contentType string) ([]byte, string, bool) {
	if len(data) < ResizeEmailMinBytes {
		return data, contentType, false
	}
	if contentType != "image/jpeg" && contentType != "image/png" {
		return data, contentType, false
	}

	decoded, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		return data, contentType, false
	}

	bounds := decoded.Bounds()
	if bounds.Dx() > MaxEmailDimension || bounds.Dy() > MaxEmailDimension {
		decoded = imaging.Fit(decoded, MaxEmailDimension, MaxEmailDimension, imaging.Lanczos)
	}
	img := imaging.Clone(decoded)

	format := imaging.PNG
	outType := "image/png"
	if contentType == "image/jpeg" || opaqueFraction(img) >= 0.9 {
		format = imaging.JPEG
		outType = "image/jpeg"
		img = flattenWhite(img)
	}

	var buf bytes.Buffer
	opts := []imaging.EncodeOption{}
	if format == imaging.JPEG {
		opts = append(opts, imaging.JPEGQuality(JpegQuality))
	}
	if err := imaging.Encode(&buf, img, format, opts...); err != nil {
		return data, contentType, false
	}
	if buf.Len() >= len(data) {
		return data, contentType, false
	}
	return buf.Bytes(), outType, true
}

// opaqueFraction returns the fraction of fully opaque pixels.
func opaqueFraction(img *stdimage.NRGBA) float64 {
	total := len(img.Pix) / 4
	if total == 0 {
		return 1
	}
	opaque := 0
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] == 0xff {
			opaque++
		}
	}
	return float64(opaque) / float64(total)
}

// flattenWhite composites the image over a white background, discarding alpha
// so it can be JPEG-encoded. White matches the default background of virtually
// every email client.
func flattenWhite(img *stdimage.NRGBA) *stdimage.NRGBA {
	dst := stdimage.NewNRGBA(img.Bounds())
	stddraw.Draw(dst, dst.Bounds(), stdimage.NewUniform(color.White), stdimage.Point{}, stddraw.Src)
	stddraw.Draw(dst, dst.Bounds(), img, img.Bounds().Min, stddraw.Over)
	return dst
}

// ToBase64DataURL converts image bytes to a data URL for multimodal AI APIs.
// Format: data:<content-type>;base64,<encoded-data>
func ToBase64DataURL(data []byte, contentType string) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded)
}

// ResizeAndEncodeForAI is a convenience function that resizes an image and returns
// it as a base64 data URL ready for AI API consumption.
func ResizeAndEncodeForAI(reader io.Reader, contentType string) (string, error) {
	data, outputContentType, err := ResizeForAI(reader, contentType)
	if err != nil {
		return "", err
	}
	return ToBase64DataURL(data, outputContentType), nil
}
