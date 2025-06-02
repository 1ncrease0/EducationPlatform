package controllers

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LessonService interface {
	CreateLesson(ctx context.Context, lesson models.Lesson, authorID uuid.UUID) (*models.Lesson, error)
	CreateModule(ctx context.Context, module models.Module, authorID uuid.UUID) (*models.Module, error)
	CourseContent(ctx context.Context, courseID uuid.UUID) ([]models.Contents, error)
	DeleteLesson(ctx context.Context, courseID, lessonID, moduleID uuid.UUID, authorID uuid.UUID) error
	DeleteModule(ctx context.Context, courseID, moduleID uuid.UUID, authorID uuid.UUID) error
	SwapLessons(ctx context.Context, lessonID1, lessonID2, authorID uuid.UUID) error
	SwapModules(ctx context.Context, moduleID1, moduleID2, courseID, authorID uuid.UUID) error
	CreateContent(ctx context.Context, content models.CourseContent, authorID uuid.UUID) (*models.CourseContent, error)
	CreateMediaContent(ctx context.Context, lessonID uuid.UUID, mediaType, filename string, file io.Reader, size int64, contentType string, authorID uuid.UUID) (*models.CourseContent, error)
	GetLessonDetail(ctx context.Context, lessonID uuid.UUID) (models.LessonDetail, error)
	SubmitQuizAnswers(ctx context.Context, lessonID uuid.UUID, userID uuid.UUID, answers []models.QuizAnswer) (float64, error)
	GetQuizResult(ctx context.Context, lessonID, userID uuid.UUID) (float64, string, error)
}

type LessonHandler struct {
	LessonService LessonService
	log           logger.Log
}

func NewLessonHandler(l logger.Log, lessonService LessonService) *LessonHandler {
	return &LessonHandler{
		LessonService: lessonService,
		log:           l,
	}
}

func (h *LessonHandler) GetLessonDetail(c *gin.Context) {
	lessonIDStr := c.Param("lesson_id")
	lessonID, err := uuid.Parse(lessonIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson_id"})
		return
	}

	detail, err := h.LessonService.GetLessonDetail(c.Request.Context(), lessonID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *LessonHandler) CreateMediaContent(c *gin.Context) {
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

	id, exists := c.Get(ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)

	content, err := h.LessonService.CreateMediaContent(c.Request.Context(), lessonID, mediaType, fileHeader.Filename, file, fileHeader.Size, ct, authorID)
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

type createLessonRequest struct {
	ModuleID    uuid.UUID `json:"module_id" binding:"required"`
	LessonTitle string    `json:"lesson_title" binding:"required"`
}

func (h *LessonHandler) SwapLessons(c *gin.Context) {
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

	id, exists := c.Get(ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)

	if err := h.LessonService.SwapLessons(c.Request.Context(), lessonID1, lessonID2, authorID); err != nil {
		h.log.ErrorErr("err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "lessons swapped"})
}

func (h *LessonHandler) SwapModules(c *gin.Context) {
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

	id, exists := c.Get(ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)

	if err := h.LessonService.SwapModules(c.Request.Context(), moduleID1, moduleID2, courseID, authorID); err != nil {
		h.log.ErrorErr("err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "modules swapped"})
}

func (h *LessonHandler) CreateLesson(c *gin.Context) {
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

	id, ok := c.Get(ClientIDCtx)
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
	createdLesson, err := h.LessonService.CreateLesson(c.Request.Context(), lesson, authorID)
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

func (h *LessonHandler) CreateModule(c *gin.Context) {
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

	id, ok := c.Get(ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)

	module := models.Module{
		CourseID: courseID,
		Title:    req.Title,
	}
	createdModule, err := h.LessonService.CreateModule(c.Request.Context(), module, authorID)
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

func (h *LessonHandler) CourseContent(c *gin.Context) {
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

	content, err := h.LessonService.CourseContent(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, content)
}

func (h *LessonHandler) DeleteLesson(c *gin.Context) {
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

	id, ok := c.Get(ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)

	if err := h.LessonService.DeleteLesson(c.Request.Context(), courseID, lessonID, moduleID, authorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "lesson deleted"})
}

func (h *LessonHandler) DeleteModule(c *gin.Context) {
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

	id, ok := c.Get(ClientIDCtx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	authorID := id.(uuid.UUID)

	if err := h.LessonService.DeleteModule(c.Request.Context(), courseID, moduleID, authorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "module deleted"})
}

type createContentRequest struct {
	LessonID uuid.UUID `json:"lesson_id" binding:"required"`
	Type     string    `json:"type" binding:"required"` // "text", "image", "video", "quiz"
	Text     *string   `json:"text,omitempty"`
	QuizJSON *string   `json:"quiz_json,omitempty"`
}

func (h *LessonHandler) CreateContent(c *gin.Context) {
	var req createContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, ok := c.Get(ClientIDCtx)
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

	createdContent, err := h.LessonService.CreateContent(c.Request.Context(), content, authorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		h.log.ErrorErr("err", err)
		return
	}
	c.JSON(http.StatusCreated, createdContent)
}

type submitQuizRequest struct {
	Answers []models.QuizAnswer `json:"answers" binding:"required"`
}

func (h *LessonHandler) SubmitQuiz(c *gin.Context) {
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

	id, exists := c.Get(ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	score, err := h.LessonService.SubmitQuizAnswers(c.Request.Context(), lessonID, userID, req.Answers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"score": score,
	})
}

func (h *LessonHandler) GetQuizResult(c *gin.Context) {
	lessonIDStr := c.Param("lesson_id")
	lessonID, err := uuid.Parse(lessonIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson_id"})
		return
	}

	id, exists := c.Get(ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	userID := id.(uuid.UUID)

	score, status, err := h.LessonService.GetQuizResult(c.Request.Context(), lessonID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"score":  score,
		"status": status,
	})
}
