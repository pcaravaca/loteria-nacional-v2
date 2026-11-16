package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"loteria-backend/internal/models"
)

// datosExportar deja una combinacion minima de registros para probar los CSV.
func TestExportacionesCSV(t *testing.T) {
	db := dbPruebas(t)
	u := usuarioPrueba(t, db, "jugador")

	l := models.Loteria{Codigo: "LN", Nombre: "Loteria Nacional", Tipo: models.TipoLoteria,
		CostoNumero: 1000, LongitudNum: 2, MaxNumeros: 1, Multiplicador: 50, Activa: true}
	if err := db.Create(&l).Error; err != nil {
		t.Fatalf("loteria: %v", err)
	}
	s := models.Sorteo{LoteriaID: l.ID, Titulo: "Sorteo de prueba", Fecha: time.Now(), Estado: models.SorteoAbierto}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("sorteo: %v", err)
	}
	a := models.Apuesta{UserID: u.ID, SorteoID: s.ID, Numeros: []int{7, 19}, Monto: 2000, Estado: models.ApuestaPendiente}
	if err := db.Create(&a).Error; err != nil {
		t.Fatalf("apuesta: %v", err)
	}
	r := models.CasinoRound{UserID: u.ID, GameCodigo: models.CasinoDados, Monto: 500,
		ClientSeed: "cs", ServerSeedHash: "hash", Nonce: 1, Resultado: map[string]any{"suma": float64(7)}}
	if err := db.Create(&r).Error; err != nil {
		t.Fatalf("ronda: %v", err)
	}
	movimientoPrueba(t, db, u.ID, models.MovRecarga, 10000, time.Now())

	h := NewExportacionesHandler(db)
	casos := []struct {
		nombre string
		fn     func(*gin.Context)
		espera []string
	}{
		{"Apuestas", h.Apuestas, []string{"numeros", "7 19", "pendiente"}},
		{"Sorteos", h.Sorteos, []string{"titulo", "Sorteo de prueba", "abierto"}},
		{"RondasCasino", h.RondasCasino, []string{"client_seed", "dados"}},
		{"Movimientos", h.Movimientos, []string{"recarga", "10000"}},
		{"Usuarios", h.Usuarios, []string{"jugador@test.local", "documento"}},
		{"Auditoria", h.Auditoria, []string{"accion"}},
	}

	for _, caso := range casos {
		q := url.Values{}
		c, w := contextoPrueba(http.MethodGet, "/api/admin/reportes/csv/"+caso.nombre, q, 0)
		caso.fn(c)

		body := w.Body.String()
		if !strings.HasPrefix(body, "\ufeff") {
			t.Errorf("%s: el CSV debia arrancar con BOM UTF-8", caso.nombre)
		}
		if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
			t.Errorf("%s: Content-Type inesperado: %s", caso.nombre, ct)
		}
		for _, espera := range caso.espera {
			if !strings.Contains(body, espera) {
				t.Errorf("%s: el CSV no incluye %q", caso.nombre, espera)
			}
		}
	}
}

func TestExportarApuestasFiltraPorUsuario(t *testing.T) {
	db := dbPruebas(t)
	u1 := usuarioPrueba(t, db, "uno")
	u2 := usuarioPrueba(t, db, "dos")

	l := models.Loteria{Codigo: "QN", Nombre: "Quiniela", Tipo: models.TipoQuiniela,
		CostoNumero: 500, LongitudNum: 2, MaxNumeros: 1, Multiplicador: 30, Activa: true}
	_ = db.Create(&l).Error
	s := models.Sorteo{LoteriaID: l.ID, Titulo: "S", Fecha: time.Now(), Estado: models.SorteoAbierto}
	_ = db.Create(&s).Error
	_ = db.Create(&models.Apuesta{UserID: u1.ID, SorteoID: s.ID, Numeros: []int{1}, Monto: 500}).Error
	_ = db.Create(&models.Apuesta{UserID: u2.ID, SorteoID: s.ID, Numeros: []int{2}, Monto: 500}).Error

	h := NewExportacionesHandler(db)
	q := url.Values{"user_id": {"1"}}
	c, w := contextoPrueba(http.MethodGet, "/api/admin/reportes/csv/apuestas", q, 0)
	h.Apuestas(c)

	body := w.Body.String()
	if strings.Count(body, "\n") != 2 { // encabezado + 1 fila
		t.Errorf("el filtro por usuario debia dejar una sola apuesta:\n%s", body)
	}
}
