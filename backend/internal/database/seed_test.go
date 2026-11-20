package database

import (
	"testing"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/config"
	"loteria-backend/internal/models"
)

// TestSeedCasino verifica que los juegos se creen con su estado correcto:
// todos quedan activos, con sus limites y techo de premio.
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
	if total != 40 {
		t.Fatalf("se esperaban 40 juegos, hay %d", total)
	}

	var activos int64
	if err := db.Table("casino_juegos").Where("activo = ?", true).Count(&activos).Error; err != nil {
		t.Fatalf("contar activos: %v", err)
	}
	if activos != 40 {
		t.Fatalf("los juegos deberian quedar activos, hay %d", activos)
	}

	// El bingo queda activo y con la apuesta máxima recortada por el techo de
	// premio (x100 de peor caso: 25 por cartón x 4 cartones).
	var bingo models.CasinoGame
	if err := db.Where("codigo = ?", models.CasinoBingo).First(&bingo).Error; err != nil {
		t.Fatalf("bingo: %v", err)
	}
	if !bingo.Activo {
		t.Error("el bingo debería quedar activo")
	}
	if maximo := casino.ApuestaMaxima(bingo.MaxApuesta, bingo.MaxMultipl, bingo.MaxPremio, bingo.Codigo); maximo != 10000 {
		t.Errorf("apuesta máxima del bingo = %d, esperaba 10000", maximo)
	}

	// Las cinco variantes de bingo quedan como entradas propias, activas.
	for _, codigo := range []string{
		models.CasinoBingo, models.CasinoBingo90, models.CasinoBingoEsquinas,
		models.CasinoBingoDiamante, models.CasinoBingoRelampago,
	} {
		var juego models.CasinoGame
		if err := db.Where("codigo = ?", codigo).First(&juego).Error; err != nil {
			t.Fatalf("%s: %v", codigo, err)
		}
		if !juego.Activo {
			t.Errorf("%s debería quedar activo", codigo)
		}
	}

	var techo int64
	if err := db.Raw("SELECT max_premio FROM casino_juegos WHERE codigo = 'keno'").Scan(&techo).Error; err != nil {
		t.Fatalf("raw: %v", err)
	}
	if techo != 1000000 {
		t.Fatalf("keno deberia tener techo 1000000, tiene %d", techo)
	}
}
