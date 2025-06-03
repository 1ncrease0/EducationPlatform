package rating

import (
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"fmt"
	"github.com/google/uuid"
)

type subscriptionRepo interface {
	SubscribeCourse(ctx context.Context, courseID, userID uuid.UUID) error
	GetSubscribedCourses(ctx context.Context, userID uuid.UUID) ([]models.Course, error)
}

type ratingRepo interface {
	RemoveRating(ctx context.Context, courseID, userID uuid.UUID) error
	AddRating(ctx context.Context, courseID, userID uuid.UUID) error
	IsRated(ctx context.Context, courseID, userID uuid.UUID) (bool, error)
}
type courseRepo interface {
	IncrementStars(ctx context.Context, courseID uuid.UUID) error
	DecrementStars(ctx context.Context, courseID uuid.UUID) error
}

type CourseRatingService struct {
	log        logger.Log
	courseRepo courseRepo
	subRepo    subscriptionRepo
	ratingRepo ratingRepo
}

func NewCourseRatingService(l logger.Log, c courseRepo, s subscriptionRepo, r ratingRepo) *CourseRatingService {
	return &CourseRatingService{
		log:        l,
		courseRepo: c,
		subRepo:    s,
		ratingRepo: r,
	}
}

func (s *CourseRatingService) RateCourse(ctx context.Context, courseID, userID uuid.UUID) error {
	courses, err := s.subRepo.GetSubscribedCourses(ctx, userID)
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

func (s *CourseRatingService) UnrateCourse(ctx context.Context, courseID, userID uuid.UUID) error {
	courses, err := s.subRepo.GetSubscribedCourses(ctx, userID)
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

func (s *CourseRatingService) GetRatingStatus(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	courses, err := s.subRepo.GetSubscribedCourses(ctx, userID)
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
