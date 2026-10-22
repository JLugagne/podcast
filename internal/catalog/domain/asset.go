package domain

import "strings"

// Asset is a stored blob described by its key, content type and size.
type Asset struct {
	Key         string
	ContentType string
	SizeBytes   int64
}

// AudioFile is an episode audio asset with its optional duration and the
// original upload file name.
type AudioFile struct {
	Asset
	DurationSeconds float64
	OriginalName    string
}

var audioExtensions = map[string]string{
	"audio/mpeg":  ".mp3",
	"audio/mp4":   ".m4a",
	"audio/x-m4a": ".m4a",
	"audio/aac":   ".aac",
	"audio/ogg":   ".ogg",
	"audio/opus":  ".opus",
	"audio/wav":   ".wav",
	"audio/x-wav": ".wav",
	"audio/webm":  ".webm",
	"audio/flac":  ".flac",
}

var imageExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// NormalizeContentType lowercases the media type and drops any parameter.
func NormalizeContentType(contentType string) string {
	ct := strings.TrimSpace(contentType)
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.ToLower(strings.TrimSpace(ct))
}

// AudioExtension returns the file extension (with dot) of a supported audio
// content type, or an empty string when it is not supported.
func AudioExtension(contentType string) string {
	return audioExtensions[NormalizeContentType(contentType)]
}

// ImageExtension returns the file extension (with dot) of a supported image
// content type, or an empty string when it is not supported.
func ImageExtension(contentType string) string {
	return imageExtensions[NormalizeContentType(contentType)]
}

// ValidateAudioContentType returns ErrUnsupportedAudioType when the content
// type is not an allowed audio format.
func ValidateAudioContentType(contentType string) error {
	if AudioExtension(contentType) == "" {
		return ErrUnsupportedAudioType
	}
	return nil
}

// ValidateImageContentType returns ErrUnsupportedImageType when the content
// type is not an allowed image format.
func ValidateImageContentType(contentType string) error {
	if ImageExtension(contentType) == "" {
		return ErrUnsupportedImageType
	}
	return nil
}
