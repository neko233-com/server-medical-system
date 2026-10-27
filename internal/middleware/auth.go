package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/config"
	"github.com/neko233-com/server-medical-system/internal/models"
	"github.com/neko233-com/server-medical-system/internal/rbac"
)

// errInvalidToken is returned when a token fails validation.
var errInvalidToken = errors.New("invalid token")

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

// TokenUserID validates a raw JWT string and returns the user id it carries.
//
// WebSocket handshakes cannot set custom HTTP headers from the browser, so the
// client passes the token as a query parameter instead; this helper lets the
// upgrade handler authenticate it with exactly the same logic as HTTP requests.
func TokenUserID(cfg *config.Config, tokenStr string) (uint, error) {
	claims, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || !claims.Valid {
		return 0, errInvalidToken
	}
	mapClaims, ok := claims.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errInvalidToken
	}
	uid, _ := mapClaims["uid"].(float64)
	if uid == 0 {
		return 0, errInvalidToken
	}
	return uint(uid), nil
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

// RequirePermission aborts with 403 unless the authenticated user holds `code`.
// Mount after AuthRequired.
func RequirePermission(db *gorm.DB, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, _ := c.Get("uid")
		if rbac.HasPermission(db, uid.(uint), code) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	}
}

// RequireAny aborts with 403 unless the user holds at least one of the codes.
// Mount after AuthRequired.
func RequireAny(db *gorm.DB, codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, _ := c.Get("uid")
		for _, code := range codes {
			if rbac.HasPermission(db, uid.(uint), code) {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	}
}
