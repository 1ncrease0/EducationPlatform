package main

import (
	"SkillForge/internal/app"
	"SkillForge/internal/config"
	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	cfg := config.MustLoad()
	app.Run(cfg)

}
