package handlers

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"loteria-backend/internal/config"
	"loteria-backend/internal/database"
	"loteria-backend/internal/models"
)

func nuevoSweepDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(&config.Config{DatabasePath: ":memory:"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return db
}

// Un sorteo vencido deja de retener la plata: se reembolsa el monto apostado.
func TestSweepReembolsaSorteoVencido(t *testing.T) {
	db := nuevoSweepDB(t)

	user := models.User{Nombre: "Test", Email: "t@t.test", Rol: models.RolCliente, Saldo: 5000}
	db.Create(&user)
	loteria := models.Loteria{Codigo: "TST", Nombre: "Prueba", Tipo: "loteria", CostoNumero: 1000,
		LongitudNum: 2, MaxNumeros: 1, Multiplicador: 50, Activa: true}
	db.Create(&loteria)
	sorteo := models.Sorteo{LoteriaID: loteria.ID, Titulo: "Vencido", Fecha: time.Now().Add(-time.Hour),
		Estado: models.SorteoAbierto}
	db.Create(&sorteo)
	apuesta := models.Apuesta{UserID: user.ID, SorteoID: sorteo.ID, Numeros: []int{7},
		Monto: 1000, Estado: models.ApuestaPendiente}
	db.Create(&apuesta)

	procesados, reembolsadas := SweepSorteosVencidos(db, true, 15*time.Minute)
	if procesados != 1 || reembolsadas != 1 {
		t.Fatalf("esperaba (1,1), dio (%d,%d)", procesados, reembolsadas)
	}

	var u models.User
	db.First(&u, user.ID)
	if u.Saldo != 6000 {
		t.Fatalf("saldo tras reembolso: esperaba 6000, dio %d", u.Saldo)
	}
	var s models.Sorteo
	db.First(&s, sorteo.ID)
	if s.Estado != models.SorteoVencido {
		t.Fatalf("estado del sorteo: esperaba %s, dio %s", models.SorteoVencido, s.Estado)
	}
}

// Con el barrido deshabilitado no se toca nada.
func TestSweepDeshabilitadoNoHaceNada(t *testing.T) {
	db := nuevoSweepDB(t)
	if p, r := SweepSorteosVencidos(db, false, 15*time.Minute); p != 0 || r != 0 {
		t.Fatalf("deshabilitado no debe procesar: (%d,%d)", p, r)
	}
}