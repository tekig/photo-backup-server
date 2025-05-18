package photo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/tekig/photo-backup-server/internal/entity"
)

type Photo struct {
	object *session.Session
}

type Config struct {
	Endpoint     string
	AccessKey    string
	AccessSecret string
	Region       string
}

func New(c Config) (*Photo, error) {
	object, err := session.NewSession(
		aws.NewConfig().
			WithEndpoint(c.Endpoint).
			WithCredentials(credentials.NewStaticCredentials(c.AccessKey, c.AccessSecret, "")).
			WithRegion(c.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("s3 session: %w", err)
	}

	return &Photo{
		object: object,
	}, nil
}

func (p *Photo) Events(ctx context.Context, events []entity.Event) error {
	var eventsByBacketID = make(map[string][]entity.Event, len(events))
	for _, event := range events {
		if !strings.HasPrefix(event.ObjectID, entity.PrefixOrigin) {
			continue
		}

		eventsByBacketID[event.BacketID] = append(eventsByBacketID[event.BacketID], event)
	}

	for backetID, events := range eventsByBacketID {
		if err := p.eventsByBacketID(ctx, backetID, events); err != nil {
			return fmt.Errorf("events by backet id %s: %w", backetID, err)
		}
	}

	return nil
}

func (p *Photo) eventsByBacketID(ctx context.Context, backetID string, events []entity.Event) error {
	downloader := s3manager.NewDownloader(p.object)
	uploader := s3manager.NewUploader(p.object)
	header := s3.New(p.object)

	buf := aws.NewWriteAtBuffer(nil)
	if _, err := downloader.DownloadWithContext(ctx, buf, &s3.GetObjectInput{
		Bucket: &backetID,
		Key:    aws.String(entity.PathMeta),
	}); err != nil {
		if !strings.Contains(err.Error(), "404") {
			return fmt.Errorf("download meta: %w", err)
		}
		buf.WriteAt([]byte("[]"), 0)
	}

	var metas []entity.Meta
	if err := json.Unmarshal(buf.Bytes(), &metas); err != nil {
		return fmt.Errorf("unmarshal meta: %w", err)
	}

	for _, event := range events {
		var metaID = slices.IndexFunc(metas, func(m entity.Meta) bool {
			return event.ObjectID == m.ObjectID
		})
		var meta *entity.Meta
		if metaID != -1 {
			meta = &metas[metaID]
		}

		name := path.Base(event.ObjectID)

		if meta != nil && meta.PreviewID != nil {
			if _, err := header.DeleteObjectsWithContext(ctx, &s3.DeleteObjectsInput{
				Bucket: &backetID,
				Delete: &s3.Delete{
					Objects: []*s3.ObjectIdentifier{
						{
							Key: aws.String(path.Join(entity.PrefixPreview, name)),
						},
					},
				},
			}); err != nil {
				return fmt.Errorf("delete preview %s: %w", event.ObjectID, err)
			}
		}

		switch event.EventType {
		case entity.EventTypeObjectCreate:
			head, err := header.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
				Bucket: &backetID,
				Key:    &event.ObjectID,
			})
			if err != nil {
				return fmt.Errorf("head object %s: %w", event.ObjectID, err)
			}

			req, _ := header.GetObjectRequest(&s3.GetObjectInput{
				Bucket: &backetID,
				Key:    &event.ObjectID,
			})
			presign, err := req.Presign(15 * time.Minute)
			if err != nil {
				return fmt.Errorf("presign url %s: %w", event.ObjectID, err)
			}

			previewPath, previewContentType, err := makePreview(ctx, presign, *head.ContentType)
			if err != nil {
				return fmt.Errorf("make preview %s: %w", event.ObjectID, err)
			}

			previewFile, err := os.Open(previewPath)
			if err != nil {
				return fmt.Errorf("open preview %s: %w", event.ObjectID, err)
			}

			previewID := filepath.Base(previewPath)

			if _, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
				Body:        previewFile,
				Bucket:      &backetID,
				ContentType: &previewContentType,
				Key:         aws.String(path.Join(entity.PrefixPreview, previewID)),
			}); err != nil {
				return fmt.Errorf("upload preview %s: %w", event.ObjectID, err)
			}

			var createdAt = time.Now().Unix()
			if value := head.Metadata["Created-At"]; value != nil {
				t, err := time.Parse(time.RFC3339, *value)
				if err != nil {
					return fmt.Errorf("Created-At parse %s: %w", event.ObjectID, err)
				}

				createdAt = t.Unix()
			}

			meta := entity.Meta{
				ObjectID:           event.ObjectID,
				ObjectContentType:  *head.ContentType,
				PreviewID:          &previewID,
				PreviewContentType: new(string),
				UpdatedAt:          head.LastModified.Unix(),
				CreatedAt:          createdAt,
			}

			if metaID != -1 {
				metas[metaID] = meta
			} else {
				metas = append(metas, meta)
			}
		case entity.EventTypeObjectDelete:
			if meta == nil {
				continue
			}

			metas = slices.DeleteFunc(metas, func(m entity.Meta) bool {
				return event.ObjectID == m.ObjectID
			})
		}
	}

	data, err := json.Marshal(metas)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}

	if _, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket:      &backetID,
		Key:         aws.String(entity.PathMeta),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	}); err != nil {
		return fmt.Errorf("upload meta: %w", err)
	}

	return nil
}
