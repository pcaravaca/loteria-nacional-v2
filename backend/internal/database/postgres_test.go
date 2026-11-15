package database

import (
	"os"
	"testing"

	"loteria-backend/internal/config"
	"loteria-backend/internal/models"
)

// TestContraPostgres corre AutoMigrate y SeedCasino contra la base de
// desarrollo real. Se salta salvo que LOTERIA_TEST_DATABASE_URL este definida,
// asi el test diario (SQLite en memoria) no depende de la red.
func TestContraPostgres(t *testing.T) {
	url := os.Getenv("LOTERIA_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("LOTERIA_TEST_DATABASE_URL no definida: se omite la prueba contra PostgreSQL")
	}

	db, err := Open(&config.Config{DatabaseURL: url})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	if err := SeedCasino(db); err != nil {
		t.Fatalf("seed casino: %v", err)
	}

	var juegos int64
	if err := db.Model(&models.CasinoGame{}).Count(&juegos).Error; err != nil {
		t.Fatalf("contar juegos: %v", err)
	}
	if juegos != 6 {
		t.Fatalf("seed casino: se esperaban 6 juegos, hay %d", juegos)
	}

	// El booleano de la columna activo debe leerse como bool en PostgreSQL.
	var activo bool
	if err := db.Raw("SELECT activo FROM casino_juegos WHERE codigo = 'blackjack'").Scan(&activo).Error; err != nil {
		t.Fatalf("leer booleano: %v", err)
	}
	if !activo {
		t.Fatal("el seed deberia dejar los seis juegos activos, blackjack aparece inactivo")
	}
}
