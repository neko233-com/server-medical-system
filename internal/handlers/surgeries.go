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

// SurgeryHandler manages operative procedures.
type SurgeryHandler struct {
	db *gorm.DB
}

func RegisterSurgeries(r gin.IRouter, db *gorm.DB) {
	h := &SurgeryHandler{db: db}
	r.GET("/surgeries", middleware.RequireAny(db, "surgery:read"), h.List)
	r.GET("/surgeries/:id", middleware.RequireAny(db, "surgery:read"), h.Get)
	r.POST("/surgeries", middleware.RequireAny(db, "surgery:manage"), h.Create)
	r.PUT("/surgeries/:id", middleware.RequireAny(db, "surgery:manage"), h.Update)
	r.PUT("/surgeries/:id/status", middleware.RequireAny(db, "surgery:manage"), h.UpdateStatus)
	r.DELETE("/surgeries/:id", middleware.RequireAny(db, "surgery:manage"), h.Delete)
}

type surgeryReq struct {
	PatientID       uint   `json:"patient_id"`
	SurgeonID       uint   `json:"surgeon_id"`
	AssistantIDs    []uint `json:"assistant_ids"`
	Anesthetist     string `json:"anesthetist"`
	RoomID          uint   `json:"room_id"`
	OperationName   string `json:"operation_name"`
	OperationNameEN string `json:"operation_name_en"`
	ScheduledAt     string `json:"scheduled_at"` // RFC3339
	AnesthesiaType  string `json:"anesthesia_type"`
	PreOpDiagnosis  string `json:"pre_op_diagnosis"`
	Notes           string `json:"notes"`
}

var validSurgeryStatus = map[string]bool{
	"planned":     true,
	"in_progress": true,
	"done":        true,
	"cancelled":   true,
}

func (h *SurgeryHandler) List(c *gin.Context) {
	patientID := c.Query("patient_id")
	status := c.Query("status")
	page, pageSize := getPagination(c)
	var total int64
	query := h.db.Model(&models.Surgery{})
	if patientID != "" {
		query = query.Where("patient_id = ?", patientID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)
	var items []models.Surgery
	paginate(query.Order("scheduled_at desc"), page, pageSize).Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "pagination": Pagination{Page: page, PageSize: pageSize, Total: total}})
}

func (h *SurgeryHandler) Get(c *gin.Context) {
	var s models.Surgery
	if err := h.db.Preload("Items").First(&s, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "surgery_not_found")})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *SurgeryHandler) Create(c *gin.Context) {
	var req surgeryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.PatientID == 0 || req.OperationName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "validation_error")})
		return
	}
	var patient models.Patient
	if h.db.First(&patient, req.PatientID).Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "patient_not_found")})
		return
	}
	var surgeon models.User
	if req.SurgeonID != 0 {
		h.db.First(&surgeon, req.SurgeonID)
	}
	uid, _ := c.Get("uid")
	var operator models.User
	h.db.First(&operator, uid)

	assistantIDs := ""
	assistantNames := ""
	if len(req.AssistantIDs) > 0 {
		ids := make([]string, 0, len(req.AssistantIDs))
		names := make([]string, 0, len(req.AssistantIDs))
		for _, id := range req.AssistantIDs {
			var a models.User
			if h.db.First(&a, id).Error == nil {
				ids = append(ids, itoa(int(id)))
				names = append(names, a.Name)
			}
		}
		assistantIDs = joinComma(ids)
		assistantNames = joinComma(names)
	}

	scheduled := time.Now()
	if req.ScheduledAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ScheduledAt); err == nil {
			scheduled = t
		}
	}

	s := models.Surgery{
		SurgeryNo:       generateSurgeryNo(scheduled, operator.ID),
		PatientID:       req.PatientID,
		PatientName:     patient.Name,
		SurgeonID:       surgeon.ID,
		SurgeonName:     surgeon.Name,
		AssistantIDs:    assistantIDs,
		AssistantNames:  assistantNames,
		Anesthetist:     req.Anesthetist,
		RoomID:          req.RoomID,
		OperationName:   req.OperationName,
		OperationNameEN: req.OperationNameEN,
		ScheduledAt:     scheduled,
		AnesthesiaType:  req.AnesthesiaType,
		PreOpDiagnosis:  req.PreOpDiagnosis,
		Notes:           req.Notes,
		Status:          "planned",
	}
	if req.RoomID != 0 {
		var room models.Room
		if h.db.First(&room, req.RoomID).Error == nil {
			s.RoomNo = room.RoomNo
		}
	}
	if err := h.db.Create(&s).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": s, "message": i18n.T(c, "created")})
}

func (h *SurgeryHandler) Update(c *gin.Context) {
	var s models.Surgery
	if err := h.db.First(&s, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "surgery_not_found")})
		return
	}
	var req surgeryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.OperationName != "" {
		s.OperationName = req.OperationName
	}
	s.OperationNameEN = req.OperationNameEN
	s.Anesthetist = req.Anesthetist
	s.AnesthesiaType = req.AnesthesiaType
	s.PreOpDiagnosis = req.PreOpDiagnosis
	s.Notes = req.Notes
	if req.RoomID != 0 {
		s.RoomID = req.RoomID
		var room models.Room
		if h.db.First(&room, req.RoomID).Error == nil {
			s.RoomNo = room.RoomNo
		}
	}
	if req.SurgeonID != 0 {
		var surgeon models.User
		if h.db.First(&surgeon, req.SurgeonID).Error == nil {
			s.SurgeonID = surgeon.ID
			s.SurgeonName = surgeon.Name
		}
	}
	if err := h.db.Save(&s).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": s, "message": i18n.T(c, "updated")})
}

// UpdateStatus drives the surgery lifecycle (planned -> in_progress -> done).
func (h *SurgeryHandler) UpdateStatus(c *gin.Context) {
	var s models.Surgery
	if err := h.db.First(&s, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "surgery_not_found")})
		return
	}
	var body struct {
		Status string `json:"status"`
		// Optional rich fields captured at transition time.
		StartedAt       string `json:"started_at"`
		EndedAt         string `json:"ended_at"`
		Findings        string `json:"findings"`
		ProcedureNotes  string `json:"procedure_notes"`
		Complications   string `json:"complications"`
		PostOpDiagnosis string `json:"post_op_diagnosis"`
		DurationMin     int    `json:"duration_min"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if !validSurgeryStatus[body.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_status")})
		return
	}
	now := time.Now()
	switch body.Status {
	case "in_progress":
		if s.StartedAt == nil {
			t := now
			s.StartedAt = &t
		}
	case "done":
		if s.EndedAt == nil {
			t := now
			s.EndedAt = &t
		}
		if s.StartedAt == nil {
			t := now
			s.StartedAt = &t
		}
	}
	s.Status = body.Status
	if body.Findings != "" {
		s.Findings = body.Findings
	}
	if body.ProcedureNotes != "" {
		s.ProcedureNotes = body.ProcedureNotes
	}
	if body.Complications != "" {
		s.Complications = body.Complications
	}
	if body.PostOpDiagnosis != "" {
		s.PostOpDiagnosis = body.PostOpDiagnosis
	}
	if body.DurationMin > 0 {
		s.DurationMin = body.DurationMin
	}
	if err := h.db.Save(&s).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": s, "message": i18n.T(c, "updated")})
}

func (h *SurgeryHandler) Delete(c *gin.Context) {
	if err := h.db.Delete(&models.Surgery{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "deleted")})
}

func generateSurgeryNo(t time.Time, operatorID uint) string {
	return "OP" + t.Format("20060102") + "-" + pad(int(operatorID)) + "-" + randomSuffix()
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ","
		}
		out += p
	}
	return out
}
