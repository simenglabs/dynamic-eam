package media

import (
	"context"
	"io"
)

// Repository: file metadata. Get -> errs.NotFound.
type Repository interface {
	Save(ctx context.Context, m *Media) error
	Get(ctx context.Context, id string) (*Media, error)
	List(ctx context.Context, ids []string) ([]*Media, error)
	CountExisting(ctx context.Context, ids []string) (int, error)
}

// FileStore: file content (local disk in production).
type FileStore interface {
	Write(id string, r io.Reader) error
	Open(id string) (io.ReadCloser, error)
	Remove(id string) error
}
