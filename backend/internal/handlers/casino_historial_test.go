package handlers

import (
	"bytes"
	"encoding/json"
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

// Pruebas del historial y la verificacion unificados: cubren que el listado
// junta casino_rondas, bingo_rondas y gemas_rondas (incluso sin premio) y que
// el endpoint unificado verifica rondas de las tres fuentes.

type historialTestEnv struct {
	h    *CasinoHandler
	db   *gorm.DB
	user models.User
}

func nuevoHistorialTestEnv(t *testing.T) *historialTestEnv {
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
	user := models.User{Nombre: "Jugador", Email: "phistorial@test", Rol: models.RolCliente, Saldo: 200000, CasinoSeed: semilla}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("usuario: %v", err)
	}
	return &historialTestEnv{h: NewCasinoHandler(db, &config.Config{}), db: db, user: user}
}

func (e *historialTestEnv) contexto(metodo, ruta string, cuerpo any, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
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

func (e *historialTestEnv) rounds(t *testing.T) []RondaHistorial {
	t.Helper()
	c, w := e.contexto("GET", "/casino/rounds", nil, gin.Params{})
	e.h.Rounds(c)
	if w.Code != http.StatusOK {
		t.Fatalf("rounds respondio %d: %s", w.Code, w.Body.String())
	}
	var out []RondaHistorial
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode rounds: %v", err)
	}
	return out
}

func (e *historialTestEnv) verify(t *testing.T, id uint, source string) *httptest.ResponseRecorder {
	t.Helper()
	ruta := "/casino/rounds/" + strconv.FormatUint(uint64(id), 10) + "/verify"
	if source != "" {
		ruta += "?source=" + source
	}
	params := gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c, w := e.contexto("GET", ruta, nil, params)
	e.h.Verify(c)
	return w
}

func (e *historialTestEnv) play(t *testing.T, monto int64) uint {
	t.Helper()
	c, w := e.contexto("POST", "/casino/play", map[string]any{
		"game": "dados", "monto": monto, "client_seed": "cli-historial", "params": map[string]any{"suma": 7},
	}, gin.Params{})
	e.h.Play(c)
	if w.Code != http.StatusOK {
		t.Fatalf("play respondio %d: %s", w.Code, w.Body.String())
	}
	var out struct {
		RondaID uint `json:"ronda_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode play: %v", err)
	}
	return out.RondaID
}

func (e *historialTestEnv) bingoStart(t *testing.T) uint {
	t.Helper()
	c, w := e.contexto("POST", "/casino/bingo/start", map[string]any{
		"monto": 500, "cartones": 1, "client_seed": "cli-historial",
	}, gin.Params{})
	e.h.BingoStart(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("bingo start respondio %d: %s", w.Code, w.Body.String())
	}
	var out struct {
		RondaID uint `json:"ronda_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode bingo: %v", err)
	}
	return out.RondaID
}

func (e *historialTestEnv) bingoHit(t *testing.T, id uint) {
	t.Helper()
	params := gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c, w := e.contexto("POST", "/casino/bingo/x/hit", nil, params)
	e.h.BingoHit(c)
	if w.Code != http.StatusOK {
		t.Fatalf("bingo hit respondio %d: %s", w.Code, w.Body.String())
	}
}

func (e *historialTestEnv) bingoClose(t *testing.T, id uint) {
	t.Helper()
	params := gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c, w := e.contexto("POST", "/casino/bingo/x/close", nil, params)
	e.h.BingoClose(c)
	if w.Code != http.StatusOK {
		t.Fatalf("bingo close respondio %d: %s", w.Code, w.Body.String())
	}
}

func (e *historialTestEnv) gemasStart(t *testing.T) (uint, []int) {
	t.Helper()
	c, w := e.contexto("POST", "/casino/gemas/start", map[string]any{
		"monto": 500, "client_seed": "cli-historial",
	}, gin.Params{})
	e.h.GemasStart(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("gemas start respondio %d: %s", w.Code, w.Body.String())
	}
	var out struct {
		RondaID uint  `json:"ronda_id"`
		Tablero []int `json:"tablero"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode gemas: %v", err)
	}
	return out.RondaID, out.Tablero
}

func (e *historialTestEnv) gemasMove(t *testing.T, id uint, a, b int) {
	t.Helper()
	params := gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c, w := e.contexto("POST", "/casino/gemas/x/move", map[string]any{"a": a, "b": b}, params)
	e.h.GemasMove(c)
	if w.Code != http.StatusOK {
		t.Fatalf("gemas move respondio %d: %s", w.Code, w.Body.String())
	}
}

func (e *historialTestEnv) gemasReventar(t *testing.T, id uint) {
	t.Helper()
	params := gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c, w := e.contexto("POST", "/casino/gemas/x/reventar", nil, params)
	e.h.GemasReventar(c)
	if w.Code != http.StatusOK {
		t.Fatalf("gemas reventar respondio %d: %s", w.Code, w.Body.String())
	}
}

type verificacionRespuesta struct {
	RondaID    uint    `json:"ronda_id"`
	Source     string  `json:"source"`
	Juego      string  `json:"juego"`
	Monto      int64   `json:"monto"`
	Multipl    float64 `json:"multiplicador"`
	Premio     int64   `json:"premio"`
	Puntos     int64   `json:"puntos"`
	Lineas     int     `json:"lineas"`
	Estado     string  `json:"estado"`
	Nonce      int64   `json:"nonce"`
	ClientSeed string  `json:"client_seed"`
	ServerSeed string  `json:"server_seed"`
	Verificado bool    `json:"verificado"`
}

func decodearVerificacion(t *testing.T, w *httptest.ResponseRecorder) verificacionRespuesta {
	t.Helper()
	var out verificacionRespuesta
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode verificacion %s: %v", w.Body.String(), err)
	}
	return out
}

func TestRoundsUnificadoIncluyeLasTresFuentes(t *testing.T) {
	env := nuevoHistorialTestEnv(t)

	casinoID := env.play(t, 1000)
	bingoID := env.bingoStart(t)
	env.bingoClose(t, bingoID) // sin cantar bolillas: premio 0
	gemasID, _ := env.gemasStart(t)
	env.gemasReventar(t, gemasID) // sin puntos: premio 0

	filas := env.rounds(t)
	if len(filas) != 3 {
		t.Fatalf("el historial trae %d filas, esperaba 3: %+v", len(filas), filas)
	}

	porFuente := map[string]RondaHistorial{}
	for _, f := range filas {
		porFuente[f.Source] = f
	}
	for _, fuente := range []string{FuenteCasino, FuenteBingo, FuenteGemas} {
		if _, ok := porFuente[fuente]; !ok {
			t.Errorf("falta una jugada de %s en el historial", fuente)
		}
	}
	if f := porFuente[FuenteCasino]; f.ID != casinoID || f.Game != models.CasinoDados {
		t.Errorf("casino: id %d game %q", f.ID, f.Game)
	}
	if f := porFuente[FuenteBingo]; f.ID != bingoID || f.Game != models.CasinoBingo || f.Premio != 0 {
		t.Errorf("bingo: id %d game %q premio %d", f.ID, f.Game, f.Premio)
	}
	if f := porFuente[FuenteGemas]; f.ID != gemasID || f.Game != models.CasinoGemas || f.Premio != 0 {
		t.Errorf("gemas: id %d game %q premio %d", f.ID, f.Game, f.Premio)
	}

	// Ordenadas de la mas nueva a la mas antigua.
	for i := 1; i < len(filas); i++ {
		if filas[i-1].CreatedAt.Before(filas[i].CreatedAt) {
			t.Errorf("historial desordenado: %v antes de %v", filas[i-1].CreatedAt, filas[i].CreatedAt)
		}
	}
}

func TestVerifyUnificadoRespondeParaLasTresFuentes(t *testing.T) {
	env := nuevoHistorialTestEnv(t)

	casinoID := env.play(t, 1000)

	bingoID := env.bingoStart(t)
	env.bingoHit(t, bingoID)
	env.bingoHit(t, bingoID)
	env.bingoClose(t, bingoID)

	gemasID, tablero := env.gemasStart(t)
	movs := casino.GemasMovimientosValidos(tablero)
	if len(movs) == 0 {
		t.Fatal("el tablero de gemas no tiene movimientos validos")
	}
	env.gemasMove(t, gemasID, movs[0][0], movs[0][1])
	env.gemasReventar(t, gemasID)

	casos := []struct {
		id     uint
		source string
		juego  string
	}{
		{casinoID, FuenteCasino, models.CasinoDados},
		{bingoID, FuenteBingo, models.CasinoBingo},
		{gemasID, FuenteGemas, models.CasinoGemas},
	}
	for _, caso := range casos {
		w := env.verify(t, caso.id, caso.source)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: verificar %d respondio %d: %s", caso.source, caso.id, w.Code, w.Body.String())
		}
		res := decodearVerificacion(t, w)
		if !res.Verificado {
			t.Errorf("%s: la verificacion no coincide", caso.source)
		}
		if res.Juego != caso.juego || res.Source != caso.source {
			t.Errorf("%s: juego %q source %q", caso.source, res.Juego, res.Source)
		}
		if res.ServerSeed == "" || res.ClientSeed != "cli-historial" {
			t.Errorf("%s: semillas server %q client %q", caso.source, res.ServerSeed, res.ClientSeed)
		}
		if res.RondaID != caso.id {
			t.Errorf("%s: ronda_id %d, esperaba %d", caso.source, res.RondaID, caso.id)
		}
	}

	// Sin `source` sigue funcionando para las rondas de casino (cliente viejo).
	w := env.verify(t, casinoID, "")
	if w.Code != http.StatusOK || !decodearVerificacion(t, w).Verificado {
		t.Errorf("verificar sin source: %d %s", w.Code, w.Body.String())
	}

	// Fuente invalida o ronda ajena.
	if w := env.verify(t, casinoID, "ruleta"); w.Code != http.StatusBadRequest {
		t.Errorf("fuente invalida respondio %d, esperaba 400", w.Code)
	}
	if w := env.verify(t, 999999, FuenteBingo); w.Code != http.StatusNotFound {
		t.Errorf("ronda inexistente respondio %d, esperaba 404", w.Code)
	}
}
