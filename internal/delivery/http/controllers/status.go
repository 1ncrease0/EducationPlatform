package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type StatusHandler struct {
}

func NewStatusHandler() *StatusHandler {
	return &StatusHandler{}
}

func (h *StatusHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "Available"})
}
