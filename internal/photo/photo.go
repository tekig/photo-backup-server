package photo

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
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

		switch {
		case strings.HasPrefix(event.ObjectID, entity.PrefixOrigin):
			eventsByBacketID[event.BacketID] = append(eventsByBacketID[event.BacketID], event)

		case strings.HasPrefix(event.ObjectID, entity.PrefixMeta):
			if event.EventType != entity.EventTypeObjectCreate {
				continue
			}

			if strings.HasSuffix(event.ObjectID, entity.NameMeta) {
				continue
			}

			if err := p.eventMeta(ctx, event); err != nil {
				return fmt.Errorf("event meta: %w", err)
			}
		}
	}

	for backetID, events := range eventsByBacketID {
		if err := p.eventsByBacketID(ctx, backetID, events); err != nil {
			return fmt.Errorf("events by backet id %s: %w", backetID, err)
		}
	}

	return nil
}

func (p *Photo) eventsByBacketID(ctx context.Context, bucketID string, events []entity.Event) error {
	uploader := s3manager.NewUploader(p.object)
	header := s3.New(p.object)

	metas, err := p.metaBuild(ctx, bucketID)
	if err != nil {
		return fmt.Errorf("build meta: %w", err)
	}

	var metaEvents []entity.Meta
	for _, event := range events {
		name := path.Base(event.ObjectID)

		var metaID = slices.IndexFunc(metas, func(m entity.Meta) bool {
			return name == m.ObjectID
		})
		if metaID != -1 && metas[metaID].PreviewID != "" {
			if _, err := header.DeleteObjectsWithContext(ctx, &s3.DeleteObjectsInput{
				Bucket: &bucketID,
				Delete: &s3.Delete{
					Objects: []*s3.ObjectIdentifier{
						{
							Key: aws.String(path.Join(entity.PrefixPreview, metas[metaID].PreviewID)),
						},
					},
				},
			}); err != nil {
				return fmt.Errorf("delete preview %s: %w", metas[metaID].PreviewID, err)
			}
		}

		switch event.EventType {
		case entity.EventTypeObjectCreate:
			head, err := header.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
				Bucket: &bucketID,
				Key:    &event.ObjectID,
			})
			if err != nil {
				return fmt.Errorf("head object %s: %w", event.ObjectID, err)
			}

			req, _ := header.GetObjectRequest(&s3.GetObjectInput{
				Bucket: &bucketID,
				Key:    &event.ObjectID,
			})
			presign, err := req.Presign(15 * time.Minute)
			if err != nil {
				return fmt.Errorf("presign url %s: %w", event.ObjectID, err)
			}

			previewPath, previewMime, err := makePreview(ctx, presign, head.ContentType)
			if err != nil {
				return fmt.Errorf("make preview %s: %w", event.ObjectID, err)
			}
			defer os.Remove(previewPath)

			previewFile, err := os.Open(previewPath)
			if err != nil {
				return fmt.Errorf("open preview %s: %w", event.ObjectID, err)
			}

			previewID := filepath.Base(previewPath)

			if _, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
				Body:        previewFile,
				Bucket:      &bucketID,
				ContentType: &previewMime,
				Key:         aws.String(path.Join(entity.PrefixPreview, previewID)),
			}); err != nil {
				return fmt.Errorf("upload preview %s: %w", event.ObjectID, err)
			}

			metaEvents = append(metaEvents, entity.Meta{
				ObjectID:      name,
				PreviewID:     previewID,
				PreviewIDAt:   time.Now().Unix(),
				PreviewMime:   previewMime,
				PreviewMimeAt: time.Now().Unix(),
				Deleted:       false,
				DeletedAt:     time.Now().Unix(),
			})
		case entity.EventTypeObjectDelete:
			metaEvents = append(metaEvents, entity.Meta{
				ObjectID:  name,
				Deleted:   true,
				DeletedAt: time.Now().Unix(),
			})
		}
	}

	data, err := json.Marshal(metaEvents)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}

	if _, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket:      &bucketID,
		Key:         aws.String(path.Join(entity.PrefixMeta, randMeta())),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	}); err != nil {
		return fmt.Errorf("upload meta: %w", err)
	}

	return nil
}

func randMeta() string {
	var name [4]byte
	rand.Read(name[:])

	return hex.EncodeToString(name[:]) + ".json"
}
