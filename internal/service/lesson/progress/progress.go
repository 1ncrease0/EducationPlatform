package progress

import (
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"strconv"
	"strings"
)

type lessonRepo interface {
	GetLessonDetail(ctx context.Context, lessonID uuid.UUID) (models.LessonDetail, error)
	UpdateLessonProgress(ctx context.Context, lessonID, userID uuid.UUID, status string, score float64) error
	GetLessonProgress(ctx context.Context, lessonID, userID uuid.UUID) (models.LessonProgress, error)
}
type LessonProgressService struct {
	log        logger.Log
	lessonRepo lessonRepo
}

func NewLessonProgressService(log logger.Log, l lessonRepo) *LessonProgressService {
	return &LessonProgressService{
		log:        log,
		lessonRepo: l,
	}
}

func (s *LessonProgressService) SubmitQuizAnswers(ctx context.Context, lessonID uuid.UUID, userID uuid.UUID, answers []models.QuizAnswer) (float64, error) {
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

func (s *LessonProgressService) GetQuizResult(ctx context.Context, lessonID, userID uuid.UUID) (float64, string, error) {
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

func normalizeText(text string) string {
	return strings.ToLower(strings.TrimSpace(text))
}
