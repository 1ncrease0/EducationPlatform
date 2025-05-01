package models

import (
	"github.com/google/uuid"
	"time"
)

const (
	StatusHidden = "hidden"
	StatusPublic = "public"
)

type Course struct {
	ID          uuid.UUID
	Title       string
	Description string
	ImageURL    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	AuthorID    uuid.UUID
	Status      string
	StarsCount  int
}
