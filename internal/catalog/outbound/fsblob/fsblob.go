// Package fsblob implements the catalog blobstore port on the local file
// system, one file per blob key plus a sidecar metadata file.
package fsblob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
)

// ErrInvalidKey is returned for a key that is empty, absolute, contains a
// backslash or a ".." sequence, or otherwise escapes the storage root.
var ErrInvalidKey = errors.New("fsblob: invalid blob key")

// metaSuffix is appended to a blob path to locate its sidecar metadata file.
const metaSuffix = ".meta"

// metaFile is the sidecar JSON describing a stored blob.
type metaFile struct {
	ContentType string `json:"content_type"`
}

// store keeps blobs under root.
type store struct{ root string }

var _ blobstore.Store = (*store)(nil)

// New creates a blob store rooted at root, creating the directory when it does
// not exist yet.
func New(root string) (blobstore.Store, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("fsblob: storage root is required")
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("fsblob: create storage root: %w", err)
	}
	return &store{root: root}, nil
}

// Put stores r under key. The content is written to a temporary file in the
// destination directory and renamed into place, and the content type is kept in
// a <file>.meta sidecar.
func (s *store) Put(ctx context.Context, key string, r io.Reader, contentType string) (int64, error) {
	dest, err := s.path(key)
	if err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return 0, fmt.Errorf("fsblob: create blob directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".put-*")
	if err != nil {
		return 0, fmt.Errorf("fsblob: create temporary file: %w", err)
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		if committed {
			return
		}
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	size, err := io.Copy(tmp, r)
	if err != nil {
		return 0, fmt.Errorf("fsblob: write blob: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return 0, fmt.Errorf("fsblob: close blob: %w", err)
	}

	metaPath := dest + metaSuffix
	metaTmp, err := writeMeta(dir, contentType)
	if err != nil {
		return 0, err
	}
	if err := os.Rename(metaTmp, metaPath); err != nil {
		_ = os.Remove(metaTmp)
		return 0, fmt.Errorf("fsblob: commit blob metadata: %w", err)
	}
	if err := os.Rename(tmpName, dest); err != nil {
		_ = os.Remove(metaPath)
		return 0, fmt.Errorf("fsblob: commit blob: %w", err)
	}
	committed = true
	return size, nil
}

// Open returns the blob stored under key, or domain.ErrAssetNotFound when it
// does not exist.
func (s *store) Open(ctx context.Context, key string) (blobstore.Blob, error) {
	dest, err := s.path(key)
	if err != nil {
		return blobstore.Blob{}, err
	}
	if err := ctx.Err(); err != nil {
		return blobstore.Blob{}, err
	}
	f, err := os.Open(dest) //nolint:gosec // G304: dest is resolved and confined under the store root by s.path
	if errors.Is(err, os.ErrNotExist) {
		return blobstore.Blob{}, domain.ErrAssetNotFound
	}
	if err != nil {
		return blobstore.Blob{}, fmt.Errorf("fsblob: open blob: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return blobstore.Blob{}, fmt.Errorf("fsblob: stat blob: %w", err)
	}
	return blobstore.Blob{
		ReadSeekCloser: f,
		Size:           info.Size(),
		ContentType:    readContentType(dest),
		ModTime:        info.ModTime(),
	}, nil
}

// Delete removes the blob and its metadata. A missing blob is not an error.
func (s *store) Delete(ctx context.Context, key string) error {
	dest, err := s.path(key)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Remove(dest); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("fsblob: delete blob: %w", err)
	}
	if err := os.Remove(dest + metaSuffix); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("fsblob: delete blob metadata: %w", err)
	}
	return nil
}

// path validates key and resolves it under the store root, refusing any path
// that would escape it.
func (s *store) path(key string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	p := filepath.Join(s.root, filepath.FromSlash(key))
	rel, err := filepath.Rel(s.root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrInvalidKey
	}
	return p, nil
}

func validateKey(key string) error {
	if key == "" ||
		strings.Contains(key, "..") ||
		strings.ContainsRune(key, '\\') ||
		strings.HasPrefix(key, "/") ||
		strings.HasSuffix(key, "/") ||
		filepath.IsAbs(key) {
		return ErrInvalidKey
	}
	return nil
}

func writeMeta(dir, contentType string) (string, error) {
	f, err := os.CreateTemp(dir, ".meta-*")
	if err != nil {
		return "", fmt.Errorf("fsblob: create temporary metadata file: %w", err)
	}
	name := f.Name()
	if err := json.NewEncoder(f).Encode(metaFile{ContentType: contentType}); err != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return "", fmt.Errorf("fsblob: write blob metadata: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", fmt.Errorf("fsblob: close blob metadata: %w", err)
	}
	return name, nil
}

func readContentType(dest string) string {
	raw, err := os.ReadFile(dest + metaSuffix) //nolint:gosec // G304: dest is resolved and confined under the store root by s.path
	if err != nil {
		return ""
	}
	var meta metaFile
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ""
	}
	return meta.ContentType
}
