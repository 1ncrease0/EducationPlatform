package content

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"fmt"
	"github.com/google/uuid"
	"io"
)

type courseRepo interface {
	CourseByID(ctx context.Context, id uuid.UUID) (*models.Course, error)
}

type lessonRepo interface {
	GetLessonDetail(ctx context.Context, lessonID uuid.UUID) (models.LessonDetail, error)
	GetLessonByID(ctx context.Context, lessonID uuid.UUID) (models.Lesson, error)
	UpsertContent(ctx context.Context, content models.CourseContent) (*models.CourseContent, error)
	CourseContent(ctx context.Context, courseID uuid.UUID) ([]models.Contents, error)
}

type mediaStorage interface {
	UploadPhoto(ctx context.Context, courseID uuid.UUID, filename string, reader io.Reader, size int64, contentType string) (objectKey string, err error)
	UploadVideo(ctx context.Context, courseID uuid.UUID, filename string, reader io.Reader, size int64, contentType string) (objectKey string, err error)
	GetPhotoURL(ctx context.Context, objectKey string) (string, error)
	GetVideoURL(ctx context.Context, objectKey string) (string, error)
}

type LessonContentService struct {
	log          logger.Log
	lessonRepo   lessonRepo
	mediaStorage mediaStorage
	courseRepo   courseRepo
}

func NewLessonContentService(log logger.Log, l lessonRepo, m mediaStorage, c courseRepo) *LessonContentService {
	return &LessonContentService{
		log:          log,
		lessonRepo:   l,
		mediaStorage: m,
		courseRepo:   c,
	}
}

func (s *LessonContentService) GetLessonDetail(ctx context.Context, lessonID uuid.UUID) (models.LessonDetail, error) {
	detail, err := s.lessonRepo.GetLessonDetail(ctx, lessonID)
	if err != nil {
		return detail, err
	}
	for i := range detail.Contents {
		if detail.Contents[i].ObjectKey != nil {
			switch detail.Contents[i].Type {
			case models.ContentTypeImage:
				url, err := s.mediaStorage.GetPhotoURL(ctx, *detail.Contents[i].ObjectKey)
				if err == nil {
					detail.Contents[i].ObjectKey = &url
				}
			case models.ContentTypeVideo:
				url, err := s.mediaStorage.GetVideoURL(ctx, *detail.Contents[i].ObjectKey)
				if err == nil {
					detail.Contents[i].ObjectKey = &url
				}
			}
		}
	}
	return detail, nil
}

func (s *LessonContentService) CreateContent(ctx context.Context, content models.CourseContent, authorID uuid.UUID) (*models.CourseContent, error) {
	lesson, err := s.lessonRepo.GetLessonByID(ctx, content.LessonID)
	if err != nil {
		return nil, err
	}
	course, err := s.courseRepo.CourseByID(ctx, lesson.CourseID)
	if err != nil {
		return nil, err
	}
	if course.AuthorID != authorID {
		return nil, app_errors.ErrNotCourseAuthor
	}
	return s.lessonRepo.UpsertContent(ctx, content)
}

func (s *LessonContentService) CreateMediaContent(ctx context.Context, lessonID uuid.UUID, mediaType, filename string, file io.Reader, size int64, contentType string, authorID uuid.UUID) (*models.CourseContent, error) {
	lesson, err := s.lessonRepo.GetLessonByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	course, err := s.courseRepo.CourseByID(ctx, lesson.CourseID)
	if err != nil {
		return nil, err
	}
	if course.AuthorID != authorID {
		return nil, app_errors.ErrNotCourseAuthor
	}

	var objectKey string
	switch mediaType {
	case models.ContentTypeImage:
		objectKey, err = s.mediaStorage.UploadPhoto(ctx, lesson.CourseID, filename, file, size, contentType)
	case models.ContentTypeVideo:
		objectKey, err = s.mediaStorage.UploadVideo(ctx, lesson.CourseID, filename, file, size, contentType)
	default:
		return nil, fmt.Errorf("unsupported media type")
	}
	if err != nil {
		return nil, err
	}

	content := models.CourseContent{
		LessonID:  lessonID,
		Type:      mediaType,
		ObjectKey: &objectKey,
	}
	return s.lessonRepo.UpsertContent(ctx, content)
}

func (s *LessonContentService) CourseContent(ctx context.Context, courseID uuid.UUID) ([]models.Contents, error) {
	_, err := s.courseRepo.CourseByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	return s.lessonRepo.CourseContent(ctx, courseID)
}
