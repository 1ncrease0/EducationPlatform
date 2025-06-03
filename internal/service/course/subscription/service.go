package subscription

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"github.com/google/uuid"
)

type courseRepo interface {
	CourseByID(ctx context.Context, id uuid.UUID) (*models.Course, error)
}

type subscriptionRepo interface {
	SubscribeCourse(ctx context.Context, courseID, userID uuid.UUID) error
}

type CourseSubscriptionService struct {
	log        logger.Log
	courseRepo courseRepo
	subRepo    subscriptionRepo
}

func NewCourseSubscriptionService(l logger.Log, c courseRepo, s subscriptionRepo) *CourseSubscriptionService {
	return &CourseSubscriptionService{
		log:        l,
		courseRepo: c,
		subRepo:    s,
	}
}

func (s *CourseSubscriptionService) Subscribe(ctx context.Context, courseID, userID uuid.UUID) error {
	course, err := s.courseRepo.CourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course.Status != models.StatusPublic {
		return app_errors.ErrCourseNotPublished
	}

	return s.subRepo.SubscribeCourse(ctx, courseID, userID)
}
