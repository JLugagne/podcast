// Package blobstore declares the outbound object-storage port for catalog
// binary assets (series covers and episode audio).
package blobstore

import (
	"context"
	"io"
	"time"
)

// Blob is a readable stored object with its metadata. The caller must close it.
type Blob struct {
	io.ReadSeekCloser
	Size        int64
	ContentType string
	ModTime     time.Time
}

// Store persists binary blobs by key.
//
// Errors: Open returns domain.ErrAssetNotFound when no blob matches the key.
// Delete treats a missing blob as success. Put returns storage errors as is.
type Store interface {
	Put(ctx context.Context, key string, r io.Reader, contentType string) (int64, error)
	Open(ctx context.Context, key string) (Blob, error)
	Delete(ctx context.Context, key string) error
}
