package models

import (
	"time"

	"gorm.io/gorm"
)

// Role identifies the privilege level of a user account.
type Role string

const (
	RoleDoctor Role = "doctor"
	RoleAdmin  Role = "admin"
)

// User is a clinician or administrator that logs into the system.
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"uniqueIndex;size:64" json:"username"`
	PasswordHash string         `gorm:"size:128" json:"-"`
	Name         string         `gorm:"size:128" json:"name"`
	Role         Role           `gorm:"size:16" json:"role"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Patient stores demographic information for a person receiving care.
type Patient struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:128" json:"name"`
	Gender    string         `gorm:"size:8" json:"gender"` // M / F
	BirthDate string         `gorm:"size:16" json:"birth_date"`
	Phone     string         `gorm:"size:32" json:"phone"`
	IDCard    string         `gorm:"size:32" json:"id_card"`
	Address   string         `gorm:"size:256" json:"address"`
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
	ID           uint           `gorm:"primaryKey" json:"id"`
	Code         string         `gorm:"size:64;index" json:"code"`
	NameZH       string         `gorm:"size:256" json:"name_zh"`
	NameEN       string         `gorm:"size:256" json:"name_en"`
	Category     string         `gorm:"size:64;index" json:"category"`
	CategoryEN   string         `gorm:"size:64" json:"category_en"`
	Spec         string         `gorm:"size:128" json:"spec"`
	Manufacturer string         `gorm:"size:128" json:"manufacturer"`
	Unit         string         `gorm:"size:32" json:"unit"`
	Price        float64        `gorm:"type:decimal(12,2)" json:"price"`
	Currency     string         `gorm:"size:8" json:"currency"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
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
