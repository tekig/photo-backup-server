package s3

import (
	"context"
	"fmt"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/tekig/photo-backup-server/internal/entity"
	"github.com/tekig/photo-backup-server/internal/repository"
)

type Storage struct {
	s      *session.Session
	bucket string
}

type StorageConfig struct {
	Endpoint     string
	AccessKey    string
	AccessSecret string
	Region       string
	Bucket       string
}

func New(c StorageConfig) (*Storage, error) {
	s, err := session.NewSession(
		aws.NewConfig().
			WithEndpoint(c.Endpoint).
			WithCredentials(credentials.NewStaticCredentials(c.AccessKey, c.AccessSecret, "")).
			WithRegion(c.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("s3 session: %w", err)
	}

	return &Storage{
		s:      s,
		bucket: c.Bucket,
	}, nil
}

func (s *Storage) Download(ctx context.Context, req repository.ObjectRequest) (*repository.ObjectResponse, error) {
	output, err := s3.New(s.s).GetObject(&s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &req.Path,
		Range:  req.Range,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return nil, fmt.Errorf("get object: %w: %w", entity.ErrNotFound, err)
			default:
				return nil, fmt.Errorf("get object: %w", err)
			}
		}
	}

	return &repository.ObjectResponse{
		ContentLength: output.ContentLength,
		ContentRange:  output.ContentRange,
		Content:       output.Body,
	}, nil
}

func (s *Storage) Upload(ctx context.Context, object repository.ObjectReader) error {
	_, err := s3manager.NewUploader(s.s).UploadWithContext(ctx, &s3manager.UploadInput{
		Body:        object.Content,
		Bucket:      &s.bucket,
		ContentType: &object.ContentType,
		Key:         &object.Path,
	})
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	return nil
}
func (s *Storage) Move(ctx context.Context, src, dst string) error {
	svc := s3.New(s.s)

	if _, err := svc.CopyObjectWithContext(ctx, &s3.CopyObjectInput{
		Bucket:     &s.bucket,
		CopySource: aws.String(path.Join(s.bucket, src)),
		Key:        &dst,
	}); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	if err := svc.WaitUntilObjectExistsWithContext(ctx, &s3.HeadObjectInput{
		Bucket: &s.bucket,
		Key:    &dst,
	}); err != nil {
		return fmt.Errorf("wait until exists: %w", err)
	}

	if _, err := svc.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &src,
	}); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}

func (s *Storage) Delete(ctx context.Context, path string) error {
	if _, err := s3.New(s.s).DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &path,
	}); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}
