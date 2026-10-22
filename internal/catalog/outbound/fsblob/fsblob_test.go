package fsblob_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
	"github.com/JLugagne/podcast/internal/catalog/outbound/fsblob"
)

func newStore(t *testing.T) (string, blobstore.Store) {
	t.Helper()
	root := t.TempDir()
	store, err := fsblob.New(root)
	if err != nil {
		t.Fatalf("New(%q): %v", root, err)
	}
	return root, store
}

func TestNewCreatesRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "nested", "blobs")
	if _, err := fsblob.New(root); err != nil {
		t.Fatalf("New(%q): %v", root, err)
	}
	info, err := os.Stat(root)
	if err != nil {
		t.Fatalf("stat root: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("root is not a directory")
	}
}

func TestNewRejectsEmptyRoot(t *testing.T) {
	for _, root := range []string{"", "   "} {
		if _, err := fsblob.New(root); err == nil {
			t.Errorf("New(%q) = nil error, want error", root)
		}
	}
}

func TestPutOpenRoundTrip(t *testing.T) {
	ctx := context.Background()
	root, store := newStore(t)
	key := "episodes/abc/audio-1.mp3"
	content := []byte("fake audio payload")

	size, err := store.Put(ctx, key, bytes.NewReader(content), "audio/mpeg")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if size != int64(len(content)) {
		t.Fatalf("Put size = %d, want %d", size, len(content))
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(key)+".meta")); err != nil {
		t.Fatalf("metadata sidecar missing: %v", err)
	}

	blob, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = blob.Close() }()

	if blob.Size != int64(len(content)) {
		t.Errorf("Blob.Size = %d, want %d", blob.Size, len(content))
	}
	if blob.ContentType != "audio/mpeg" {
		t.Errorf("Blob.ContentType = %q, want audio/mpeg", blob.ContentType)
	}
	if blob.ModTime.IsZero() {
		t.Error("Blob.ModTime is zero")
	}
	got, err := io.ReadAll(blob)
	if err != nil {
		t.Fatalf("read blob: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("blob content = %q, want %q", got, content)
	}
	if _, err := blob.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("seek blob: %v", err)
	}
	again, err := io.ReadAll(blob)
	if err != nil {
		t.Fatalf("read blob after seek: %v", err)
	}
	if !bytes.Equal(again, content) {
		t.Errorf("blob content after seek = %q, want %q", again, content)
	}
}

func TestOpenMissingReturnsAssetNotFound(t *testing.T) {
	ctx := context.Background()
	_, store := newStore(t)

	_, err := store.Open(ctx, "series/missing/cover-1.png")
	if !errors.Is(err, domain.ErrAssetNotFound) {
		t.Fatalf("Open missing = %v, want domain.ErrAssetNotFound", err)
	}
}

func TestPutReplacesExistingBlob(t *testing.T) {
	ctx := context.Background()
	root, store := newStore(t)
	key := "series/abc/cover-1.png"

	if _, err := store.Put(ctx, key, strings.NewReader("old"), "image/png"); err != nil {
		t.Fatalf("first Put: %v", err)
	}
	if _, err := store.Put(ctx, key, strings.NewReader("new content"), "image/webp"); err != nil {
		t.Fatalf("second Put: %v", err)
	}

	blob, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = blob.Close() }()
	got, err := io.ReadAll(blob)
	if err != nil {
		t.Fatalf("read blob: %v", err)
	}
	if string(got) != "new content" {
		t.Errorf("blob content = %q, want %q", got, "new content")
	}
	if blob.ContentType != "image/webp" {
		t.Errorf("Blob.ContentType = %q, want image/webp", blob.ContentType)
	}

	entries, err := os.ReadDir(filepath.Join(root, "series", "abc"))
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("directory has %d entries, want 2 (blob + metadata)", len(entries))
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".put-") || strings.HasPrefix(entry.Name(), ".meta-") {
			t.Errorf("temporary file %q was left behind", entry.Name())
		}
	}
}

func TestDeleteRemovesBlobAndMetadata(t *testing.T) {
	ctx := context.Background()
	root, store := newStore(t)
	key := "episodes/abc/audio-1.mp3"

	if _, err := store.Put(ctx, key, strings.NewReader("payload"), "audio/mpeg"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	for _, path := range []string{key, key + ".meta"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("stat %s = %v, want not exist", path, err)
		}
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Errorf("Delete missing = %v, want nil", err)
	}
}

func TestRejectsInvalidKeys(t *testing.T) {
	ctx := context.Background()
	_, store := newStore(t)

	keys := []string{
		"",
		"..",
		"../escape.mp3",
		"series/../../escape.mp3",
		"series/abc/..",
		"/absolute.mp3",
		"series\\abc\\file.mp3",
		"series/abc/",
	}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			if _, err := store.Put(ctx, key, strings.NewReader("x"), "audio/mpeg"); !errors.Is(err, fsblob.ErrInvalidKey) {
				t.Errorf("Put(%q) = %v, want fsblob.ErrInvalidKey", key, err)
			}
			if _, err := store.Open(ctx, key); !errors.Is(err, fsblob.ErrInvalidKey) {
				t.Errorf("Open(%q) = %v, want fsblob.ErrInvalidKey", key, err)
			}
			if err := store.Delete(ctx, key); !errors.Is(err, fsblob.ErrInvalidKey) {
				t.Errorf("Delete(%q) = %v, want fsblob.ErrInvalidKey", key, err)
			}
		})
	}
}

func TestPutCleansUpAfterReadError(t *testing.T) {
	ctx := context.Background()
	root, store := newStore(t)
	key := "series/abc/cover-1.png"

	_, err := store.Put(ctx, key, iotest.ErrReader(errors.New("boom")), "image/png")
	if err == nil {
		t.Fatal("Put with a failing reader = nil error, want error")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(key))); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("blob file exists after failed Put: %v", err)
	}
	if _, err := store.Open(ctx, key); !errors.Is(err, domain.ErrAssetNotFound) {
		t.Errorf("Open after failed Put = %v, want domain.ErrAssetNotFound", err)
	}
}

func TestPutHonoursCancelledContext(t *testing.T) {
	_, store := newStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := store.Put(ctx, "series/abc/cover-1.png", strings.NewReader("x"), "image/png"); err == nil {
		t.Fatal("Put with a cancelled context = nil error, want error")
	}
}
