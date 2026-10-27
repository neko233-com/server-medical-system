package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/i18n"
	"github.com/neko233-com/server-medical-system/internal/middleware"
	"github.com/neko233-com/server-medical-system/internal/models"
)

// PrescriptionHandler issues and reads prescriptions ("开药").
type PrescriptionHandler struct {
	db *gorm.DB
}

// RegisterPrescriptions wires prescription routes with RBAC permission checks.
func RegisterPrescriptions(r gin.IRouter, db *gorm.DB) {
	h := &PrescriptionHandler{db: db}
	r.GET("/prescriptions", middleware.RequireAny(db, "prescription:read"), h.List)
	r.POST("/prescriptions", middleware.RequireAny(db, "prescription:create"), h.Create)
	r.GET("/prescriptions/:id", middleware.RequireAny(db, "prescription:read"), h.Get)
}

type rxItemReq struct {
	MedicineID uint   `json:"medicine_id"`
	Quantity   int    `json:"quantity"`
	Dosage     string `json:"dosage"`
	Frequency  string `json:"frequency"`
}

type prescriptionReq struct {
	PatientID uint        `json:"patient_id"`
	CaseID    uint        `json:"case_id"`
	RoomID    uint        `json:"room_id"`
	Items     []rxItemReq `json:"items"`
	Status    string      `json:"status"`
}

// List returns prescriptions, optionally filtered by patient.
func (h *PrescriptionHandler) List(c *gin.Context) {
	patientID := c.Query("patient_id")
	page, pageSize := getPagination(c)
	var total int64
	query := h.db.Model(&models.Prescription{})
	if patientID != "" {
		query = query.Where("patient_id = ?", patientID)
	}
	query.Count(&total)
	var items []models.Prescription
	paginate(query.Order("id desc"), page, pageSize).Preload("Items").Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "pagination": Pagination{Page: page, PageSize: pageSize, Total: total}})
}

// Get returns a prescription with its line items.
func (h *PrescriptionHandler) Get(c *gin.Context) {
	var rx models.Prescription
	if err := h.db.Preload("Items").First(&rx, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	c.JSON(http.StatusOK, rx)
}

// Create issues a prescription, computing line subtotals and the total.
func (h *PrescriptionHandler) Create(c *gin.Context) {
	var req prescriptionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.PatientID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "patient_not_found")})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "validation_error")})
		return
	}
	uid, _ := c.Get("uid")
	var doctor models.User
	h.db.First(&doctor, uid)

	now := time.Now()
	rx := models.Prescription{
		PrescriptionNo: fmt.Sprintf("RX%s%04d", now.Format("20060102"), now.Nanosecond()%10000),
		PatientID:      req.PatientID,
		DoctorID:       doctor.ID,
		DoctorName:     doctor.Name,
		CaseID:         req.CaseID,
		RoomID:         req.RoomID,
		Currency:       "CNY",
		Status:         req.Status,
	}
	if req.RoomID != 0 {
		var room models.Room
		if h.db.First(&room, req.RoomID).Error == nil {
			rx.RoomNo = room.RoomNo
		}
	}
	if rx.Status == "" {
		rx.Status = "issued"
	}

	var total float64
	for _, it := range req.Items {
		if it.MedicineID == 0 || it.Quantity <= 0 {
			continue
		}
		var med models.Medicine
		if err := h.db.First(&med, it.MedicineID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "medicine_not_found")})
			return
		}
		name := med.NameZH
		if name == "" {
			name = med.NameEN
		}
		subtotal := med.Price * float64(it.Quantity)
		total += subtotal
		rx.Items = append(rx.Items, models.PrescriptionItem{
			MedicineID:   med.ID,
			MedicineName: name,
			Spec:         med.Spec,
			Quantity:     it.Quantity,
			Dosage:       it.Dosage,
			Frequency:    it.Frequency,
			Price:        med.Price,
			Subtotal:     subtotal,
		})
		rx.Currency = med.Currency
	}
	rx.TotalAmount = total

	if err := h.db.Create(&rx).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": rx, "message": i18n.T(c, "prescription_created")})
}
