package entity

type Meta struct {
	// ObjectID without prefix `origin/`
	ObjectID       string `json:"object_id,omitempty"`
	ObjectMime     string `json:"object_mime,omitempty"`
	ObjectMimeAt   int64  `json:"object_mime_at,omitempty"`
	LastModified   int64  `json:"last_modified,omitempty"`
	LastModifiedAt int64  `json:"last_modified_at,omitempty"`
	// PreviewID without prefix `preview/`
	PreviewID     string `json:"preview_id,omitempty"`
	PreviewIDAt   int64  `json:"preview_id_at,omitempty"`
	PreviewMime   string `json:"preview_mime,omitempty"`
	PreviewMimeAt int64  `json:"preview_mime_at,omitempty"`
	Deleted       bool   `json:"deleted,omitempty"`
	DeletedAt     int64  `json:"deleted_at,omitempty"`
}
