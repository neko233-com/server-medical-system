package seed

import (
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/models"
)

func hash(p string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("seed: hash password: %v", err)
	}
	return string(h)
}

// SeedUsers creates the initial admin and demo doctor accounts.
func SeedUsers(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}
	users := []models.User{
		{Username: "admin", PasswordHash: hash("admin123"), Name: "系统管理员", Role: models.RoleAdmin},
		{Username: "doctor", PasswordHash: hash("doctor123"), Name: "张医生", Role: models.RoleDoctor},
	}
	if err := db.Create(&users).Error; err != nil {
		log.Fatalf("seed: create users: %v", err)
	}
	log.Println("seed: created default users (admin/admin123, doctor/doctor123)")
}

// SeedMedicines populates the common drug formulary on first run.
func SeedMedicines(db *gorm.DB) {
	var count int64
	db.Model(&models.Medicine{}).Count(&count)
	if count > 0 {
		return
	}
	items := commonMedicines()
	if err := db.CreateInBatches(items, 50).Error; err != nil {
		log.Fatalf("seed: create medicines: %v", err)
	}
	log.Printf("seed: created %d common medicines", len(items))
}
