package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"loteria-backend/internal/config"
)

// AuthRequired valida el Bearer token y deja userID y rol en el contexto.
func AuthRequired(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no autorizado"})
			return
		}

		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token vacio"})
			return
		}

		token, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalido"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token sin claims"})
			return
		}

		uid, ok := claims["uid"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token sin usuario"})
			return
		}
		rol, _ := claims["rol"].(string)

		c.Set("userID", uint(uid))
		c.Set("rol", rol)
		c.Next()
	}
}

// RoleRequired restringe el acceso a los roles indicados.
func RoleRequired(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(c *gin.Context) {
		rol, _ := c.Get("rol")
		if !allowed[rol.(string)] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "sin permisos"})
			return
		}
		c.Next()
	}
}
