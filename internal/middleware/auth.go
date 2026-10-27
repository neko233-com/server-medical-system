package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/neko233-com/server-medical-system/internal/config"
	"github.com/neko233-com/server-medical-system/internal/models"
)

// CORS allows the SPA dev server (and configured origins) to call the API.
func CORS(allowed string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", allowed)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// GenerateToken mints a short-lived JWT for the given user.
func GenerateToken(cfg *config.Config, user *models.User) (string, error) {
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":  user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(72 * time.Hour).Unix(),
	})
	return claims.SignedString([]byte(cfg.JWTSecret))
}

// AuthRequired validates the bearer token and stores uid/role in context.
func AuthRequired(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		hdr := c.GetHeader("Authorization")
		if hdr == "" || !strings.HasPrefix(hdr, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		tokenStr := strings.TrimPrefix(hdr, "Bearer ")
		claims, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !claims.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		mapClaims, ok := claims.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		uid, _ := mapClaims["uid"].(float64)
		role, _ := mapClaims["role"].(string)
		c.Set("uid", uint(uid))
		c.Set("role", role)
		c.Next()
	}
}

// AdminRequired ensures the caller has the admin role. Mount after AuthRequired.
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != string(models.RoleAdmin) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}
