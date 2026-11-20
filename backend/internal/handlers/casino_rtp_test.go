package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"loteria-backend/internal/config"
	"loteria-backend/internal/database"
	"loteria-backend/internal/models"
)

// TestGamesExpongoDobleRTP verifica que el catalogo publico exponga el RTP
// habitual y el RTP casual de los juegos con decisiones, y que los de azar
// puro sigan trayendo rtp_casual = 0.
func TestGamesExpongoDobleRTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := database.Open(&config.Config{DatabasePath: ":memory:"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := database.SeedCasino(db); err != nil {
		t.Fatalf("seed casino: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/casino/games?todas=1", nil)
	c.Set("userID", uint(1))
	c.Set("rol", models.RolAdmin)
	NewCasinoHandler(db, &config.Config{}).Games(c)

	if w.Code != http.StatusOK {
		t.Fatalf("estado = %d, cuerpo = %s", w.Code, w.Body.String())
	}
	var filas []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &filas); err != nil {
		t.Fatalf("json: %v", err)
	}
	porCodigo := map[string]map[string]any{}
	for _, f := range filas {
		codigo, _ := f["codigo"].(string)
		porCodigo[codigo] = f
	}
	for _, codigo := range []string{
		models.CasinoDomino, models.CasinoAjedrez, models.CasinoBackgammon, models.CasinoTetris,
	} {
		fila, ok := porCodigo[codigo]
		if !ok {
			t.Fatalf("%s no esta en el catalogo", codigo)
		}
		if _, ok := fila["rtp"]; !ok {
			t.Errorf("%s: el catalogo no expone rtp", codigo)
		}
		casual, ok := fila["rtp_casual"].(float64)
		if !ok {
			t.Errorf("%s: el catalogo no expone rtp_casual", codigo)
			continue
		}
		if casual <= 0 {
			t.Errorf("%s: rtp_casual = %.2f, debe estar declarado", codigo, casual)
		}
	}
	for _, codigo := range []string{models.CasinoRuleta, models.CasinoKeno} {
		fila, ok := porCodigo[codigo]
		if !ok {
			t.Fatalf("%s no esta en el catalogo", codigo)
		}
		casual, ok := fila["rtp_casual"].(float64)
		if !ok {
			t.Errorf("%s: el catalogo no expone rtp_casual", codigo)
			continue
		}
		if casual != 0 {
			t.Errorf("%s: rtp_casual = %.2f, deberia ser 0 (no aplica)", codigo, casual)
		}
	}
}
