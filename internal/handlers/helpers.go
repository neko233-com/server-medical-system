package handlers

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// atoui parses a path/query parameter into a uint, returning 0 on failure.
func atoui(s string) uint {
	n, _ := strconv.ParseUint(s, 10, 64)
	return uint(n)
}

// Pagination is returned alongside list endpoints.
type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// getPagination reads and clamps page/page_size from the query string.
func getPagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

// paginate applies offset/limit for the given page.
func paginate(db *gorm.DB, page, pageSize int) *gorm.DB {
	return db.Offset((page - 1) * pageSize).Limit(pageSize)
}

// currentUser returns the authenticated user's id and role.
func currentUser(c *gin.Context) (uint, string) {
	uid, _ := c.Get("uid")
	role, _ := c.Get("role")
	uidVal, _ := uid.(uint)
	roleVal, _ := role.(string)
	return uidVal, roleVal
}

// pad left-pads an int to at least 2 digits for readable identifiers.
func pad(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// randomSuffix returns a short pseudo-random alphanumeric suffix.
func randomSuffix() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 4)
	for i := range b {
		b[i] = chars[(int(time.Now().UnixNano())>>(i*5))%len(chars)]
	}
	return string(b)
}

// generateNo builds a human-readable business number with the given prefix,
// e.g. generateNo("APT") -> "APT20261028-4821".
//
// The format is `<PREFIX><YYYYMMDD>-<random>` which keeps numbers sortable by
// creation date while remaining unique across concurrent inserts.
func generateNo(prefix string) string {
	return prefix + time.Now().Format("20060102") + "-" + randomSuffix()
}
