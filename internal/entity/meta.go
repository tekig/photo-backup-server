package entity

type Meta struct {
	ObjectID           string  `json:"object_id,omitempty"`
	ObjectContentType  string  `json:"object_content_type,omitempty"`
	PreviewID          *string `json:"preview_id,omitempty"`
	PreviewContentType *string `json:"preview_content_type,omitempty"`
	UpdatedAt          int64   `json:"updated_at,omitempty"`
	CreatedAt          int64   `json:"created_at,omitempty"`
}
