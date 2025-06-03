package auth

import (
	"SkillForge/internal/app_errors"
	"SkillForge/internal/delivery/http/controllers/middleware"
	"SkillForge/internal/models"
	"SkillForge/pkg/logger"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"net/http"
)

type AuthService interface {
	CreateUser(ctx context.Context, user models.User) (*models.User, error)
	LoginUser(ctx context.Context, username, password string) (accessToken, refreshToken string, err error)
	ParseToken(ctx context.Context, token string) (*jwt.Token, error)
	IsAccessToken(ctx context.Context, token *jwt.Token) bool
	AccessClaims(ctx context.Context, token string) (userID uuid.UUID, roles []string, err error)
	User(ctx context.Context, id uuid.UUID) (*models.User, error)
	RefreshTokens(ctx context.Context, token string) (*models.TokenPair, error)
}

type AuthHandler struct {
	AuthService AuthService
	log         logger.Log
}

func NewAuthHandler(l logger.Log, auth AuthService) *AuthHandler {
	return &AuthHandler{
		AuthService: auth,
		log:         l,
	}
}

type meResponse struct {
	UserId   string   `json:"userId"`
	Username string   `json:"username" binding:"required"`
	Email    string   `json:"email" binding:"required"`
	Role     []string `json:"role" binding:"required"`
}

func (h *AuthHandler) Me(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.ClientIDCtx)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user id"})
		return
	}
	user, err := h.AuthService.User(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		h.log.ErrorErr("error retrieving user", err, c)
		return
	}

	resp := meResponse{
		UserId:   userID.String(),
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Roles,
	}
	c.JSON(http.StatusOK, resp)
}

type registerRequest struct {
	Username string   `json:"username" binding:"required"`
	Password string   `json:"password" binding:"required"`
	Email    string   `json:"email" binding:"required"`
	Role     []string `json:"role" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input registerRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.User{
		Username: input.Username,
		Password: input.Password,
		Email:    input.Email,
		Roles:    make([]string, 0),
	}
	for _, role := range input.Role {
		user.Roles = append(user.Roles, role)
	}

	_, err := h.AuthService.CreateUser(c.Request.Context(), user)
	if err != nil {
		if errors.Is(err, app_errors.ErrUserExists) || errors.Is(err, app_errors.ErrIncorrectPassword) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		h.log.ErrorErr("error handling register user", err, c)

		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "registration success"})
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input loginRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, err := h.AuthService.LoginUser(c.Request.Context(), input.Username, input.Password)
	if err != nil {
		if errors.Is(err, app_errors.ErrUserNotFound) || errors.Is(err, app_errors.ErrIncorrectPassword) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		h.log.ErrorErr("Error handling login user", err, c)
		return
	}

	response := loginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	c.JSON(http.StatusOK, response)

}

type tokenRefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type tokenRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var input tokenRefreshRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenPair, err := h.AuthService.RefreshTokens(c.Request.Context(), input.RefreshToken)
	if err != nil {
		if errors.Is(err, app_errors.ErrUserNotFound) || errors.Is(err, app_errors.ErrTokenExpired) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tokenRefreshResponse{
		AccessToken:  tokenPair.AccessToken.Raw,
		RefreshToken: tokenPair.RefreshToken.Raw,
	})

}
