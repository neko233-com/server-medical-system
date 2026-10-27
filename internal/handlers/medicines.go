package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/config"
	"github.com/neko233-com/server-medical-system/internal/i18n"
	"github.com/neko233-com/server-medical-system/internal/middleware"
	"github.com/neko233-com/server-medical-system/internal/models"
)

// MedicineHandler manages the drug formulary (admin-extensible).
type MedicineHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// RegisterMedicines wires medicine routes. Create/Update/Delete require admin.
func RegisterMedicines(r gin.IRouter, db *gorm.DB, cfg *config.Config) {
	h := &MedicineHandler{db: db, cfg: cfg}
	r.GET("/medicines", h.List)
	r.POST("/medicines", middleware.AdminRequired(), h.Create)
	r.PUT("/medicines/:id", middleware.AdminRequired(), h.Update)
	r.DELETE("/medicines/:id", middleware.AdminRequired(), h.Delete)
	// Pricing is just the price field, but a dedicated route documents intent.
	r.PUT("/medicines/:id/price", middleware.AdminRequired(), h.UpdatePrice)
}

type medicineReq struct {
	Code         string  `json:"code"`
	NameZH       string  `json:"name_zh"`
	NameEN       string  `json:"name_en"`
	Category     string  `json:"category"`
	CategoryEN   string  `json:"category_en"`
	Spec         string  `json:"spec"`
	Manufacturer string  `json:"manufacturer"`
	Unit         string  `json:"unit"`
	Price        float64 `json:"price"`
	Currency     string  `json:"currency"`
	IsActive     *bool   `json:"is_active"`
}

// List returns paginated medicines, filterable by keyword and category.
func (h *MedicineHandler) List(c *gin.Context) {
	q := c.Query("q")
	category := c.Query("category")
	page, pageSize := getPagination(c)
	var total int64
	query := h.db.Model(&models.Medicine{})
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("name_zh LIKE ? OR name_en LIKE ? OR code LIKE ?", like, like, like)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	query.Count(&total)
	var items []models.Medicine
	paginate(query.Order("category, id"), page, pageSize).Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "pagination": Pagination{Page: page, PageSize: pageSize, Total: total}})
}

// Create adds a new medicine to the formulary (admin only).
func (h *MedicineHandler) Create(c *gin.Context) {
	var req medicineReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.NameZH == "" && req.NameEN == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "name_required")})
		return
	}
	var dup int64
	h.db.Model(&models.Medicine{}).Where("code = ?", req.Code).Count(&dup)
	if req.Code != "" && dup > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": i18n.T(c, "medicine_code_exists")})
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	currency := req.Currency
	if currency == "" {
		currency = "CNY"
	}
	med := models.Medicine{
		Code:         req.Code,
		NameZH:       req.NameZH,
		NameEN:       req.NameEN,
		Category:     req.Category,
		CategoryEN:   req.CategoryEN,
		Spec:         req.Spec,
		Manufacturer: req.Manufacturer,
		Unit:         req.Unit,
		Price:        req.Price,
		Currency:     currency,
		IsActive:     active,
	}
	if err := h.db.Create(&med).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": med, "message": i18n.T(c, "created")})
}

// Update edits a medicine (admin only).
func (h *MedicineHandler) Update(c *gin.Context) {
	var med models.Medicine
	if err := h.db.First(&med, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "medicine_not_found")})
		return
	}
	var req medicineReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.NameZH != "" {
		med.NameZH = req.NameZH
	}
	if req.NameEN != "" {
		med.NameEN = req.NameEN
	}
	med.Category = req.Category
	med.CategoryEN = req.CategoryEN
	med.Spec = req.Spec
	med.Manufacturer = req.Manufacturer
	med.Unit = req.Unit
	if req.Price >= 0 {
		med.Price = req.Price
	}
	if req.Currency != "" {
		med.Currency = req.Currency
	}
	if req.IsActive != nil {
		med.IsActive = *req.IsActive
	}
	if err := h.db.Save(&med).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": med, "message": i18n.T(c, "updated")})
}

// UpdatePrice is a focused endpoint for configuring medicine pricing (admin only).
func (h *MedicineHandler) UpdatePrice(c *gin.Context) {
	var med models.Medicine
	if err := h.db.First(&med, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "medicine_not_found")})
		return
	}
	var body struct {
		Price    float64 `json:"price"`
		Currency string  `json:"currency"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if body.Price >= 0 {
		med.Price = body.Price
	}
	if body.Currency != "" {
		med.Currency = body.Currency
	}
	if err := h.db.Save(&med).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": med, "message": i18n.T(c, "updated")})
}

// Delete soft-removes a medicine (admin only).
func (h *MedicineHandler) Delete(c *gin.Context) {
	if err := h.db.Delete(&models.Medicine{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "deleted")})
}
