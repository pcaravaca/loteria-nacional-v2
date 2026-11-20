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

type bingoTestEnv struct {
	h    *CasinoHandler
	db   *gorm.DB
	user models.User
}

func nuevoBingoTestEnv(t *testing.T) *bingoTestEnv {
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
	user := models.User{Nombre: "Jugador", Email: "pbingo@test", Rol: models.RolCliente, Saldo: 50000, CasinoSeed: semilla}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("usuario: %v", err)
	}
	return &bingoTestEnv{h: NewCasinoHandler(db, &config.Config{}), db: db, user: user}
}

func (e *bingoTestEnv) contexto(metodo, ruta string, cuerpo any, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
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

func (e *bingoTestEnv) start(cuerpo any) *httptest.ResponseRecorder {
	c, w := e.contexto("POST", "/casino/bingo/start", cuerpo, gin.Params{})
	e.h.BingoStart(c)
	return w
}

func (e *bingoTestEnv) hit(id uint) *httptest.ResponseRecorder {
	params := gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c, w := e.contexto("POST", "/casino/bingo/x/hit", nil, params)
	e.h.BingoHit(c)
	return w
}

func (e *bingoTestEnv) close(id uint) *httptest.ResponseRecorder {
	params := gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c, w := e.contexto("POST", "/casino/bingo/x/close", nil, params)
	e.h.BingoClose(c)
	return w
}

func (e *bingoTestEnv) saldo() int64 {
	var u models.User
	e.db.First(&u, e.user.ID)
	return u.Saldo
}

type bingoRespuesta struct {
	RondaID          uint      `json:"ronda_id"`
	Monto            int64     `json:"monto"`
	MontoPorCarton   int64     `json:"monto_por_carton"`
	Cartones         int       `json:"cartones"`
	Grids            [][][]int `json:"grids"`
	Orden            []int     `json:"orden"`
	Cantadas         []int     `json:"cantadas"`
	Bolilla          int       `json:"bolilla"`
	CantidadCantadas int       `json:"cantidad_cantadas"`
	Estado           string    `json:"estado"`
	Lineas           int       `json:"lineas"`
	Lleno            bool      `json:"lleno"`
	Tiers            []int64   `json:"tiers"`
	Premio           int64     `json:"premio"`
	PremioPagado     int64     `json:"premio_pagado"`
	Saldo            int64     `json:"saldo"`
	ServerSeedHash   string    `json:"server_seed_hash"`
	Nonce            int64     `json:"nonce"`
}

func decodearBingo(t *testing.T, w *httptest.ResponseRecorder) bingoRespuesta {
	t.Helper()
	var out bingoRespuesta
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	return out
}

func TestBingoStartCobraYAbre(t *testing.T) {
	env := nuevoBingoTestEnv(t)
	w := env.start(map[string]any{"monto": 1000, "cartones": 2, "client_seed": "cli-test"})
	if w.Code != http.StatusCreated {
		t.Fatalf("start respondio %d: %s", w.Code, w.Body.String())
	}
	res := decodearBingo(t, w)
	if res.Monto != 2000 || res.MontoPorCarton != 1000 || res.Cartones != 2 {
		t.Errorf("montos: total %d, por carton %d, cartones %d", res.Monto, res.MontoPorCarton, res.Cartones)
	}
	if len(res.Grids) != 2 {
		t.Fatalf("se generaron %d cartones", len(res.Grids))
	}
	for n, g := range res.Grids {
		if len(g) != 5 {
			t.Fatalf("cartón %d: %d filas", n, len(g))
		}
		for _, fila := range g {
			if len(fila) != 5 {
				t.Fatalf("cartón %d: fila de %d columnas", n, len(fila))
			}
		}
	}
	if len(res.Orden) != 75 {
		t.Errorf("el orden tiene %d bolillas", len(res.Orden))
	}
	if res.Estado != "abierta" || len(res.Cantadas) != 0 {
		t.Errorf("estado %q con %d cantadas", res.Estado, len(res.Cantadas))
	}
	if env.saldo() != 48000 {
		t.Errorf("saldo tras abrir: %d", env.saldo())
	}
}

func TestBingoHitYClose(t *testing.T) {
	env := nuevoBingoTestEnv(t)
	w := env.start(map[string]any{"monto": 500, "cartones": 1})
	if w.Code != http.StatusCreated {
		t.Fatalf("start respondio %d: %s", w.Code, w.Body.String())
	}
	res := decodearBingo(t, w)

	w = env.hit(res.RondaID)
	if w.Code != http.StatusOK {
		t.Fatalf("hit respondio %d: %s", w.Code, w.Body.String())
	}
	res = decodearBingo(t, w)
	if len(res.Cantadas) != 1 || res.Bolilla != res.Cantadas[0] {
		t.Errorf("cantadas %v, bolilla %d", res.Cantadas, res.Bolilla)
	}
	if res.Estado != "abierta" {
		t.Errorf("estado %q", res.Estado)
	}
	if env.saldo() != 49500 {
		t.Errorf("saldo tras cantar: %d", env.saldo())
	}

	w = env.close(res.RondaID)
	if w.Code != http.StatusOK {
		t.Fatalf("close respondio %d: %s", w.Code, w.Body.String())
	}
	res = decodearBingo(t, w)
	if res.Estado != "pagada" {
		t.Errorf("estado %q", res.Estado)
	}
	if res.PremioPagado != res.Premio {
		t.Errorf("premio pagado %d, premio %d", res.PremioPagado, res.Premio)
	}
	if res.Premio < 0 || res.Premio%500 != 0 {
		t.Errorf("premio %d no es múltiplo de la apuesta", res.Premio)
	}
	if env.saldo() != 49500+res.PremioPagado {
		t.Errorf("saldo final %d, esperaba %d", env.saldo(), 49500+res.PremioPagado)
	}

	// una ronda cerrada no se puede volver a cerrar ni cantar
	if w := env.close(res.RondaID); w.Code != http.StatusBadRequest {
		t.Errorf("re-cerrar responde %d, esperaba 400", w.Code)
	}
	if w := env.hit(res.RondaID); w.Code != http.StatusBadRequest {
		t.Errorf("cantar en cerrada responde %d, esperaba 400", w.Code)
	}
}

func TestBingoSeCierraAlLlegarALas40(t *testing.T) {
	env := nuevoBingoTestEnv(t)
	w := env.start(map[string]any{"monto": 500, "cartones": 1})
	if w.Code != http.StatusCreated {
		t.Fatalf("start respondio %d: %s", w.Code, w.Body.String())
	}
	res := decodearBingo(t, w)

	var ult bingoRespuesta
	for i := 0; i < casino.BingoCantadas; i++ {
		w := env.hit(res.RondaID)
		if w.Code != http.StatusOK {
			t.Fatalf("hit %d respondio %d: %s", i+1, w.Code, w.Body.String())
		}
		ult = decodearBingo(t, w)
	}
	if ult.Estado != "pagada" {
		t.Errorf("estado tras %d cantadas: %q", casino.BingoCantadas, ult.Estado)
	}
	if len(ult.Cantadas) != casino.BingoCantadas {
		t.Errorf("cantadas %d, esperaba %d", len(ult.Cantadas), casino.BingoCantadas)
	}
	if ult.PremioPagado != ult.Premio {
		t.Errorf("premio pagado %d, premio %d", ult.PremioPagado, ult.Premio)
	}
	if env.saldo() != 49500+ult.PremioPagado {
		t.Errorf("saldo final %d, esperaba %d", env.saldo(), 49500+ult.PremioPagado)
	}

	// no se puede cantar de más
	if w := env.hit(res.RondaID); w.Code != http.StatusBadRequest {
		t.Errorf("hit 41 responde %d, esperaba 400", w.Code)
	}
}

func TestBingoCantidadesLimite(t *testing.T) {
	env := nuevoBingoTestEnv(t)
	w := env.start(map[string]any{"monto": 500, "cartones": 0})
	if w.Code != http.StatusCreated {
		t.Fatalf("start respondio %d: %s", w.Code, w.Body.String())
	}
	if res := decodearBingo(t, w); res.Cartones != 1 {
		t.Errorf("cartones 0 deberia redondear a 1, fue %d", res.Cartones)
	}

	w = env.start(map[string]any{"monto": 500, "cartones": 9})
	if w.Code != http.StatusCreated {
		t.Fatalf("start respondio %d: %s", w.Code, w.Body.String())
	}
	if res := decodearBingo(t, w); res.Cartones != casino.BingoMaxCartones {
		t.Errorf("cartones 9 deberia recortarse a %d, fue %d", casino.BingoMaxCartones, res.Cartones)
	}
}

func TestBingoApuestaMaximaPorTecho(t *testing.T) {
	env := nuevoBingoTestEnv(t)
	// Con techo 1.000.000 y multiplicador máximo 100 (25 x 4 cartones), la
	// apuesta por cartón no puede pasar de 10.000.
	w := env.start(map[string]any{"monto": 20000, "cartones": 1})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("start con 20000 respondio %d, esperaba 400: %s", w.Code, w.Body.String())
	}
	w = env.start(map[string]any{"monto": 10000, "cartones": 4})
	if w.Code != http.StatusCreated {
		t.Fatalf("start con 10000 x 4 respondio %d: %s", w.Code, w.Body.String())
	}
	if res := decodearBingo(t, w); res.Monto != 40000 {
		t.Errorf("apuesta total %d, esperaba 40000", res.Monto)
	}
}

func TestBingoSemillasQuedanGuardadas(t *testing.T) {
	env := nuevoBingoTestEnv(t)
	w := env.start(map[string]any{"monto": 500, "cartones": 2, "client_seed": "cli-bingo"})
	if w.Code != http.StatusCreated {
		t.Fatalf("start respondio %d: %s", w.Code, w.Body.String())
	}
	res := decodearBingo(t, w)
	var ronda models.BingoRonda
	if err := env.db.First(&ronda, res.RondaID).Error; err != nil {
		t.Fatalf("ronda: %v", err)
	}
	if ronda.ServerSeedHash == "" || ronda.Nonce != 0 {
		t.Errorf("hash %q nonce %d", ronda.ServerSeedHash, ronda.Nonce)
	}
	if len(ronda.Orden) != 75 || len(ronda.Grids) != 2 {
		t.Errorf("orden %d cartones %d", len(ronda.Orden), len(ronda.Grids))
	}
	if ronda.ClientSeed != "cli-bingo" {
		t.Errorf("client seed %q", ronda.ClientSeed)
	}
}
