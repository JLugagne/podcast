// Package blobstoretest provides function-based test doubles for the blobstore
// port. An unset func panics so unexpected calls surface.
package blobstoretest

import (
	"context"
	"io"

	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
)

// MockStore is a function-field test double for blobstore.Store.
type MockStore struct {
	PutFn    func(ctx context.Context, key string, r io.Reader, contentType string) (int64, error)
	OpenFn   func(ctx context.Context, key string) (blobstore.Blob, error)
	DeleteFn func(ctx context.Context, key string) error
}

var _ blobstore.Store = (*MockStore)(nil)

// Put calls PutFn, panicking when it is not set.
func (m *MockStore) Put(ctx context.Context, key string, r io.Reader, contentType string) (int64, error) {
	if m.PutFn == nil {
		panic("MockStore.PutFn not set")
	}
	return m.PutFn(ctx, key, r, contentType)
}

// Open calls OpenFn, panicking when it is not set.
func (m *MockStore) Open(ctx context.Context, key string) (blobstore.Blob, error) {
	if m.OpenFn == nil {
		panic("MockStore.OpenFn not set")
	}
	return m.OpenFn(ctx, key)
}

// Delete calls DeleteFn, panicking when it is not set.
func (m *MockStore) Delete(ctx context.Context, key string) error {
	if m.DeleteFn == nil {
		panic("MockStore.DeleteFn not set")
	}
	return m.DeleteFn(ctx, key)
}
