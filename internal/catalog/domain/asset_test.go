package domain_test

import (
	"errors"
	"testing"

	"github.com/JLugagne/podcast/internal/catalog/domain"
)

func TestContentTypes(t *testing.T) {
	audioTests := []struct {
		contentType string
		extension   string
	}{
		{"audio/mpeg", ".mp3"},
		{"audio/mp4", ".m4a"},
		{"audio/x-m4a", ".m4a"},
		{"audio/aac", ".aac"},
		{"audio/ogg", ".ogg"},
		{"audio/opus", ".opus"},
		{"audio/wav", ".wav"},
		{"audio/x-wav", ".wav"},
		{"audio/webm", ".webm"},
		{"audio/flac", ".flac"},
		{"audio/mpeg; charset=binary", ".mp3"},
		{" AUDIO/MPEG ", ".mp3"},
	}
	for _, tc := range audioTests {
		t.Run(tc.contentType, func(t *testing.T) {
			if got := domain.AudioExtension(tc.contentType); got != tc.extension {
				t.Fatalf("AudioExtension(%q) = %q, want %q", tc.contentType, got, tc.extension)
			}
			if err := domain.ValidateAudioContentType(tc.contentType); err != nil {
				t.Fatalf("ValidateAudioContentType(%q) error = %v", tc.contentType, err)
			}
		})
	}

	imageTests := []struct {
		contentType string
		extension   string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/webp", ".webp"},
		{"image/png; charset=binary", ".png"},
	}
	for _, tc := range imageTests {
		t.Run(tc.contentType, func(t *testing.T) {
			if got := domain.ImageExtension(tc.contentType); got != tc.extension {
				t.Fatalf("ImageExtension(%q) = %q, want %q", tc.contentType, got, tc.extension)
			}
			if err := domain.ValidateImageContentType(tc.contentType); err != nil {
				t.Fatalf("ValidateImageContentType(%q) error = %v", tc.contentType, err)
			}
		})
	}

	t.Run("rejects unknown audio", func(t *testing.T) {
		if err := domain.ValidateAudioContentType("application/octet-stream"); !errors.Is(err, domain.ErrUnsupportedAudioType) {
			t.Fatalf("error = %v, want ErrUnsupportedAudioType", err)
		}
		if ext := domain.AudioExtension("audio/unknown"); ext != "" {
			t.Fatalf("AudioExtension() = %q, want empty", ext)
		}
	})

	t.Run("rejects unknown image", func(t *testing.T) {
		if err := domain.ValidateImageContentType("image/gif"); !errors.Is(err, domain.ErrUnsupportedImageType) {
			t.Fatalf("error = %v, want ErrUnsupportedImageType", err)
		}
		if ext := domain.ImageExtension("image/svg+xml"); ext != "" {
			t.Fatalf("ImageExtension() = %q, want empty", ext)
		}
	})
}

func TestNormalizeContentType(t *testing.T) {
	tests := map[string]string{
		"image/png":                     "image/png",
		" IMAGE/PNG ; charset=binary  ": "image/png",
		"audio/mpeg":                    "audio/mpeg",
	}
	for in, want := range tests {
		if got := domain.NormalizeContentType(in); got != want {
			t.Fatalf("NormalizeContentType(%q) = %q, want %q", in, got, want)
		}
	}
}
