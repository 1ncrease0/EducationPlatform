package controllers

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CourseService interface {
	CreateCourse(ctx context.Context, course models.Course) (uuid.UUID, error)
	Publish(ctx context.Context, id uuid.UUID, authorID uuid.UUID) error
	Hide(ctx context.Context, id uuid.UUID, authorID uuid.UUID) error
	CoursesPreview(ctx context.Context, count int, offset int) ([]models.CoursePreview, int, error)
	SearchCoursesPreview(ctx context.Context, query string, count int, offset int) ([]models.CoursePreview, int, error)
	UploadCourseLogo(ctx context.Context, courseID, authorID uuid.UUID, filename string, reader io.Reader, size int64, contentType string) (string, error)
	CourseByID(ctx context.Context, id uuid.UUID) (*models.CoursePreview, error)
	Subscribe(ctx context.Context, courseID, userID uuid.UUID) error
	GetSubscribedCourses(ctx context.Context, userID uuid.UUID) ([]models.CoursePreview, error)
	GetMyCourses(ctx context.Context, authorID uuid.UUID) ([]models.CoursePreview, error)
	GetCourseStatus(ctx context.Context, courseID uuid.UUID) (string, error)
	RateCourse(ctx context.Context, courseID, userID uuid.UUID) error
	UnrateCourse(ctx context.Context, courseID, userID uuid.UUID) error
	GetRatingStatus(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]bool, error)
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

func (h *CourseHandler) CourseByID(c *gin.Context) {
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		h.log.ErrorErr("err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}

	preview, err := h.CourseService.CourseByID(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, preview)
}

func (h *CourseHandler) UploadCourseLogo(c *gin.Context) {
	courseIDParam := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}
	authorIDValue, exists := c.Get(ClientIDCtx)
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

	url, err := h.CourseService.UploadCourseLogo(
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

func (h *CourseHandler) ListCoursePreview(c *gin.Context) {
	ctx := c.Request.Context()
	limit := 10
	if s := c.Query("limit"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v <= 0 {
			h.log.ErrorErr("invalid limit parameter", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
			return
		}
		limit = v
	}

	offset := 0
	if s := c.Query("offset"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 {
			h.log.ErrorErr("invalid offset parameter", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "offset must be a non-negative integer"})
			return
		}
		offset = v
	}

	var total int
	q := c.Query("query")
	var previews []models.CoursePreview
	var err error
	if q != "" {
		previews, total, err = h.CourseService.SearchCoursesPreview(ctx, q, limit, offset)
	} else {
		previews, total, err = h.CourseService.CoursesPreview(ctx, limit, offset)
	}
	if err != nil {
		h.log.ErrorErr("ListCourses failed", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch courses"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"total":   total,
		"courses": previews,
	})
}

type newCourseRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
}

func (h *CourseHandler) CreateCourse(c *gin.Context) {
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
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (h *CourseHandler) PublishCourse(c *gin.Context) {
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

	userID, ex := c.Get(ClientIDCtx)
	if !ex {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userUUID := userID.(uuid.UUID)
	err = h.CourseService.Publish(c.Request.Context(), courseID, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func (h *CourseHandler) HideCourse(c *gin.Context) {
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

	userID, ex := c.Get(ClientIDCtx)
	if !ex {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userUUID := userID.(uuid.UUID)
	err = h.CourseService.Hide(c.Request.Context(), courseID, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func (h *CourseHandler) SubscribeCourse(c *gin.Context) {
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}

	id, ok := c.Get(ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	err = h.CourseService.Subscribe(c.Request.Context(), courseID, userID)
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

func (h *CourseHandler) GetSubscribedCourses(c *gin.Context) {
	id, ok := c.Get(ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	previews, err := h.CourseService.GetSubscribedCourses(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"courses": previews})
}

func (h *CourseHandler) GetMyCourses(c *gin.Context) {
	id, ok := c.Get(ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)

	previews, err := h.CourseService.GetMyCourses(c.Request.Context(), authorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"courses": previews})
}

func (h *CourseHandler) GetCourseStatus(c *gin.Context) {
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}

	status, err := h.CourseService.GetCourseStatus(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"course_id": courseID.String(),
		"status":    status,
	})
}

func (h *CourseHandler) RateCourse(c *gin.Context) {
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}
	id, exists := c.Get(ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	if err := h.CourseService.RateCourse(c.Request.Context(), courseID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "rated"})
}

func (h *CourseHandler) UnrateCourse(c *gin.Context) {
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}
	id, exists := c.Get(ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	if err := h.CourseService.UnrateCourse(c.Request.Context(), courseID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "unrated"})
}

func (h *CourseHandler) GetRatingStatus(c *gin.Context) {
	id, ok := c.Get(ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	statusMap, err := h.CourseService.GetRatingStatus(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rated_status": statusMap})
}
