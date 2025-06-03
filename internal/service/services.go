package service

import (
	"SkillForge/internal/service/auth"
	cm "SkillForge/internal/service/course/management"
	"SkillForge/internal/service/course/query"
	"SkillForge/internal/service/course/rating"
	"SkillForge/internal/service/course/subscription"
	"SkillForge/internal/service/lesson/content"
	"SkillForge/internal/service/lesson/progress"

	lm "SkillForge/internal/service/lesson/management"
)

type Collection struct {
	*auth.AuthService

	*cm.CourseManagementService
	*rating.CourseRatingService
	*subscription.CourseSubscriptionService
	*query.CourseQueryService

	*lm.LessonManagementService
	*content.LessonContentService
	*progress.LessonProgressService
}
