package handlers

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/i18n"
	"github.com/neko233-com/server-medical-system/internal/middleware"
	"github.com/neko233-com/server-medical-system/internal/models"
)

// =============================================================================
//  AppointmentHandler — 预约挂号
// =============================================================================
//  覆盖门诊入口：预约建档 → 到诊确认 → 完成/取消/爽约。
//  状态机：booked → arrived → completed，旁路为 cancelled / no_show。
// =============================================================================

type AppointmentHandler struct{ db *gorm.DB }

// RegisterAppointments mounts the appointment routes.
func RegisterAppointments(r gin.IRouter, db *gorm.DB) {
	h := &AppointmentHandler{db: db}
	r.GET("/appointments", middleware.RequireAny(db, "appointment:read"), h.List)
	r.GET("/appointments/:id", middleware.RequireAny(db, "appointment:read"), h.Get)
	r.POST("/appointments", middleware.RequireAny(db, "appointment:manage"), h.Create)
	r.PUT("/appointments/:id", middleware.RequireAny(db, "appointment:manage"), h.Update)
	r.PUT("/appointments/:id/status", middleware.RequireAny(db, "appointment:manage"), h.UpdateStatus)
	r.DELETE("/appointments/:id", middleware.RequireAny(db, "appointment:manage"), h.Delete)
}

// validApptStatus enumerates the allowed appointment states.
var validApptStatus = map[string]bool{
	"booked": true, "arrived": true, "completed": true, "cancelled": true, "no_show": true,
}

type apptReq struct {
	PatientID  uint   `json:"patient_id"`
	DoctorID   uint   `json:"doctor_id"`
	Department string `json:"department"`
	RoomID     uint   `json:"room_id"`
	SlotAt     string `json:"slot_at"` // RFC3339
	Source     string `json:"source"`
	ApptType   string `json:"appt_type"`
	Reason     string `json:"reason"`
	Notes      string `json:"notes"`
}

// List returns appointments, filterable by patient / doctor / status / date.
func (h *AppointmentHandler) List(c *gin.Context) {
	page, pageSize := getPagination(c)
	var total int64
	q := h.db.Model(&models.Appointment{})
	if v := c.Query("patient_id"); v != "" {
		q = q.Where("patient_id = ?", atoui(v))
	}
	if v := c.Query("doctor_id"); v != "" {
		q = q.Where("doctor_id = ?", atoui(v))
	}
	if v := c.Query("status"); v != "" {
		q = q.Where("status = ?", v)
	}
	if v := c.Query("date"); v != "" {
		// 按自然日过滤，兼容三种数据库（不使用方言日期函数）
		if d, err := time.Parse("2006-01-02", v); err == nil {
			next := d.AddDate(0, 0, 1)
			q = q.Where("slot_at >= ? AND slot_at < ?", d, next)
		}
	}
	q.Count(&total)
	var items []models.Appointment
	paginate(q.Order("slot_at desc"), page, pageSize).Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "pagination": Pagination{Page: page, PageSize: pageSize, Total: total}})
}

// Get returns one appointment.
func (h *AppointmentHandler) Get(c *gin.Context) {
	var a models.Appointment
	if err := h.db.First(&a, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	c.JSON(http.StatusOK, a)
}

// Create books a new appointment and generates its appointment number.
func (h *AppointmentHandler) Create(c *gin.Context) {
	var req apptReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.PatientID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "patient_required")})
		return
	}
	var p models.Patient
	if h.db.First(&p, req.PatientID).Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "patient_not_found")})
		return
	}
	a := models.Appointment{
		ApptNo:      generateNo("APT"),
		PatientID:   req.PatientID,
		PatientName: p.Name,
		DoctorID:    req.DoctorID,
		Department:  req.Department,
		RoomID:      req.RoomID,
		Source:      orDefault(req.Source, "walkin"),
		ApptType:    orDefault(req.ApptType, "general"),
		Reason:      req.Reason,
		Notes:       req.Notes,
		Status:      "booked",
	}
	// Resolve doctor / room display fields so lists render without extra joins.
	if req.DoctorID != 0 {
		var u models.User
		if h.db.First(&u, req.DoctorID).Error == nil {
			a.DoctorName = u.Name
			if a.Department == "" {
				a.Department = u.Department
			}
		}
	}
	if req.RoomID != 0 {
		var rm models.Room
		if h.db.First(&rm, req.RoomID).Error == nil {
			a.RoomNo = rm.RoomNo
		}
	}
	if req.SlotAt != "" {
		if t, err := time.Parse(time.RFC3339, req.SlotAt); err == nil {
			a.SlotAt = t
		}
	}
	if a.SlotAt.IsZero() {
		a.SlotAt = time.Now()
	}
	if err := h.db.Create(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": a, "message": i18n.T(c, "created")})
}

// Update edits an appointment.
func (h *AppointmentHandler) Update(c *gin.Context) {
	var a models.Appointment
	if err := h.db.First(&a, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	var req apptReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.DoctorID != 0 {
		a.DoctorID = req.DoctorID
		var u models.User
		if h.db.First(&u, req.DoctorID).Error == nil {
			a.DoctorName = u.Name
		}
	}
	if req.RoomID != 0 {
		a.RoomID = req.RoomID
		var rm models.Room
		if h.db.First(&rm, req.RoomID).Error == nil {
			a.RoomNo = rm.RoomNo
		}
	}
	if req.Department != "" {
		a.Department = req.Department
	}
	if req.Reason != "" {
		a.Reason = req.Reason
	}
	a.Notes = req.Notes
	if req.SlotAt != "" {
		if t, err := time.Parse(time.RFC3339, req.SlotAt); err == nil {
			a.SlotAt = t
		}
	}
	if err := h.db.Save(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": a, "message": i18n.T(c, "updated")})
}

// UpdateStatus advances the appointment lifecycle.
func (h *AppointmentHandler) UpdateStatus(c *gin.Context) {
	var a models.Appointment
	if err := h.db.First(&a, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if !validApptStatus[body.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_status")})
		return
	}
	a.Status = body.Status
	if err := h.db.Save(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": a, "message": i18n.T(c, "updated")})
}

// Delete removes an appointment.
func (h *AppointmentHandler) Delete(c *gin.Context) {
	if err := h.db.Delete(&models.Appointment{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "deleted")})
}

// =============================================================================
//  BedHandler — 病区与住院床位
// =============================================================================
//  覆盖住院场景：病区 → 床位，支持入院占床、出院释放、预留与维修态。
//  占用率由 operations 看板汇总（见 operations.go）。
// =============================================================================

type BedHandler struct{ db *gorm.DB }

// RegisterBeds mounts ward/bed routes.
func RegisterBeds(r gin.IRouter, db *gorm.DB) {
	h := &BedHandler{db: db}
	r.GET("/wards", middleware.RequireAny(db, "bed:read"), h.ListWards)
	r.POST("/wards", middleware.RequireAny(db, "bed:manage"), h.CreateWard)
	r.DELETE("/wards/:id", middleware.RequireAny(db, "bed:manage"), h.DeleteWard)

	r.GET("/beds", middleware.RequireAny(db, "bed:read"), h.ListBeds)
	r.GET("/beds/:id", middleware.RequireAny(db, "bed:read"), h.GetBed)
	r.POST("/beds", middleware.RequireAny(db, "bed:manage"), h.CreateBed)
	r.PUT("/beds/:id", middleware.RequireAny(db, "bed:manage"), h.UpdateBed)
	r.PUT("/beds/:id/admit", middleware.RequireAny(db, "bed:manage"), h.Admit)         // 入院占床
	r.PUT("/beds/:id/discharge", middleware.RequireAny(db, "bed:manage"), h.Discharge) // 出院释放
	r.PUT("/beds/:id/status", middleware.RequireAny(db, "bed:manage"), h.SetStatus)
	r.DELETE("/beds/:id", middleware.RequireAny(db, "bed:manage"), h.DeleteBed)
}

// validBedStatus enumerates bed states.
var validBedStatus = map[string]bool{
	"available": true, "occupied": true, "reserved": true, "maintenance": true,
}

// ListWards returns all wards.
func (h *BedHandler) ListWards(c *gin.Context) {
	var items []models.Ward
	h.db.Order("code").Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type wardReq struct {
	Code       string `json:"code"`
	NameZH     string `json:"name_zh"`
	NameEN     string `json:"name_en"`
	Department string `json:"department"`
	Floor      string `json:"floor"`
	DoctorID   uint   `json:"doctor_id"`
	Status     string `json:"status"`
}

// CreateWard adds an inpatient ward.
func (h *BedHandler) CreateWard(c *gin.Context) {
	var req wardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.Code == "" || req.NameZH == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "name_required")})
		return
	}
	w := models.Ward{
		Code: req.Code, NameZH: req.NameZH, NameEN: req.NameEN,
		Department: req.Department, Floor: req.Floor, DoctorID: req.DoctorID,
		Status: orDefault(req.Status, "active"),
	}
	if req.DoctorID != 0 {
		var u models.User
		if h.db.First(&u, req.DoctorID).Error == nil {
			w.DoctorName = u.Name
		}
	}
	if err := h.db.Create(&w).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": w, "message": i18n.T(c, "created")})
}

// DeleteWard removes a ward (only when it holds no beds).
func (h *BedHandler) DeleteWard(c *gin.Context) {
	var n int64
	h.db.Model(&models.Bed{}).Where("ward_id = ?", c.Param("id")).Count(&n)
	if n > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": i18n.T(c, "ward_has_beds")})
		return
	}
	if err := h.db.Delete(&models.Ward{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "deleted")})
}

// ListBeds returns beds, filterable by ward / status.
func (h *BedHandler) ListBeds(c *gin.Context) {
	page, pageSize := getPagination(c)
	var total int64
	q := h.db.Model(&models.Bed{})
	if v := c.Query("ward_id"); v != "" {
		q = q.Where("ward_id = ?", atoui(v))
	}
	if v := c.Query("status"); v != "" {
		q = q.Where("status = ?", v)
	}
	q.Count(&total)
	var items []models.Bed
	paginate(q.Order("bed_no"), page, pageSize).Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "pagination": Pagination{Page: page, PageSize: pageSize, Total: total}})
}

// GetBed returns one bed.
func (h *BedHandler) GetBed(c *gin.Context) {
	var b models.Bed
	if err := h.db.First(&b, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	c.JSON(http.StatusOK, b)
}

type bedReq struct {
	WardID    uint    `json:"ward_id"`
	BedNo     string  `json:"bed_no"`
	Status    string  `json:"status"`
	DailyRate float64 `json:"daily_rate"`
}

// CreateBed adds a bed to a ward, deriving its display code.
func (h *BedHandler) CreateBed(c *gin.Context) {
	var req bedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.WardID == 0 || req.BedNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	var w models.Ward
	if h.db.First(&w, req.WardID).Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	b := models.Bed{
		WardID:    req.WardID,
		WardName:  w.NameZH,
		BedNo:     req.BedNo,
		Code:      fmt.Sprintf("%s-%s", w.Code, req.BedNo),
		Status:    orDefault(req.Status, "available"),
		DailyRate: req.DailyRate,
	}
	if err := h.db.Create(&b).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": b, "message": i18n.T(c, "created")})
}

// UpdateBed edits a bed.
func (h *BedHandler) UpdateBed(c *gin.Context) {
	var b models.Bed
	if err := h.db.First(&b, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	var req bedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.BedNo != "" {
		b.BedNo = req.BedNo
	}
	if req.DailyRate > 0 {
		b.DailyRate = req.DailyRate
	}
	if req.Status != "" && validBedStatus[req.Status] {
		b.Status = req.Status
	}
	if err := h.db.Save(&b).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": b, "message": i18n.T(c, "updated")})
}

// Admit assigns a patient to the bed (入院占床).
func (h *BedHandler) Admit(c *gin.Context) {
	var b models.Bed
	if err := h.db.First(&b, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	if b.Status == "occupied" {
		c.JSON(http.StatusConflict, gin.H{"error": i18n.T(c, "bed_occupied")})
		return
	}
	var body struct {
		PatientID uint `json:"patient_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if body.PatientID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "patient_required")})
		return
	}
	var p models.Patient
	if h.db.First(&p, body.PatientID).Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "patient_not_found")})
		return
	}
	b.PatientID = body.PatientID
	b.PatientName = p.Name
	b.Status = "occupied"
	now := time.Now()
	b.AdmittedAt = &now
	if err := h.db.Save(&b).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": b, "message": i18n.T(c, "updated")})
}

// Discharge frees the bed (出院释放).
func (h *BedHandler) Discharge(c *gin.Context) {
	var b models.Bed
	if err := h.db.First(&b, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	b.PatientID = 0
	b.PatientName = ""
	b.AdmittedAt = nil
	b.Status = "available"
	if err := h.db.Save(&b).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": b, "message": i18n.T(c, "updated")})
}

// SetStatus switches a bed among available / reserved / maintenance.
func (h *BedHandler) SetStatus(c *gin.Context) {
	var b models.Bed
	if err := h.db.First(&b, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if !validBedStatus[body.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_status")})
		return
	}
	b.Status = body.Status
	if body.Status != "occupied" {
		b.PatientID = 0
		b.PatientName = ""
		b.AdmittedAt = nil
	}
	if err := h.db.Save(&b).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": b, "message": i18n.T(c, "updated")})
}

// DeleteBed removes a bed.
func (h *BedHandler) DeleteBed(c *gin.Context) {
	if err := h.db.Delete(&models.Bed{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "deleted")})
}

// =============================================================================
//  BillingHandler — 收费结算
// =============================================================================
//  账单：明细行 → 自动汇总金额；支持支付、退款、作废。
//  金额一律以「明细 qty × unit_price」为准重算，避免手工改价导致账目不平。
// =============================================================================

type BillingHandler struct{ db *gorm.DB }

// RegisterBilling mounts invoice routes.
func RegisterBilling(r gin.IRouter, db *gorm.DB) {
	h := &BillingHandler{db: db}
	r.GET("/invoices", middleware.RequireAny(db, "billing:read"), h.List)
	r.GET("/invoices/:id", middleware.RequireAny(db, "billing:read"), h.Get)
	r.POST("/invoices", middleware.RequireAny(db, "billing:manage"), h.Create)
	r.PUT("/invoices/:id", middleware.RequireAny(db, "billing:manage"), h.Update)
	r.POST("/invoices/:id/items", middleware.RequireAny(db, "billing:manage"), h.AddItem)
	r.DELETE("/invoices/items/:itemId", middleware.RequireAny(db, "billing:manage"), h.RemoveItem)
	r.POST("/invoices/:id/pay", middleware.RequireAny(db, "billing:manage"), h.Pay)
	r.POST("/invoices/:id/refund", middleware.RequireAny(db, "billing:manage"), h.Refund)
	r.DELETE("/invoices/:id", middleware.RequireAny(db, "billing:manage"), h.Delete)
}

// List returns invoices, filterable by patient and status.
func (h *BillingHandler) List(c *gin.Context) {
	page, pageSize := getPagination(c)
	var total int64
	q := h.db.Model(&models.Invoice{})
	if v := c.Query("patient_id"); v != "" {
		q = q.Where("patient_id = ?", atoui(v))
	}
	if v := c.Query("status"); v != "" {
		q = q.Where("status = ?", v)
	}
	q.Count(&total)
	var items []models.Invoice
	paginate(q.Order("id desc"), page, pageSize).Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "pagination": Pagination{Page: page, PageSize: pageSize, Total: total}})
}

// Get returns an invoice together with its line items.
func (h *BillingHandler) Get(c *gin.Context) {
	var inv models.Invoice
	if err := h.db.Preload("Items").First(&inv, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	c.JSON(http.StatusOK, inv)
}

type invoiceReq struct {
	PatientID      uint    `json:"patient_id"`
	CaseID         uint    `json:"case_id"`
	PrescriptionID uint    `json:"prescription_id"`
	Department     string  `json:"department"`
	DoctorID       uint    `json:"doctor_id"`
	Discount       float64 `json:"discount"`
	Currency       string  `json:"currency"`
	Remark         string  `json:"remark"`
}

// Create opens a new (pending) invoice for a patient.
func (h *BillingHandler) Create(c *gin.Context) {
	var req invoiceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.PatientID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "patient_required")})
		return
	}
	var p models.Patient
	if h.db.First(&p, req.PatientID).Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "patient_not_found")})
		return
	}
	inv := models.Invoice{
		InvoiceNo:      generateNo("INV"),
		PatientID:      req.PatientID,
		PatientName:    p.Name,
		CaseID:         req.CaseID,
		PrescriptionID: req.PrescriptionID,
		Department:     req.Department,
		DoctorID:       req.DoctorID,
		Discount:       req.Discount,
		Currency:       orDefault(req.Currency, "CNY"),
		Remark:         req.Remark,
		Status:         "pending",
	}
	if req.DoctorID != 0 {
		var u models.User
		if h.db.First(&u, req.DoctorID).Error == nil {
			inv.DoctorName = u.Name
			if inv.Department == "" {
				inv.Department = u.Department
			}
		}
	}
	if err := h.db.Create(&inv).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": inv, "message": i18n.T(c, "created")})
}

// Update edits invoice header fields (never the totals — those are recomputed).
func (h *BillingHandler) Update(c *gin.Context) {
	var inv models.Invoice
	if err := h.db.First(&inv, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	var req invoiceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.Department != "" {
		inv.Department = req.Department
	}
	if req.DoctorID != 0 {
		inv.DoctorID = req.DoctorID
		var u models.User
		if h.db.First(&u, req.DoctorID).Error == nil {
			inv.DoctorName = u.Name
		}
	}
	if req.Discount > 0 {
		inv.Discount = req.Discount
	}
	inv.Remark = req.Remark
	if err := recomputeInvoice(h.db, &inv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": inv, "message": i18n.T(c, "updated")})
}

type itemReq struct {
	ItemType  string  `json:"item_type"`
	RefID     uint    `json:"ref_id"`
	NameZH    string  `json:"name_zh"`
	NameEN    string  `json:"name_en"`
	Qty       float64 `json:"qty"`
	Unit      string  `json:"unit"`
	UnitPrice float64 `json:"unit_price"`
}

// AddItem appends a chargeable line and recalculates the invoice totals.
func (h *BillingHandler) AddItem(c *gin.Context) {
	var inv models.Invoice
	if err := h.db.First(&inv, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	var req itemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.Qty <= 0 {
		req.Qty = 1
	}
	if req.NameZH == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "name_required")})
		return
	}
	item := models.InvoiceItem{
		InvoiceID: inv.ID,
		ItemType:  orDefault(req.ItemType, "other"),
		RefID:     req.RefID,
		NameZH:    req.NameZH,
		NameEN:    req.NameEN,
		Qty:       req.Qty,
		Unit:      req.Unit,
		UnitPrice: req.UnitPrice,
		Amount:    roundMoney(req.Qty * req.UnitPrice),
	}
	if err := h.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	if err := recomputeInvoice(h.db, &inv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": item, "invoice": inv, "message": i18n.T(c, "created")})
}

// RemoveItem deletes a line and recalculates the totals.
func (h *BillingHandler) RemoveItem(c *gin.Context) {
	var item models.InvoiceItem
	if err := h.db.First(&item, c.Param("itemId")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	invoiceID := item.InvoiceID
	if err := h.db.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	var inv models.Invoice
	if h.db.First(&inv, invoiceID).Error == nil {
		_ = recomputeInvoice(h.db, &inv)
	}
	c.JSON(http.StatusOK, gin.H{"invoice": inv, "message": i18n.T(c, "deleted")})
}

// Pay marks an invoice as paid and stamps the payment time.
func (h *BillingHandler) Pay(c *gin.Context) {
	var inv models.Invoice
	if err := h.db.First(&inv, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	var body struct {
		PaymentMethod string `json:"payment_method"`
	}
	_ = c.ShouldBindJSON(&body)
	if inv.Status == "paid" {
		c.JSON(http.StatusConflict, gin.H{"error": i18n.T(c, "invoice_already_paid")})
		return
	}
	inv.Status = "paid"
	inv.PaymentMethod = orDefault(body.PaymentMethod, "cash")
	now := time.Now()
	inv.PaidAt = &now
	if err := h.db.Save(&inv).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": inv, "message": i18n.T(c, "updated")})
}

// Refund marks a paid invoice as refunded.
func (h *BillingHandler) Refund(c *gin.Context) {
	var inv models.Invoice
	if err := h.db.First(&inv, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	if inv.Status != "paid" {
		c.JSON(http.StatusConflict, gin.H{"error": i18n.T(c, "invoice_not_paid")})
		return
	}
	inv.Status = "refunded"
	if err := h.db.Save(&inv).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": inv, "message": i18n.T(c, "updated")})
}

// Delete removes a pending invoice only (已收款单据禁止删除，保证账目可追溯).
func (h *BillingHandler) Delete(c *gin.Context) {
	var inv models.Invoice
	if err := h.db.First(&inv, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "not_found")})
		return
	}
	if inv.Status == "paid" {
		c.JSON(http.StatusConflict, gin.H{"error": i18n.T(c, "invoice_paid_cannot_delete")})
		return
	}
	h.db.Where("invoice_id = ?", inv.ID).Delete(&models.InvoiceItem{})
	if err := h.db.Delete(&inv).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "deleted")})
}

// ---------------------------------------------------------------------------
//  金额工具
// ---------------------------------------------------------------------------

// recomputeInvoice recalculates subtotal/total from the persisted line items so
// the header can never drift from its details — a core accounting guarantee.
func recomputeInvoice(db *gorm.DB, inv *models.Invoice) error {
	var items []models.InvoiceItem
	if err := db.Where("invoice_id = ?", inv.ID).Find(&items).Error; err != nil {
		return err
	}
	var sub float64
	for _, it := range items {
		sub += it.Amount
	}
	inv.Subtotal = roundMoney(sub)
	if inv.Discount > inv.Subtotal {
		inv.Discount = inv.Subtotal // 折扣不得超过小计，避免出现负数应收
	}
	inv.Total = roundMoney(inv.Subtotal - inv.Discount)
	return db.Save(inv).Error
}

// roundMoney rounds to two decimals for money-safe storage/display.
// It preserves the sign so that refunds (negative adjustments) stay negative.
func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

// orDefault returns v when non-empty, otherwise def.
func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
