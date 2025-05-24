package photo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/tekig/photo-backup-server/internal/entity"
)

func (p *Photo) eventMeta(ctx context.Context, event entity.Event) error {
	metas, err := p.metaCompact(ctx, event.BacketID, event.ObjectID)
	if err != nil {
		return fmt.Errorf("meta compact: %w", err)
	}

	metaRaw, err := json.MarshalIndent(metas, "", "\t")
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}

	if _, err := s3manager.NewUploader(p.object).UploadWithContext(ctx, &s3manager.UploadInput{
		Body:        aws.ReadSeekCloser(bytes.NewReader(metaRaw)),
		Bucket:      &event.BacketID,
		Key:         aws.String(entity.PathMeta),
		ContentType: aws.String("application/json"),
	}); err != nil {
		return fmt.Errorf("upload meta: %w", err)
	}

	if _, err := s3.New(p.object).DeleteObject(&s3.DeleteObjectInput{
		Bucket: &event.BacketID,
		Key:    &event.ObjectID,
	}); err != nil {
		return fmt.Errorf("delete wal: %w", err)
	}

	return nil
}

func (p *Photo) metaBuild(ctx context.Context, bucketID string) ([]entity.Meta, error) {
	header := s3.New(p.object)

	metas, err := header.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: &bucketID,
		Prefix: aws.String(entity.PrefixMeta),
	})
	if err != nil {
		return nil, fmt.Errorf("list metas: %w", err)
	}

	var wals []string
	for _, o := range metas.Contents {
		if strings.HasSuffix(*o.Key, entity.NameMeta) {
			continue
		}
		wals = append(wals, *o.Key)
	}

	return p.metaCompact(ctx, bucketID)
}

func (p *Photo) metaCompact(ctx context.Context, bucketID string, wals ...string) ([]entity.Meta, error) {
	downloader := s3manager.NewDownloader(p.object)

	var events []entity.Meta
	for _, wal := range wals {
		walBuf := aws.NewWriteAtBuffer(nil)
		if _, err := downloader.DownloadWithContext(ctx, walBuf, &s3.GetObjectInput{
			Bucket: &bucketID,
			Key:    &wal,
		}); err != nil {
			return nil, fmt.Errorf("download wal: %w", err)
		}

		var partEvents []entity.Meta
		if err := json.Unmarshal(walBuf.Bytes(), &partEvents); err != nil {
			return nil, fmt.Errorf("unmarshal wal: %w", err)
		}

		events = append(events, partEvents...)
	}

	metaBuf := aws.NewWriteAtBuffer(nil)
	if _, err := downloader.DownloadWithContext(ctx, metaBuf, &s3.GetObjectInput{
		Bucket: &bucketID,
		Key:    aws.String(entity.PathMeta),
	}); err != nil {
		if !strings.Contains(err.Error(), "404") {
			return nil, fmt.Errorf("download meta: %w", err)
		}
		metaBuf.WriteAt([]byte("[]"), 0)
	}

	var metas []entity.Meta
	if err := json.Unmarshal(metaBuf.Bytes(), &metas); err != nil {
		return nil, fmt.Errorf("unmarshal meta: %w", err)
	}
	var metaByID = make(map[string]entity.Meta, len(metas))
	for _, v := range metas {
		metaByID[v.ObjectID] = v
	}

	for _, event := range events {
		meta, ok := metaByID[event.ObjectID]
		if !ok {
			meta.ObjectID = event.ObjectID
		}

		if event.UploadAt != 0 && meta.UploadAt < event.UploadAt {
			meta.UploadAt = event.UploadAt
		}
		if event.ObjectMimeAt != 0 && meta.ObjectMimeAt < event.ObjectMimeAt {
			meta.ObjectMimeAt = event.ObjectMimeAt
			meta.ObjectMime = event.ObjectMime
		}
		if event.LastModifiedAt != 0 && meta.LastModifiedAt < event.LastModifiedAt {
			meta.LastModifiedAt = event.LastModifiedAt
			meta.LastModified = event.LastModified
		}
		if event.PreviewIDAt != 0 && meta.PreviewIDAt < event.PreviewIDAt {
			meta.PreviewIDAt = event.PreviewIDAt
			meta.PreviewID = event.PreviewID
		}
		if event.PreviewMimeAt != 0 && meta.PreviewMimeAt < event.PreviewMimeAt {
			meta.PreviewMimeAt = event.PreviewMimeAt
			meta.PreviewMime = event.PreviewMime
		}
		if event.DeletedAt != 0 && meta.DeletedAt < event.DeletedAt {
			meta.Deleted = event.Deleted
			meta.DeletedAt = event.DeletedAt
		}

		metaByID[event.ObjectID] = meta
	}

	metas = metas[:0]
	for _, m := range metaByID {
		metas = append(metas, m)
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].ObjectID > metas[j].ObjectID })

	return metas, nil
}
