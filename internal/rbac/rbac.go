package rbac

import (
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/models"
)

// UserPermissions resolves the effective permission set for a user: the union of
// the role's permissions and any direct user_permission grants, minus explicit
// revokes. It returns a set for O(1) checks.
func UserPermissions(db *gorm.DB, uid uint) (map[string]bool, error) {
	var user models.User
	if err := db.First(&user, uid).Error; err != nil {
		return nil, err
	}
	set := map[string]bool{}

	// Role permissions.
	var rolePerms []models.RolePermission
	if err := db.Where("role_code = ?", user.Role).Find(&rolePerms).Error; err != nil {
		return nil, err
	}
	for _, rp := range rolePerms {
		set[rp.PermissionCode] = true
	}

	// Direct user grants / revokes (layered on top of the role).
	var userPerms []models.UserPermission
	if err := db.Where("user_id = ?", uid).Find(&userPerms).Error; err != nil {
		return nil, err
	}
	for _, up := range userPerms {
		if up.Grant {
			set[up.PermissionCode] = true
		} else {
			delete(set, up.PermissionCode)
		}
	}
	return set, nil
}

// HasPermission reports whether the user holds the given permission.
func HasPermission(db *gorm.DB, uid uint, code string) bool {
	set, err := UserPermissions(db, uid)
	if err != nil {
		return false
	}
	return set[code]
}
