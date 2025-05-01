package controllers

import (
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
)

type CourseService interface {
	CreateCourse(ctx context.Context, course models.Course) (uuid.UUID, error)
	Publish(ctx context.Context, id uuid.UUID, authorID uuid.UUID) error
	Hide(ctx context.Context, id uuid.UUID, authorID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID, authorID uuid.UUID) error
}

type CourseHandler struct {
	CourseService CourseService
	log           logger.Log
}

func NewCourseHandler(l logger.Log, courseService CourseService) *CourseHandler {
	return &CourseHandler{
		CourseService: courseService,
		log:           l,
	}
}

type newCourseRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
}

func (h *CourseHandler) NewCourse(c *gin.Context) {
	var input newCourseRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authorID, ok := c.Get(ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	course := models.Course{
		Title:       input.Title,
		Description: input.Description,
		AuthorID:    authorID.(uuid.UUID),
		Status:      models.StatusHidden,
	}
	id, err := h.CourseService.CreateCourse(c.Request.Context(), course)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (h *CourseHandler) PublishCourse(c *gin.Context) {
	id, ok := c.Params.Get("course_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_id is required"})
	}
	courseID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	userID, ex := c.Get(ClientIDCtx)
	if !ex {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
	userUUID := userID.(uuid.UUID)
	err = h.CourseService.Publish(c.Request.Context(), courseID, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	c.JSON(http.StatusOK, gin.H{})
}

func (h *CourseHandler) HideCourse(c *gin.Context) {
	id, ok := c.Params.Get("course_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_id is required"})
	}
	courseID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	userID, ex := c.Get(ClientIDCtx)
	if !ex {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
	userUUID := userID.(uuid.UUID)
	err = h.CourseService.Hide(c.Request.Context(), courseID, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	c.JSON(http.StatusOK, gin.H{})
}

func (h *CourseHandler) DeleteCourse(c *gin.Context) {
	id, ok := c.Params.Get("course_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_id is required"})
	}
	courseID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	userID, ex := c.Get(ClientIDCtx)
	if !ex {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
	userUUID := userID.(uuid.UUID)
	err = h.CourseService.Delete(c.Request.Context(), courseID, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	c.JSON(http.StatusOK, gin.H{})
}
