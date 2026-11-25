package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"loteria-backend/internal/config"
	"loteria-backend/internal/database"
	"loteria-backend/internal/models"
)

func TestTicketResumenTermico(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := database.Open(&config.Config{DatabasePath: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	u1 := models.User{Nombre: "Uno", Email: "u1@test", Rol: models.RolCliente, Saldo: 0}
	u2 := models.User{Nombre: "Dos", Email: "u2@test", Rol: models.RolCliente, Saldo: 0}
	if err := db.Create(&u1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&u2).Error; err != nil {
		t.Fatal(err)
	}
	l := models.Loteria{Codigo: "LN", Nombre: "Lotería Nacional", Tipo: models.TipoLoteria}
	if err := db.Create(&l).Error; err != nil {
		t.Fatal(err)
	}
	s := models.Sorteo{LoteriaID: l.ID, Titulo: "Sorteo", Estado: models.SorteoCerrado}
	if err := db.Create(&s).Error; err != nil {
		t.Fatal(err)
	}
	haceMucho := time.Now().AddDate(0, 0, -10)
	if err := db.Create(&models.Apuesta{UserID: u1.ID, SorteoID: s.ID, Monto: 3000, Premio: 1000, Estado: models.ApuestaPagada, CreatedAt: haceMucho}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Apuesta{UserID: u1.ID, SorteoID: s.ID, Monto: 5000, Premio: 0, Estado: models.ApuestaPendiente}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Apuesta{UserID: u2.ID, SorteoID: s.ID, Monto: 9000, Premio: 0, Estado: models.ApuestaPendiente}).Error; err != nil {
		t.Fatal(err)
	}

	h := NewReportesHandler(db)
	r := gin.New()
	r.GET("/reportes/ticket", func(c *gin.Context) {
		c.Set("rol", c.Query("rol"))
		c.Set("userID", u1.ID)
		h.TicketResumen(c)
	})

	// Cliente: ve sólo sus apuestas; sin filtro de fechas aparecen las dos suyas.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/reportes/ticket?rol=cliente&ancho=58", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("estado %d", w.Code)
	}
	html := w.Body.String()
	if !strings.Contains(html, "width: 463px") || !strings.Contains(html, "₡8.000") {
		t.Fatalf("reporte del cliente incorrecto: %s", html[:300])
	}
	if strings.Contains(html, "₡17.000") { // 3000+5000+9000 sería el total global
		t.Fatal("el cliente vio apuestas de otro usuario")
	}

	// Rango que excluye la apuesta vieja: queda sólo ₡5.000.
	hoy := time.Now().Format("2006-01-02")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/reportes/ticket?rol=cliente&desde="+hoy, nil))
	if !strings.Contains(w2.Body.String(), "₡5.000") || strings.Contains(w2.Body.String(), "₡8.000") {
		t.Fatalf("filtro de fechas no aplicado: %s", w2.Body.String()[:300])
	}

	// Admin: ve todo, incluida la apuesta del otro usuario.
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/reportes/ticket?rol=admin&ancho=80", nil))
	if !strings.Contains(w3.Body.String(), "width: 639px") || !strings.Contains(w3.Body.String(), "₡17.000") {
		t.Fatal("el admin debe ver el total global en 80 mm")
	}
}
