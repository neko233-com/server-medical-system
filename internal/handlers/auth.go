package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/config"
	"github.com/neko233-com/server-medical-system/internal/i18n"
	"github.com/neko233-com/server-medical-system/internal/middleware"
	"github.com/neko233-com/server-medical-system/internal/models"
)

// AuthHandler exposes login / current-user endpoints.
type AuthHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// RegisterAuth wires auth routes onto the given router group.
func RegisterAuth(r gin.IRouter, db *gorm.DB, cfg *config.Config) {
	h := &AuthHandler{db: db, cfg: cfg}
	r.POST("/auth/login", h.Login)
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login validates credentials and returns a JWT plus the user profile.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "username_required")})
		return
	}
	if req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "password_required")})
		return
	}
	var user models.User
	if err := h.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": i18n.T(c, "invalid_credentials")})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": i18n.T(c, "invalid_credentials")})
		return
	}
	token, err := middleware.GenerateToken(h.cfg, &user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

// Me returns the currently authenticated user. Mounted with AuthRequired.
func Me(c *gin.Context, db *gorm.DB) {
	uid, _ := c.Get("uid")
	var user models.User
	if err := db.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	c.JSON(http.StatusOK, user)
}
