package photo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"sync"

	"github.com/tekig/photo-backup-server/internal/entity"
	"github.com/tekig/photo-backup-server/internal/repository"
)

const (
	OriginalsPath  = "originals"
	ThumbnailsPath = "thumbnails"
	TrushPath      = "trush"
	ContentName    = "content.json"
)

type Photo struct {
	storage   repository.Storage
	thumbnail repository.Thumbnail
	contents  []entity.Content

	mu sync.RWMutex
}

func New(storage repository.Storage, thumbnail repository.Thumbnail) (*Photo, error) {
	r, err := storage.Download(context.TODO(), repository.ObjectRequest{
		Path: ContentName,
	})
	if err != nil && !errors.Is(err, entity.ErrNotFound) {
		return nil, fmt.Errorf("download contents: %w", err)
	}
	if r != nil {
		defer r.Content.Close()
	}

	var contents = make([]entity.Content, 0)
	if r != nil {
		if err := json.NewDecoder(r.Content).Decode(&contents); err != nil {
			return nil, fmt.Errorf("decode contents: %w", err)
		}
	} else {
		fmt.Println("Use empty content: contents not found")
	}

	return &Photo{
		storage:   storage,
		thumbnail: thumbnail,
		contents:  contents,
	}, nil
}

func (p *Photo) Contents(ctx context.Context) ([]entity.Content, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.contents, nil
}

func (p *Photo) ContentOriginal(ctx context.Context, req entity.ObjectRequest) (*entity.ObjectReader, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	idx := slices.IndexFunc(p.contents, func(c entity.Content) bool { return c.Original.ID == req.ID })
	if idx == -1 {
		return nil, fmt.Errorf("search content: %w", entity.ErrNotFound)
	}

	content := p.contents[idx]

	if req.IfModifiedSince != nil {
		if content.Original.LastModified == *req.IfModifiedSince {
			return nil, entity.ErrNotModified
		}
	}

	object, err := p.storage.Download(ctx, repository.ObjectRequest{
		Path:  path.Join(OriginalsPath, req.ID),
		Range: req.Range,
	})
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}

	return &entity.ObjectReader{
		Object:        content.Original,
		Content:       object.Content,
		ContentLength: object.ContentLength,
		ContentRange:  object.ContentRange,
	}, nil
}

func (p *Photo) ContentThumbnail(ctx context.Context, id string, ifModifiedSince *int64) (*entity.ObjectReader, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	idx := slices.IndexFunc(p.contents, func(c entity.Content) bool { return c.Original.ID == id })
	if idx == -1 {
		return nil, fmt.Errorf("search content: %w", entity.ErrNotFound)
	}

	content := p.contents[idx]

	if ifModifiedSince != nil {
		if content.Thumbnail.LastModified == *ifModifiedSince {
			return nil, entity.ErrNotModified
		}
	}

	object, err := p.storage.Download(ctx, repository.ObjectRequest{
		Path: path.Join(ThumbnailsPath, content.Thumbnail.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}

	return &entity.ObjectReader{
		Object:  content.Thumbnail,
		Content: object.Content,
	}, nil
}

func (p *Photo) ContentUpload(ctx context.Context, original entity.ObjectReader) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	tmp, err := os.MkdirTemp("", "photo-*")
	if err != nil {
		return fmt.Errorf("mkdir temp: %w", err)
	}
	defer os.RemoveAll(tmp)

	fOrigin, err := os.Create(path.Join(tmp, original.ID))
	if err != nil {
		return fmt.Errorf("create original: %w", err)
	}
	defer fOrigin.Close()

	if _, err := io.Copy(fOrigin, original.Content); err != nil {
		return fmt.Errorf("copy original: %w", err)
	}

	if _, err := fOrigin.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek: %w", err)
	}
	original.Content = fOrigin

	th, err := p.thumbnail.Create(ctx, repository.Object{
		Path:        fOrigin.Name(),
		ContentType: original.ContentType,
	})
	if err != nil {
		return fmt.Errorf("thumbnail: %w", err)
	}
	fThumbnail, err := os.Open(th.Path)
	if err != nil {
		return fmt.Errorf("open thumbnail: %w", err)
	}
	defer fThumbnail.Close()

	thumbnail := entity.ObjectReader{
		Object: entity.Object{
			ID:           path.Base(th.Path),
			ContentType:  th.ContentType,
			LastModified: original.LastModified,
		},
		Content: fThumbnail,
	}

	if err := p.storage.Upload(ctx, repository.ObjectReader{
		Path:        path.Join(OriginalsPath, original.ID),
		ContentType: original.ContentType,
		Content:     original.Content,
	}); err != nil {
		return fmt.Errorf("upload original: %w", err)
	}

	if err := p.storage.Upload(ctx, repository.ObjectReader{
		Path:        path.Join(ThumbnailsPath, thumbnail.ID),
		ContentType: thumbnail.ContentType,
		Content:     thumbnail.Content,
	}); err != nil {
		return fmt.Errorf("upload thumbnail: %w", err)
	}

	content := entity.Content{
		Original:  original.Object,
		Thumbnail: thumbnail.Object,
	}

	idx := slices.IndexFunc(p.contents, func(c entity.Content) bool { return c.Original.ID == content.Original.ID })
	if idx != -1 {
		p.contents[idx] = content
	} else {
		p.contents = append(p.contents, content)
	}

	if err := p.contentsUpload(ctx); err != nil {
		return fmt.Errorf("contents upload: %w", err)
	}

	return nil
}

func (p *Photo) ContentDelete(ctx context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	idx := slices.IndexFunc(p.contents, func(c entity.Content) bool { return c.Original.ID == id })
	if idx == -1 {
		return nil
	}
	content := p.contents[idx]

	if err := p.storage.Delete(ctx, path.Join(ThumbnailsPath, content.Thumbnail.ID)); err != nil {
		return fmt.Errorf("thumbnail delete: %w", err)
	}

	if err := p.storage.Move(ctx, path.Join(OriginalsPath, content.Original.ID), path.Join(TrushPath, content.Original.ID)); err != nil {
		return fmt.Errorf("original trush: %w", err)
	}

	p.contents = slices.DeleteFunc(p.contents, func(c entity.Content) bool { return c.Original.ID == id })

	if err := p.contentsUpload(ctx); err != nil {
		return fmt.Errorf("contents upload: %w", err)
	}

	return nil
}

func (p *Photo) contentsUpload(ctx context.Context) error {
	data, err := json.Marshal(p.contents)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := p.storage.Upload(ctx, repository.ObjectReader{
		Path:        ContentName,
		ContentType: "applicaltion/json",
		Content:     bytes.NewReader(data),
	}); err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	return nil
}
