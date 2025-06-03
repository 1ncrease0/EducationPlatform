package http

import (
	"SkillForge/internal/delivery/http/controllers/auth"
	"SkillForge/internal/delivery/http/controllers/course"
	"SkillForge/internal/delivery/http/controllers/lesson"
	"SkillForge/internal/delivery/http/controllers/middleware"
	"SkillForge/internal/delivery/http/controllers/status"
	"SkillForge/internal/models"
	"SkillForge/internal/service"
	"SkillForge/pkg/logger"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func InitRoutes(l logger.Log, u service.Collection) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	config := cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	r.Use(cors.New(config))

	statusHandler := status.NewStatusHandler()
	authHandler := auth.NewAuthHandler(l, u.AuthService)

	authMiddlewareProvider := middleware.NewAuthMiddlewareProvider(l, u.AuthService)

	courseManagementHandler := course.NewManagementHandler(l, u.CourseManagementService)
	courseQueryHandler := course.NewQueryHandler(l, u.CourseQueryService)
	courseSubscriptionHandler := course.NewSubscriptionHandler(l, u.CourseSubscriptionService)
	courseRatingHandler := course.NewRatingHandler(l, u.CourseRatingService)

	lessonManagementHandler := lesson.NewManagementHandler(l, u.LessonManagementService)
	lessonProgressHandler := lesson.NewProgressHandler(l, u.LessonProgressService)
	lessonContentHandler := lesson.NewContentHandler(l, u.LessonContentService)

	v1 := r.Group("/v1", middleware.LoggingMiddleware(l))
	{
		v1.GET("/status", statusHandler.Status)

		v1.GET("/me", authMiddlewareProvider.AuthMiddleware, authHandler.Me)

		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/refresh", authHandler.Refresh)
		}

		courses := v1.Group("/courses")
		{
			courses.GET("", courseQueryHandler.ListCoursePreview)
			courses.GET("/:course_id/preview", courseQueryHandler.CourseByID)
			courses.GET("/:course_id/content", lessonContentHandler.CourseContent)
			courses.GET("/:course_id/status", courseManagementHandler.GetCourseStatus)

			author := courses.Group("", authMiddlewareProvider.AuthMiddleware, middleware.RequireRoles(models.AuthorRole))
			{
				author.PUT("/:course_id/logo", courseManagementHandler.UploadCourseLogo)
				author.POST("", courseManagementHandler.CreateCourse)
				author.PATCH("/:course_id/publish", courseManagementHandler.PublishCourse)
				author.PATCH("/:course_id/hide", courseManagementHandler.HideCourse)
				author.POST("/:course_id/create-lesson", lessonManagementHandler.CreateLesson)
				author.POST("/:course_id/create-module", lessonManagementHandler.CreateModule)
				author.DELETE("/:course_id/module/:module_id/lesson/:lesson_id", lessonManagementHandler.DeleteLesson)
				author.DELETE("/:course_id/module/:module_id", lessonManagementHandler.DeleteModule)
				author.GET("/my-courses", courseQueryHandler.GetMyCourses)
				author.PATCH("/:course_id/lessons/swap", lessonManagementHandler.SwapLessons)
				author.PATCH("/:course_id/modules/swap", lessonManagementHandler.SwapModules)
				author.POST("/:course_id/lesson/content", lessonContentHandler.CreateContent)
				author.POST("/:course_id/lesson/content/media", lessonContentHandler.CreateMediaContent)
				author.GET("/:course_id/lessons/:lesson_id", lessonContentHandler.GetLessonDetail)
			}

			client := courses.Group("", authMiddlewareProvider.AuthMiddleware, middleware.RequireRoles(models.ClientRole))
			{
				client.POST("/:course_id/subscribe", courseSubscriptionHandler.SubscribeCourse)
				client.GET("/subscriptions", courseQueryHandler.GetSubscribedCourses)
				client.GET("/lessons/:lesson_id", lessonContentHandler.GetLessonDetail)
				client.POST("/lessons/:lesson_id/quiz/submit", lessonProgressHandler.SubmitQuiz)
				client.GET("/lessons/:lesson_id/quiz/result", lessonProgressHandler.GetQuizResult)
				client.POST("/:course_id/star", courseRatingHandler.RateCourse)
				client.DELETE("/:course_id/star", courseRatingHandler.UnrateCourse)
				client.GET("/rated-status", courseRatingHandler.GetRatingStatus)
			}

		}

	}
	return r
}
