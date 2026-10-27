package models

import (
	"time"

	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Accounts & RBAC
// ---------------------------------------------------------------------------

// Role codes used throughout the system. A user's Role field stores one of these
// codes; each resolves to a row in the roles table for permission lookup.
const (
	RoleAdmin        = "admin"
	RoleDoctor       = "doctor"
	RoleNurse        = "nurse"
	RoleReceptionist = "receptionist"
	RoleRadiologist  = "radiologist"
	RoleSurgeon      = "surgeon"
	RolePharmacist   = "pharmacist"
)

// User is a clinician or administrator that logs into the system.
// Role holds the role *code* (e.g. "admin", "doctor") which resolves to a row
// in the roles table; effective permissions are computed from that role plus
// any direct user_permission grants.
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"uniqueIndex;size:64" json:"username"`
	PasswordHash string         `gorm:"size:128" json:"-"`
	Name         string         `gorm:"size:128" json:"name"`
	Role         string         `gorm:"size:32;index" json:"role"` // role code
	Department   string         `gorm:"size:128" json:"department"`
	Title        string         `gorm:"size:64" json:"title"`                 // e.g. 主治医师
	Status       string         `gorm:"size:16;default:active" json:"status"` // active / disabled
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Role is a named set of permissions (RBAC).
type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"uniqueIndex;size:32" json:"code"`
	NameZH      string    `gorm:"size:128" json:"name_zh"`
	NameEN      string    `gorm:"size:128" json:"name_en"`
	Description string    `gorm:"size:256" json:"description"`
	IsSystem    bool      `gorm:"default:false" json:"is_system"` // system roles cannot be deleted
	Sort        int       `json:"sort"`
	CreatedAt   time.Time `json:"created_at"`
}

// Permission is a single capability in the catalog.
type Permission struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Code        string `gorm:"uniqueIndex;size:64" json:"code"`
	NameZH      string `gorm:"size:128" json:"name_zh"`
	NameEN      string `gorm:"size:128" json:"name_en"`
	Group       string `gorm:"size:64;index" json:"group"` // grouping for UI (患者/处方/...)
	Description string `gorm:"size:256" json:"description"`
}

// RolePermission links a role code to a permission code.
type RolePermission struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	RoleCode       string `gorm:"uniqueIndex:uk_rp;size:32" json:"role_code"`
	PermissionCode string `gorm:"uniqueIndex:uk_rp;size:64" json:"permission_code"`
}

// UserPermission grants (or, when Grant=false, revokes) a permission directly
// to a user, layered on top of the role's permissions.
type UserPermission struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	UserID         uint   `gorm:"uniqueIndex:uk_up;index" json:"user_id"`
	PermissionCode string `gorm:"uniqueIndex:uk_up;size:64" json:"permission_code"`
	Grant          bool   `gorm:"default:true" json:"grant"`
}

// ---------------------------------------------------------------------------
// Clinical core (existing)
// ---------------------------------------------------------------------------

// Patient stores demographic information for a person receiving care.
type Patient struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:128" json:"name"`
	Gender    string         `gorm:"size:8" json:"gender"` // M / F
	BirthDate string         `gorm:"size:16" json:"birth_date"`
	Phone     string         `gorm:"size:32" json:"phone"`
	IDCard    string         `gorm:"size:32" json:"id_card"`
	Address   string         `gorm:"size:256" json:"address"`
	BloodType string         `gorm:"size:8" json:"blood_type"`
	Allergies string         `gorm:"type:text" json:"allergies"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// MedicalCase is a single visit / consultation record for a patient.
type MedicalCase struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	PatientID      uint      `gorm:"index" json:"patient_id"`
	DoctorID       uint      `gorm:"index" json:"doctor_id"`
	DoctorName     string    `gorm:"size:128" json:"doctor_name"`
	RoomID         uint      `gorm:"index" json:"room_id"`
	RoomNo         string    `gorm:"size:32" json:"room_no"`
	VisitDate      time.Time `json:"visit_date"`
	ChiefComplaint string    `gorm:"type:text" json:"chief_complaint"`
	PresentIllness string    `gorm:"type:text" json:"present_illness"`
	PastHistory    string    `gorm:"type:text" json:"past_history"`
	Diagnosis      string    `gorm:"type:text" json:"diagnosis"`
	Notes          string    `gorm:"type:text" json:"notes"`
	CaseNo         string    `gorm:"size:64;index" json:"case_no"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Medicine is a single entry in the drug formulary. Names are bilingual to
// support the globalization requirement; the active admin can extend the list.
type Medicine struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	Code         string  `gorm:"size:64;index" json:"code"`
	NameZH       string  `gorm:"size:256" json:"name_zh"`
	NameEN       string  `gorm:"size:256" json:"name_en"`
	Category     string  `gorm:"size:64;index" json:"category"`
	CategoryEN   string  `gorm:"size:64" json:"category_en"`
	Spec         string  `gorm:"size:128" json:"spec"`
	Manufacturer string  `gorm:"size:128" json:"manufacturer"`
	Unit         string  `gorm:"size:32" json:"unit"`
	Price        float64 `gorm:"type:decimal(12,2)" json:"price"`
	Currency     string  `gorm:"size:8" json:"currency"`
	IsActive     bool    `gorm:"default:true" json:"is_active"`
	// StockQty is the on-hand quantity in the pharmacy (units of Unit).
	// Together with ReorderLevel it drives low-stock warnings on the operations
	// dashboard, which is how the system flags items needing replenishment.
	StockQty float64 `gorm:"type:decimal(12,2);default:0" json:"stock_qty"`
	// ReorderLevel is the low-stock threshold; StockQty <= ReorderLevel triggers
	// a purchase / replenishment alert.
	ReorderLevel float64        `gorm:"type:decimal(12,2);default:0" json:"reorder_level"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Prescription is an order for medicines issued during a case.
type Prescription struct {
	ID             uint               `gorm:"primaryKey" json:"id"`
	PrescriptionNo string             `gorm:"size:64;uniqueIndex" json:"prescription_no"`
	PatientID      uint               `gorm:"index" json:"patient_id"`
	DoctorID       uint               `gorm:"index" json:"doctor_id"`
	DoctorName     string             `gorm:"size:128" json:"doctor_name"`
	RoomID         uint               `gorm:"index" json:"room_id"`
	RoomNo         string             `gorm:"size:32" json:"room_no"`
	CaseID         uint               `json:"case_id"`
	TotalAmount    float64            `gorm:"type:decimal(12,2)" json:"total_amount"`
	Currency       string             `gorm:"size:8" json:"currency"`
	Status         string             `gorm:"size:16" json:"status"` // draft / issued / dispensed
	CreatedAt      time.Time          `json:"created_at"`
	Items          []PrescriptionItem `gorm:"foreignKey:PrescriptionID" json:"items,omitempty"`
}

// PrescriptionItem is a single line within a prescription.
type PrescriptionItem struct {
	ID             uint    `gorm:"primaryKey" json:"id"`
	PrescriptionID uint    `gorm:"index" json:"prescription_id"`
	MedicineID     uint    `gorm:"index" json:"medicine_id"`
	MedicineName   string  `gorm:"size:256" json:"medicine_name"`
	Spec           string  `gorm:"size:128" json:"spec"`
	Quantity       int     `json:"quantity"`
	Dosage         string  `gorm:"size:128" json:"dosage"`
	Frequency      string  `gorm:"size:128" json:"frequency"`
	Price          float64 `gorm:"type:decimal(12,2)" json:"price"`
	Subtotal       float64 `gorm:"type:decimal(12,2)" json:"subtotal"`
}

// ---------------------------------------------------------------------------
// Outpatient rooms
// ---------------------------------------------------------------------------

// Room is an outpatient / operating / radiology room. Rooms can be assigned a
// default doctor and a default printer.
type Room struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	RoomNo      string         `gorm:"size:32;uniqueIndex" json:"room_no"` // e.g. "201"
	NameZH      string         `gorm:"size:128" json:"name_zh"`
	NameEN      string         `gorm:"size:128" json:"name_en"`
	Type        string         `gorm:"size:32;index" json:"type"` // consultation / surgery / imaging / emergency / ward
	Department  string         `gorm:"size:128" json:"department"`
	Floor       string         `gorm:"size:16" json:"floor"`
	DoctorID    uint           `gorm:"index" json:"doctor_id"` // assigned doctor (0 = none)
	DoctorName  string         `gorm:"size:128" json:"doctor_name"`
	PrinterID   uint           `gorm:"index" json:"printer_id"`              // default printer (0 = none)
	Status      string         `gorm:"size:16;default:active" json:"status"` // active / maintenance
	Description string         `gorm:"size:256" json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// ---------------------------------------------------------------------------
// Printers
// ---------------------------------------------------------------------------

// Printer is a physical or network printer bound to a doctor and/or a room.
type Printer struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Name       string         `gorm:"size:128" json:"name"`
	Type       string         `gorm:"size:32" json:"type"`       // label / receipt / a4 / thermal
	Connection string         `gorm:"size:16" json:"connection"` // network / usb / browser
	IPAddress  string         `gorm:"size:64" json:"ip_address"`
	Port       string         `gorm:"size:16" json:"port"`
	DoctorID   uint           `gorm:"index" json:"doctor_id"` // bound doctor (0 = none)
	DoctorName string         `gorm:"size:128" json:"doctor_name"`
	RoomID     uint           `gorm:"index" json:"room_id"` // bound room (0 = none)
	RoomNo     string         `gorm:"size:32" json:"room_no"`
	IsDefault  bool           `gorm:"default:false" json:"is_default"`
	Status     string         `gorm:"size:16;default:active" json:"status"` // active / offline
	Notes      string         `gorm:"size:256" json:"notes"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// PrintJob records a print request for audit + routing purposes.
type PrintJob struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PrinterID   uint      `gorm:"index" json:"printer_id"`
	PrinterName string    `gorm:"size:128" json:"printer_name"`
	DoctorID    uint      `gorm:"index" json:"doctor_id"`
	DoctorName  string    `gorm:"size:128" json:"doctor_name"`
	RoomID      uint      `gorm:"index" json:"room_id"`
	RoomNo      string    `gorm:"size:32" json:"room_no"`
	PatientID   uint      `gorm:"index" json:"patient_id"`
	DocType     string    `gorm:"size:32" json:"doc_type"` // prescription / case / label / report
	RefID       uint      `gorm:"index" json:"ref_id"`
	Summary     string    `gorm:"size:256" json:"summary"`
	Status      string    `gorm:"size:16;default:printed" json:"status"` // printed / failed
	CreatedAt   time.Time `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Surgery management
// ---------------------------------------------------------------------------

// Surgery is an operative procedure scheduled and performed in an OR room.
type Surgery struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	SurgeryNo       string     `gorm:"size:64;uniqueIndex" json:"surgery_no"`
	PatientID       uint       `gorm:"index" json:"patient_id"`
	PatientName     string     `gorm:"size:128" json:"patient_name"`
	SurgeonID       uint       `gorm:"index" json:"surgeon_id"` // lead surgeon
	SurgeonName     string     `gorm:"size:128" json:"surgeon_name"`
	AssistantIDs    string     `gorm:"size:256" json:"assistant_ids"` // comma separated user ids
	AssistantNames  string     `gorm:"size:256" json:"assistant_names"`
	Anesthetist     string     `gorm:"size:128" json:"anesthetist"`
	RoomID          uint       `gorm:"index" json:"room_id"`
	RoomNo          string     `gorm:"size:32" json:"room_no"`
	OperationName   string     `gorm:"size:256" json:"operation_name"`
	OperationNameEN string     `gorm:"size:256" json:"operation_name_en"`
	ScheduledAt     time.Time  `json:"scheduled_at"`
	StartedAt       *time.Time `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at"`
	Status          string     `gorm:"size:16;default:planned" json:"status"` // planned / in_progress / done / cancelled
	AnesthesiaType  string     `gorm:"size:64" json:"anesthesia_type"`
	DurationMin     int        `json:"duration_min"`
	PreOpDiagnosis  string     `gorm:"type:text" json:"pre_op_diagnosis"`
	PostOpDiagnosis string     `gorm:"type:text" json:"post_op_diagnosis"`
	Findings        string     `gorm:"type:text" json:"findings"`
	ProcedureNotes  string     `gorm:"type:text" json:"procedure_notes"`
	Complications   string     `gorm:"type:text" json:"complications"`
	Notes           string     `gorm:"type:text" json:"notes"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// Imaging (CT / X-Ray / MRI / Ultrasound ...)
// ---------------------------------------------------------------------------

// ImagingStudy is an order for diagnostic imaging for a patient.
type ImagingStudy struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	StudyNo     string         `gorm:"size:64;uniqueIndex" json:"study_no"`
	PatientID   uint           `gorm:"index" json:"patient_id"`
	PatientName string         `gorm:"size:128" json:"patient_name"`
	DoctorID    uint           `gorm:"index" json:"doctor_id"` // ordering doctor
	DoctorName  string         `gorm:"size:128" json:"doctor_name"`
	RoomID      uint           `gorm:"index" json:"room_id"`
	RoomNo      string         `gorm:"size:32" json:"room_no"`
	Modality    string         `gorm:"size:32;index" json:"modality"` // CT / XR / MRI / US / ENDO
	BodyPart    string         `gorm:"size:128" json:"body_part"`
	StudyDate   time.Time      `json:"study_date"`
	Indication  string         `gorm:"type:text" json:"indication"`
	Findings    string         `gorm:"type:text" json:"findings"`
	Impression  string         `gorm:"type:text" json:"impression"`
	Status      string         `gorm:"size:16;default:ordered" json:"status"` // ordered / done / reported
	ReportedBy  string         `gorm:"size:128" json:"reported_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Images      []ImagingImage `gorm:"foreignKey:StudyID" json:"images,omitempty"`
}

// ImagingImage is a single uploaded image (film / scan) attached to a study.
type ImagingImage struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	StudyID     uint      `gorm:"index" json:"study_id"`
	FileName    string    `gorm:"size:256" json:"file_name"`
	StoredName  string    `gorm:"size:256" json:"stored_name"` // on disk
	ContentType string    `gorm:"size:64" json:"content_type"`
	FileSize    int64     `json:"file_size"`
	Caption     string    `gorm:"size:256" json:"caption"`
	UploadedBy  uint      `gorm:"index" json:"uploaded_by"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// =============================================================================
//  内部即时通讯（浮窗聊天）
// =============================================================================

// ChatMessage is a single chat entry, either a direct (private) message between
// two users or a message in the hospital-wide group.
//
// Addressing model:
//   - group messages  : Scope = "group",  GroupKey = CarouselKey, ReceiverID = 0
//   - direct messages : Scope = "direct", PeerKey  = peerKey(a,b), ReceiverID = peer id
//
// Retention records are plain rows so that the background retention job can purge
// everything older than the configured window (see internal/chat/retention.go).
type ChatMessage struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Scope       string    `gorm:"size:16;index" json:"scope"`     // group | direct
	GroupKey    string    `gorm:"size:64;index" json:"group_key"` // destination group key
	PeerKey     string    `gorm:"size:64;index" json:"peer_key"`  // canonical "<minID>-<maxID>"
	SenderID    uint      `gorm:"index" json:"sender_id"`
	SenderName  string    `gorm:"size:128" json:"sender_name"`
	ReceiverID  uint      `gorm:"index" json:"receiver_id"`    // 0 for group messages
	ContentType string    `gorm:"size:16" json:"content_type"` // text | image
	Content     string    `gorm:"type:text" json:"content"`
	FileName    string    `gorm:"size:256" json:"file_name"` // stored file name (image)
	OrigName    string    `gorm:"size:256" json:"orig_name"` // original file name
	FileSize    int64     `json:"file_size"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}
