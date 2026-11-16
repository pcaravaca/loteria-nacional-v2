package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/config"
	"loteria-backend/internal/database"
	"loteria-backend/internal/models"
)

type pokerTestEnv struct {
	h    *CasinoHandler
	db   *gorm.DB
	user models.User
}

func nuevoPokerTestEnv(t *testing.T) *pokerTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := database.Open(&config.Config{DatabasePath: ":memory:"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := database.SeedCasino(db); err != nil {
		t.Fatalf("seed casino: %v", err)
	}
	semilla, _, err := casino.NuevaSemilla()
	if err != nil {
		t.Fatalf("semilla: %v", err)
	}
	user := models.User{Nombre: "Jugador", Email: "pjoker@test", Rol: models.RolCliente, Saldo: 50000, CasinoSeed: semilla}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("usuario: %v", err)
	}
	return &pokerTestEnv{h: NewCasinoHandler(db, &config.Config{}), db: db, user: user}
}

// contexto arma un gin.Context de prueba con el usuario y el cuerpo dados.
func (e *pokerTestEnv) contexto(metodo, ruta string, cuerpo any, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	var lector *bytes.Reader
	if cuerpo == nil {
		lector = bytes.NewReader(nil)
	} else {
		b, _ := json.Marshal(cuerpo)
		lector = bytes.NewReader(b)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(metodo, ruta, lector)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("userID", e.user.ID)
	c.Params = params
	return c, w
}

func (e *pokerTestEnv) repartir(cuerpo any) *httptest.ResponseRecorder {
	c, w := e.contexto("POST", "/casino/poker", cuerpo, gin.Params{})
	e.h.PokerRepartir(c)
	return w
}

func (e *pokerTestEnv) cobrar(id uint, cuerpo any) *httptest.ResponseRecorder {
	params := gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c, w := e.contexto("POST", "/casino/poker/x/cobrar", cuerpo, params)
	e.h.PokerCobrar(c)
	return w
}

func (e *pokerTestEnv) saldo() int64 {
	var u models.User
	e.db.First(&u, e.user.ID)
	return u.Saldo
}

func decodear[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	return out
}

func TestPokerRepartirYCobrar(t *testing.T) {
	env := nuevoPokerTestEnv(t)
	w := env.repartir(map[string]any{"juego": "jacksorbetter", "monto": 1000, "client_seed": "cli-test"})
	if w.Code != http.StatusCreated {
		t.Fatalf("repartir respondio %d: %s", w.Code, w.Body.String())
	}
	type reparto struct {
		ManoID uint     `json:"mano_id"`
		Ronda  uint     `json:"ronda_id"`
		Cartas []string `json:"cartas"`
	}
	rep := decodear[reparto](t, w)
	if len(rep.Cartas) != 5 {
		t.Fatalf("llegaron %d cartas", len(rep.Cartas))
	}
	if env.saldo() != 49000 {
		t.Errorf("saldo tras repartir: %d", env.saldo())
	}

	w = env.cobrar(rep.ManoID, map[string]any{"retener": []int{0, 1}})
	if w.Code != http.StatusOK {
		t.Fatalf("cobrar respondio %d: %s", w.Code, w.Body.String())
	}
	type cobro struct {
		Premio    int64          `json:"premio"`
		Estado    string         `json:"estado"`
		Resultado map[string]any `json:"resultado"`
	}
	res := decodear[cobro](t, w)
	if res.Estado != "pagada" {
		t.Errorf("estado %q", res.Estado)
	}
	if res.Premio < 0 {
		t.Errorf("premio negativo: %d", res.Premio)
	}
	if env.saldo() != 49000+res.Premio {
		t.Errorf("saldo final %d, esperaba %d", env.saldo(), 49000+res.Premio)
	}

	var ronda models.CasinoRound
	if err := env.db.First(&ronda, rep.Ronda).Error; err != nil {
		t.Fatalf("ronda: %v", err)
	}
	mult, _ := res.Resultado["multipl"].(float64)
	if ronda.Multiplicador != int64(mult) {
		t.Errorf("la ronda quedo x%d y el cierre fue x%v", ronda.Multiplicador, res.Resultado["multipl"])
	}

	// una mano cerrada no se puede cobrar dos veces
	w = env.cobrar(rep.ManoID, map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Errorf("re-cobrar responde %d, esperaba 400", w.Code)
	}
}

func TestPokerCaducadaSeResuelveSola(t *testing.T) {
	env := nuevoPokerTestEnv(t)
	w := env.repartir(map[string]any{"juego": "allamerican", "monto": 1000})
	if w.Code != http.StatusCreated {
		t.Fatalf("repartir respondio %d: %s", w.Code, w.Body.String())
	}
	type reparto struct {
		ManoID uint `json:"mano_id"`
	}
	rep := decodear[reparto](t, w)
	env.db.Model(&models.PokerMano{}).Where("id = ?", rep.ManoID).
		Update("created_at", time.Now().Add(-time.Hour))

	saldoAntes := env.saldo()
	w = env.cobrar(rep.ManoID, map[string]any{})
	if w.Code != http.StatusOK {
		t.Fatalf("cobrar respondio %d: %s", w.Code, w.Body.String())
	}
	type cobro struct {
		Estado string `json:"estado"`
		Premio int64  `json:"premio"`
	}
	res := decodear[cobro](t, w)
	if res.Estado != "caducada" {
		t.Errorf("estado %q, esperaba caducada", res.Estado)
	}
	if env.saldo() != saldoAntes+res.Premio {
		t.Errorf("saldo %d, esperaba %d", env.saldo(), saldoAntes+res.Premio)
	}
}
