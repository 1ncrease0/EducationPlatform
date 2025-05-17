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
	UpdatedAt time.Time `json:"updated_at"`
}

type LessonDetail struct {
	Lesson   Lesson          `json:"lesson"`
	Contents []CourseContent `json:"contents"`
}
