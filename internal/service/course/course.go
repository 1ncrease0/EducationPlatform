package course

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"github.com/google/uuid"
)

type CourseRepo interface {
	NewCourse(ctx context.Context, course *models.Course) (uuid.UUID, error)
	CourseByID(ctx context.Context, id uuid.UUID) (*models.Course, error)
	ChangeStatus(ctx context.Context, id uuid.UUID, status string) error
}

type SearchRepo interface {
	Index(ctx context.Context, course models.Course) error
	Search(ctx context.Context, query string, size int) ([]uuid.UUID, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type CourseService struct {
	log        logger.Log
	courseRepo CourseRepo
	searchRepo SearchRepo
}

func NewCourseService(log logger.Log, courseRepo CourseRepo, searchRepo SearchRepo) *CourseService {
	return &CourseService{
		log:        log,
		courseRepo: courseRepo,
		searchRepo: searchRepo,
	}
}

func (s *CourseService) CourseByID(ctx context.Context, id uuid.UUID) (*models.Course, error) {
	course, err := s.courseRepo.CourseByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return course, nil
}

func (s *CourseService) CreateCourse(ctx context.Context, course models.Course) (uuid.UUID, error) {
	id, err := s.courseRepo.NewCourse(ctx, &course)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (s *CourseService) Publish(ctx context.Context, id uuid.UUID, authorID uuid.UUID) error {
	course, err := s.courseRepo.CourseByID(ctx, id)
	if err != nil {
		return err
	}
	if authorID != course.AuthorID {
		return app_errors.ErrNotCourseAuthor
	}
	err = s.courseRepo.ChangeStatus(ctx, id, models.StatusPublic)
	if err != nil {
		return err
	}

	err = s.searchRepo.Index(ctx, *course)
	if err != nil {
		s.log.ErrorErr("error indexing course", err)
		return err
	}
	return nil
}

func (s *CourseService) Hide(ctx context.Context, id uuid.UUID, authorID uuid.UUID) error {
	course, err := s.courseRepo.CourseByID(ctx, id)
	if err != nil {
		return err
	}
	if authorID != course.AuthorID {
		return app_errors.ErrNotCourseAuthor
	}
	err = s.courseRepo.ChangeStatus(ctx, id, models.StatusHidden)
	if err != nil {
		return err
	}
	err = s.searchRepo.Delete(ctx, id)
	if err != nil {
		s.log.ErrorErr("error hiding course", err)
		return err
	}
	return nil
}
