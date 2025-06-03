package management

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"github.com/google/uuid"
	"io"
	"mime"
	"path/filepath"
	"strings"
)

const (
	maxLogoSizeBytes = 2 << 40
)

type userRepo interface {
	UserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type logoRepo interface {
	GetLogoURL(ctx context.Context, objectKey string) (string, error)
	UploadLogo(ctx context.Context, courseID uuid.UUID, filename string, reader io.Reader, size int64, contentType string) (objectKey string, err error)
	DeleteLogo(ctx context.Context, objectKey string) error
}

type courseRepo interface {
	NewCourse(ctx context.Context, course *models.Course) (uuid.UUID, error)
	CourseByID(ctx context.Context, id uuid.UUID) (*models.Course, error)
	ChangeStatus(ctx context.Context, id uuid.UUID, status string) error
	ListCoursesByAuthor(ctx context.Context, authorID uuid.UUID) ([]models.Course, error)
	UpdateCourseLogo(ctx context.Context, courseID uuid.UUID, logoObjectKey string) error
}

type searchRepo interface {
	Index(ctx context.Context, course models.Course) error
	Search(ctx context.Context, query string, size int) ([]uuid.UUID, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context, query string) (int, error)
}

type CourseManagementService struct {
	log        logger.Log
	userRepo   userRepo
	courseRepo courseRepo
	searchRepo searchRepo
	logoRepo   logoRepo
}

func NewCourseManagementService(log logger.Log, u userRepo, c courseRepo, s searchRepo, l logoRepo) *CourseManagementService {
	return &CourseManagementService{
		log:        log,
		userRepo:   u,
		courseRepo: c,
		searchRepo: s,
		logoRepo:   l,
	}
}

func (s *CourseManagementService) CreateCourse(ctx context.Context, course models.Course) (uuid.UUID, error) {
	id, err := s.courseRepo.NewCourse(ctx, &course)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (s *CourseManagementService) Publish(ctx context.Context, id uuid.UUID, authorID uuid.UUID) error {
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

func (s *CourseManagementService) Hide(ctx context.Context, id uuid.UUID, authorID uuid.UUID) error {
	course, err := s.courseRepo.CourseByID(ctx, id)
	if err != nil {
		return err
	}
	if authorID != course.AuthorID {
		return app_errors.ErrNotCourseAuthor
	}
	if err := s.courseRepo.ChangeStatus(ctx, id, models.StatusHidden); err != nil {
		return err
	}
	return nil
}

func (s *CourseManagementService) GetCourseStatus(ctx context.Context, id uuid.UUID) (string, error) {
	course, err := s.courseRepo.CourseByID(ctx, id)
	if err != nil {
		return "", err
	}
	return course.Status, nil
}

func (s *CourseManagementService) UploadCourseLogo(
	ctx context.Context,
	courseID, authorID uuid.UUID,
	filename string,
	reader io.Reader,
	size int64,
	contentType string,
) (string, error) {
	course, err := s.courseRepo.CourseByID(ctx, courseID)
	if err != nil {
		return "", err
	}
	if course.AuthorID != authorID {
		return "", app_errors.ErrNotCourseAuthor
	}

	if size > maxLogoSizeBytes {
		return "", app_errors.ErrFileSize
	}

	if contentType == "" {
		contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	}
	if !strings.HasPrefix(contentType, "image/") {
		return "", app_errors.ErrNotImage
	}

	if course.LogoObjectKey != "" {
		if err := s.logoRepo.DeleteLogo(ctx, course.LogoObjectKey); err != nil {
			s.log.ErrorErr("failed to delete previous logo", err)
		}
	}

	objectKey, err := s.logoRepo.UploadLogo(ctx, courseID, filename, reader, size, contentType)
	if err != nil {
		s.log.ErrorErr("failed to upload logo to storage", err)
		return "", err
	}

	if err = s.courseRepo.UpdateCourseLogo(ctx, courseID, objectKey); err != nil {
		s.log.ErrorErr("failed to save logo key to db", err)
		return "", err
	}
	url, err := s.logoRepo.GetLogoURL(ctx, objectKey)
	if err != nil {
		s.log.ErrorErr("failed to get presigned URL", err)
		return "", err
	}

	return url, nil
}
