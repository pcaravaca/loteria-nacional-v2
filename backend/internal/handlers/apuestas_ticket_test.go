package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"loteria-backend/internal/config"
	"loteria-backend/internal/database"
	"loteria-backend/internal/models"
)

func TestTicketApuesta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := database.Open(&config.Config{DatabasePath: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	u := models.User{Nombre: "Player", Email: "p@test", Rol: models.RolCliente, Saldo: 100000}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	l := models.Loteria{Codigo: "LN", Nombre: "Lotería Nacional", Tipo: models.TipoLoteria, LongitudNum: 2, CostoNumero: 1000, MaxNumeros: 2}
	if err := db.Create(&l).Error; err != nil {
		t.Fatal(err)
	}
	s := models.Sorteo{LoteriaID: l.ID, Titulo: "Sorteo de prueba", Estado: models.SorteoAbierto}
	if err := db.Create(&s).Error; err != nil {
		t.Fatal(err)
	}
	a := models.Apuesta{UserID: u.ID, SorteoID: s.ID, Numeros: []int{3, 7}, Monto: 2000, Estado: models.ApuestaPendiente}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}

	h := NewApuestasHandler(db, &config.Config{})
	r := gin.New()
	r.GET("/apuestas/:id/ticket", func(c *gin.Context) {
		if c.Query("uid") == "otro" {
			c.Set("userID", u.ID+1)
		} else {
			c.Set("userID", u.ID)
		}
		h.Ticket(c)
	})

	// 80 mm: ancho de la POS-80 del mostrador.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/apuestas/1/ticket?ancho=80", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("estado %d: %s", w.Code, w.Body.String())
	}
	html := w.Body.String()
	for _, esperado := range []string{"width: 639px", "Sorteo de prueba", "03", "07", "₡2.000", "LOTERÍA NACIONAL"} {
		if !strings.Contains(html, esperado) {
			t.Fatalf("falta %q en el tique", esperado)
		}
	}

	// Otro usuario no puede ver el comprobante ajeno.
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/apuestas/1/ticket?uid=otro", nil))
	if w2.Code != http.StatusForbidden {
		t.Fatalf("otro usuario recibió %d, quería 403", w2.Code)
	}

	// Un ancho no soportado cae a 58 mm (Sunmi V2).
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/apuestas/1/ticket?ancho=72", nil))
	if !strings.Contains(w3.Body.String(), "width: 463px") {
		t.Fatal("ancho no soportado debe caer a 58 mm")
	}
}
