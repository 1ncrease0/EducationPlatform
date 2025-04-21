package auth

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"time"
)

const (
	AccessTokenType  = "access"
	RefreshTokenType = "refresh"
)

var signingMethod = jwt.SigningMethodHS256

type JWTManager struct {
	secretKey  string
	accessTTL  time.Duration
	refreshTTL time.Duration
	issuer     string
}

func NewJWTManager(secretKey, issuer string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:  secretKey,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		issuer:     issuer,
	}
}

type AccessTokenClaims struct {
	TokenType string    `json:"token_type"`
	UserID    uuid.UUID `json:"user_id"`
	Roles     []string  `json:"roles"`
	jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
	TokenType string    `json:"token_type"`
	UserID    uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

func (j *JWTManager) AccessClaims(tokenStr string) (*AccessTokenClaims, error) {
	claims := &AccessTokenClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != signingMethod {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secretKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse access token: %w", err)
	}

	if claims.TokenType != AccessTokenType {
		return nil, fmt.Errorf("wrong token type: expected %q, got %q", AccessTokenType, claims.TokenType)
	}

	return claims, nil
}

func (j *JWTManager) Parse(token string) (*jwt.Token, error) {
	parser := jwt.Parser{}
	jwtToken, err := parser.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if token.Method != signingMethod {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secretKey), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, app_errors.ErrTokenExpired
		}
		return nil, fmt.Errorf("failed to parse access token: %w", err)
	}
	return jwtToken, nil
}

func (j *JWTManager) TokenType(token *jwt.Token, t string) bool {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		fmt.Println(1)
		return false
	}
	if tokenType, ok := claims["token_type"].(string); ok {
		return tokenType == t
	}

	return false
}

func (j *JWTManager) GenerateTokenPair(userID uuid.UUID, roles []string) (*models.TokenPair, error) {
	now := time.Now()
	accessToken := jwt.NewWithClaims(signingMethod, AccessTokenClaims{
		TokenType: AccessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    j.issuer,
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID: userID,
		Roles:  roles,
	})

	accessKey := []byte(j.secretKey)
	signedAccessToken, err := accessToken.SignedString(accessKey)
	if err != nil {
		return nil, fmt.Errorf("access token signing failed: %v", err)
	}
	accessToken, err = j.Parse(signedAccessToken)
	if err != nil {
		return nil, fmt.Errorf("access token parsinh failed: %v", err)
	}

	refreshToken := jwt.NewWithClaims(signingMethod, RefreshTokenClaims{
		TokenType: RefreshTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    j.issuer,
			ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID: userID,
	})

	key := []byte(j.secretKey)
	signedRefreshToken, err := refreshToken.SignedString(key)
	if err != nil {
		return nil, fmt.Errorf("refresh token signing failed: %v", err)
	}
	refreshToken, err = j.Parse(signedRefreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh token parsinh failed: %v", err)
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
