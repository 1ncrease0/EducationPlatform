package query

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
	ListPublicCourses(ctx context.Context, limit int, offset int) ([]models.Course, error)
	CountPublicCourses(ctx context.Context) (int, error)
	ListCoursesByAuthor(ctx context.Context, authorID uuid.UUID) ([]models.Course, error)
}

type userRepo interface {
	UserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type logoRepo interface {
	GetLogoURL(ctx context.Context, objectKey string) (string, error)
}

type searchRepo interface {
	Search(ctx context.Context, query string, size int) ([]uuid.UUID, error)
	Count(ctx context.Context, query string) (int, error)
}

type subRepo interface {
	GetSubscribedCourses(ctx context.Context, userID uuid.UUID) ([]models.Course, error)
}

type CourseQueryService struct {
	log        logger.Log
	userRepo   userRepo
	courseRepo courseRepo
	logoRepo   logoRepo
	searchRepo searchRepo
	subRepo    subRepo
}

func NewCourseQueryService(log logger.Log, c courseRepo, l logoRepo, u userRepo, s searchRepo, sub subRepo) *CourseQueryService {
	return &CourseQueryService{
		log:        log,
		courseRepo: c,
		logoRepo:   l,
		userRepo:   u,
		searchRepo: s,
		subRepo:    sub,
	}
}

func (s *CourseQueryService) CourseByID(ctx context.Context, id uuid.UUID) (*models.CoursePreview, error) {
	course, err := s.courseRepo.CourseByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var logoURL string
	if course.LogoObjectKey != "" {
		logoURL, err = s.logoRepo.GetLogoURL(ctx, course.LogoObjectKey)
		if err != nil {
			s.log.ErrorErr("CourseByID: failed to get logo URL", err)
		}
	}

	author, err := s.userRepo.UserByID(ctx, course.AuthorID)
	if err != nil {
		s.log.ErrorErr("CourseByID: failed to get author", err)
		author = &models.User{Username: ""}
	}

	preview := models.CoursePreview{
		ID:          course.ID,
		Title:       course.Title,
		Description: course.Description,
		AuthorName:  author.Username,
		LogoURL:     logoURL,
		StarsCount:  course.StarsCount,
	}

	return &preview, nil
}

func (s *CourseQueryService) CoursesPreview(ctx context.Context, count int, offset int) ([]models.CoursePreview, int, error) {
	courses, err := s.courseRepo.ListPublicCourses(ctx, count, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.courseRepo.CountPublicCourses(ctx)
	if err != nil {
		return nil, 0, err
	}

	previews := make([]models.CoursePreview, 0, len(courses))
	for _, c := range courses {
		desc := c.Description
		if len(desc) > 200 {
			desc = desc[:200] + "…"
		}

		logoURL := ""
		if c.LogoObjectKey != "" {
			u, err := s.logoRepo.GetLogoURL(ctx, c.LogoObjectKey)
			if err != nil {
				s.log.ErrorErr("preview: failed to get logo URL", err)
			} else {
				logoURL = u
			}
		}

		author, err := s.userRepo.UserByID(ctx, c.AuthorID)
		if err != nil {
			s.log.ErrorErr("search preview: failed to get author by id", err)
		}

		previews = append(previews, models.CoursePreview{
			ID:          c.ID,
			Title:       c.Title,
			AuthorName:  author.Username,
			Description: c.Description,
			LogoURL:     logoURL,
			StarsCount:  c.StarsCount,
		})
	}

	return previews, total, nil
}

func (s *CourseQueryService) SearchCoursesPreview(ctx context.Context, query string, count int, offset int) ([]models.CoursePreview, int, error) {
	ids, err := s.searchRepo.Search(ctx, query, count+offset)
	if err != nil {
		return nil, 0, fmt.Errorf("search preview: elastic search failed: %w", err)
	}

	if len(ids) > offset {
		ids = ids[offset:]
	}
	if len(ids) > count {
		ids = ids[:count]
	}

	if len(ids) == 0 {
		return []models.CoursePreview{}, 0, nil
	}

	total, err := s.searchRepo.Count(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("search count failed: %w", err)
	}

	previews := make([]models.CoursePreview, 0, len(ids))
	for _, id := range ids {
		course, err := s.courseRepo.CourseByID(ctx, id)
		if err != nil {
			s.log.ErrorErr("search preview: failed to load course by id", err)
			continue
		}

		desc := course.Description
		if len(desc) > 200 {
			desc = desc[:200] + "…"
		}

		logoURL := ""
		if course.LogoObjectKey != "" {
			u, err := s.logoRepo.GetLogoURL(ctx, course.LogoObjectKey)
			if err != nil {
				s.log.ErrorErr("search preview: failed to get logo URL", err)
			} else {
				logoURL = u
			}
		}

		author, err := s.userRepo.UserByID(ctx, course.AuthorID)
		if err != nil {
			s.log.ErrorErr("search preview: failed to get author by id", err)
		}
		previews = append(previews, models.CoursePreview{
			ID:          course.ID,
			Title:       course.Title,
			Description: course.Description,
			LogoURL:     logoURL,
			AuthorName:  author.Username,
			StarsCount:  course.StarsCount,
		})
	}

	return previews, total, nil
}

func (s *CourseQueryService) GetCourseLogoURL(ctx context.Context, courseID uuid.UUID) (string, error) {
	course, err := s.courseRepo.CourseByID(ctx, courseID)
	if err != nil {
		return "", err
	}
	if course.Status != models.StatusPublic {
		return "", app_errors.ErrCourseNotPublished
	}
	if course.LogoObjectKey == "" {
		return "", app_errors.ErrImageNotFound
	}
	url, err := s.logoRepo.GetLogoURL(ctx, course.LogoObjectKey)
	if err != nil {
		s.log.ErrorErr("failed to get logo URL", err)
		return "", err
	}
	return url, nil
}

func (s *CourseQueryService) GetMyCourses(ctx context.Context, authorID uuid.UUID) ([]models.CoursePreview, error) {
	courses, err := s.courseRepo.ListCoursesByAuthor(ctx, authorID)
	if err != nil {
		return nil, err
	}

	var previews []models.CoursePreview
	for _, course := range courses {
		var logoURL string
		if course.LogoObjectKey != "" {
			logoURL, err = s.logoRepo.GetLogoURL(ctx, course.LogoObjectKey)
			if err != nil {
				s.log.ErrorErr("GetMyCourses: failed to get logo URL", err)
			}
		}

		author, err := s.userRepo.UserByID(ctx, course.AuthorID)
		if err != nil {
			s.log.ErrorErr("GetMyCourses: failed to get author", err)
			author = &models.User{Username: ""}
		}

		preview := models.CoursePreview{
			ID:          course.ID,
			Title:       course.Title,
			Description: course.Description,
			AuthorName:  author.Username,
			LogoURL:     logoURL,
			StarsCount:  course.StarsCount,
		}
		previews = append(previews, preview)
	}

	return previews, nil
}

func (s *CourseQueryService) GetSubscribedCourses(ctx context.Context, userID uuid.UUID) ([]models.CoursePreview, error) {
	courses, err := s.subRepo.GetSubscribedCourses(ctx, userID)
	if err != nil {
		return nil, err
	}

	var previews []models.CoursePreview
	for _, course := range courses {
		var logoURL string
		if course.LogoObjectKey != "" {
			logoURL, err = s.logoRepo.GetLogoURL(ctx, course.LogoObjectKey)
			if err != nil {
				s.log.ErrorErr("GetSubscribedCourses: failed to get logo URL", err)
			}
		}

		author, err := s.userRepo.UserByID(ctx, course.AuthorID)
		if err != nil {
			s.log.ErrorErr("GetSubscribedCourses: failed to get author", err)
			author = &models.User{Username: ""}
		}

		preview := models.CoursePreview{
			ID:          course.ID,
			Title:       course.Title,
			Description: course.Description,
			AuthorName:  author.Username,
			LogoURL:     logoURL,
			StarsCount:  course.StarsCount,
		}
		previews = append(previews, preview)
	}

	return previews, nil
}
