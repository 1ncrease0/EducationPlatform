package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	ContentTypeText  = "text"
	ContentTypeImage = "image"
	ContentTypeVideo = "video"
	ContentTypeQuiz  = "quiz"

	LessonStatusPassed = "passed"
	LessonStatusFailed = "failed"
)

type Lesson struct {
	ID          uuid.UUID `json:"id"`
	CourseID    uuid.UUID `json:"course_id"`
	ModuleID    uuid.UUID `json:"module_id"`
	LessonTitle string    `json:"lesson_title"`
	LessonOrder int       `json:"lesson_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CourseContent struct {
	ID        uuid.UUID `json:"id"`
	LessonID  uuid.UUID `json:"lesson_id"`
	Type      string    `json:"type"`
	Order     int       `json:"order"`
	Text      *string   `json:"text,omitempty"`
	ObjectKey *string   `json:"object_key,omitempty"`
	QuizJSON  *string   `json:"quiz_json,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LessonProgress struct {
	UserID    uuid.UUID `json:"user_id"`
	LessonID  uuid.UUID `json:"lesson_id"`
	Status    string    `json:"status"`
	Score     float64   `json:"score"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LessonDetail struct {
	Lesson   Lesson          `json:"lesson"`
	Contents []CourseContent `json:"contents"`
}

type QuizJSON struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Questions   []QuizQuestion `json:"questions"`
	MinScore    float64        `json:"minScore"`
}

type QuizQuestion struct {
	ID            string       `json:"id"`
	Text          string       `json:"text"`
	Type          string       `json:"type"`
	Options       []QuizOption `json:"options"`
	Required      bool         `json:"required"`
	CorrectAnswer string       `json:"correctAnswer"`
}

type QuizOption struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"isCorrect"`
}

type QuizAnswer struct {
	QuestionID string   `json:"question_id"`
	OptionIDs  []string `json:"option_ids,omitempty"`
	TextAnswer string   `json:"text_answer,omitempty"`
}
