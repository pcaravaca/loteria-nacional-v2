package rifasauto

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"

	"loteria-backend/internal/config"
	"loteria-backend/internal/database"
	"loteria-backend/internal/models"
	"loteria-backend/internal/tiempo"
)

func nuevaDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(&config.Config{
		DatabasePath: fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if crudo, err := db.DB(); err == nil {
			_ = crudo.Close()
		}
	})
	return db
}

// sembrar arma una lotería nacional, una rifa vencida con un número vendido y
// el sorteo de LN del mismo día.
func sembrar(t *testing.T, db *gorm.DB, conResultado bool) (models.Rifa, models.User) {
	t.Helper()
	loteria := models.Loteria{Nombre: "Lotería Nacional", Codigo: "LN", Tipo: "numeros",
		LongitudNum: 4, CostoNumero: 100, Multiplicador: 10, Activa: true}
	if err := db.Create(&loteria).Error; err != nil {
		t.Fatalf("loteria: %v", err)
	}
	user := models.User{Nombre: "Jugador", Email: "rifa@test", Rol: models.RolCliente, Saldo: 1000}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("usuario: %v", err)
	}
	operador := models.User{Nombre: "Op", Email: "op@test", Rol: models.RolAdmin}
	if err := db.Create(&operador).Error; err != nil {
		t.Fatalf("operador: %v", err)
	}

	fecha := time.Date(2026, 11, 20, 23, 0, 0, 0, time.UTC)
	sorteo := models.Sorteo{LoteriaID: loteria.ID, Titulo: "Lotería Nacional · Sorteo",
		Fecha: fecha.In(tiempo.CR), Estado: models.SorteoPagado}
	if conResultado {
		sorteo.NumerosGanadores = []int{7}
		sorteo.NumerosRaw = "[7]"
	}
	if err := db.Create(&sorteo).Error; err != nil {
		t.Fatalf("sorteo: %v", err)
	}

	rifa := models.Rifa{Titulo: "Rifa de prueba", Premio: "TV", PremioMonto: 5000,
		PrecioNumero: 100, DesdeNumero: 0, HastaNumero: 49,
		FechaSorteo: fecha, Estado: models.RifaAbierta, TipoSorteo: models.RifaSorteoAzar,
		CreadoPor: operador.ID}
	if err := db.Create(&rifa).Error; err != nil {
		t.Fatalf("rifa: %v", err)
	}
	for n := 0; n < 50; n++ {
		var uid *uint
		if n == 7 {
			u := user.ID
			uid = &u
		}
		if err := db.Create(&models.RifaNumero{RifaID: rifa.ID, Numero: n, UserID: uid}).Error; err != nil {
			t.Fatalf("numero: %v", err)
		}
	}
	return rifa, user
}

// TestGraduaConLoteriaNacional verifica que una rifa vencida se gradúa con el
// número del sorteo de LN del mismo día y paga al que lo tiene.
func TestGraduaConLoteriaNacional(t *testing.T) {
	db := nuevaDB(t)
	rifa, user := sembrar(t, db, true)

	ok, motivo := GraduarUna(db, rifa.ID)
	if !ok {
		t.Fatalf("no se graduó: %s", motivo)
	}
	var actual models.Rifa
	db.First(&actual, rifa.ID)
	if actual.Estado != models.RifaSorteada {
		t.Errorf("estado %q", actual.Estado)
	}
	if len(actual.NumerosGanadores) != 1 || actual.NumerosGanadores[0] != 7 {
		t.Errorf("ganadores %v", actual.NumerosGanadores)
	}
	var saldo models.User
	db.First(&saldo, user.ID)
	if saldo.Saldo != 1000+5000 {
		t.Errorf("saldo del ganador %d, esperaba 6000", saldo.Saldo)
	}
}

// TestSinResultadoQuedaPendiente verifica que si el sorteo de LN no tiene
// números, la rifa pasa a pendiente y deja alerta.
func TestSinResultadoQuedaPendiente(t *testing.T) {
	db := nuevaDB(t)
	rifa, _ := sembrar(t, db, false)

	if _, motivo := GraduarUna(db, rifa.ID); motivo == "" {
		t.Fatal("se graduó sin resultado")
	}
	var actual models.Rifa
	db.First(&actual, rifa.ID)
	if actual.Estado != models.RifaPendiente {
		t.Errorf("estado %q, esperaba pendiente", actual.Estado)
	}
	var alertas int64
	db.Model(&models.AlertaCS{}).Where("tipo = ?", models.AlertaRifaPendiente).Count(&alertas)
	if alertas == 0 {
		t.Error("no se dejó la alerta de rifa pendiente")
	}
}
