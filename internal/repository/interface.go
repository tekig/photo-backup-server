package repository

import (
	"context"
	"io"
)

type Object struct {
	Path        string
	ContentType string
}

type ObjectReader struct {
	Path        string
	ContentType string
	Content     io.Reader
}

type ObjectRequest struct {
	Path  string
	Range *string
}

type ObjectResponse struct {
	ContentLength *int64
	ContentRange  *string
	Content       io.ReadCloser
}

type Storage interface {
	Download(ctx context.Context, req ObjectRequest) (*ObjectResponse, error)
	Upload(ctx context.Context, object ObjectReader) error
	Move(ctx context.Context, src, dst string) error
	Delete(ctx context.Context, path string) error
}

type Thumbnail interface {
	Create(ctx context.Context, object Object) (*Object, error)
}
