package resultados

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// abrirDBAislada usa un nombre distinto de base en memoria para no chocar con
// las loterías que crean otros tests.
func abrirDBAislada(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:calendario_ventana?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if err := db.AutoMigrate(&models.Loteria{}, &models.Sorteo{}, &models.Apuesta{}, &models.User{}, &models.Movimiento{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestGenerarSorteosVentanaIncluyePasado verifica que la ventana crea sorteos
// también hacia atrás: sin eso, los días que quedaron sin sorteo no pueden
// recibir resultados aunque la fuente los publique.
func TestGenerarSorteosVentanaIncluyePasado(t *testing.T) {
	db := abrirDBAislada(t)
	loteria := models.Loteria{Codigo: "NT", Nombre: "Nuevos Tiempos", LongitudNum: 2, Activa: true}
	if err := db.Create(&loteria).Error; err != nil {
		t.Fatalf("loteria: %v", err)
	}
	zona := time.FixedZone("CR", -6*3600)
	hoy := time.Now().In(zona).Truncate(24 * time.Hour)

	n, err := GenerarSorteosVentana(db, zona, 14, 21)
	if err != nil {
		t.Fatalf("GenerarSorteosVentana: %v", err)
	}
	if n == 0 {
		t.Fatal("se esperaba que se crearan sorteos")
	}

	ayer := hoy.AddDate(0, 0, -1)
	var nAyer int64
	if err := db.Model(&models.Sorteo{}).
		Where("loteria_id = ? AND fecha >= ? AND fecha < ?", loteria.ID, ayer, ayer.AddDate(0, 0, 1)).
		Count(&nAyer).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if nAyer != 3 {
		t.Errorf("ayer tuvo %d sorteos, quería 3 (medio día, tarde, noche)", nAyer)
	}

	// Debe ser idempotente: la segunda pasada no crea duplicados.
	n2, err := GenerarSorteosVentana(db, zona, 14, 21)
	if err != nil {
		t.Fatalf("segunda pasada: %v", err)
	}
	if n2 != 0 {
		t.Errorf("la segunda pasada creó %d sorteos, quería 0", n2)
	}
}
