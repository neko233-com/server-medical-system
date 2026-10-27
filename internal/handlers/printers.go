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

// PrinterHandler manages printers and records print jobs.
type PrinterHandler struct {
	db *gorm.DB
}

func RegisterPrinters(r gin.IRouter, db *gorm.DB) {
	h := &PrinterHandler{db: db}
	r.GET("/printers", middleware.RequireAny(db, "printer:read"), h.List)
	r.GET("/printers/:id", middleware.RequireAny(db, "printer:read"), h.Get)
	r.POST("/printers", middleware.RequireAny(db, "printer:manage"), h.Create)
	r.PUT("/printers/:id", middleware.RequireAny(db, "printer:manage"), h.Update)
	r.DELETE("/printers/:id", middleware.RequireAny(db, "printer:manage"), h.Delete)
	r.PUT("/printers/:id/default", middleware.RequireAny(db, "printer:manage"), h.SetDefault)
	// Resolve the printer a document should be routed to for the caller's
	// doctor + room context, then record the print job.
	r.GET("/printers/resolve", middleware.RequireAny(db, "printer:read"), h.Resolve)
	r.POST("/print-jobs", h.CreateJob)
	r.GET("/print-jobs", middleware.RequireAny(db, "printer:read"), h.ListJobs)
}

type printerReq struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Connection string `json:"connection"`
	IPAddress  string `json:"ip_address"`
	Port       string `json:"port"`
	DoctorID   uint   `json:"doctor_id"`
	RoomID     uint   `json:"room_id"`
	IsDefault  bool   `json:"is_default"`
	Status     string `json:"status"`
	Notes      string `json:"notes"`
}

func (h *PrinterHandler) List(c *gin.Context) {
	q := c.Query("q")
	roomID := c.Query("room_id")
	doctorID := c.Query("doctor_id")
	page, pageSize := getPagination(c)
	var total int64
	query := h.db.Model(&models.Printer{})
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("name LIKE ? OR ip_address LIKE ?", like, like)
	}
	if roomID != "" {
		query = query.Where("room_id = ?", roomID)
	}
	if doctorID != "" {
		query = query.Where("doctor_id = ?", doctorID)
	}
	query.Count(&total)
	var items []models.Printer
	paginate(query.Order("id"), page, pageSize).Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "pagination": Pagination{Page: page, PageSize: pageSize, Total: total}})
}

func (h *PrinterHandler) Get(c *gin.Context) {
	var p models.Printer
	if err := h.db.First(&p, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "printer_not_found")})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *PrinterHandler) Create(c *gin.Context) {
	var req printerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "name_required")})
		return
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	p := h.applyPrinter(&models.Printer{}, &req, status)
	if req.IsDefault {
		h.clearDefaults()
		p.IsDefault = true
	}
	if err := h.db.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": p, "message": i18n.T(c, "created")})
}

func (h *PrinterHandler) Update(c *gin.Context) {
	var p models.Printer
	if err := h.db.First(&p, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "printer_not_found")})
		return
	}
	var req printerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	status := p.Status
	if req.Status != "" {
		status = req.Status
	}
	p2 := h.applyPrinter(&p, &req, status)
	if req.IsDefault {
		h.clearDefaults()
		p2.IsDefault = true
	}
	if err := h.db.Save(&p2).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": p2, "message": i18n.T(c, "updated")})
}

func (h *PrinterHandler) applyPrinter(p *models.Printer, req *printerReq, status string) *models.Printer {
	if req.Name != "" {
		p.Name = req.Name
	}
	p.Type = req.Type
	p.Connection = req.Connection
	p.IPAddress = req.IPAddress
	p.Port = req.Port
	p.DoctorID = req.DoctorID
	p.RoomID = req.RoomID
	p.Status = status
	p.Notes = req.Notes
	if req.DoctorID != 0 {
		var u models.User
		if h.db.First(&u, req.DoctorID).Error == nil {
			p.DoctorName = u.Name
		}
	} else {
		p.DoctorName = ""
	}
	if req.RoomID != 0 {
		var room models.Room
		if h.db.First(&room, req.RoomID).Error == nil {
			p.RoomNo = room.RoomNo
		}
	} else {
		p.RoomNo = ""
	}
	return p
}

func (h *PrinterHandler) clearDefaults() {
	h.db.Model(&models.Printer{}).Where("is_default = ?", true).Update("is_default", false)
}

func (h *PrinterHandler) SetDefault(c *gin.Context) {
	var p models.Printer
	if err := h.db.First(&p, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "printer_not_found")})
		return
	}
	h.clearDefaults()
	p.IsDefault = true
	h.db.Save(&p)
	c.JSON(http.StatusOK, gin.H{"item": p, "message": i18n.T(c, "updated")})
}

func (h *PrinterHandler) Delete(c *gin.Context) {
	if err := h.db.Delete(&models.Printer{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "deleted")})
}

// Resolve picks the best printer for the caller (doctor) given an optional room.
// Priority: doctor-bound printer > room's default printer > global default.
func (h *PrinterHandler) Resolve(c *gin.Context) {
	uid, _ := c.Get("uid")
	roomID := atoui(c.Query("room_id"))
	printer := h.resolvePrinter(uid.(uint), roomID)
	if printer == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "no_target_printer")})
		return
	}
	c.JSON(http.StatusOK, printer)
}

func (h *PrinterHandler) resolvePrinter(uid uint, roomID uint) *models.Printer {
	// 1. doctor-bound
	var byDoctor models.Printer
	if h.db.Where("doctor_id = ? AND status = ?", uid, "active").Order("is_default desc").First(&byDoctor).Error == nil {
		return &byDoctor
	}
	// 2. room default (the room's bound printer, or its own printer_id)
	if roomID != 0 {
		var room models.Room
		if h.db.First(&room, roomID).Error == nil && room.PrinterID != 0 {
			var p models.Printer
			if h.db.First(&p, room.PrinterID).Error == nil && p.Status == "active" {
				return &p
			}
		}
		var byRoom models.Printer
		if h.db.Where("room_id = ? AND status = ?", roomID, "active").Order("is_default desc").First(&byRoom).Error == nil {
			return &byRoom
		}
	}
	// 3. global default
	var def models.Printer
	if h.db.Where("is_default = ? AND status = ?", true, "active").First(&def).Error == nil {
		return &def
	}
	return nil
}

// CreateJob records a print job, resolving the target printer automatically when
// none is supplied. This is the audit + routing entry point for printing.
func (h *PrinterHandler) CreateJob(c *gin.Context) {
	var body struct {
		PrinterID uint   `json:"printer_id"`
		RoomID    uint   `json:"room_id"`
		PatientID uint   `json:"patient_id"`
		DocType   string `json:"doc_type"`
		RefID     uint   `json:"ref_id"`
		Summary   string `json:"summary"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	uid, _ := c.Get("uid")
	printer := &models.Printer{}
	if body.PrinterID != 0 {
		if h.db.First(printer, body.PrinterID).Error != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "printer_not_found")})
			return
		}
	} else {
		resolved := h.resolvePrinter(uid.(uint), body.RoomID)
		if resolved == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "no_target_printer")})
			return
		}
		printer = resolved
	}
	var doctor models.User
	h.db.First(&doctor, uid)

	job := models.PrintJob{
		PrinterID:   printer.ID,
		PrinterName: printer.Name,
		DoctorID:    doctor.ID,
		DoctorName:  doctor.Name,
		RoomID:      body.RoomID,
		PatientID:   body.PatientID,
		DocType:     body.DocType,
		RefID:       body.RefID,
		Summary:     body.Summary,
		Status:      "printed",
		CreatedAt:   time.Now(),
	}
	if body.RoomID != 0 {
		var room models.Room
		if h.db.First(&room, body.RoomID).Error == nil {
			job.RoomNo = room.RoomNo
		}
	}
	if err := h.db.Create(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": job, "printer": printer, "message": i18n.T(c, "created")})
}

func (h *PrinterHandler) ListJobs(c *gin.Context) {
	page, pageSize := getPagination(c)
	var total int64
	query := h.db.Model(&models.PrintJob{})
	query.Count(&total)
	var items []models.PrintJob
	paginate(query.Order("id desc"), page, pageSize).Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "pagination": Pagination{Page: page, PageSize: pageSize, Total: total}})
}
