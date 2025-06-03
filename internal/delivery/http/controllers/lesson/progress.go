package lesson

import (
	"SkillForge/internal/delivery/http/controllers/middleware"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
)

type ProgressService interface {
	SubmitQuizAnswers(ctx context.Context, lessonID uuid.UUID, userID uuid.UUID, answers []models.QuizAnswer) (float64, error)
	GetQuizResult(ctx context.Context, lessonID, userID uuid.UUID) (float64, string, error)
}

type ProgressHandler struct {
	log     logger.Log
	service ProgressService
}

func NewProgressHandler(log logger.Log, service ProgressService) *ProgressHandler {
	return &ProgressHandler{log, service}

}

type submitQuizRequest struct {
	Answers []models.QuizAnswer `json:"answers" binding:"required"`
}

func (h *ProgressHandler) SubmitQuiz(c *gin.Context) {
	lessonIDStr := c.Param("lesson_id")
	lessonID, err := uuid.Parse(lessonIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson_id"})
		return
	}

	var req submitQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, exists := c.Get(middleware.ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	score, err := h.service.SubmitQuizAnswers(c.Request.Context(), lessonID, userID, req.Answers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"score": score,
	})
}

func (h *ProgressHandler) GetQuizResult(c *gin.Context) {
	lessonIDStr := c.Param("lesson_id")
	lessonID, err := uuid.Parse(lessonIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson_id"})
		return
	}

	id, exists := c.Get(middleware.ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	score, status, err := h.service.GetQuizResult(c.Request.Context(), lessonID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"score":  score,
		"status": status,
	})
}
