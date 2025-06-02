package lesson

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type lessonRepo interface {
	CreateLesson(ctx context.Context, lesson models.Lesson) (*models.Lesson, error)
	CreateModule(ctx context.Context, module models.Module) (*models.Module, error)
	GetLessonByID(ctx context.Context, lessonID uuid.UUID) (models.Lesson, error)
	GetModuleByID(ctx context.Context, moduleID uuid.UUID) (models.Module, error)
	CourseContent(ctx context.Context, courseID uuid.UUID) ([]models.Contents, error)
	DeleteLessonAndUpdateOrder(ctx context.Context, lessonID, moduleID uuid.UUID, lessonOrder int) error
	DeleteModuleAndUpdateOrder(ctx context.Context, moduleID, courseID uuid.UUID, moduleOrder int) error
	GetMaxModuleOrder(ctx context.Context, courseID uuid.UUID) (int, error)
	GetMaxLessonOrder(ctx context.Context, moduleID uuid.UUID) (int, error)
	SwapLessons(ctx context.Context, lessonID1, lessonID2 uuid.UUID) error
	SwapModules(ctx context.Context, moduleID1, moduleID2 uuid.UUID) error
	CreateContent(ctx context.Context, content models.CourseContent) (*models.CourseContent, error)
	GetLessonDetail(ctx context.Context, lessonID uuid.UUID) (models.LessonDetail, error)
	UpsertContent(ctx context.Context, content models.CourseContent) (*models.CourseContent, error)
	LessonsByModule(ctx context.Context, moduleID uuid.UUID) ([]models.Lesson, error)
	UpdateLessonProgress(ctx context.Context, lessonID, userID uuid.UUID, status string, score float64) error
	GetLessonProgress(ctx context.Context, lessonID, userID uuid.UUID) (models.LessonProgress, error)
}
type courseRepo interface {
	CourseByID(ctx context.Context, id uuid.UUID) (*models.Course, error)
}

type mediaStorage interface {
	UploadPhoto(ctx context.Context, courseID uuid.UUID, filename string, reader io.Reader, size int64, contentType string) (objectKey string, err error)
	UploadVideo(ctx context.Context, courseID uuid.UUID, filename string, reader io.Reader, size int64, contentType string) (objectKey string, err error)
	GetPhotoURL(ctx context.Context, objectKey string) (string, error)
	GetVideoURL(ctx context.Context, objectKey string) (string, error)
	DeleteVideo(ctx context.Context, objectKey string) error
	DeletePhoto(ctx context.Context, objectKey string) error
}

type LessonService struct {
	log          logger.Log
	lessonRepo   lessonRepo
	courseRepo   courseRepo
	mediaStorage mediaStorage
}

func NewLessonService(l logger.Log, lessonRepo lessonRepo, courseRepo courseRepo, mediaStorage mediaStorage) *LessonService {
	return &LessonService{
		log:          l,
		lessonRepo:   lessonRepo,
		courseRepo:   courseRepo,
		mediaStorage: mediaStorage,
	}
}

func (s *LessonService) CreateContent(ctx context.Context, content models.CourseContent, authorID uuid.UUID) (*models.CourseContent, error) {
	lesson, err := s.lessonRepo.GetLessonByID(ctx, content.LessonID)
	if err != nil {
		return nil, err
	}
	course, err := s.courseRepo.CourseByID(ctx, lesson.CourseID)
	if err != nil {
		return nil, err
	}
	if course.AuthorID != authorID {
		return nil, app_errors.ErrNotCourseAuthor
	}
	return s.lessonRepo.UpsertContent(ctx, content)
}

func (s *LessonService) CreateMediaContent(ctx context.Context, lessonID uuid.UUID, mediaType, filename string, file io.Reader, size int64, contentType string, authorID uuid.UUID) (*models.CourseContent, error) {
	lesson, err := s.lessonRepo.GetLessonByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	course, err := s.courseRepo.CourseByID(ctx, lesson.CourseID)
	if err != nil {
		return nil, err
	}
	if course.AuthorID != authorID {
		return nil, app_errors.ErrNotCourseAuthor
	}

	var objectKey string
	switch mediaType {
	case models.ContentTypeImage:
		objectKey, err = s.mediaStorage.UploadPhoto(ctx, lesson.CourseID, filename, file, size, contentType)
	case models.ContentTypeVideo:
		objectKey, err = s.mediaStorage.UploadVideo(ctx, lesson.CourseID, filename, file, size, contentType)
	default:
		return nil, fmt.Errorf("unsupported media type")
	}
	if err != nil {
		return nil, err
	}

	content := models.CourseContent{
		LessonID:  lessonID,
		Type:      mediaType,
		ObjectKey: &objectKey,
	}
	return s.lessonRepo.UpsertContent(ctx, content)
}
func (s *LessonService) SwapLessons(ctx context.Context, lessonID1, lessonID2, authorID uuid.UUID) error {
	lesson1, err := s.lessonRepo.GetLessonByID(ctx, lessonID1)
	if err != nil {
		return err
	}
	lesson2, err := s.lessonRepo.GetLessonByID(ctx, lessonID2)
	if err != nil {
		return err
	}
	if lesson1.ModuleID != lesson2.ModuleID {
		return fmt.Errorf("lessons belong to different modules")
	}

	course, err := s.courseRepo.CourseByID(ctx, lesson1.CourseID)
	if err != nil {
		return err
	}
	if course.AuthorID != authorID {
		return app_errors.ErrNotCourseAuthor
	}

	return s.lessonRepo.SwapLessons(ctx, lessonID1, lessonID2)
}

func (s *LessonService) SwapModules(ctx context.Context, moduleID1, moduleID2, courseID, authorID uuid.UUID) error {
	module1, err := s.lessonRepo.GetModuleByID(ctx, moduleID1)
	if err != nil {
		return err
	}
	module2, err := s.lessonRepo.GetModuleByID(ctx, moduleID2)
	if err != nil {
		return err
	}
	if module1.CourseID != module2.CourseID {
		return fmt.Errorf("modules belong to different courses")
	}
	if module1.CourseID != courseID {
		return fmt.Errorf("courseID mismatch")
	}
	course, err := s.courseRepo.CourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course.AuthorID != authorID {
		return app_errors.ErrNotCourseAuthor
	}

	return s.lessonRepo.SwapModules(ctx, moduleID1, moduleID2)
}

func (s *LessonService) CreateLesson(ctx context.Context, lesson models.Lesson, authorID uuid.UUID) (*models.Lesson, error) {
	course, err := s.courseRepo.CourseByID(ctx, lesson.CourseID)
	if err != nil {
		return nil, err
	}
	if course.AuthorID != authorID {
		return nil, app_errors.ErrNotCourseAuthor
	}

	maxOrder, err := s.lessonRepo.GetMaxLessonOrder(ctx, lesson.ModuleID)
	if err != nil {
		return nil, err
	}
	lesson.LessonOrder = maxOrder + 1

	l, err := s.lessonRepo.CreateLesson(ctx, lesson)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (s *LessonService) CreateModule(ctx context.Context, module models.Module, authorID uuid.UUID) (*models.Module, error) {
	course, err := s.courseRepo.CourseByID(ctx, module.CourseID)
	if err != nil {
		return nil, err
	}
	if course.AuthorID != authorID {
		return nil, app_errors.ErrNotCourseAuthor
	}

	maxOrder, err := s.lessonRepo.GetMaxModuleOrder(ctx, module.CourseID)
	if err != nil {
		return nil, err
	}
	module.Order = maxOrder + 1

	m, err := s.lessonRepo.CreateModule(ctx, module)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *LessonService) CourseContent(ctx context.Context, courseID uuid.UUID) ([]models.Contents, error) {
	_, err := s.courseRepo.CourseByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	return s.lessonRepo.CourseContent(ctx, courseID)
}

func (s *LessonService) DeleteLesson(ctx context.Context, courseID, lessonID, moduleID, authorID uuid.UUID) error {
	course, err := s.courseRepo.CourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course.AuthorID != authorID {
		return app_errors.ErrNotCourseAuthor
	}

	detail, err := s.lessonRepo.GetLessonDetail(ctx, lessonID)
	if err != nil {
		return err
	}

	for _, content := range detail.Contents {
		if content.ObjectKey != nil {
			switch content.Type {
			case models.ContentTypeImage:
				if err := s.mediaStorage.DeletePhoto(ctx, *content.ObjectKey); err != nil {
					s.log.Error("failed to delete image from minio", err)
				}
			case models.ContentTypeVideo:
				if err := s.mediaStorage.DeleteVideo(ctx, *content.ObjectKey); err != nil {
					s.log.Error("failed to delete video from minio", err)
				}
			}
		}
	}

	return s.lessonRepo.DeleteLessonAndUpdateOrder(ctx, lessonID, moduleID, detail.Lesson.LessonOrder)
}

func (s *LessonService) DeleteModule(ctx context.Context, courseID, moduleID, authorID uuid.UUID) error {
	course, err := s.courseRepo.CourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course.AuthorID != authorID {
		return app_errors.ErrNotCourseAuthor
	}

	module, err := s.lessonRepo.GetModuleByID(ctx, moduleID)
	if err != nil {
		return err
	}

	lessons, err := s.lessonRepo.LessonsByModule(ctx, moduleID)
	if err != nil {
		return err
	}
	for _, lesson := range lessons {
		detail, err := s.lessonRepo.GetLessonDetail(ctx, lesson.ID)
		if err != nil {
			s.log.Error("failed to get lesson detail", err)
			continue
		}
		for _, content := range detail.Contents {
			if content.ObjectKey != nil {
				switch content.Type {
				case models.ContentTypeImage:
					if err := s.mediaStorage.DeletePhoto(ctx, *content.ObjectKey); err != nil {
						s.log.Error("failed to delete image from MinIO", err)
					}
				case models.ContentTypeVideo:
					if err := s.mediaStorage.DeleteVideo(ctx, *content.ObjectKey); err != nil {
						s.log.Error("failed to delete video from MinIO", err)
					}
				}
			}
		}
	}

	return s.lessonRepo.DeleteModuleAndUpdateOrder(ctx, moduleID, courseID, module.Order)
}

func (s *LessonService) GetLessonDetail(ctx context.Context, lessonID uuid.UUID) (models.LessonDetail, error) {
	detail, err := s.lessonRepo.GetLessonDetail(ctx, lessonID)
	if err != nil {
		return detail, err
	}
	for i := range detail.Contents {
		if detail.Contents[i].ObjectKey != nil {
			switch detail.Contents[i].Type {
			case models.ContentTypeImage:
				url, err := s.mediaStorage.GetPhotoURL(ctx, *detail.Contents[i].ObjectKey)
				if err == nil {
					detail.Contents[i].ObjectKey = &url
				}
			case models.ContentTypeVideo:
				url, err := s.mediaStorage.GetVideoURL(ctx, *detail.Contents[i].ObjectKey)
				if err == nil {
					detail.Contents[i].ObjectKey = &url
				}
			}
		}
	}
	return detail, nil
}

func normalizeText(text string) string {
	return strings.ToLower(strings.TrimSpace(text))
}

func (s *LessonService) SubmitQuizAnswers(ctx context.Context, lessonID uuid.UUID, userID uuid.UUID, answers []models.QuizAnswer) (float64, error) {
	detail, err := s.lessonRepo.GetLessonDetail(ctx, lessonID)
	if err != nil {
		return 0, err
	}

	var quizContent *models.CourseContent
	for _, content := range detail.Contents {
		if content.Type == models.ContentTypeQuiz {
			quizContent = &content
			break
		}
	}

	if quizContent == nil || quizContent.QuizJSON == nil {
		return 0, fmt.Errorf("quiz not found in lesson")
	}

	s.log.Info("!!quizContent", quizContent)
	s.log.Info("!!quizContent.QuizJSON", *quizContent.QuizJSON)

	var quiz models.QuizJSON
	if err := json.Unmarshal([]byte(*quizContent.QuizJSON), &quiz); err != nil {
		s.log.Error("Failed to unmarshal quiz JSON", err)
		s.log.Error("Quiz JSON content:", *quizContent.QuizJSON)
		return 0, fmt.Errorf("invalid quiz format: %w", err)
	}

	s.log.Info("Parsed quiz:", quiz)
	s.log.Info("User answers:", answers)
	score := 0.0
	answeredQuestions := 0.0
	totalQuestions := float64(len(quiz.Questions))

	for _, question := range quiz.Questions {
		s.log.Info("Processing question:", question)
		var userAnswer *models.QuizAnswer
		for _, answer := range answers {
			if answer.QuestionID == question.ID {
				userAnswer = &answer
				break
			}
		}

		if question.Required && userAnswer == nil {
			s.log.Info("Skipping required question without answer:", question.ID)
			continue
		}

		answeredQuestions++
		s.log.Info("Answered questions count:", answeredQuestions)

		if userAnswer != nil {
			s.log.Info("Processing user answer:", userAnswer)
			switch question.Type {
			case "single", "single_choice":
				if len(userAnswer.OptionIDs) == 1 {
					optionIndex := 0
					if strings.HasPrefix(userAnswer.OptionIDs[0], "option_") {
						optionIndex, _ = strconv.Atoi(strings.TrimPrefix(userAnswer.OptionIDs[0], "option_"))
					}
					s.log.Info("Single choice answer details:", map[string]interface{}{
						"optionIndex":    optionIndex,
						"totalOptions":   len(question.Options),
						"selectedOption": question.Options[optionIndex],
					})
					if optionIndex < len(question.Options) && question.Options[optionIndex].IsCorrect {
						score++
						s.log.Info("Correct single choice answer, score:", score)
					}
				}
			case "multiple", "multiple_choice":
				correctCount := 0
				for _, option := range question.Options {
					if option.IsCorrect {
						correctCount++
					}
				}
				s.log.Info("Multiple choice question, correct options count:", correctCount)
				if correctCount > 0 {
					userCorrectCount := 0
					userIncorrectCount := 0
					for _, optionID := range userAnswer.OptionIDs {
						optionIndex := 0
						if strings.HasPrefix(optionID, "option_") {
							optionIndex, _ = strconv.Atoi(strings.TrimPrefix(optionID, "option_"))
						}
						if optionIndex < len(question.Options) {
							if question.Options[optionIndex].IsCorrect {
								userCorrectCount++
							} else {
								userIncorrectCount++
							}
						}
					}
					s.log.Info("Multiple choice answer details:", map[string]interface{}{
						"userCorrectCount":   userCorrectCount,
						"userIncorrectCount": userIncorrectCount,
						"totalCorrectCount":  correctCount,
					})
					if userCorrectCount == correctCount && userIncorrectCount == 0 {
						score++
						s.log.Info("Correct multiple choice answer, score:", score)
					}
				}
			case "text":
				if userAnswer.TextAnswer != "" && question.CorrectAnswer != "" {
					normalizedUserAnswer := normalizeText(userAnswer.TextAnswer)
					normalizedCorrectAnswer := normalizeText(question.CorrectAnswer)
					s.log.Info("Text answer comparison:", map[string]string{
						"user":    normalizedUserAnswer,
						"correct": normalizedCorrectAnswer,
					})
					if normalizedUserAnswer == normalizedCorrectAnswer {
						score++
						s.log.Info("Correct text answer, score:", score)
					}
				}
			}
		}
	}

	finalScore := 0.0
	if totalQuestions > 0 {
		finalScore = (score / totalQuestions) * 100
	}
	s.log.Info("Final score calculation:", map[string]float64{
		"score":          score,
		"totalQuestions": totalQuestions,
		"finalScore":     finalScore,
	})

	status := models.LessonStatusFailed
	if finalScore == 100 {
		status = models.LessonStatusPassed
	}
	s.log.Info("Final status:", status)

	if err := s.lessonRepo.UpdateLessonProgress(ctx, lessonID, userID, status, finalScore); err != nil {
		return 0, fmt.Errorf("failed to update lesson progress: %w", err)
	}

	return finalScore, nil
}

func (s *LessonService) GetQuizResult(ctx context.Context, lessonID, userID uuid.UUID) (float64, string, error) {
	detail, err := s.lessonRepo.GetLessonDetail(ctx, lessonID)
	if err != nil {
		return 0, "", err
	}

	var quizContent *models.CourseContent
	for _, content := range detail.Contents {
		if content.Type == models.ContentTypeQuiz {
			quizContent = &content
			break
		}
	}

	if quizContent == nil || quizContent.QuizJSON == nil {
		return 0, "", fmt.Errorf("quiz not found in lesson")
	}

	var quiz models.QuizJSON
	if err := json.Unmarshal([]byte(*quizContent.QuizJSON), &quiz); err != nil {
		return 0, "", fmt.Errorf("invalid quiz format: %w", err)
	}

	progress, err := s.lessonRepo.GetLessonProgress(ctx, lessonID, userID)
	if err != nil {
		return 0, "", fmt.Errorf("quiz not taken yet")
	}

	if progress.Score == 0 && progress.Status == "" {
		return 0, "", fmt.Errorf("quiz not taken yet")
	}

	status := models.LessonStatusFailed
	if progress.Score == 100 {
		status = models.LessonStatusPassed
	}

	return progress.Score, status, nil
}
