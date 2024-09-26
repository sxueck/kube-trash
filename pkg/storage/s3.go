package storage

import (
	"context"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sxueck/kube-trash/config"
)

type S3Storage struct {
	client *minio.Client
	bucket string
}

func NewS3Storage(cfg config.S3Config) (*S3Storage, error) {
	minioClient, err := minio.New(fmt.Sprintf("%s:%d", cfg.Endpoint, cfg.Port), &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: false, // Set to true if HTTPS is used
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %v", err)
	}

	return &S3Storage{
		client: minioClient,
		bucket: cfg.Bucket,
	}, nil
}

func (s *S3Storage) Upload(ctx context.Context, objectName string, reader io.Reader, objectSize int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, objectSize, minio.PutObjectOptions{ContentType: "text/plain"})
	if err != nil {
		return fmt.Errorf("failed to upload object: %v", err)
	}

	log.Infof("Successfully uploaded %s of size %d\n", objectName, objectSize)
	return nil
}

func (s *S3Storage) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, s.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %v", err)
	}

	return object, nil
}

func (s *S3Storage) Delete(ctx context.Context, objectName string) error {
	err := s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %v", err)
	}

	log.Infof("Successfully deleted object: %s\n", objectName)
	return nil
}
