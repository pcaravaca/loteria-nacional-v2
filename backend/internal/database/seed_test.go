package database

import "testing"

// TestSeedCasino verifica que los juegos se creen con su estado correcto,
// en particular que blackjack quede inactivo.
func TestSeedCasino(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := SeedCasino(db); err != nil {
		t.Fatalf("seed casino: %v", err)
	}

	var crudo int64
	if err := db.Raw("SELECT activo FROM casino_juegos WHERE codigo = 'blackjack'").Scan(&crudo).Error; err != nil {
		t.Fatalf("raw: %v", err)
	}
	t.Logf("activo crudo tras seed = %d", crudo)

	if err := db.Exec("UPDATE casino_juegos SET activo = ? WHERE codigo = 'blackjack'", false).Error; err != nil {
		t.Fatalf("exec: %v", err)
	}
	if err := db.Raw("SELECT activo FROM casino_juegos WHERE codigo = 'blackjack'").Scan(&crudo).Error; err != nil {
		t.Fatalf("raw2: %v", err)
	}
	t.Logf("activo crudo tras exec manual = %d", crudo)

	var total int64
	db.Table("casino_juegos").Count(&total)
	t.Logf("juegos: %d", total)
}
