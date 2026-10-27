package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/i18n"
	"github.com/neko233-com/server-medical-system/internal/middleware"
	"github.com/neko233-com/server-medical-system/internal/models"
)

// RBACHandler exposes roles, permissions and user-account management.
type RBACHandler struct {
	db *gorm.DB
}

// RegisterRBAC wires RBAC routes onto the router group. Each route declares its
// own permission requirement via RequirePermission / RequireAny.
func RegisterRBAC(r gin.IRouter, db *gorm.DB) {
	h := &RBACHandler{db: db}

	// Permission catalog (readable by any authenticated user).
	r.GET("/rbac/permissions", h.ListPermissions)
	// Staff directory: minimal user info for selection dropdowns across modules.
	// Available to any authenticated user (operational lookup, not account admin).
	r.GET("/rbac/directory", h.Directory)

	// Roles.
	r.GET("/rbac/roles", middleware.RequireAny(db, "role:read"), h.ListRoles)
	r.POST("/rbac/roles", middleware.RequireAny(db, "role:manage"), h.CreateRole)
	r.PUT("/rbac/roles/:code", middleware.RequireAny(db, "role:manage"), h.UpdateRole)
	r.DELETE("/rbac/roles/:code", middleware.RequireAny(db, "role:manage"), h.DeleteRole)
	r.GET("/rbac/roles/:code/permissions", middleware.RequireAny(db, "role:read"), h.GetRolePermissions)
	r.PUT("/rbac/roles/:code/permissions", middleware.RequireAny(db, "role:manage"), h.SetRolePermissions)

	// Users.
	r.GET("/rbac/users", middleware.RequireAny(db, "user:read"), h.ListUsers)
	r.POST("/rbac/users", middleware.RequireAny(db, "user:manage"), h.CreateUser)
	r.PUT("/rbac/users/:id", middleware.RequireAny(db, "user:manage"), h.UpdateUser)
	r.DELETE("/rbac/users/:id", middleware.RequireAny(db, "user:manage"), h.DeleteUser)
	r.PUT("/rbac/users/:id/role", middleware.RequireAny(db, "user:manage"), h.AssignRole)
	r.GET("/rbac/users/:id/permissions", middleware.RequireAny(db, "user:read"), h.GetUserPermissions)
	r.PUT("/rbac/users/:id/permissions", middleware.RequireAny(db, "user:manage"), h.SetUserPermissions)
}

// ---------------------------------------------------------------------------
// Permissions
// ---------------------------------------------------------------------------

// ListPermissions returns the full permission catalog grouped by `group`.
func (h *RBACHandler) ListPermissions(c *gin.Context) {
	var perms []models.Permission
	h.db.Order("`group`, code").Find(&perms)
	c.JSON(http.StatusOK, perms)
}

// Directory returns minimal user info (id/name/role/title/department) for use in
// selection dropdowns (e.g. assigning surgeons, assistants, room doctors).
func (h *RBACHandler) Directory(c *gin.Context) {
	var users []models.User
	h.db.Order("id").Find(&users)
	out := make([]gin.H, 0, len(users))
	for _, u := range users {
		out = append(out, gin.H{
			"id":         u.ID,
			"name":       u.Name,
			"role":       u.Role,
			"title":      u.Title,
			"department": u.Department,
			"status":     u.Status,
		})
	}
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// Roles
// ---------------------------------------------------------------------------

// ListRoles returns all roles with their permission codes attached.
func (h *RBACHandler) ListRoles(c *gin.Context) {
	var roles []models.Role
	h.db.Order("sort, id").Find(&roles)
	out := make([]gin.H, 0, len(roles))
	for _, role := range roles {
		var rps []models.RolePermission
		h.db.Where("role_code = ?", role.Code).Find(&rps)
		codes := make([]string, 0, len(rps))
		for _, rp := range rps {
			codes = append(codes, rp.PermissionCode)
		}
		out = append(out, gin.H{
			"id":          role.ID,
			"code":        role.Code,
			"name_zh":     role.NameZH,
			"name_en":     role.NameEN,
			"description": role.Description,
			"is_system":   role.IsSystem,
			"sort":        role.Sort,
			"permissions": codes,
		})
	}
	c.JSON(http.StatusOK, out)
}

type roleReq struct {
	Code        string `json:"code"`
	NameZH      string `json:"name_zh"`
	NameEN      string `json:"name_en"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
}

// CreateRole adds a new (non-system) role.
func (h *RBACHandler) CreateRole(c *gin.Context) {
	var req roleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.Code == "" || req.NameZH == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "name_required")})
		return
	}
	var dup int64
	h.db.Model(&models.Role{}).Where("code = ?", req.Code).Count(&dup)
	if dup > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": i18n.T(c, "role_code_exists")})
		return
	}
	role := models.Role{
		Code:        req.Code,
		NameZH:      req.NameZH,
		NameEN:      req.NameEN,
		Description: req.Description,
		IsSystem:    false,
		Sort:        req.Sort,
	}
	if err := h.db.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": role, "message": i18n.T(c, "created")})
}

// UpdateRole edits a role's metadata (not its code).
func (h *RBACHandler) UpdateRole(c *gin.Context) {
	var role models.Role
	if err := h.db.Where("code = ?", c.Param("code")).First(&role).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "role_not_found")})
		return
	}
	var req roleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.NameZH != "" {
		role.NameZH = req.NameZH
	}
	role.NameEN = req.NameEN
	role.Description = req.Description
	role.Sort = req.Sort
	if err := h.db.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": role, "message": i18n.T(c, "updated")})
}

// DeleteRole removes a non-system role.
func (h *RBACHandler) DeleteRole(c *gin.Context) {
	var role models.Role
	if err := h.db.Where("code = ?", c.Param("code")).First(&role).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "role_not_found")})
		return
	}
	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": i18n.T(c, "cannot_delete_system_role")})
		return
	}
	h.db.Where("role_code = ?", role.Code).Delete(&models.RolePermission{})
	h.db.Delete(&role)
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "deleted")})
}

// GetRolePermissions returns the permission codes assigned to a role.
func (h *RBACHandler) GetRolePermissions(c *gin.Context) {
	var rps []models.RolePermission
	h.db.Where("role_code = ?", c.Param("code")).Find(&rps)
	codes := make([]string, 0, len(rps))
	for _, rp := range rps {
		codes = append(codes, rp.PermissionCode)
	}
	c.JSON(http.StatusOK, gin.H{"role_code": c.Param("code"), "permissions": codes})
}

// SetRolePermissions replaces a role's permission set.
func (h *RBACHandler) SetRolePermissions(c *gin.Context) {
	var role models.Role
	if err := h.db.Where("code = ?", c.Param("code")).First(&role).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "role_not_found")})
		return
	}
	var body struct {
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	h.db.Where("role_code = ?", role.Code).Delete(&models.RolePermission{})
	if len(body.Permissions) > 0 {
		bulk := make([]models.RolePermission, 0, len(body.Permissions))
		for _, code := range body.Permissions {
			bulk = append(bulk, models.RolePermission{RoleCode: role.Code, PermissionCode: code})
		}
		h.db.Create(&bulk)
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "updated"), "permissions": body.Permissions})
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

// ListUsers returns accounts (password hash omitted).
func (h *RBACHandler) ListUsers(c *gin.Context) {
	var users []models.User
	h.db.Order("id").Find(&users)
	c.JSON(http.StatusOK, users)
}

type userReq struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Department string `json:"department"`
	Title      string `json:"title"`
	Status     string `json:"status"`
}

// CreateUser adds a new account.
func (h *RBACHandler) CreateUser(c *gin.Context) {
	var req userReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.Username == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "name_required")})
		return
	}
	var dup int64
	h.db.Model(&models.User{}).Where("username = ?", req.Username).Count(&dup)
	if dup > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": i18n.T(c, "username_exists")})
		return
	}
	role := req.Role
	if role == "" {
		role = models.RoleDoctor
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	user := models.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		Name:         req.Name,
		Role:         role,
		Department:   req.Department,
		Title:        req.Title,
		Status:       status,
	}
	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": user, "message": i18n.T(c, "created")})
}

// UpdateUser edits an account (role/status excluded here — see AssignRole).
func (h *RBACHandler) UpdateUser(c *gin.Context) {
	var user models.User
	if err := h.db.First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "user_not_found")})
		return
	}
	var req userReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Department != "" {
		user.Department = req.Department
	}
	user.Title = req.Title
	if req.Status != "" {
		user.Status = req.Status
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
			return
		}
		user.PasswordHash = string(hash)
	}
	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": user, "message": i18n.T(c, "updated")})
}

// DeleteUser disables an account (cannot disable yourself).
func (h *RBACHandler) DeleteUser(c *gin.Context) {
	uid, _ := c.Get("uid")
	if uid.(uint) == atoui(c.Param("id")) {
		c.JSON(http.StatusForbidden, gin.H{"error": i18n.T(c, "cannot_disable_self")})
		return
	}
	var user models.User
	if err := h.db.First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "user_not_found")})
		return
	}
	h.db.Delete(&user)
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "deleted")})
}

// AssignRole sets the role code for a user.
func (h *RBACHandler) AssignRole(c *gin.Context) {
	var user models.User
	if err := h.db.First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "user_not_found")})
		return
	}
	var body struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Role == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	user.Role = body.Role
	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": user, "message": i18n.T(c, "updated")})
}

// GetUserPermissions returns direct (per-user) permission grants.
func (h *RBACHandler) GetUserPermissions(c *gin.Context) {
	var ups []models.UserPermission
	h.db.Where("user_id = ?", c.Param("id")).Find(&ups)
	c.JSON(http.StatusOK, ups)
}

// SetUserPermissions replaces a user's direct permission grants.
func (h *RBACHandler) SetUserPermissions(c *gin.Context) {
	var body struct {
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	h.db.Where("user_id = ?", c.Param("id")).Delete(&models.UserPermission{})
	if len(body.Permissions) > 0 {
		bulk := make([]models.UserPermission, 0, len(body.Permissions))
		for _, code := range body.Permissions {
			bulk = append(bulk, models.UserPermission{UserID: atoui(c.Param("id")), PermissionCode: code, Grant: true})
		}
		h.db.Create(&bulk)
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "updated"), "permissions": body.Permissions})
}
