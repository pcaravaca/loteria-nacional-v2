package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"loteria-backend/internal/models"
)

// contextoPrueba arma un gin.Context con query params y opcionalmente un
// usuario en el contexto (sin pasar por el middleware JWT).
func contextoPrueba(metodo, ruta string, query url.Values, userID uint) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(metodo, ruta+"?"+query.Encode(), nil)
	if userID != 0 {
		c.Set("userID", userID)
	}
	return c, w
}

func TestRegistrarAuditoria(t *testing.T) {
	db := dbPruebas(t)
	u := usuarioPrueba(t, db, "admin")

	RegistrarAuditoria(db, nil, u.ID, "saldo.ajustar", "usuario", fmt.Sprintf("%d", u.ID), "monto=100")

	var entradas []models.AuditLog
	if err := db.Find(&entradas).Error; err != nil {
		t.Fatalf("leer auditoria: %v", err)
	}
	if len(entradas) != 1 {
		t.Fatalf("se esperaba 1 entrada, hay %d", len(entradas))
	}
	e := entradas[0]
	if e.Accion != "saldo.ajustar" || e.Entidad != "usuario" || e.ActorID != u.ID {
		t.Errorf("entrada mal registrada: %+v", e)
	}
	if e.ActorName != "admin" {
		t.Errorf("el nombre del actor debia resolverse, fue %q", e.ActorName)
	}
}

func TestRangoFechas(t *testing.T) {
	q := url.Values{"desde": {"2026-11-01"}, "hasta": {"2026-11-15"}}
	c, _ := contextoPrueba(http.MethodGet, "/", q, 0)

	ini, fin, ok := rangoFechas(c)
	if !ok {
		t.Fatal("con desde/hasta presentes el rango debe ser valido")
	}
	if ini.Day() != 1 || ini.Month() != time.November {
		t.Errorf("inicio inesperado: %v", ini)
	}
	// Hasta el dia 15 se interpreta como todo ese dia: fin al inicio del 16.
	if fin.Day() != 16 {
		t.Errorf("el fin debia cubrir hasta el 15 completo, fue %v", fin)
	}

	q.Set("desde", "abc")
	cMal, _ := contextoPrueba(http.MethodGet, "/", q, 0)
	if _, _, ok := rangoFechas(cMal); ok {
		t.Error("una fecha con formato distinto deberia invalidar el rango")
	}
}

func TestAuditoriaListarFiltros(t *testing.T) {
	db := dbPruebas(t)
	u := usuarioPrueba(t, db, "actor")
	RegistrarAuditoria(db, nil, u.ID, "kyc.verificar", "usuario", "7", "")
	RegistrarAuditoria(db, nil, u.ID, "sorteo.cerrar", "sorteo", "3", "")

	h := NewAuditoriaHandler(db)
	q := url.Values{"accion": {"kyc.verificar"}}
	c, w := contextoPrueba(http.MethodGet, "/api/admin/auditoria", q, u.ID)
	h.Listar(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "kyc.verificar") {
		t.Error("la respuesta no incluye la accion filtrada")
	}
	if strings.Contains(w.Body.String(), "sorteo.cerrar") {
		t.Error("el filtro por accion no funciono")
	}
}
