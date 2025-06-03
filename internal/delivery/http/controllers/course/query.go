package course

import (
	"SkillForge/internal/delivery/http/controllers/middleware"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"strconv"
)

type QueryService interface {
	GetMyCourses(ctx context.Context, authorID uuid.UUID) ([]models.CoursePreview, error)
	CourseByID(ctx context.Context, id uuid.UUID) (*models.CoursePreview, error)
	CoursesPreview(ctx context.Context, count int, offset int) ([]models.CoursePreview, int, error)
	SearchCoursesPreview(ctx context.Context, query string, count int, offset int) ([]models.CoursePreview, int, error)
	GetSubscribedCourses(ctx context.Context, userID uuid.UUID) ([]models.CoursePreview, error)
}

type QueryHandler struct {
	log     logger.Log
	service QueryService
}

func NewQueryHandler(log logger.Log, s QueryService) *QueryHandler {
	return &QueryHandler{
		log:     log,
		service: s,
	}
}

func (h *QueryHandler) GetMyCourses(c *gin.Context) {
	id, ok := c.Get(middleware.ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)

	previews, err := h.service.GetMyCourses(c.Request.Context(), authorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"courses": previews})
}

func (h *QueryHandler) CourseByID(c *gin.Context) {
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		h.log.ErrorErr("err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}

	preview, err := h.service.CourseByID(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, preview)
}

func (h *QueryHandler) ListCoursePreview(c *gin.Context) {
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
		previews, total, err = h.service.SearchCoursesPreview(ctx, q, limit, offset)
	} else {
		previews, total, err = h.service.CoursesPreview(ctx, limit, offset)
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

func (h *QueryHandler) GetSubscribedCourses(c *gin.Context) {
	id, ok := c.Get(middleware.ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	previews, err := h.service.GetSubscribedCourses(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"courses": previews})
}
