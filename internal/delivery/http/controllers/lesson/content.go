package lesson

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/delivery/http/controllers/middleware"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"mime"
	"net/http"
	"path/filepath"
)

type ContentService interface {
	GetLessonDetail(ctx context.Context, lessonID uuid.UUID) (models.LessonDetail, error)
	CreateContent(ctx context.Context, content models.CourseContent, authorID uuid.UUID) (*models.CourseContent, error)
	CreateMediaContent(ctx context.Context, lessonID uuid.UUID, mediaType, filename string, file io.Reader, size int64, contentType string, authorID uuid.UUID) (*models.CourseContent, error)
	CourseContent(ctx context.Context, courseID uuid.UUID) ([]models.Contents, error)
}

type ContentHandler struct {
	log     logger.Log
	service ContentService
}

func NewContentHandler(log logger.Log, service ContentService) *ContentHandler {
	return &ContentHandler{
		log:     log,
		service: service,
	}
}

func (h *ContentHandler) GetLessonDetail(c *gin.Context) {
	lessonIDStr := c.Param("lesson_id")
	lessonID, err := uuid.Parse(lessonIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson_id"})
		return
	}

	detail, err := h.service.GetLessonDetail(c.Request.Context(), lessonID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *ContentHandler) CreateMediaContent(c *gin.Context) {
	lessonIDStr := c.PostForm("lesson_id")
	lessonID, err := uuid.Parse(lessonIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson_id"})
		return
	}
	mediaType := c.PostForm("type")
	if mediaType != models.ContentTypeImage && mediaType != models.ContentTypeVideo {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be either 'image' or 'video'"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open file"})
		return
	}
	defer file.Close()

	ct := fileHeader.Header.Get("CourseContent-Type")
	if ct == "" {
		ct = mime.TypeByExtension(filepath.Ext(fileHeader.Filename))
		if ct == "" {
			ct = "application/octet-stream"
		}
	}

	id, exists := c.Get(middleware.ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)

	content, err := h.service.CreateMediaContent(c.Request.Context(), lessonID, mediaType, fileHeader.Filename, file, fileHeader.Size, ct, authorID)
	if err != nil {
		if errors.Is(err, app_errors.ErrNotCourseAuthor) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, content)
}

func (h *ContentHandler) CourseContent(c *gin.Context) {
	courseIDStr, ok := c.Params.Get("course_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_id is required"})
		return
	}
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	content, err := h.service.CourseContent(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, content)
}

type createContentRequest struct {
	LessonID uuid.UUID `json:"lesson_id" binding:"required"`
	Type     string    `json:"type" binding:"required"` // "text", "image", "video", "quiz"
	Text     *string   `json:"text,omitempty"`
	QuizJSON *string   `json:"quiz_json,omitempty"`
}

func (h *ContentHandler) CreateContent(c *gin.Context) {
	var req createContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, ok := c.Get(middleware.ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)
	content := models.CourseContent{
		LessonID: req.LessonID,
		Type:     req.Type,
		Text:     req.Text,
		QuizJSON: req.QuizJSON,
	}

	createdContent, err := h.service.CreateContent(c.Request.Context(), content, authorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		h.log.ErrorErr("err", err)
		return
	}
	c.JSON(http.StatusCreated, createdContent)
}
