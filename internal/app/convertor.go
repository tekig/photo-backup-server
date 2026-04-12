package app

import "github.com/tekig/photo-backup-server/internal/repository"

func toStorageClass(src map[string]string) map[repository.ObjectType]string {
	m := map[string]repository.ObjectType{
		"Original":       repository.ObjectTypeOriginal,
		"Thumbnail":      repository.ObjectTypeThumbnail,
		"Contents":       repository.ObjectTypeContents,
		"TrashOriginal": repository.ObjectTypeTrashOriginal,
	}

	dst := make(map[repository.ObjectType]string, len(src))
	for k, s := range src {
		if s == "" {
			continue
		}
		dst[m[k]] = s
	}

	return dst
}
