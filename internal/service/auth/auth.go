package auth

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type AuthRepo interface {
	CreateUser(ctx context.Context, user models.User) (*models.User, error)
	UserByName(ctx context.Context, username string) (*models.User, error)
	UserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type tokenRepo interface {
	Create(ctx context.Context, userID uuid.UUID, token *jwt.Token) (*models.RefreshToken, error)
	ByPrimaryKey(ctx context.Context, userID uuid.UUID, token *jwt.Token) (*models.RefreshToken, error)
	DeleteUserTokens(ctx context.Context, userID uuid.UUID) error
}

type AuthService struct {
	log        logger.Log
	jwtManager *JWTManager
	authRepo   AuthRepo
	tokenRepo  tokenRepo
}

func NewAuthService(l logger.Log, manager *JWTManager, aRepo AuthRepo, tRepo tokenRepo) *AuthService {
	return &AuthService{
		log:        l,
		jwtManager: manager,
		authRepo:   aRepo,
		tokenRepo:  tRepo,
	}
}

func (u *AuthService) RefreshTokens(ctx context.Context, token string) (*models.TokenPair, error) {
	curToken, err := u.jwtManager.Parse(token)
	if err != nil {
		return nil, err
	}
	userIdStr, err := curToken.Claims.GetSubject()
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(userIdStr)
	if err != nil {
		return nil, err
	}
	tokenRecord, err := u.tokenRepo.ByPrimaryKey(ctx, userID, curToken)
	if err != nil {
		return nil, err
	}
	user, err := u.authRepo.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if tokenRecord.ExpiresAt.Before(time.Now()) {
		return nil, app_errors.ErrTokenExpired
	}
	tokenPair, err := u.jwtManager.GenerateTokenPair(user.ID, user.Roles)
	if err != nil {
		return nil, err
	}
	if err := u.tokenRepo.DeleteUserTokens(ctx, userID); err != nil {
		return nil, err
	}
	if _, err := u.tokenRepo.Create(ctx, user.ID, tokenPair.RefreshToken); err != nil {
		return nil, err
	}
	return tokenPair, nil

}

func (u *AuthService) ParseToken(ctx context.Context, token string) (*jwt.Token, error) {
	return u.jwtManager.Parse(token)
}

func (u *AuthService) IsAccessToken(ctx context.Context, token *jwt.Token) bool {
	return u.jwtManager.TokenType(token, AccessTokenType)
}

func (u *AuthService) AccessClaims(ctx context.Context, token string) (userID uuid.UUID, roles []string, err error) {
	claims, err := u.jwtManager.AccessClaims(token)
	if err != nil {
		return uuid.Nil, nil, err
	}
	roles = claims.Roles
	userID = claims.UserID
	err = nil
	return
}

func (u *AuthService) User(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := u.authRepo.UserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (u *AuthService) LoginUser(ctx context.Context, username, password string) (accessToken, refreshToken string, err error) {
	user, err := u.authRepo.UserByName(ctx, username)
	if err != nil {
		return "", "", err
	}

	if !checkPasswordHash(password, user.Password) {
		return "", "", app_errors.ErrIncorrectPassword
	}

	tokenPair, err := u.jwtManager.GenerateTokenPair(user.ID, user.Roles)
	if err != nil {
		return "", "", err
	}

	err = u.tokenRepo.DeleteUserTokens(ctx, user.ID)
	if err != nil {
		return "", "", err
	}
	_, err = u.tokenRepo.Create(ctx, user.ID, tokenPair.RefreshToken)
	if err != nil {
		return "", "", err
	}

	return tokenPair.AccessToken.Raw, tokenPair.RefreshToken.Raw, nil
}

func (u *AuthService) CreateUser(ctx context.Context, user models.User) (*models.User, error) {
	var err error

	if len(user.Password) > 16 || len(user.Password) < 6 {
		return nil, app_errors.ErrIncorrectPassword
	}

	user.Password, err = hashPassword(user.Password)
	if err != nil {
		return nil, err
	}

	createdUser, err := u.authRepo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return createdUser, nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
