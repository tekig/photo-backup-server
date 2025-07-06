package entity

import "io"

type Content struct {
	Original  Object `json:"original,omitempty"`
	Thumbnail Object `json:"thumbnail,omitempty"`
}

type Object struct {
	ID           string `json:"id,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
	LastModified int64  `json:"last_modified,omitempty"`
}

type ObjectReader struct {
	Object
	Content io.ReadCloser
}
