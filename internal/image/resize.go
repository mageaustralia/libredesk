// resize.go provides image resizing utilities for multimodal AI support.
package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
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
