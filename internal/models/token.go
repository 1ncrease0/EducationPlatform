package models

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"time"
)

type RefreshToken struct {
	UserID      uuid.UUID
	HashedToken string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type TokenPair struct {
	AccessToken  *jwt.Token
	RefreshToken *jwt.Token
}
