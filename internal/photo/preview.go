package photo

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

func init() {
	mime.AddExtensionType(".heic", "image/heic")
	mime.AddExtensionType(".heif", "image/heif")
}

func makePreview(ctx context.Context, source, contentType string) (string, string, error) {
	u, err := url.Parse(source)
	if err != nil {
		return "", "", fmt.Errorf("parse url: %w", err)
	}

	name := path.Base(u.Path)

	if contentType == "application/x-www-form-urlencoded" {
		ext := path.Ext(name)
		if next := mime.TypeByExtension(ext); next != "" {
			contentType = next
		}
	}
	switch {
	case strings.HasPrefix(contentType, "video/"):
		preview := path.Join(os.TempDir(), name+".mp4")

		if err := cmd(
			ctx, "ffmpeg",
			"-i", source,
			"-t", "3",
			"-an",
			"-vf", `scale='if(gt(iw,ih),-1,256)':'if(gt(iw,ih),256,-1)',scale=trunc(iw/2)*2:trunc(ih/2)*2`,
			preview,
		); err != nil {
			return "", "", fmt.Errorf("ffmpeg convert: %w", err)
		}

		return preview, "video/mp4", nil
	case strings.HasPrefix(contentType, "image/"):
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return "", "", fmt.Errorf("new request: %w", err)
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", "", fmt.Errorf("do: %w", err)
		}
		defer res.Body.Close()

		f, err := os.Create(path.Join(os.TempDir(), name))
		if err != nil {
			return "", "", fmt.Errorf("create source: %w", err)
		}
		defer os.Remove(f.Name())
		defer f.Close()

		if _, err := io.Copy(f, res.Body); err != nil {
			return "", "", fmt.Errorf("download source: %w", err)
		}

		preview := path.Join(os.TempDir(), name+".jpg")

		// ffmpeg -i temp.jpg -vf "scale='if(gt(iw,ih),-1,256)':'if(gt(iw,ih),256,-1)'" -q:v 2 output.jpg
		// magick temp.jpg -resize 256x256^ output.jpg
		if err := cmd(
			ctx, "magick",
			f.Name(),
			"-resize", "256x256^",
			preview,
		); err != nil {
			return "", "", fmt.Errorf("magick convert: %w", err)
		}

		return preview, "image/jpeg", nil
	default:
		return "", "", fmt.Errorf("not support content type `%s`", contentType)

	}
}
