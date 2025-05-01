package minioStorage

import (
	"SkillForge/internal/config"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStorage struct {
	client  *minio.Client
	buckets map[string]config.BucketConfig
}

func NewMinioStorage(endpoint, accessKey, secretKey string, useSSL bool, buckets map[string]config.BucketConfig) (*MinioStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	for _, bc := range buckets {
		exists, err := client.BucketExists(ctx, bc.Name)
		if err != nil {
			return nil, fmt.Errorf("error checking bucket %s: %w", bc.Name, err)
		}
		if !exists {
			if err := client.MakeBucket(ctx, bc.Name, minio.MakeBucketOptions{}); err != nil {
				return nil, fmt.Errorf("error creating bucket %s: %w", bc.Name, err)
			}
		}
	}

	return &MinioStorage{client: client, buckets: buckets}, nil
}
