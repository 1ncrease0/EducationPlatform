package lesson

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
)

type lessonRepo interface {
	CreateLesson(ctx context.Context, lesson models.Lesson) (*models.Lesson, error)
	CreateModule(ctx context.Context, module models.Module) (*models.Module, error)
	GetLessonByID(ctx context.Context, lessonID uuid.UUID) (models.Lesson, error)
	GetModuleByID(ctx context.Context, moduleID uuid.UUID) (models.Module, error)
	CourseContent(ctx context.Context, courseID uuid.UUID) ([]models.Contents, error)
	DeleteLessonAndUpdateOrder(ctx context.Context, lessonID, moduleID uuid.UUID, lessonOrder int) error
	DeleteModuleAndUpdateOrder(ctx context.Context, moduleID, courseID uuid.UUID, moduleOrder int) error
	GetMaxModuleOrder(ctx context.Context, courseID uuid.UUID) (int, error)
	GetMaxLessonOrder(ctx context.Context, moduleID uuid.UUID) (int, error)
	SwapLessons(ctx context.Context, lessonID1, lessonID2 uuid.UUID) error
	SwapModules(ctx context.Context, moduleID1, moduleID2 uuid.UUID) error
	CreateContent(ctx context.Context, content models.CourseContent) (*models.CourseContent, error)
	GetLessonDetail(ctx context.Context, lessonID uuid.UUID) (models.LessonDetail, error)
	UpsertContent(ctx context.Context, content models.CourseContent) (*models.CourseContent, error)
	LessonsByModule(ctx context.Context, moduleID uuid.UUID) ([]models.Lesson, error)
}
type courseRepo interface {
	CourseByID(ctx context.Context, id uuid.UUID) (*models.Course, error)
}

type mediaStorage interface {
	UploadPhoto(ctx context.Context, courseID uuid.UUID, filename string, reader io.Reader, size int64, contentType string) (objectKey string, err error)
	UploadVideo(ctx context.Context, courseID uuid.UUID, filename string, reader io.Reader, size int64, contentType string) (objectKey string, err error)
	GetPhotoURL(ctx context.Context, objectKey string) (string, error)
	GetVideoURL(ctx context.Context, objectKey string) (string, error)
	DeleteVideo(ctx context.Context, objectKey string) error
	DeletePhoto(ctx context.Context, objectKey string) error
}

type LessonService struct {
	log          logger.Log
	lessonRepo   lessonRepo
	courseRepo   courseRepo
	mediaStorage mediaStorage
}

func NewLessonService(l logger.Log, lessonRepo lessonRepo, courseRepo courseRepo, mediaStorage mediaStorage) *LessonService {
	return &LessonService{
		log:          l,
		lessonRepo:   lessonRepo,
		courseRepo:   courseRepo,
		mediaStorage: mediaStorage,
	}
}

func (s *LessonService) CreateContent(ctx context.Context, content models.CourseContent, authorID uuid.UUID) (*models.CourseContent, error) {
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

func (s *LessonService) CreateMediaContent(ctx context.Context, lessonID uuid.UUID, mediaType, filename string, file io.Reader, size int64, contentType string, authorID uuid.UUID) (*models.CourseContent, error) {
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
func (s *LessonService) SwapLessons(ctx context.Context, lessonID1, lessonID2, authorID uuid.UUID) error {
	lesson1, err := s.lessonRepo.GetLessonByID(ctx, lessonID1)
	if err != nil {
		return err
	}
	lesson2, err := s.lessonRepo.GetLessonByID(ctx, lessonID2)
	if err != nil {
		return err
	}
	if lesson1.ModuleID != lesson2.ModuleID {
		return fmt.Errorf("lessons belong to different modules")
	}

	course, err := s.courseRepo.CourseByID(ctx, lesson1.CourseID)
	if err != nil {
		return err
	}
	if course.AuthorID != authorID {
		return app_errors.ErrNotCourseAuthor
	}

	return s.lessonRepo.SwapLessons(ctx, lessonID1, lessonID2)
}

func (s *LessonService) SwapModules(ctx context.Context, moduleID1, moduleID2, courseID, authorID uuid.UUID) error {
	module1, err := s.lessonRepo.GetModuleByID(ctx, moduleID1)
	if err != nil {
		return err
	}
	module2, err := s.lessonRepo.GetModuleByID(ctx, moduleID2)
	if err != nil {
		return err
	}
	if module1.CourseID != module2.CourseID {
		return fmt.Errorf("modules belong to different courses")
	}
	if module1.CourseID != courseID {
		return fmt.Errorf("courseID mismatch")
	}
	course, err := s.courseRepo.CourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course.AuthorID != authorID {
		return app_errors.ErrNotCourseAuthor
	}

	return s.lessonRepo.SwapModules(ctx, moduleID1, moduleID2)
}

func (s *LessonService) CreateLesson(ctx context.Context, lesson models.Lesson, authorID uuid.UUID) (*models.Lesson, error) {
	course, err := s.courseRepo.CourseByID(ctx, lesson.CourseID)
	if err != nil {
		return nil, err
	}
	if course.AuthorID != authorID {
		return nil, app_errors.ErrNotCourseAuthor
	}

	maxOrder, err := s.lessonRepo.GetMaxLessonOrder(ctx, lesson.ModuleID)
	if err != nil {
		return nil, err
	}
	lesson.LessonOrder = maxOrder + 1

	l, err := s.lessonRepo.CreateLesson(ctx, lesson)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (s *LessonService) CreateModule(ctx context.Context, module models.Module, authorID uuid.UUID) (*models.Module, error) {
	course, err := s.courseRepo.CourseByID(ctx, module.CourseID)
	if err != nil {
		return nil, err
	}
	if course.AuthorID != authorID {
		return nil, app_errors.ErrNotCourseAuthor
	}

	maxOrder, err := s.lessonRepo.GetMaxModuleOrder(ctx, module.CourseID)
	if err != nil {
		return nil, err
	}
	module.Order = maxOrder + 1

	m, err := s.lessonRepo.CreateModule(ctx, module)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *LessonService) CourseContent(ctx context.Context, courseID uuid.UUID) ([]models.Contents, error) {
	_, err := s.courseRepo.CourseByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	return s.lessonRepo.CourseContent(ctx, courseID)
}

func (s *LessonService) DeleteLesson(ctx context.Context, courseID, lessonID, moduleID, authorID uuid.UUID) error {
	course, err := s.courseRepo.CourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course.AuthorID != authorID {
		return app_errors.ErrNotCourseAuthor
	}

	detail, err := s.lessonRepo.GetLessonDetail(ctx, lessonID)
	if err != nil {
		return err
	}

	for _, content := range detail.Contents {
		if content.ObjectKey != nil {
			switch content.Type {
			case models.ContentTypeImage:
				if err := s.mediaStorage.DeletePhoto(ctx, *content.ObjectKey); err != nil {
					s.log.Error("failed to delete image from minio", err)
				}
			case models.ContentTypeVideo:
				if err := s.mediaStorage.DeleteVideo(ctx, *content.ObjectKey); err != nil {
					s.log.Error("failed to delete video from minio", err)
				}
			}
		}
	}

	return s.lessonRepo.DeleteLessonAndUpdateOrder(ctx, lessonID, moduleID, detail.Lesson.LessonOrder)
}

func (s *LessonService) DeleteModule(ctx context.Context, courseID, moduleID, authorID uuid.UUID) error {
	course, err := s.courseRepo.CourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course.AuthorID != authorID {
		return app_errors.ErrNotCourseAuthor
	}

	module, err := s.lessonRepo.GetModuleByID(ctx, moduleID)
	if err != nil {
		return err
	}

	lessons, err := s.lessonRepo.LessonsByModule(ctx, moduleID)
	if err != nil {
		return err
	}
	for _, lesson := range lessons {
		detail, err := s.lessonRepo.GetLessonDetail(ctx, lesson.ID)
		if err != nil {
			s.log.Error("failed to get lesson detail", err)
			continue
		}
		for _, content := range detail.Contents {
			if content.ObjectKey != nil {
				switch content.Type {
				case models.ContentTypeImage:
					if err := s.mediaStorage.DeletePhoto(ctx, *content.ObjectKey); err != nil {
						s.log.Error("failed to delete image from MinIO", err)
					}
				case models.ContentTypeVideo:
					if err := s.mediaStorage.DeleteVideo(ctx, *content.ObjectKey); err != nil {
						s.log.Error("failed to delete video from MinIO", err)
					}
				}
			}
		}
	}

	return s.lessonRepo.DeleteModuleAndUpdateOrder(ctx, moduleID, courseID, module.Order)
}

func (s *LessonService) GetLessonDetail(ctx context.Context, lessonID uuid.UUID) (models.LessonDetail, error) {
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
