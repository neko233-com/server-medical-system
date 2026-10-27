package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/i18n"
	"github.com/neko233-com/server-medical-system/internal/middleware"
	"github.com/neko233-com/server-medical-system/internal/models"
)

// CaseHandler manages consultation / visit records.
type CaseHandler struct {
	db *gorm.DB
}

// RegisterCases wires medical-case routes with RBAC permission checks.
func RegisterCases(r gin.IRouter, db *gorm.DB) {
	h := &CaseHandler{db: db}
	r.GET("/cases", middleware.RequireAny(db, "case:read"), h.List)
	r.POST("/cases", middleware.RequireAny(db, "case:write"), h.Create)
	r.GET("/cases/:id", middleware.RequireAny(db, "case:read"), h.Get)
	r.PUT("/cases/:id", middleware.RequireAny(db, "case:write"), h.Update)
}

type caseReq struct {
	PatientID      uint   `json:"patient_id"`
	RoomID         uint   `json:"room_id"`
	VisitDate      string `json:"visit_date"` // RFC3339; optional
	ChiefComplaint string `json:"chief_complaint"`
	PresentIllness string `json:"present_illness"`
	PastHistory    string `json:"past_history"`
	Diagnosis      string `json:"diagnosis"`
	Notes          string `json:"notes"`
}

// List returns cases, optionally filtered by patient.
func (h *CaseHandler) List(c *gin.Context) {
	patientID := c.Query("patient_id")
	page, pageSize := getPagination(c)
	var total int64
	query := h.db.Model(&models.MedicalCase{})
	if patientID != "" {
		query = query.Where("patient_id = ?", patientID)
	}
	query.Count(&total)
	var items []models.MedicalCase
	paginate(query.Order("id desc"), page, pageSize).Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "pagination": Pagination{Page: page, PageSize: pageSize, Total: total}})
}

// Get returns a single case.
func (h *CaseHandler) Get(c *gin.Context) {
	var medCase models.MedicalCase
	if err := h.db.First(&medCase, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "case_not_found")})
		return
	}
	c.JSON(http.StatusOK, medCase)
}

// Create records a new consultation. Caller identity comes from the JWT.
func (h *CaseHandler) Create(c *gin.Context) {
	var req caseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.PatientID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "patient_not_found")})
		return
	}
	uid, _ := c.Get("uid")
	role, _ := c.Get("role")
	var doctor models.User
	h.db.First(&doctor, uid)
	visit := time.Now()
	if req.VisitDate != "" {
		if t, err := time.Parse(time.RFC3339, req.VisitDate); err == nil {
			visit = t
		}
	}
	medCase := models.MedicalCase{
		PatientID:      req.PatientID,
		DoctorID:       doctor.ID,
		DoctorName:     doctor.Name,
		RoomID:         req.RoomID,
		VisitDate:      visit,
		ChiefComplaint: req.ChiefComplaint,
		PresentIllness: req.PresentIllness,
		PastHistory:    req.PastHistory,
		Diagnosis:      req.Diagnosis,
		Notes:          req.Notes,
		CaseNo:         generateCaseNo(visit, doctor.ID),
	}
	if req.RoomID != 0 {
		var room models.Room
		if h.db.First(&room, req.RoomID).Error == nil {
			medCase.RoomNo = room.RoomNo
		}
	}
	_ = role
	if err := h.db.Create(&medCase).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": medCase, "message": i18n.T(c, "created")})
}

// Update edits an existing case.
func (h *CaseHandler) Update(c *gin.Context) {
	var medCase models.MedicalCase
	if err := h.db.First(&medCase, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "case_not_found")})
		return
	}
	var req caseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	medCase.ChiefComplaint = req.ChiefComplaint
	medCase.PresentIllness = req.PresentIllness
	medCase.PastHistory = req.PastHistory
	medCase.Diagnosis = req.Diagnosis
	medCase.Notes = req.Notes
	if err := h.db.Save(&medCase).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": medCase, "message": i18n.T(c, "updated")})
}

// generateCaseNo builds a readable, unique-ish case number.
func generateCaseNo(t time.Time, doctorID uint) string {
	return t.Format("20060102") + "-" + pad(int(doctorID)) + "-" + randomSuffix()
}
