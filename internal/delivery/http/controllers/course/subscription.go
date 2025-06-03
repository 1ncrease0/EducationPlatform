package course

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/delivery/http/controllers/middleware"
	"SkillForge/pkg/logger"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
)

type SubscriptionService interface {
	Subscribe(ctx context.Context, courseID, userID uuid.UUID) error
}

type SubscriptionHandler struct {
	log     logger.Log
	service SubscriptionService
}

func NewSubscriptionHandler(log logger.Log, s SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{
		log:     log,
		service: s,
	}
}

func (h *SubscriptionHandler) SubscribeCourse(c *gin.Context) {
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}

	id, ok := c.Get(middleware.ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	err = h.service.Subscribe(c.Request.Context(), courseID, userID)
	if err != nil {
		if err == app_errors.ErrAlreadySubscribed {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "subscribed"})
}
