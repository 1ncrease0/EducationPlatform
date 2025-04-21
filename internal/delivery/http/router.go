package http

import (
	"SkillForge/internal/delivery/http/controllers"
	"SkillForge/internal/service"
	"SkillForge/pkg/logger"
	"github.com/gin-gonic/gin"
)

func InitRoutes(l logger.Log, u service.Collection) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	statusController := controllers.NewStatusHandler()
	authController := controllers.NewAuthHandler(l, u.AuthService)

	v1 := r.Group("/v1", controllers.LoggingMiddleware(l))
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authController.Login)
			auth.POST("/register", authController.Register)
			auth.POST("/refresh", authController.Refresh)
		}

		status := v1.Group("/status", authController.AuthMiddleware)
		{
			status.GET("/1", statusController.Status)
		}

	}
	return r
}
