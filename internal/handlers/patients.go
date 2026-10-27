package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/i18n"
	"github.com/neko233-com/server-medical-system/internal/middleware"
	"github.com/neko233-com/server-medical-system/internal/models"
)

// PatientHandler manages patient demographics.
type PatientHandler struct {
	db *gorm.DB
}

// RegisterPatients wires patient CRUD routes with RBAC permission checks.
func RegisterPatients(r gin.IRouter, db *gorm.DB) {
	h := &PatientHandler{db: db}
	r.GET("/patients", middleware.RequireAny(db, "patient:read"), h.List)
	r.POST("/patients", middleware.RequireAny(db, "patient:write"), h.Create)
	r.GET("/patients/:id", middleware.RequireAny(db, "patient:read"), h.Get)
	r.PUT("/patients/:id", middleware.RequireAny(db, "patient:write"), h.Update)
	r.DELETE("/patients/:id", middleware.RequireAny(db, "patient:write"), h.Delete)
}

type patientReq struct {
	Name      string `json:"name"`
	Gender    string `json:"gender"`
	BirthDate string `json:"birth_date"`
	Phone     string `json:"phone"`
	IDCard    string `json:"id_card"`
	Address   string `json:"address"`
}

// List returns paginated patients, optionally filtered by a search term.
func (h *PatientHandler) List(c *gin.Context) {
	q := c.Query("q")
	page, pageSize := getPagination(c)
	var total int64
	query := h.db.Model(&models.Patient{})
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("name LIKE ? OR phone LIKE ? OR id_card LIKE ?", like, like, like)
	}
	query.Count(&total)
	var items []models.Patient
	paginate(query.Order("id desc"), page, pageSize).Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "pagination": Pagination{Page: page, PageSize: pageSize, Total: total}})
}

// Get returns a single patient.
func (h *PatientHandler) Get(c *gin.Context) {
	var patient models.Patient
	if err := h.db.First(&patient, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "patient_not_found")})
		return
	}
	c.JSON(http.StatusOK, patient)
}

// Create registers a new patient.
func (h *PatientHandler) Create(c *gin.Context) {
	var req patientReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "name_required")})
		return
	}
	patient := models.Patient{
		Name:      req.Name,
		Gender:    req.Gender,
		BirthDate: req.BirthDate,
		Phone:     req.Phone,
		IDCard:    req.IDCard,
		Address:   req.Address,
	}
	if err := h.db.Create(&patient).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": patient, "message": i18n.T(c, "created")})
}

// Update modifies an existing patient.
func (h *PatientHandler) Update(c *gin.Context) {
	var patient models.Patient
	if err := h.db.First(&patient, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "patient_not_found")})
		return
	}
	var req patientReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.Name != "" {
		patient.Name = req.Name
	}
	patient.Gender = req.Gender
	patient.BirthDate = req.BirthDate
	patient.Phone = req.Phone
	patient.IDCard = req.IDCard
	patient.Address = req.Address
	if err := h.db.Save(&patient).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": patient, "message": i18n.T(c, "updated")})
}

// Delete soft-deletes a patient.
func (h *PatientHandler) Delete(c *gin.Context) {
	if err := h.db.Delete(&models.Patient{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "deleted")})
}
