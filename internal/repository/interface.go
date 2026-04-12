package repository

import (
	"context"
	"io"
)

type Object struct {
	Path        string
	ContentType string
}

type ObjectType int

const (
	ObjectTypeOriginal ObjectType = iota + 1
	ObjectTypeThumbnail
	ObjectTypeContents
	ObjectTypeTrashOriginal
)

type ObjectReader struct {
	Type        ObjectType
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

type ObjectMove struct {
	Type        ObjectType
	Source      string
	Destination string
}

type Storage interface {
	Download(ctx context.Context, req ObjectRequest) (*ObjectResponse, error)
	Upload(ctx context.Context, object ObjectReader) error
	Move(ctx context.Context, move ObjectMove) error
	Delete(ctx context.Context, path string) error
}

type Thumbnail interface {
	Create(ctx context.Context, object Object) (*Object, error)
}
