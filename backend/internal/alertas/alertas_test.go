package alertas

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"

	"loteria-backend/internal/config"
	"loteria-backend/internal/database"
	"loteria-backend/internal/models"
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

func countAlertas(t *testing.T, db *gorm.DB, tipo string) int64 {
	t.Helper()
	var n int64
	db.Model(&models.AlertaCS{}).Where("tipo = ?", tipo).Count(&n)
	return n
}

// TestDetectarBingoRetenida verifica que una ronda abierta hace más de 15
// minutos se convierte en alerta.
func TestDetectarBingoRetenida(t *testing.T) {
	db := nuevaDB(t)
	usuario := models.User{Nombre: "Jugador", Email: "cs1@test", Rol: models.RolCliente, Saldo: 50000}
	if err := db.Create(&usuario).Error; err != nil {
		t.Fatalf("usuario: %v", err)
	}
	ronda := models.BingoRonda{
		UserID: usuario.ID, Monto: 500, MontoPorCarton: 500, Cartones: 1,
		Estado: "abierta", Cantadas: []int{}, Orden: []int{1, 2, 3},
	}
	if err := db.Create(&ronda).Error; err != nil {
		t.Fatalf("ronda: %v", err)
	}
	// La envejecemos por debajo de la tabla: 30 minutos.
	db.Model(&models.BingoRonda{}).Where("id = ?", ronda.ID).
		Update("created_at", time.Now().Add(-30*time.Minute))

	if n := Detectar(db); n == 0 {
		t.Fatal("no se creó ninguna alerta")
	}
	if countAlertas(t, db, models.AlertaBingoRetenida) != 1 {
		t.Errorf("alertas de bingo retenida: %d", countAlertas(t, db, models.AlertaBingoRetenida))
	}

	// Segunda pasada: la misma alerta no debe duplicarse.
	_ = Detectar(db)
	if countAlertas(t, db, models.AlertaBingoRetenida) != 1 {
		t.Errorf("la alerta se duplicó: %d", countAlertas(t, db, models.AlertaBingoRetenida))
	}
}

// TestDetectarReintentoYDobleCobro cubre las alertas que delatan un corte y un
// pago resuelto dos veces.
func TestDetectarReintentoYDobleCobro(t *testing.T) {
	db := nuevaDB(t)
	usuario := models.User{Nombre: "Jugador", Email: "cs2@test", Rol: models.RolCliente, Saldo: 50000}
	if err := db.Create(&usuario).Error; err != nil {
		t.Fatalf("usuario: %v", err)
	}
	db.Create(&models.AuditLog{
		ActorID: usuario.ID, Accion: "bingo_hit_sincroniza", Entidad: "bingo_ronda",
		EntidadID: "7", RequestID: "req-reintento", Detalle: "sincroniza", IP: "1.2.3.4",
	})
	db.Create(&models.AuditLog{
		ActorID: usuario.ID, Accion: "casino_play", Entidad: "casino_ronda",
		EntidadID: "1", RequestID: "req-doble", Detalle: "juego=mina-de-oro",
	})
	db.Create(&models.AuditLog{
		ActorID: usuario.ID, Accion: "casino_play", Entidad: "casino_ronda",
		EntidadID: "2", RequestID: "req-doble", Detalle: "juego=mina-de-oro",
	})

	_ = Detectar(db)

	if countAlertas(t, db, models.AlertaReintento) != 1 {
		t.Errorf("alertas de reintento: %d", countAlertas(t, db, models.AlertaReintento))
	}
	if countAlertas(t, db, models.AlertaDobleCobro) != 1 {
		t.Errorf("alertas de doble cobro: %d", countAlertas(t, db, models.AlertaDobleCobro))
	}
}
