package handlers

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/casino"
	"loteria-backend/internal/config"
	"loteria-backend/internal/database"
	"loteria-backend/internal/models"
)

// Prueba de punta a punta de la partida completa de ajedrez: se abre cobrando,
// se juegan turnos hasta el final, se liquida y el verificador reproduce la
// partida y la jugada de la casa desde las semillas.

type ajedrezPartidaEnv struct {
	h    *CasinoHandler
	db   *gorm.DB
	user models.User
}

func nuevoAjedrezPartidaEnv(t *testing.T) *ajedrezPartidaEnv {
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
	user := models.User{Nombre: "Jugador", Email: "pajedrezpartida@test", Rol: models.RolCliente, Saldo: 100000, CasinoSeed: semilla}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("usuario: %v", err)
	}
	return &ajedrezPartidaEnv{h: NewCasinoHandler(db, &config.Config{}), db: db, user: user}
}

func (e *ajedrezPartidaEnv) ctx(metodo, ruta string, cuerpo any, id uint) (*gin.Context, *httptest.ResponseRecorder) {
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
	if id > 0 {
		c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	}
	return c, w
}

func (e *ajedrezPartidaEnv) start(t *testing.T, monto int64, color string) (int, map[string]any) {
	t.Helper()
	c, w := e.ctx("POST", "/casino/ajedrez/partida/start", map[string]any{"monto": monto, "client_seed": "cli-partida", "color": color}, 0)
	e.h.AjedrezPartidaStart(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("start respondio %d: %s", w.Code, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode start: %v", err)
	}
	id := int(out["partida_id"].(float64))
	return id, out
}

func (e *ajedrezPartidaEnv) move(t *testing.T, id int, uci string) (int, map[string]any) {
	t.Helper()
	c, w := e.ctx("POST", "/casino/ajedrez/partida/x/move", map[string]any{"uci": uci}, uint(id))
	e.h.AjedrezPartidaMove(c)
	var out map[string]any
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode move: %v", err)
		}
	}
	return w.Code, out
}

func (e *ajedrezPartidaEnv) verificar(t *testing.T, id int) map[string]any {
	t.Helper()
	c, w := e.ctx("GET", "/casino/ajedrez/partida/x/verificar", nil, uint(id))
	e.h.AjedrezPartidaVerificar(c)
	if w.Code != http.StatusOK {
		t.Fatalf("verificar respondio %d: %s", w.Code, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode verificar: %v", err)
	}
	return out
}

// TestAjedrezPartidaCompletaPuntaAPunto juega una partida completo por HTTP y
// verifica que el verificador la reproduce.
func TestAjedrezPartidaCompletaPuntaAPunto(t *testing.T) {
	e := nuevoAjedrezPartidaEnv(t)
	saldoInicial := e.saldo()
	id, estado := e.start(t, 2000, "blancas")
	if estado["estado"] != "abierta" {
		t.Fatalf("estado = %v", estado["estado"])
	}
	if got := e.saldo(); got != saldoInicial-2000 {
		t.Fatalf("saldo = %d, esperaba %d", got, saldoInicial-2000)
	}
	rng := rand.New(rand.NewSource(7))
	jugadas := 0
	for {
		legales, _ := estado["legales"].([]any)
		if len(legales) == 0 {
			break
		}
		uci := legales[rng.Intn(len(legales))].(string)
		codigo, nuevo := e.move(t, id, uci)
		if codigo != http.StatusOK {
			t.Fatalf("move %s respondio %d: %v", uci, codigo, nuevo)
		}
		estado = nuevo
		jugadas++
		if estado["estado"] == "pagada" {
			break
		}
		if jugadas > 400 {
			t.Fatal("la partida no termino")
		}
	}
	if estado["estado"] != "pagada" {
		t.Fatalf("la partida quedo en %v tras %d turnos", estado["estado"], jugadas)
	}
	t.Logf("partida %d: resultado=%v motivo=%v jugadas=%d premio=%v", id, estado["resultado"], estado["motivo"], jugadas, estado["premio"])
	verificacion := e.verificar(t, id)
	if verificacion["verificado"] != true {
		t.Fatalf("verificacion fallo: %v", verificacion)
	}
	// No se puede seguir jugando una partida cerrada.
	if codigo, _ := e.move(t, id, "e2e4"); codigo != http.StatusBadRequest {
		t.Errorf("move sobre partida cerrada = %d, esperaba 400", codigo)
	}
}

// TestAjedrezPartidaJugadaIlegal verifica que el servidor rechaza una jugada
// fuera de la lista legal.
func TestAjedrezPartidaJugadaIlegal(t *testing.T) {
	e := nuevoAjedrezPartidaEnv(t)
	id, estado := e.start(t, 2000, "blancas")
	legales, _ := estado["legales"].([]any)
	if len(legales) == 0 {
		t.Fatal("la posicion inicial no tiene jugadas")
	}
	// e2e5 nunca es legal con esta configuracion.
	if codigo, _ := e.move(t, id, "e2e5"); codigo != http.StatusBadRequest {
		t.Errorf("jugada ilegal respondio %d, esperaba 400", codigo)
	}
	// Una vez cerrada, verify sigue funcionando.
	verificacion := e.verificar(t, id)
	if verificacion["verificado"] != true {
		t.Fatalf("verificacion de partida abierta fallo: %v", verificacion)
	}
}

// TestAjedrezPartidaNegrasVerifica juega con negras (la casa mueve primero) y
// verifica que la primera jugada de la casa se reproduce.
func TestAjedrezPartidaNegrasVerifica(t *testing.T) {
	e := nuevoAjedrezPartidaEnv(t)
	id, estado := e.start(t, 2000, "negras")
	jugadas, _ := estado["jugadas"].([]any)
	if len(jugadas) != 1 {
		t.Fatalf("con negras la casa debia mover primero: %d jugadas", len(jugadas))
	}
	if estado["turno"] != "b" {
		t.Errorf("turno = %v, esperaba b", estado["turno"])
	}
	verificacion := e.verificar(t, id)
	if verificacion["verificado"] != true {
		t.Fatalf("verificacion con negras fallo: %v", verificacion)
	}
}

// TestAjedrezPartidaLimiteDiario verifica que el tope de la tabla de partidas se
// suma al de las rondas relampago.
func TestAjedrezPartidaLimiteDiario(t *testing.T) {
	e := nuevoAjedrezPartidaEnv(t)
	if err := e.db.Model(&models.CasinoGame{}).Where("codigo = ?", models.CasinoAjedrez).
		Update("limite_diario", 3000).Error; err != nil {
		t.Fatalf("update limite: %v", err)
	}
	if _, estado := e.start(t, 2000, "blancas"); estado["estado"] != "abierta" {
		t.Fatalf("primer start: %v", estado)
	}
	c, w := e.ctx("POST", "/casino/ajedrez/partida/start", map[string]any{"monto": 2000, "client_seed": "cli-partida"}, 0)
	e.h.AjedrezPartidaStart(c)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("segundo start respondio %d, esperaba 422: %s", w.Code, w.Body.String())
	}
}

func (e *ajedrezPartidaEnv) saldo() int64 {
	var u models.User
	e.db.First(&u, e.user.ID)
	return u.Saldo
}
