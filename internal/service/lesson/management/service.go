package management

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"fmt"
	"github.com/google/uuid"
)

type courseRepo interface {
	CourseByID(ctx context.Context, id uuid.UUID) (*models.Course, error)
}

type lessonRepo interface {
	CreateLesson(ctx context.Context, lesson models.Lesson) (*models.Lesson, error)
	CreateModule(ctx context.Context, module models.Module) (*models.Module, error)
	GetLessonByID(ctx context.Context, lessonID uuid.UUID) (models.Lesson, error)
	GetModuleByID(ctx context.Context, moduleID uuid.UUID) (models.Module, error)
	SwapLessons(ctx context.Context, lessonID1, lessonID2 uuid.UUID) error
	SwapModules(ctx context.Context, moduleID1, moduleID2 uuid.UUID) error
	GetMaxModuleOrder(ctx context.Context, courseID uuid.UUID) (int, error)
	GetMaxLessonOrder(ctx context.Context, moduleID uuid.UUID) (int, error)
	GetLessonDetail(ctx context.Context, lessonID uuid.UUID) (models.LessonDetail, error)
	DeleteLessonAndUpdateOrder(ctx context.Context, lessonID, moduleID uuid.UUID, lessonOrder int) error
	DeleteModuleAndUpdateOrder(ctx context.Context, moduleID, courseID uuid.UUID, moduleOrder int) error
	LessonsByModule(ctx context.Context, moduleID uuid.UUID) ([]models.Lesson, error)
}

type mediaStorage interface {
	DeleteVideo(ctx context.Context, objectKey string) error
	DeletePhoto(ctx context.Context, objectKey string) error
}

type LessonManagementService struct {
	log          logger.Log
	courseRepo   courseRepo
	lessonRepo   lessonRepo
	mediaStorage mediaStorage
}

func NewLessonManagementService(log logger.Log, c courseRepo, l lessonRepo, m mediaStorage) *LessonManagementService {
	return &LessonManagementService{
		log:          log,
		courseRepo:   c,
		lessonRepo:   l,
		mediaStorage: m,
	}
}

func (s *LessonManagementService) SwapLessons(ctx context.Context, lessonID1, lessonID2, authorID uuid.UUID) error {
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

func (s *LessonManagementService) SwapModules(ctx context.Context, moduleID1, moduleID2, courseID, authorID uuid.UUID) error {
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

func (s *LessonManagementService) CreateLesson(ctx context.Context, lesson models.Lesson, authorID uuid.UUID) (*models.Lesson, error) {
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

func (s *LessonManagementService) CreateModule(ctx context.Context, module models.Module, authorID uuid.UUID) (*models.Module, error) {
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

func (s *LessonManagementService) DeleteLesson(ctx context.Context, courseID, lessonID, moduleID, authorID uuid.UUID) error {
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

func (s *LessonManagementService) DeleteModule(ctx context.Context, courseID, moduleID, authorID uuid.UUID) error {
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
