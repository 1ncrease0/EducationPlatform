package controllers

import (
	"SkillForge/internal/app_errors"
	"SkillForge/pkg/logger"
	_ "encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

const (
	ClientIDCtx    = "client_id"
	ClientRolesCtx = "client_roles"
)

func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	roleSet := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		roleSet[r] = struct{}{}
	}
	return func(c *gin.Context) {
		raw, exists := c.Get(ClientRolesCtx)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "roles not found"})
			return
		}

		roles, ok := raw.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid roles format"})
			return
		}

		for _, role := range roles {
			if _, allowed := roleSet[role]; allowed {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}

func (h *AuthHandler) AuthMiddleware(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	var token string
	if parts := strings.Split(authHeader, "Bearer "); len(parts) == 2 {
		token = parts[1]
	}
	if token == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	parsedToken, err := h.AuthService.ParseToken(c.Request.Context(), token)
	if err != nil {
		h.log.Info("failed to parse token", err)
		if errors.Is(err, app_errors.ErrTokenExpired) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": app_errors.ErrTokenExpired.Error()})
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "cant parse token"})
		return
	}
	if !h.AuthService.IsAccessToken(c.Request.Context(), parsedToken) {
		//h.log.Error("not access")\
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not access token"})

		return
	}

	userID, roles, err := h.AuthService.AccessClaims(c.Request.Context(), token)
	if err != nil {
		//h.log.Error("claims")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	user, err := h.AuthService.User(c.Request.Context(), userID)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.Set(ClientIDCtx, user.ID)
	c.Set(ClientRolesCtx, roles)
	c.Next()
}

func LoggingMiddleware(logger logger.Log) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		if rawQuery != "" {
			path = fmt.Sprintf("%s?%s", path, rawQuery)
		}
		status := c.Writer.Status()

		msg := fmt.Sprintf("%s %s", method, path)

		logger.Info(msg,
			"status", status,
			"latency", latency,
			"client_ip", clientIP,
		)

		for _, ginErr := range c.Errors {

			logger.ErrorErr("HTTP request error", ginErr.Err,
				"status", status,
				"method", method,
				"path", path,
			)
		}
	}
}
