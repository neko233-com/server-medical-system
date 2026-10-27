package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/i18n"
	"github.com/neko233-com/server-medical-system/internal/middleware"
	"github.com/neko233-com/server-medical-system/internal/models"
)

// RoomHandler manages outpatient / operating / radiology rooms.
type RoomHandler struct {
	db *gorm.DB
}

func RegisterRooms(r gin.IRouter, db *gorm.DB) {
	h := &RoomHandler{db: db}
	r.GET("/rooms", middleware.RequireAny(db, "room:read"), h.List)
	r.GET("/rooms/:id", middleware.RequireAny(db, "room:read"), h.Get)
	r.POST("/rooms", middleware.RequireAny(db, "room:manage"), h.Create)
	r.PUT("/rooms/:id", middleware.RequireAny(db, "room:manage"), h.Update)
	r.DELETE("/rooms/:id", middleware.RequireAny(db, "room:manage"), h.Delete)
	// Assign / unassign the doctor bound to a room.
	r.PUT("/rooms/:id/doctor", middleware.RequireAny(db, "room:manage"), h.AssignDoctor)
}

type roomReq struct {
	RoomNo      string `json:"room_no"`
	NameZH      string `json:"name_zh"`
	NameEN      string `json:"name_en"`
	Type        string `json:"type"`
	Department  string `json:"department"`
	Floor       string `json:"floor"`
	DoctorID    uint   `json:"doctor_id"`
	PrinterID   uint   `json:"printer_id"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

func (h *RoomHandler) List(c *gin.Context) {
	q := c.Query("q")
	typ := c.Query("type")
	page, pageSize := getPagination(c)
	var total int64
	query := h.db.Model(&models.Room{})
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("room_no LIKE ? OR name_zh LIKE ? OR name_en LIKE ? OR department LIKE ?", like, like, like, like)
	}
	if typ != "" {
		query = query.Where("type = ?", typ)
	}
	query.Count(&total)
	var items []models.Room
	paginate(query.Order("room_no"), page, pageSize).Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "pagination": Pagination{Page: page, PageSize: pageSize, Total: total}})
}

func (h *RoomHandler) Get(c *gin.Context) {
	var room models.Room
	if err := h.db.First(&room, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "room_not_found")})
		return
	}
	c.JSON(http.StatusOK, room)
}

func (h *RoomHandler) Create(c *gin.Context) {
	var req roomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.RoomNo == "" || req.NameZH == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "name_required")})
		return
	}
	var dup int64
	h.db.Model(&models.Room{}).Where("room_no = ?", req.RoomNo).Count(&dup)
	if dup > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": i18n.T(c, "medicine_code_exists")})
		return
	}
	room := h.applyRoom(&models.Room{}, &req)
	if err := h.db.Create(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": room, "message": i18n.T(c, "created")})
}

func (h *RoomHandler) Update(c *gin.Context) {
	var room models.Room
	if err := h.db.First(&room, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "room_not_found")})
		return
	}
	var req roomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	room2 := h.applyRoom(&room, &req)
	if err := h.db.Save(&room2).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": room2, "message": i18n.T(c, "updated")})
}

// applyRoom copies request fields into a Room, resolving doctor/printer names.
func (h *RoomHandler) applyRoom(room *models.Room, req *roomReq) *models.Room {
	room.RoomNo = req.RoomNo
	if req.NameZH != "" {
		room.NameZH = req.NameZH
	}
	room.NameEN = req.NameEN
	room.Type = req.Type
	room.Department = req.Department
	room.Floor = req.Floor
	room.DoctorID = req.DoctorID
	room.PrinterID = req.PrinterID
	if req.Status != "" {
		room.Status = req.Status
	} else if room.Status == "" {
		room.Status = "active"
	}
	room.Description = req.Description
	if req.DoctorID != 0 {
		var u models.User
		if h.db.First(&u, req.DoctorID).Error == nil {
			room.DoctorName = u.Name
		}
	} else {
		room.DoctorName = ""
	}
	if req.PrinterID != 0 {
		var p models.Printer
		if h.db.First(&p, req.PrinterID).Error == nil {
			room.PrinterID = p.ID
		}
	}
	return room
}

// AssignDoctor binds (or unbinds, when doctor_id == 0) a doctor to a room.
func (h *RoomHandler) AssignDoctor(c *gin.Context) {
	var room models.Room
	if err := h.db.First(&room, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "room_not_found")})
		return
	}
	var body struct {
		DoctorID uint `json:"doctor_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	room.DoctorID = body.DoctorID
	if body.DoctorID != 0 {
		var u models.User
		if h.db.First(&u, body.DoctorID).Error == nil {
			room.DoctorName = u.Name
		}
	} else {
		room.DoctorName = ""
	}
	if err := h.db.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": room, "message": i18n.T(c, "updated")})
}

func (h *RoomHandler) Delete(c *gin.Context) {
	if err := h.db.Delete(&models.Room{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "deleted")})
}
