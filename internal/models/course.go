package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusHidden = "hidden"
	StatusPublic = "public"
)

type Course struct {
	ID            uuid.UUID `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	LogoObjectKey string    `json:"logo_object_key"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	AuthorID      uuid.UUID `json:"author_id"`
	Status        string    `json:"status"`
	StarsCount    int       `json:"stars_count"`
}

type CoursePreview struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	AuthorName  string    `json:"author_name"`
	LogoURL     string    `json:"logo_url"`
	StarsCount  int       `json:"stars_count"`
}
