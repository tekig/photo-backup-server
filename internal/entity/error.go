package entity

import "errors"

var (
	ErrNotFound    = errors.New("not found")
	ErrNotModified = errors.New("not modified")
)
