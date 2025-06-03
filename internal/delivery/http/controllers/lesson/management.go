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
	"net/http"
)

type ManagementService interface {
	CreateLesson(ctx context.Context, lesson models.Lesson, authorID uuid.UUID) (*models.Lesson, error)
	CreateModule(ctx context.Context, module models.Module, authorID uuid.UUID) (*models.Module, error)
	DeleteLesson(ctx context.Context, courseID, lessonID, moduleID uuid.UUID, authorID uuid.UUID) error
	DeleteModule(ctx context.Context, courseID, moduleID uuid.UUID, authorID uuid.UUID) error
	SwapLessons(ctx context.Context, lessonID1, lessonID2, authorID uuid.UUID) error
	SwapModules(ctx context.Context, moduleID1, moduleID2, courseID, authorID uuid.UUID) error
}

type ManagementHandler struct {
	log     logger.Log
	service ManagementService
}

func NewManagementHandler(log logger.Log, service ManagementService) *ManagementHandler {
	return &ManagementHandler{log, service}
}

func (h *ManagementHandler) DeleteLesson(c *gin.Context) {
	courseIDStr, ok := c.Params.Get("course_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_id is required"})
		return
	}
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}

	moduleIDStr, ok := c.Params.Get("module_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "module_id is required"})
		return
	}
	moduleID, err := uuid.Parse(moduleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module_id"})
		return
	}

	lessonIDStr, ok := c.Params.Get("lesson_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lesson_id is required"})
		return
	}
	lessonID, err := uuid.Parse(lessonIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson_id"})
		return
	}

	id, ok := c.Get(middleware.ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)

	if err := h.service.DeleteLesson(c.Request.Context(), courseID, lessonID, moduleID, authorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "lesson deleted"})
}

func (h *ManagementHandler) DeleteModule(c *gin.Context) {
	courseIDStr, ok := c.Params.Get("course_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_id is required"})
		return
	}
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}

	moduleIDStr, ok := c.Params.Get("module_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "module_id is required"})
		return
	}
	moduleID, err := uuid.Parse(moduleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module_id"})
		return
	}

	id, ok := c.Get(middleware.ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)

	if err := h.service.DeleteModule(c.Request.Context(), courseID, moduleID, authorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "module deleted"})
}

type createLessonRequest struct {
	ModuleID    uuid.UUID `json:"module_id" binding:"required"`
	LessonTitle string    `json:"lesson_title" binding:"required"`
}

func (h *ManagementHandler) CreateLesson(c *gin.Context) {
	var input createLessonRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	courseIDStr, ok := c.Params.Get("course_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_id is required"})
		return
	}
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
	authorID := id.(uuid.UUID)

	lesson := models.Lesson{
		CourseID:    courseID,
		ModuleID:    input.ModuleID,
		LessonTitle: input.LessonTitle,
	}
	createdLesson, err := h.service.CreateLesson(c.Request.Context(), lesson, authorID)
	if err != nil {
		if errors.Is(err, app_errors.ErrDuplicateLesson) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, createdLesson)
}

type createModuleRequest struct {
	Title string `json:"title" binding:"required"`
}

func (h *ManagementHandler) CreateModule(c *gin.Context) {
	var req createModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	courseIDStr, ok := c.Params.Get("course_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_id is required"})
		return
	}
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
	authorID := id.(uuid.UUID)

	module := models.Module{
		CourseID: courseID,
		Title:    req.Title,
	}
	createdModule, err := h.service.CreateModule(c.Request.Context(), module, authorID)
	if err != nil {
		if errors.Is(err, app_errors.ErrDuplicateModule) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, createdModule)
}

func (h *ManagementHandler) SwapLessons(c *gin.Context) {
	courseIDStr := c.Param("course_id")
	_, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}

	lessonID1Str := c.Query("lesson_id_1")
	lessonID2Str := c.Query("lesson_id_2")
	if lessonID1Str == "" || lessonID2Str == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lesson_id_1 and lesson_id_2 required"})
		return
	}
	lessonID1, err := uuid.Parse(lessonID1Str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson_id_1"})
		return
	}
	lessonID2, err := uuid.Parse(lessonID2Str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson_id_2"})
		return
	}

	id, exists := c.Get(middleware.ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)

	if err := h.service.SwapLessons(c.Request.Context(), lessonID1, lessonID2, authorID); err != nil {
		h.log.ErrorErr("err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "lessons swapped"})
}

func (h *ManagementHandler) SwapModules(c *gin.Context) {
	courseIDStr := c.Param("course_id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course_id"})
		return
	}

	moduleID1Str := c.Query("module_id_1")
	moduleID2Str := c.Query("module_id_2")
	if moduleID1Str == "" || moduleID2Str == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "module_id_1 and module_id_2 required"})
		return
	}
	moduleID1, err := uuid.Parse(moduleID1Str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module_id_1"})
		return
	}
	moduleID2, err := uuid.Parse(moduleID2Str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module_id_2"})
		return
	}

	id, exists := c.Get(middleware.ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)

	if err := h.service.SwapModules(c.Request.Context(), moduleID1, moduleID2, courseID, authorID); err != nil {
		h.log.ErrorErr("err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "modules swapped"})
}
