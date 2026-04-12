package app

import "github.com/tekig/photo-backup-server/internal/repository"

var StorageClass = map[string]repository.ObjectType{
	"Original":      repository.ObjectTypeOriginal,
	"Thumbnail":     repository.ObjectTypeThumbnail,
	"Contents":      repository.ObjectTypeContents,
	"TrashOriginal": repository.ObjectTypeTrashOriginal,
}

type Config struct {
	Gateway struct {
		Address string `yaml:"Address"`
	} `yaml:"Gateway"`
	Storage struct {
		Endpoint     string            `yaml:"Endpoint"`
		AccessKey    string            `yaml:"AccessKey"`
		AccessSecret string            `yaml:"AccessSecret"`
		Region       string            `yaml:"Region"`
		Bucket       string            `yaml:"Bucket"`
		StorageClass map[string]string `yaml:"StorageClass"`
	} `yaml:"Storage"`
}
