package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/i18n"
	"github.com/neko233-com/server-medical-system/internal/middleware"
	"github.com/neko233-com/server-medical-system/internal/models"
)

// ImagingHandler manages diagnostic imaging studies and their images.
type ImagingHandler struct {
	db        *gorm.DB
	uploadDir string
}

func RegisterImaging(r gin.IRouter, db *gorm.DB, uploadDir string) {
	h := &ImagingHandler{db: db, uploadDir: uploadDir}
	r.GET("/imaging", middleware.RequireAny(db, "imaging:read"), h.ListStudies)
	r.GET("/imaging/:id", middleware.RequireAny(db, "imaging:read"), h.GetStudy)
	r.POST("/imaging", middleware.RequireAny(db, "imaging:upload"), h.CreateStudy)
	r.PUT("/imaging/:id", middleware.RequireAny(db, "imaging:upload"), h.UpdateStudy)
	r.PUT("/imaging/:id/status", middleware.RequireAny(db, "imaging:report"), h.UpdateStatus)
	r.DELETE("/imaging/:id", middleware.RequireAny(db, "imaging:upload"), h.DeleteStudy)
	// Images.
	r.POST("/imaging/:id/images", middleware.RequireAny(db, "imaging:upload"), h.UploadImage)
	r.GET("/imaging/:id/images", middleware.RequireAny(db, "imaging:read"), h.ListImages)
	r.DELETE("/imaging/images/:imageId", middleware.RequireAny(db, "imaging:upload"), h.DeleteImage)
	// Serve stored image bytes (auth required).
	r.GET("/imaging/images/:id/file", middleware.RequireAny(db, "imaging:read"), h.ServeImage)
}

type studyReq struct {
	PatientID  uint   `json:"patient_id"`
	DoctorID   uint   `json:"doctor_id"`
	RoomID     uint   `json:"room_id"`
	Modality   string `json:"modality"` // CT / XR / MRI / US / ENDO
	BodyPart   string `json:"body_part"`
	StudyDate  string `json:"study_date"`
	Indication string `json:"indication"`
	Findings   string `json:"findings"`
	Impression string `json:"impression"`
}

var validModality = map[string]bool{
	"CT": true, "XR": true, "MRI": true, "US": true, "ENDO": true, "OTHER": true,
}

var validStudyStatus = map[string]bool{
	"ordered": true, "done": true, "reported": true,
}

func (h *ImagingHandler) ListStudies(c *gin.Context) {
	patientID := c.Query("patient_id")
	modality := c.Query("modality")
	status := c.Query("status")
	page, pageSize := getPagination(c)
	var total int64
	query := h.db.Model(&models.ImagingStudy{})
	if patientID != "" {
		query = query.Where("patient_id = ?", patientID)
	}
	if modality != "" {
		query = query.Where("modality = ?", modality)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)
	var items []models.ImagingStudy
	paginate(query.Order("study_date desc"), page, pageSize).Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "pagination": Pagination{Page: page, PageSize: pageSize, Total: total}})
}

func (h *ImagingHandler) GetStudy(c *gin.Context) {
	var s models.ImagingStudy
	if err := h.db.Preload("Images").First(&s, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "study_not_found")})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *ImagingHandler) CreateStudy(c *gin.Context) {
	var req studyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.PatientID == 0 || req.Modality == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "validation_error")})
		return
	}
	if !validModality[req.Modality] {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_status")})
		return
	}
	var patient models.Patient
	if h.db.First(&patient, req.PatientID).Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "patient_not_found")})
		return
	}
	uid, _ := c.Get("uid")
	var doctor models.User
	h.db.First(&doctor, uid)
	studyDate := time.Now()
	if req.StudyDate != "" {
		if t, err := time.Parse(time.RFC3339, req.StudyDate); err == nil {
			studyDate = t
		}
	}
	s := models.ImagingStudy{
		StudyNo:     generateStudyNo(studyDate, doctor.ID),
		PatientID:   req.PatientID,
		PatientName: patient.Name,
		DoctorID:    doctor.ID,
		DoctorName:  doctor.Name,
		RoomID:      req.RoomID,
		Modality:    req.Modality,
		BodyPart:    req.BodyPart,
		StudyDate:   studyDate,
		Indication:  req.Indication,
		Findings:    req.Findings,
		Impression:  req.Impression,
		Status:      "ordered",
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

func (h *ImagingHandler) UpdateStudy(c *gin.Context) {
	var s models.ImagingStudy
	if err := h.db.First(&s, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "study_not_found")})
		return
	}
	var req studyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if req.Modality != "" && validModality[req.Modality] {
		s.Modality = req.Modality
	}
	s.BodyPart = req.BodyPart
	s.Indication = req.Indication
	s.Findings = req.Findings
	s.Impression = req.Impression
	if req.RoomID != 0 {
		s.RoomID = req.RoomID
		var room models.Room
		if h.db.First(&room, req.RoomID).Error == nil {
			s.RoomNo = room.RoomNo
		}
	}
	if err := h.db.Save(&s).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": s, "message": i18n.T(c, "updated")})
}

func (h *ImagingHandler) UpdateStatus(c *gin.Context) {
	var s models.ImagingStudy
	if err := h.db.First(&s, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "study_not_found")})
		return
	}
	var body struct {
		Status     string `json:"status"`
		Findings   string `json:"findings"`
		Impression string `json:"impression"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if !validStudyStatus[body.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_status")})
		return
	}
	s.Status = body.Status
	if body.Findings != "" {
		s.Findings = body.Findings
	}
	if body.Impression != "" {
		s.Impression = body.Impression
	}
	if body.Status == "reported" {
		uid, _ := c.Get("uid")
		var u models.User
		if h.db.First(&u, uid).Error == nil {
			s.ReportedBy = u.Name
		}
	}
	if err := h.db.Save(&s).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": s, "message": i18n.T(c, "updated")})
}

func (h *ImagingHandler) DeleteStudy(c *gin.Context) {
	var s models.ImagingStudy
	if err := h.db.Preload("Images").First(&s, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "study_not_found")})
		return
	}
	// Remove stored files.
	for _, img := range s.Images {
		_ = os.Remove(h.imagePath(&img))
	}
	h.db.Delete(&s)
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "deleted")})
}

// UploadImage handles a multipart image upload attached to a study.
func (h *ImagingHandler) UploadImage(c *gin.Context) {
	var study models.ImagingStudy
	if err := h.db.First(&study, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "study_not_found")})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if file.Size > 25<<20 { // 25MB
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "file_too_large")})
		return
	}
	ct := file.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") && ct != "application/dicom" && ct != "application/octet-stream" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "unsupported_file_type")})
		return
	}
	caption := c.PostForm("caption")
	uid, _ := c.Get("uid")

	dir := filepath.Join(h.uploadDir, "images", itoa(int(study.ID)))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	stored := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(file.Filename))
	// Prevent path traversal in the supplied filename.
	stored = filepath.Base(stored)
	dst := filepath.Join(dir, stored)
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	img := models.ImagingImage{
		StudyID:     study.ID,
		FileName:    file.Filename,
		StoredName:  filepath.ToSlash(filepath.Join("images", itoa(int(study.ID)), stored)),
		ContentType: ct,
		FileSize:    file.Size,
		Caption:     caption,
		UploadedBy:  uid.(uint),
		UploadedAt:  time.Now(),
	}
	if err := h.db.Create(&img).Error; err != nil {
		_ = os.Remove(dst)
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": img, "message": i18n.T(c, "created")})
}

func (h *ImagingHandler) ListImages(c *gin.Context) {
	var study models.ImagingStudy
	if err := h.db.First(&study, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "study_not_found")})
		return
	}
	var imgs []models.ImagingImage
	h.db.Where("study_id = ?", study.ID).Order("id").Find(&imgs)
	c.JSON(http.StatusOK, imgs)
}

func (h *ImagingHandler) DeleteImage(c *gin.Context) {
	var img models.ImagingImage
	if err := h.db.First(&img, c.Param("imageId")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "image_not_found")})
		return
	}
	_ = os.Remove(h.imagePath(&img))
	h.db.Delete(&img)
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(c, "deleted")})
}

// ServeImage streams a stored image to the browser (auth required).
func (h *ImagingHandler) ServeImage(c *gin.Context) {
	var img models.ImagingImage
	if err := h.db.First(&img, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "image_not_found")})
		return
	}
	path := filepath.Join(h.uploadDir, filepath.FromSlash(img.StoredName))
	if _, err := os.Stat(path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(c, "image_not_found")})
		return
	}
	c.File(path)
}

func (h *ImagingHandler) imagePath(img *models.ImagingImage) string {
	return filepath.Join(h.uploadDir, filepath.FromSlash(img.StoredName))
}

func generateStudyNo(t time.Time, doctorID uint) string {
	return "IMG" + t.Format("20060102") + "-" + pad(int(doctorID)) + "-" + randomSuffix()
}
