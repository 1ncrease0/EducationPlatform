package course

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/delivery/http/controllers/middleware"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

type ManagementService interface {
	CreateCourse(ctx context.Context, course models.Course) (uuid.UUID, error)
	Publish(ctx context.Context, id uuid.UUID, authorID uuid.UUID) error
	Hide(ctx context.Context, id uuid.UUID, authorID uuid.UUID) error
	UploadCourseLogo(ctx context.Context, courseID, authorID uuid.UUID, filename string, reader io.Reader, size int64, contentType string) (string, error)
	GetCourseStatus(ctx context.Context, id uuid.UUID) (string, error)
}

type ManagementHandler struct {
	log     logger.Log
	service ManagementService
}

func NewManagementHandler(l logger.Log, s ManagementService) *ManagementHandler {
	return &ManagementHandler{
		log:     l,
		service: s,
	}
}

type newCourseRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
}

func (h *ManagementHandler) CreateCourse(c *gin.Context) {
	var input newCourseRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authorID, ok := c.Get(middleware.ClientIDCtx)
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
	id, err := h.service.CreateCourse(c.Request.Context(), course)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (h *ManagementHandler) PublishCourse(c *gin.Context) {
	id, ok := c.Params.Get("course_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_id is required"})
		return
	}
	courseID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ex := c.Get(middleware.ClientIDCtx)
	if !ex {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userUUID := userID.(uuid.UUID)
	err = h.service.Publish(c.Request.Context(), courseID, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func (h *ManagementHandler) HideCourse(c *gin.Context) {
	id, ok := c.Params.Get("course_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_id is required"})
		return
	}
	courseID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ex := c.Get(middleware.ClientIDCtx)
	if !ex {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userUUID := userID.(uuid.UUID)
	err = h.service.Hide(c.Request.Context(), courseID, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func (h *ManagementHandler) GetCourseStatus(c *gin.Context) {
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}

	status, err := h.service.GetCourseStatus(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"course_id": courseID.String(),
		"status":    status,
	})
}

func (h *ManagementHandler) UploadCourseLogo(c *gin.Context) {
	courseIDParam := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}
	authorIDValue, exists := c.Get(middleware.ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID, ok := authorIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user_id in context"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open uploaded file"})
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("CourseContent-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(fileHeader.Filename)))
	}

	url, err := h.service.UploadCourseLogo(
		c.Request.Context(),
		courseID,
		authorID,
		fileHeader.Filename,
		file,
		fileHeader.Size,
		contentType,
	)
	if err != nil {
		switch err {
		case app_errors.ErrNotCourseAuthor:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case app_errors.ErrFileSize:
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
		case app_errors.ErrNotImage:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			h.log.ErrorErr("UploadCourseLogo failed", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"url":    url,
	})
}
