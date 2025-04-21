package postgres

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type TokensPostgres struct {
	db *pgxpool.Pool
}

func NewTokensPostgres(db *pgxpool.Pool) *TokensPostgres {
	return &TokensPostgres{db: db}
}

func (r *TokensPostgres) hashToken(token *jwt.Token) (string, error) {
	h := sha256.New()
	h.Write([]byte(token.Raw))
	hashedBytes := h.Sum(nil)
	base64TokenHash := base64.StdEncoding.EncodeToString(hashedBytes)
	return base64TokenHash, nil
}

func (r *TokensPostgres) Create(ctx context.Context, userID uuid.UUID, token *jwt.Token) (*models.RefreshToken, error) {
	hashedToken, err := r.hashToken(token)
	if err != nil {
		return nil, err
	}
	expiresAt, err := token.Claims.GetExpirationTime()
	if err != nil {
		return nil, err
	}
	expiresAtFormat := expiresAt.Format(time.RFC3339)
	query := `
		INSERT INTO refresh_tokens (user_id, hashed_token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING created_at, expires_at
	`
	refreshToken := &models.RefreshToken{
		UserID:      userID,
		HashedToken: hashedToken,
	}
	err = r.db.QueryRow(ctx, query, userID, hashedToken, expiresAtFormat).Scan(&refreshToken.CreatedAt, &refreshToken.ExpiresAt)
	if err != nil {
		return nil, err
	}

	return refreshToken, nil
}

func (r *TokensPostgres) ByPrimaryKey(ctx context.Context, userID uuid.UUID, token *jwt.Token) (*models.RefreshToken, error) {
	hashedToken, err := r.hashToken(token)
	if err != nil {
		return nil, err
	}
	query := `SELECT * FROM refresh_tokens WHERE user_id = $1 AND hashed_token = $2`
	refreshToken := models.RefreshToken{}
	err = r.db.QueryRow(ctx, query, userID, hashedToken).Scan(&refreshToken.UserID, &refreshToken.HashedToken, &refreshToken.CreatedAt, &refreshToken.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, app_errors.ErrTokenNotFound
		}
		return nil, err
	}
	return &refreshToken, nil
}

func (r *TokensPostgres) DeleteUserTokens(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return err
	}
	return nil
}
