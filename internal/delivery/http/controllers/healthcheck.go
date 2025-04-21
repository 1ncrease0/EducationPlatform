package controllers

import (
	"github.com/gin-gonic/gin"
)

type StatusHandler struct {
}

func NewStatusHandler() *StatusHandler {
	return &StatusHandler{}
}

func (h *StatusHandler) Status(c *gin.Context) {
	c.JSON(200, gin.H{"status": "Available"})
}
