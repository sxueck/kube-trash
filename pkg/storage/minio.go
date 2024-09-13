package storage

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sxueck/kube-record/config"
)

type MinioStorage struct {
	client *minio.Client
	bucket string
}

func NewMinioStorage(cfg config.MinioConfig) (*MinioStorage, error) {
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: false, // Set to true if HTTPS is used
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %v", err)
	}

	return &MinioStorage{
		client: minioClient,
		bucket: cfg.Bucket,
	}, nil
}

func (s *MinioStorage) Upload(ctx context.Context, objectName string, reader io.Reader, objectSize int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, objectSize, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return fmt.Errorf("failed to upload object: %v", err)
	}

	log.Printf("Successfully uploaded %s of size %d\n", objectName, objectSize)
	return nil
}

func (s *MinioStorage) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, s.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %v", err)
	}

	return object, nil
}

func (s *MinioStorage) Delete(ctx context.Context, objectName string) error {
	err := s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %v", err)
	}

	log.Printf("Successfully deleted object: %s\n", objectName)
	return nil
}
