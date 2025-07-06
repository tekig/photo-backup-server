package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/tekig/photo-backup-server/internal/repository"
)

type CMD struct {
}

func New() *CMD {
	return &CMD{}
}

func (c *CMD) Create(ctx context.Context, original repository.Object) (*repository.Object, error) {
	dir, name := path.Split(original.Path)

	switch {
	case strings.HasPrefix(original.ContentType, "video/"):
		preview := path.Join(dir, name+".mp4")

		if err := cmd(
			ctx, "ffmpeg",
			"-i", original.Path,
			"-t", "3",
			"-an",
			"-vf", `scale='if(gt(iw,ih),-1,256)':'if(gt(iw,ih),256,-1)',scale=trunc(iw/2)*2:trunc(ih/2)*2`,
			preview,
		); err != nil {
			return nil, fmt.Errorf("ffmpeg convert: %w", err)
		}

		return &repository.Object{
			Path:        preview,
			ContentType: "video/mp4",
		}, nil
	case strings.HasPrefix(original.ContentType, "image/"):
		preview := path.Join(dir, name+".jpg")

		// ffmpeg -i temp.jpg -vf "scale='if(gt(iw,ih),-1,256)':'if(gt(iw,ih),256,-1)'" -q:v 2 output.jpg
		// magick temp.jpg -resize 256x256^ output.jpg
		if err := cmd(
			ctx, "magick",
			original.Path,
			"-resize", "256x256^",
			preview,
		); err != nil {
			return nil, fmt.Errorf("magick convert: %w", err)
		}

		return &repository.Object{
			Path:        preview,
			ContentType: "image/jpeg",
		}, nil
	default:
		return nil, fmt.Errorf("not support content type `%s`", original.ContentType)
	}
}

func cmd(ctx context.Context, prog string, args ...string) error {
	cmd := exec.CommandContext(ctx, prog, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run stderr=`%s`: %w", stderr.String(), err)
	}

	return nil
}
