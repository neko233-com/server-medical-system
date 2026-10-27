package db

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/models"
	"github.com/neko233-com/server-medical-system/internal/seed"
)

// Init opens the database, runs migrations and returns the handle.
func Init(path string) (*gorm.DB, error) {
	database, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := database.AutoMigrate(
		&models.User{},
		&models.Patient{},
		&models.MedicalCase{},
		&models.Medicine{},
		&models.Prescription{},
		&models.PrescriptionItem{},
	); err != nil {
		return nil, err
	}
	return database, nil
}

// Seed populates default users and the common medicine formulary.
func Seed(database *gorm.DB) {
	seed.SeedUsers(database)
	seed.SeedMedicines(database)
}
