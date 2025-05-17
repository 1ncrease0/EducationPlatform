package minio_storage

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"io"
	"mime"
	"net/url"
	"path/filepath"
	"time"
)

type LessonStorage struct {
	storage      *MinioStorage
	bucket       string
	presignedTTL time.Duration
}

func NewLessonStorage(storage *MinioStorage, bucketName string, presignedTTL time.Duration) (*LessonStorage, error) {
	exists, err := storage.client.BucketExists(context.Background(), bucketName)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err = storage.client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}
	return &LessonStorage{storage: storage, bucket: bucketName, presignedTTL: presignedTTL}, nil
}

func (s *LessonStorage) UploadPhoto(
	ctx context.Context,
	courseID uuid.UUID,
	filename string,
	reader io.Reader,
	size int64,
	contentType string,
) (objectKey string, err error) {
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".bin"
	}

	objectKey = fmt.Sprintf("lessons/%s/photo%s", courseID.String(), ext)

	if contentType == "" {
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	_, err = s.storage.client.PutObject(
		ctx,
		s.bucket,
		objectKey,
		reader,
		size,
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", err
	}
	return objectKey, nil
}

func (s *LessonStorage) GetPhotoURL(ctx context.Context, objectKey string) (string, error) {
	reqParams := make(url.Values)
	url, err := s.storage.client.PresignedGetObject(
		ctx,
		s.bucket,
		objectKey,
		s.presignedTTL,
		reqParams,
	)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (s *LessonStorage) DeletePhoto(ctx context.Context, objectKey string) error {
	return s.storage.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

func (s *LessonStorage) UploadVideo(
	ctx context.Context,
	courseID uuid.UUID,
	filename string,
	reader io.Reader,
	size int64,
	contentType string,
) (objectKey string, err error) {
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".bin"
	}

	objectKey = fmt.Sprintf("lessons/%s/video%s", courseID.String(), ext)

	if contentType == "" {
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	_, err = s.storage.client.PutObject(
		ctx,
		s.bucket,
		objectKey,
		reader,
		size,
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", err
	}
	return objectKey, nil
}

func (s *LessonStorage) GetVideoURL(ctx context.Context, objectKey string) (string, error) {
	reqParams := make(url.Values)
	url, err := s.storage.client.PresignedGetObject(
		ctx,
		s.bucket,
		objectKey,
		s.presignedTTL,
		reqParams,
	)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (s *LessonStorage) DeleteVideo(ctx context.Context, objectKey string) error {
	return s.storage.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}
