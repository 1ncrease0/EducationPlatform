package course

import (
	"SkillForge/internal/delivery/http/controllers/middleware"
	"SkillForge/pkg/logger"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
)

type RatingService interface {
	RateCourse(ctx context.Context, courseID, userID uuid.UUID) error
	GetRatingStatus(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]bool, error)
	UnrateCourse(ctx context.Context, courseID, userID uuid.UUID) error
}

type RatingHandler struct {
	log     logger.Log
	service RatingService
}

func NewRatingHandler(log logger.Log, s RatingService) *RatingHandler {
	return &RatingHandler{
		log:     log,
		service: s,
	}
}

func (h *RatingHandler) RateCourse(c *gin.Context) {
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}
	id, exists := c.Get(middleware.ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	if err := h.service.RateCourse(c.Request.Context(), courseID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "rated"})
}

func (h *RatingHandler) UnrateCourse(c *gin.Context) {
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}
	id, exists := c.Get(middleware.ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	if err := h.service.UnrateCourse(c.Request.Context(), courseID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "unrated"})
}

func (h *RatingHandler) GetRatingStatus(c *gin.Context) {
	id, ok := c.Get(middleware.ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	statusMap, err := h.service.GetRatingStatus(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rated_status": statusMap})
}
