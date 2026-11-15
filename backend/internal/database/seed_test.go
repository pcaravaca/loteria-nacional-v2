package database

import (
	"testing"

	"loteria-backend/internal/config"
)

// TestSeedCasino verifica que los juegos se creen con su estado correcto: los
// seis quedan activos, con sus limites y techo de premio.
func TestSeedCasino(t *testing.T) {
	db, err := Open(&config.Config{DatabasePath: ":memory:"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := SeedCasino(db); err != nil {
		t.Fatalf("seed casino: %v", err)
	}

	var total int64
	if err := db.Table("casino_juegos").Count(&total).Error; err != nil {
		t.Fatalf("contar: %v", err)
	}
	if total != 6 {
		t.Fatalf("se esperaban 6 juegos, hay %d", total)
	}

	var activos int64
	if err := db.Table("casino_juegos").Where("activo = ?", true).Count(&activos).Error; err != nil {
		t.Fatalf("contar activos: %v", err)
	}
	if activos != 6 {
		t.Fatalf("los seis juegos deberian quedar activos, hay %d", activos)
	}

	var techo int64
	if err := db.Raw("SELECT max_premio FROM casino_juegos WHERE codigo = 'keno'").Scan(&techo).Error; err != nil {
		t.Fatalf("raw: %v", err)
	}
	if techo != 1000000 {
		t.Fatalf("keno deberia tener techo 1000000, tiene %d", techo)
	}
}
