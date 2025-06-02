package http

import (
	"SkillForge/internal/delivery/http/controllers"
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

	statusController := controllers.NewStatusHandler()
	authController := controllers.NewAuthHandler(l, u.AuthService)
	courseController := controllers.NewCourseHandler(l, u.CourseService)
	lessonController := controllers.NewLessonHandler(l, u.LessonService)

	v1 := r.Group("/v1", controllers.LoggingMiddleware(l))
	{
		v1.GET("/status", statusController.Status)

		v1.GET("/me", authController.AuthMiddleware, authController.Me)

		auth := v1.Group("/auth")
		{
			auth.POST("/login", authController.Login)
			auth.POST("/register", authController.Register)
			auth.POST("/refresh", authController.Refresh)
		}

		courses := v1.Group("/courses")
		{
			courses.GET("", courseController.ListCoursePreview)
			courses.GET("/:course_id/preview", courseController.CourseByID)
			courses.GET("/:course_id/content", lessonController.CourseContent)
			courses.GET("/:course_id/status", courseController.GetCourseStatus)

			author := courses.Group("", authController.AuthMiddleware, controllers.RequireRoles(models.AuthorRole))
			{
				author.PUT("/:course_id/logo", courseController.UploadCourseLogo)
				author.POST("", courseController.CreateCourse)
				author.PATCH("/:course_id/publish", courseController.PublishCourse)
				author.PATCH("/:course_id/hide", courseController.HideCourse)
				author.POST("/:course_id/create-lesson", lessonController.CreateLesson)
				author.POST("/:course_id/create-module", lessonController.CreateModule)
				author.DELETE("/:course_id/module/:module_id/lesson/:lesson_id", lessonController.DeleteLesson)
				author.DELETE("/:course_id/module/:module_id", lessonController.DeleteModule)
				author.GET("/my-courses", courseController.GetMyCourses)
				author.PATCH("/:course_id/lessons/swap", lessonController.SwapLessons)
				author.PATCH("/:course_id/modules/swap", lessonController.SwapModules)
				author.POST("/:course_id/lesson/content", lessonController.CreateContent)
				author.POST("/:course_id/lesson/content/media", lessonController.CreateMediaContent)
				author.GET("/:course_id/lessons/:lesson_id", lessonController.GetLessonDetail)
			}

			client := courses.Group("", authController.AuthMiddleware, controllers.RequireRoles(models.ClientRole))
			{
				client.POST("/:course_id/subscribe", courseController.SubscribeCourse)
				client.GET("/subscriptions", courseController.GetSubscribedCourses)
				client.GET("/lessons/:lesson_id", lessonController.GetLessonDetail)
				client.POST("/lessons/:lesson_id/quiz/submit", lessonController.SubmitQuiz)
				client.GET("/lessons/:lesson_id/quiz/result", lessonController.GetQuizResult)
				client.POST("/:course_id/star", courseController.RateCourse)
				client.DELETE("/:course_id/star", courseController.UnrateCourse)
				client.GET("/rated-status", courseController.GetRatingStatus)
			}

		}

	}
	return r
}
