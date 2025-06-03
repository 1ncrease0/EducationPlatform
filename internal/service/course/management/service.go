package managment

import (
	"SkillForge/internal/models"
	"github.com/google/uuid"
)

type ManagmentService struct {
}

func (s *CourseService) CreateCourse(ctx context.Context, course models.Course) (uuid.UUID, error) {
	id, err := s.courseRepo.NewCourse(ctx, &course)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
