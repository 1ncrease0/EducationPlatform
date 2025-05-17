package service

import (
	"SkillForge/internal/service/auth"
	"SkillForge/internal/service/course"
	"SkillForge/internal/service/lesson"
)

type Collection struct {
	*auth.AuthService
	*course.CourseService
	*lesson.LessonService
}
