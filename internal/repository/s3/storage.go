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
	s              *session.Session
	bucket         string
	storageClasses map[repository.ObjectType]string
}

type StorageConfig struct {
	Endpoint     string
	AccessKey    string
	AccessSecret string
	Region       string
	Bucket       string
	StorageClass map[repository.ObjectType]string
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
		s:              s,
		bucket:         c.Bucket,
		storageClasses: c.StorageClass,
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
		Body:         object.Content,
		Bucket:       &s.bucket,
		ContentType:  &object.ContentType,
		Key:          &object.Path,
		StorageClass: s.storageClass(object.Type),
	})
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	return nil
}

func (s *Storage) Move(ctx context.Context, move repository.ObjectMove) error {
	svc := s3.New(s.s)

	if _, err := svc.CopyObjectWithContext(ctx, &s3.CopyObjectInput{
		Bucket:       &s.bucket,
		CopySource:   aws.String(path.Join(s.bucket, move.Source)),
		Key:          &move.Destination,
		StorageClass: s.storageClass(move.Type),
	}); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	if err := svc.WaitUntilObjectExistsWithContext(ctx, &s3.HeadObjectInput{
		Bucket: &s.bucket,
		Key:    &move.Destination,
	}); err != nil {
		return fmt.Errorf("wait until exists: %w", err)
	}

	if _, err := svc.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &move.Source,
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

func (s *Storage) storageClass(o repository.ObjectType) *string {
	v, ok := s.storageClasses[o]
	if !ok {
		return nil
	}

	return &v
}
