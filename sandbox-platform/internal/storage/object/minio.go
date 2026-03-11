// Package object provides a MinIO client for storing tool artifacts.
package object

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOConfig holds connection parameters.
type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// MinIOClient wraps the minio.Client with a fixed bucket.
type MinIOClient struct {
	client *minio.Client
	bucket string
}

// NewMinIOClient creates a MinIOClient and ensures the bucket exists.
func NewMinIOClient(cfg MinIOConfig) (*MinIOClient, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio new: %w", err)
	}

	ctx := context.Background()
	exists, err := mc.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("bucket exists check: %w", err)
	}
	if !exists {
		if err := mc.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("make bucket: %w", err)
		}
	}

	return &MinIOClient{client: mc, bucket: cfg.Bucket}, nil
}

// Put uploads data to the given object key.
func (m *MinIOClient) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, m.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// Get downloads an object and returns its content.
func (m *MinIOClient) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := m.client.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %q: %w", key, err)
	}
	return obj, nil
}

// Delete removes an object.
func (m *MinIOClient) Delete(ctx context.Context, key string) error {
	return m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
}

// PresignedGetURL returns a time-limited download URL for the given key.
func (m *MinIOClient) PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	params := make(url.Values)
	u, err := m.client.PresignedGetObject(ctx, m.bucket, key, expiry, params)
	if err != nil {
		return "", fmt.Errorf("presign %q: %w", key, err)
	}
	return u.String(), nil
}
