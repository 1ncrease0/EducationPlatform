package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
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
