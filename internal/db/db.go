package db

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/config"
	"github.com/neko233-com/server-medical-system/internal/models"
	"github.com/neko233-com/server-medical-system/internal/seed"
)

// Init opens the database based on cfg.DBDriver, runs migrations and returns the
// handle. By default (driver == "sqlite") it uses the pure-Go SQLite driver with
// a local file. Set DB_DRIVER=postgres|mysql + DB_DSN to use an external DB.
func Init(cfg *config.Config) (*gorm.DB, error) {
	var (
		database *gorm.DB
		err      error
	)
	switch cfg.DBDriver {
	case "postgres":
		if cfg.DBDSN == "" {
			return nil, fmt.Errorf("DB_DSN is required when DB_DRIVER=postgres")
		}
		database, err = gorm.Open(postgres.Open(cfg.DBDSN), &gorm.Config{})
	case "mysql":
		if cfg.DBDSN == "" {
			return nil, fmt.Errorf("DB_DSN is required when DB_DRIVER=mysql")
		}
		database, err = gorm.Open(mysql.Open(cfg.DBDSN), &gorm.Config{})
	default: // sqlite
		database, err = gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	}
	if err != nil {
		return nil, err
	}
	if err := database.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.UserPermission{},
		&models.Patient{},
		&models.MedicalCase{},
		&models.Medicine{},
		&models.Prescription{},
		&models.PrescriptionItem{},
		&models.Room{},
		&models.Printer{},
		&models.PrintJob{},
		&models.Surgery{},
		&models.ImagingStudy{},
		&models.ImagingImage{},
		&models.ChatMessage{},
	); err != nil {
		return nil, err
	}
	return database, nil
}

// Seed populates the RBAC catalog, default accounts, rooms, printers and the
// common medicine formulary. All seeders are idempotent.
func Seed(database *gorm.DB, seedDemo bool) {
	seed.SeedRBAC(database)
	seed.SeedUsers(database, seedDemo)
	seed.SeedRooms(database, seedDemo)
	seed.SeedPrinters(database, seedDemo)
	seed.SeedMedicines(database)
}
