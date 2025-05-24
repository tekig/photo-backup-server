package entity

import "path"

const (
	PrefixOrigin  = "origin"
	PrefixPreview = "preview"
	PrefixMeta    = "meta"
	NameMeta      = "meta.json"
)

var (
	PathMeta = path.Join(PrefixMeta, NameMeta)
)
