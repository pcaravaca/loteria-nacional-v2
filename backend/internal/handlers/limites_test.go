package handlers

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"loteria-backend/internal/config"
	"loteria-backend/internal/database"
	"loteria-backend/internal/models"
	"loteria-backend/internal/tiempo"
)

// dbPruebas abre una base SQLite en memoria con el esquema completo.
func dbPruebas(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(&config.Config{DatabasePath: ":memory:"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return db
}

func usuarioPrueba(t *testing.T, db *gorm.DB, nombre string) models.User {
	t.Helper()
	u := models.User{Nombre: nombre, Email: nombre + "@test.local", PasswordHash: "x"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("crear usuario: %v", err)
	}
	return u
}

func movimientoPrueba(t *testing.T, db *gorm.DB, userID uint, tipo string, monto int64, cuando time.Time) {
	t.Helper()
	m := models.Movimiento{UserID: userID, Tipo: tipo, Monto: monto, Descripcion: "test", CreatedAt: cuando}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("crear movimiento: %v", err)
	}
}

func TestEdadEn(t *testing.T) {
	nac := time.Date(2000, 6, 15, 0, 0, 0, 0, tiempo.CR)
	if edad := EdadEn(nac, time.Date(2026, 6, 14, 12, 0, 0, 0, tiempo.CR)); edad != 25 {
		t.Errorf("el dia 14 aun no cumplio: esperaba 25, dio %d", edad)
	}
	if edad := EdadEn(nac, time.Date(2026, 6, 15, 0, 0, 0, 1, tiempo.CR)); edad != 26 {
		t.Errorf("en el dia del cumpleanos debe ser 26: dio %d", edad)
	}
}

func TestInicioSemanaCRDevuelveLunes(t *testing.T) {
	inicio := tiempo.InicioSemana()
	if inicio.Weekday() != time.Monday {
		t.Errorf("la semana debia empezar en lunes, empezo en %s", inicio.Weekday())
	}
	if !inicio.Equal(time.Date(inicio.Year(), inicio.Month(), inicio.Day(), 0, 0, 0, 0, tiempo.CR)) {
		t.Error("el inicio de semana no esta a medianoche")
	}
}

func TestEvaluarLimites(t *testing.T) {
	pares := []limitePar{
		{Nombre: "diario", Limite: 1000, Usado: 800},
		{Nombre: "semanal", Limite: 5000, Usado: 900},
	}

	excede, disponible, motivo := EvaluarLimites(pares, 300)
	if !excede {
		t.Error("con 300 mas se superaria el limite diario")
	}
	if motivo != "diario" {
		t.Errorf("el motivo debia ser diario, fue %s", motivo)
	}
	if disponible != 200 {
		t.Errorf("disponible debia ser 200, fue %d", disponible)
	}

	excede, _, _ = EvaluarLimites(pares, 200)
	if excede {
		t.Error("justo en el limite no deberia exceder")
	}

	sinLimites := []limitePar{{Nombre: "diario", Limite: 0, Usado: 0}}
	excede, disponible, _ = EvaluarLimites(sinLimites, 999999)
	if excede || disponible != 0 {
		t.Error("sin limites configurados nada deberia exceder")
	}
}

func TestEvaluarGastoYDeposito(t *testing.T) {
	db := dbPruebas(t)
	u := usuarioPrueba(t, db, "jugador")

	// Sin limites, nada se rechaza.
	if excede, _, _ := EvaluarGasto(db, u.ID, 5000); excede {
		t.Fatal("sin limites no deberia rechazar el gasto")
	}

	// Limite diario de gasto de 1000 con 800 ya apostados.
	l := models.LimitesJugador{UserID: u.ID, GastoDiario: 1000}
	if err := db.Save(&l).Error; err != nil {
		t.Fatalf("guardar limites: %v", err)
	}
	movimientoPrueba(t, db, u.ID, models.MovApuesta, -800, time.Now())

	if excede, disponible, motivo := EvaluarGasto(db, u.ID, 300); !excede || motivo != "diario" || disponible != 200 {
		t.Errorf("debia exceder por diario con 200 disponibles: excede=%v disp=%d motivo=%s", excede, disponible, motivo)
	}
	if excede, _, _ := EvaluarGasto(db, u.ID, 200); excede {
		t.Error("llegar justo al limite deberia permitirse")
	}

	// Los premios no cuentan como gasto.
	movimientoPrueba(t, db, u.ID, models.MovPremio, 1000, time.Now())
	if excede, _, _ := EvaluarGasto(db, u.ID, 200); excede {
		t.Error("un premio no deberia contar como gasto")
	}

	// Depositos: limite semanal de 2000.
	ld := models.LimitesJugador{UserID: u.ID, DepositoSemanal: 2000}
	_ = db.Save(&ld)
	movimientoPrueba(t, db, u.ID, models.MovRecarga, 1500, time.Now())
	if excede, disponible, motivo := EvaluarDeposito(db, u.ID, 600); !excede || motivo != "semanal" || disponible != 500 {
		t.Errorf("deposito debia exceder por semanal con 500 disponibles: excede=%v disp=%d motivo=%s", excede, disponible, motivo)
	}
}

func TestPeriodoDescartaMovimientosViejos(t *testing.T) {
	db := dbPruebas(t)
	u := usuarioPrueba(t, db, "viejo")

	hace8Dias := time.Now().AddDate(0, 0, -8)
	movimientoPrueba(t, db, u.ID, models.MovApuesta, -9000, hace8Dias)

	if excede, _, _ := EvaluarGasto(db, u.ID, 5000); excede {
		t.Error("una apuesta de hace 8 dias no deberia contar para el limite diario")
	}
}
