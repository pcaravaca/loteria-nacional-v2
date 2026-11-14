package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"loteria-backend/internal/models"
)

// Open abre (y crea si falta) la base SQLite y aplica AutoMigrate.
func Open(path string) (*gorm.DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("crear directorio de datos: %w", err)
		}
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("abrir sqlite %s: %w", path, err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Loteria{},
		&models.Sorteo{},
		&models.Apuesta{},
		&models.Movimiento{},
		&models.CasinoGame{},
		&models.CasinoRound{},
		&models.BlackjackMano{},
	); err != nil {
		return nil, fmt.Errorf("automigrate: %w", err)
	}

	return db, nil
}
