package course

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	maxLogoSizeBytes = 2 << 40
)

type userRepo interface {
	UserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}
type courseRepo interface {
	NewCourse(ctx context.Context, course *models.Course) (uuid.UUID, error)
	CourseByID(ctx context.Context, id uuid.UUID) (*models.Course, error)
	ChangeStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateCourseLogo(ctx context.Context, courseID uuid.UUID, logoObjectKey string) error
	ListPublicCourses(ctx context.Context, limit int, offset int) ([]models.Course, error)
	CountPublicCourses(ctx context.Context) (int, error)
	ListCoursesByAuthor(ctx context.Context, authorID uuid.UUID) ([]models.Course, error)
	IncrementStars(ctx context.Context, courseID uuid.UUID) error
	DecrementStars(ctx context.Context, courseID uuid.UUID) error
}

type searchRepo interface {
	Index(ctx context.Context, course models.Course) error
	Search(ctx context.Context, query string, size int) ([]uuid.UUID, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context, query string) (int, error)
}

type logoRepo interface {
	GetLogoURL(ctx context.Context, objectKey string) (string, error)
	UploadLogo(ctx context.Context, courseID uuid.UUID, filename string, reader io.Reader, size int64, contentType string) (objectKey string, err error)
	DeleteLogo(ctx context.Context, objectKey string) error
}

type lessonRepo interface {
	LessonsByCourse(ctx context.Context, courseID uuid.UUID) ([]models.Lesson, error)
}

type subscriptionRepo interface {
	SubscribeCourse(ctx context.Context, courseID, userID uuid.UUID) error
	GetSubscribedCourses(ctx context.Context, userID uuid.UUID) ([]models.Course, error)
}
type ratingRepo interface {
	RemoveRating(ctx context.Context, courseID, userID uuid.UUID) error
	AddRating(ctx context.Context, courseID, userID uuid.UUID) error
	IsRated(ctx context.Context, courseID, userID uuid.UUID) (bool, error)
}

type CourseService struct {
	log          logger.Log
	courseRepo   courseRepo
	searchRepo   searchRepo
	logoRepo     logoRepo
	lessonRepo   lessonRepo
	userRepo     userRepo
	subscription subscriptionRepo
	ratingRepo   ratingRepo
}

func NewCourseService(log logger.Log, courseRepo courseRepo, searchRepo searchRepo,
	logoRepo logoRepo, lessonRepo lessonRepo,
	useRepo userRepo, subRepo subscriptionRepo,
	ratingRepo ratingRepo,

) *CourseService {
	return &CourseService{
		log:          log,
		courseRepo:   courseRepo,
		searchRepo:   searchRepo,
		logoRepo:     logoRepo,
		lessonRepo:   lessonRepo,
		userRepo:     useRepo,
		subscription: subRepo,
		ratingRepo:   ratingRepo,
	}
}

func (s *CourseService) RateCourse(ctx context.Context, courseID, userID uuid.UUID) error {
	courses, err := s.subscription.GetSubscribedCourses(ctx, userID)
	if err != nil {
		return err
	}

	var ok bool
	for _, c := range courses {
		if c.ID == courseID {
			ok = true
		}
	}
	if !ok {
		return fmt.Errorf("course %s not found in subscription", courseID)
	}
	if err := s.ratingRepo.AddRating(ctx, courseID, userID); err != nil {
		return err
	}
	return s.courseRepo.IncrementStars(ctx, courseID)
}

func (s *CourseService) UnrateCourse(ctx context.Context, courseID, userID uuid.UUID) error {
	courses, err := s.subscription.GetSubscribedCourses(ctx, userID)
	if err != nil {
		return err
	}

	var ok bool
	for _, c := range courses {
		if c.ID == courseID {
			ok = true
		}
	}
	if !ok {
		return fmt.Errorf("course %s not found in subscription", courseID)
	}
	if err := s.ratingRepo.RemoveRating(ctx, courseID, userID); err != nil {
		return err
	}
	return s.courseRepo.DecrementStars(ctx, courseID)
}

func (s *CourseService) CourseByID(ctx context.Context, id uuid.UUID) (*models.CoursePreview, error) {
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

func (s *CourseService) CoursesPreview(ctx context.Context, count int, offset int) ([]models.CoursePreview, int, error) {
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

func (s *CourseService) SearchCoursesPreview(ctx context.Context, query string, count int, offset int) ([]models.CoursePreview, int, error) {
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
	if err := s.courseRepo.ChangeStatus(ctx, id, models.StatusHidden); err != nil {
		return err
	}
	return nil
}

func (s *CourseService) UploadCourseLogo(
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
func (s *CourseService) GetCourseLogoURL(ctx context.Context, courseID uuid.UUID) (string, error) {
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

func (s *CourseService) GetCourseStatus(ctx context.Context, id uuid.UUID) (string, error) {
	course, err := s.courseRepo.CourseByID(ctx, id)
	if err != nil {
		return "", err
	}
	return course.Status, nil
}

func (s *CourseService) Subscribe(ctx context.Context, courseID, userID uuid.UUID) error {
	course, err := s.courseRepo.CourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course.Status != models.StatusPublic {
		return app_errors.ErrCourseNotPublished
	}

	return s.subscription.SubscribeCourse(ctx, courseID, userID)
}

func (s *CourseService) GetSubscribedCourses(ctx context.Context, userID uuid.UUID) ([]models.CoursePreview, error) {
	courses, err := s.subscription.GetSubscribedCourses(ctx, userID)
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

func (s *CourseService) GetMyCourses(ctx context.Context, authorID uuid.UUID) ([]models.CoursePreview, error) {
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

func (s *CourseService) GetRatingStatus(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]bool, error) {

	courses, err := s.subscription.GetSubscribedCourses(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]bool)
	for _, course := range courses {
		rated, err := s.ratingRepo.IsRated(ctx, course.ID, userID)
		if err != nil {
			result[course.ID] = false
		} else {
			result[course.ID] = rated
		}
	}
	return result, nil
}
