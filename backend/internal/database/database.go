package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"loteria-backend/internal/config"
	"loteria-backend/internal/models"
	"loteria-backend/internal/tiempo"
)

// Open abre la base y aplica AutoMigrate. Usa PostgreSQL si cfg.DatabaseURL
// esta definido; si no, crea/usa el archivo SQLite de cfg.DatabasePath.
func Open(cfg *config.Config) (*gorm.DB, error) {
	// Los topes y reportes se calculan en hora CR: fijar la zona antes de
	// cualquier consulta evita que el resultado dependa del TZ del host.
	tiempo.Fijar()
	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)}

	var (
		db  *gorm.DB
		err error
	)
	if cfg.DatabaseURL != "" {
		db, err = gorm.Open(postgres.Open(cfg.DatabaseURL), gormCfg)
		if err != nil {
			return nil, fmt.Errorf("abrir postgres: %w", err)
		}
	} else {
		if dir := filepath.Dir(cfg.DatabasePath); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("crear directorio de datos: %w", err)
			}
		}
		db, err = gorm.Open(sqlite.Open(cfg.DatabasePath), gormCfg)
		if err != nil {
			return nil, fmt.Errorf("abrir sqlite %s: %w", cfg.DatabasePath, err)
		}
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
		&models.BingoRonda{},
		&models.GemasRonda{},
		&models.PokerMano{},
		&models.AuditLog{},
		&models.LimitesJugador{},
		&models.Rifa{},
		&models.RifaNumero{},
		&models.Ajuste{},
		&models.GrupoJugadores{},
		&models.JugadorGrupo{},
		&models.AccesoContenido{},
	); err != nil {
		return nil, fmt.Errorf("automigrate: %w", err)
	}

	return db, nil
}
