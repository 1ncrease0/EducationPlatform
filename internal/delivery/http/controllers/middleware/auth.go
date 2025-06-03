package middleware

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

type AuthService interface {
	ParseToken(ctx context.Context, token string) (*jwt.Token, error)
	IsAccessToken(ctx context.Context, token *jwt.Token) bool
	AccessClaims(ctx context.Context, token string) (userID uuid.UUID, roles []string, err error)
	User(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type AuthMiddlewareProvider struct {
	log     logger.Log
	service AuthService
}

func NewAuthMiddlewareProvider(log logger.Log, s AuthService) *AuthMiddlewareProvider {
	return &AuthMiddlewareProvider{
		log:     log,
		service: s,
	}
}
func (h *AuthMiddlewareProvider) AuthMiddleware(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	var token string
	if parts := strings.Split(authHeader, "Bearer "); len(parts) == 2 {
		token = parts[1]
	}
	if token == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	parsedToken, err := h.service.ParseToken(c.Request.Context(), token)
	if err != nil {
		h.log.Info("failed to parse token", err)
		if errors.Is(err, app_errors.ErrTokenExpired) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": app_errors.ErrTokenExpired.Error()})
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "cant parse token"})
		return
	}
	if !h.service.IsAccessToken(c.Request.Context(), parsedToken) {
		//h.log.Error("not access")\
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not access token"})

		return
	}

	userID, roles, err := h.service.AccessClaims(c.Request.Context(), token)
	if err != nil {
		//h.log.Error("claims")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	user, err := h.service.User(c.Request.Context(), userID)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.Set(ClientIDCtx, user.ID)
	c.Set(ClientRolesCtx, roles)
	c.Next()
}
