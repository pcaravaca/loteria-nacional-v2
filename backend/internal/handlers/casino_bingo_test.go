package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
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

// basesSeq distingue las bases en memoria cuando un mismo test crea varios env.
var basesSeq atomic.Int64

func nuevoBingoTestEnv(t *testing.T) *bingoTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	// Cada llamada necesita su propia base: con ":memory:" simple el pool de
	// conexiones comparte la misma base entre pruebas (y entre iteraciones de
	// un mismo bucle) y las filas de una se mezclan con las de otra.
	db, err := database.Open(&config.Config{
		DatabasePath: fmt.Sprintf("file:%s_%d?mode=memory&cache=shared",
			nombreBase(t.Name()), basesSeq.Add(1)),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if crudo, err := db.DB(); err == nil {
			_ = crudo.Close()
		}
	})
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

// hitCon manda el estado que el cliente ya ve, como hace la app cuando pide
// una bolilla. Es lo que evita cantar dos veces tras un error de red.
func (e *bingoTestEnv) hitCon(id uint, cantadasEsperadas int) *httptest.ResponseRecorder {
	params := gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c, w := e.contexto("POST", "/casino/bingo/x/hit",
		map[string]any{"cantadas_esperadas": cantadasEsperadas}, params)
	e.h.BingoHit(c)
	return w
}

func (e *bingoTestEnv) reanudar() *httptest.ResponseRecorder {
	c, w := e.contexto("GET", "/casino/bingo/abierta", nil, gin.Params{})
	e.h.BingoReanudarHandler(c)
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
	Variante         string    `json:"variante"`
	NombreVariante   string    `json:"nombre_variante"`
	Monto            int64     `json:"monto"`
	MontoPorCarton   int64     `json:"monto_por_carton"`
	Cartones         int       `json:"cartones"`
	Grids            [][][]int `json:"grids"`
	Orden            []int     `json:"orden"`
	Cantadas         []int     `json:"cantadas"`
	Bolilla          int       `json:"bolilla"`
	CantidadCantadas int       `json:"cantidad_cantadas"`
	TopeCantadas     int       `json:"tope_cantadas"`
	Bolillas         int       `json:"bolillas"`
	Filas            int       `json:"filas"`
	Columnas         int       `json:"columnas"`
	CentroLibre      bool      `json:"centro_libre"`
	Estado           string    `json:"estado"`
	Lineas           int       `json:"lineas"`
	Lleno            bool      `json:"lleno"`
	Patron           string    `json:"patron"`
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

	// una ronda cerrada sigue respondiendo el resultado, pero no vuelve a pagar
	if w := env.close(res.RondaID); w.Code != http.StatusOK {
		t.Errorf("re-cerrar responde %d, esperaba 200", w.Code)
	}
	if w := env.hit(res.RondaID); w.Code != http.StatusOK {
		t.Errorf("cantar en cerrada responde %d, esperaba 200", w.Code)
	}
	if env.saldo() != 49500+res.PremioPagado {
		t.Errorf("el reintento duplico el pago: saldo %d", env.saldo())
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

	// un hit despues de cerrada no vuelve a pagar ni a cantar
	w = env.hit(res.RondaID)
	if w.Code != http.StatusOK {
		t.Errorf("hit 41 responde %d, esperaba 200", w.Code)
	}
	if desp := decodearBingo(t, w); desp.Estado != "pagada" || len(desp.Cantadas) != casino.BingoCantadas {
		t.Errorf("reintento altero la ronda: estado %q con %d cantadas", desp.Estado, len(desp.Cantadas))
	}
	if env.saldo() != 49500+ult.PremioPagado {
		t.Errorf("el reintento duplico el pago: saldo %d", env.saldo())
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
	if ronda.Variante != models.CasinoBingo {
		t.Errorf("variante %q, esperaba %q", ronda.Variante, models.CasinoBingo)
	}
}

// TestBingoStartVariantes abre una jugada por cada variante y verifica que el
// sorteo y el tope respeten la geometría declarada.
func TestBingoStartVariantes(t *testing.T) {
	casos := []struct {
		codigo   string
		bolillas int
		filas    int
		columnas int
		tope     int
		centro   bool
	}{
		{"bingo", 75, 5, 5, 42, true},
		{"bingo90", 90, 3, 9, 60, false},
		{"bingo-esquinas", 75, 5, 5, 42, true},
		{"bingo-diamante", 75, 5, 5, 42, true},
		{"bingo-relampago", 75, 5, 5, 20, true},
	}
	for _, c := range casos {
		env := nuevoBingoTestEnv(t)
		w := env.start(map[string]any{"monto": 500, "cartones": 1, "variante": c.codigo})
		if w.Code != http.StatusCreated {
			t.Fatalf("%s: start respondio %d: %s", c.codigo, w.Code, w.Body.String())
		}
		res := decodearBingo(t, w)
		if res.Variante != c.codigo || res.Bolillas != c.bolillas || res.Filas != c.filas || res.Columnas != c.columnas {
			t.Errorf("%s: geometria %+v", c.codigo, res)
		}
		if res.TopeCantadas != c.tope || res.CantidadCantadas != c.tope {
			t.Errorf("%s: tope %d/%d, esperaba %d", c.codigo, res.TopeCantadas, res.CantidadCantadas, c.tope)
		}
		if res.CentroLibre != c.centro {
			t.Errorf("%s: centro libre %v", c.codigo, res.CentroLibre)
		}
		if len(res.Grids) != 1 || len(res.Grids[0]) != c.filas || len(res.Grids[0][0]) != c.columnas {
			t.Errorf("%s: ticket %+v", c.codigo, res.Grids)
		}
		if len(res.Orden) != c.bolillas {
			t.Errorf("%s: orden de %d bolillas", c.codigo, len(res.Orden))
		}
		var ronda models.BingoRonda
		if err := env.db.First(&ronda, res.RondaID).Error; err != nil {
			t.Fatalf("%s: ronda: %v", c.codigo, err)
		}
		if ronda.Variante != c.codigo {
			t.Errorf("%s: la ronda guardo la variante %q", c.codigo, ronda.Variante)
		}
	}
}

// TestBingoVarianteInvalida verifica que una variante desconocida no abra ronda.
func TestBingoVarianteInvalida(t *testing.T) {
	env := nuevoBingoTestEnv(t)
	w := env.start(map[string]any{"monto": 500, "variante": "bingo-no-existe"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("start con variante invalida respondio %d, esperaba 400", w.Code)
	}
	if env.saldo() != 50000 {
		t.Errorf("la apuesta no deberia descontarse: saldo %d", env.saldo())
	}
}

// TestBingoCierrePorVariante crea rondas con patrones armados y verifica que el
// cierre pague según la variante.
func TestBingoCierrePorVariante(t *testing.T) {
	env := nuevoBingoTestEnv(t)
	grid75 := [][]int{
		{1, 16, 31, 46, 61},
		{2, 17, 32, 47, 62},
		{3, 18, 0, 48, 63},
		{4, 19, 33, 49, 64},
		{5, 20, 34, 50, 65},
	}
	grid90 := [][]int{
		{1, 10, 20, 30, 0, 0, 60, 0, 0},
		{2, 11, 21, 31, 0, 0, 0, 70, 0},
		{3, 12, 0, 0, 40, 50, 0, 0, 80},
	}
	casos := []struct {
		variante string
		grid     [][]int
		cantadas []int
		premio   int64
		patron   string
	}{
		{"bingo-esquinas", grid75, []int{1, 61, 5, 65}, 9000, "Cuatro esquinas"},
		{"bingo-diamante", grid75, []int{32, 18, 48, 33}, 9000, "Diamante"},
		{"bingo90", grid90, []int{1, 10, 20, 30, 60}, 2000, "Una linea"},
	}
	for _, c := range casos {
		ronda := models.BingoRonda{
			UserID:         env.user.ID,
			Variante:       c.variante,
			Monto:          1000,
			MontoPorCarton: 1000,
			Cartones:       1,
			Grids:          [][][]int{c.grid},
			Orden:          []int{1, 2, 3},
			Cantadas:       c.cantadas,
			Estado:         "abierta",
			ClientSeed:     "x",
			ServerSeedHash: "y",
		}
		if err := env.db.Create(&ronda).Error; err != nil {
			t.Fatalf("%s: crear ronda: %v", c.variante, err)
		}
		w := env.close(ronda.ID)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: close respondio %d: %s", c.variante, w.Code, w.Body.String())
		}
		res := decodearBingo(t, w)
		if res.Premio != c.premio || res.PremioPagado != c.premio {
			t.Errorf("%s: premio %d pagado %d, esperaba %d", c.variante, res.Premio, res.PremioPagado, c.premio)
		}
		if res.Patron != c.patron {
			t.Errorf("%s: patron %q, esperaba %q", c.variante, res.Patron, c.patron)
		}
		var guardada models.BingoRonda
		if err := env.db.First(&guardada, ronda.ID).Error; err != nil {
			t.Fatalf("%s: ronda: %v", c.variante, err)
		}
		if guardada.Estado != "pagada" || guardada.Premio != c.premio {
			t.Errorf("%s: ronda guardada estado %q premio %d", c.variante, guardada.Estado, guardada.Premio)
		}
	}
}

// TestBingoStartEsIdempotente verifica que reenviar el start con la misma clave
// (sin datos, timeout, 502) no cobra dos veces: devuelve la ronda ya creada.
func TestBingoStartEsIdempotente(t *testing.T) {
	env := nuevoBingoTestEnv(t)
	cuerpo := map[string]any{"monto": 500, "cartones": 2, "client_seed": "cli-idem", "idempotency_key": "clave-1"}

	w := env.start(cuerpo)
	if w.Code != http.StatusCreated {
		t.Fatalf("start respondio %d: %s", w.Code, w.Body.String())
	}
	primera := decodearBingo(t, w)
	if env.saldo() != 49000 {
		t.Errorf("saldo tras el primer start: %d", env.saldo())
	}

	// Segundo start con la misma clave: debe responder la ronda existente.
	w = env.start(cuerpo)
	if w.Code != http.StatusOK {
		t.Fatalf("start repetido respondio %d: %s", w.Code, w.Body.String())
	}
	var repetida struct {
		Repetida bool           `json:"repetida"`
		Resumen  bingoRespuesta `json:"resumen"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &repetida); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	if !repetida.Repetida {
		t.Errorf("el segundo start no se marco como repetido: %s", w.Body.String())
	}
	if repetida.Resumen.RondaID != primera.RondaID {
		t.Errorf("el repetido abrio otra ronda: %d != %d", repetida.Resumen.RondaID, primera.RondaID)
	}
	if env.saldo() != 49000 {
		t.Errorf("el reintento volvio a cobrar: saldo %d", env.saldo())
	}
	var filas int64
	env.db.Model(&models.BingoRonda{}).Count(&filas)
	if filas != 1 {
		t.Errorf("quedaron %d rondas para la misma clave", filas)
	}
}

// TestBingoHitNoSeRepite verifica que un hit reenviado no canta dos veces.
func TestBingoHitNoSeRepite(t *testing.T) {
	env := nuevoBingoTestEnv(t)
	w := env.start(map[string]any{"monto": 500, "cartones": 1})
	res := decodearBingo(t, w)

	w = env.hitCon(res.RondaID, 0)
	if w.Code != http.StatusOK {
		t.Fatalf("hit respondio %d: %s", w.Code, w.Body.String())
	}
	primera := decodearBingo(t, w)
	if len(primera.Cantadas) != 1 {
		t.Fatalf("primer hit dejo %d cantadas", len(primera.Cantadas))
	}

	// Reintento con el mismo estado: el servidor ya canto, no debe avanzar.
	w = env.hitCon(res.RondaID, 0)
	if w.Code != http.StatusOK {
		t.Fatalf("hit repetido respondio %d: %s", w.Code, w.Body.String())
	}
	if reps := decodearBingo(t, w); len(reps.Cantadas) != 1 {
		t.Errorf("el reintento canto de mas: %d cantadas", len(reps.Cantadas))
	}

	// Al avanzar el contador del cliente, si canta la siguiente.
	w = env.hitCon(res.RondaID, 1)
	if w.Code != http.StatusOK {
		t.Fatalf("hit 2 respondio %d: %s", w.Code, w.Body.String())
	}
	if seg := decodearBingo(t, w); len(seg.Cantadas) != 2 {
		t.Errorf("segundo hit dejo %d cantadas", len(seg.Cantadas))
	}
}

// TestBingoSeReanuda verifica que la ronda abierta se puede retomar.
func TestBingoSeReanuda(t *testing.T) {
	env := nuevoBingoTestEnv(t)
	w := env.start(map[string]any{"monto": 500, "cartones": 1})
	res := decodearBingo(t, w)

	w = env.reanudar()
	if w.Code != http.StatusOK {
		t.Fatalf("reanudar respondio %d: %s", w.Code, w.Body.String())
	}
	var salida struct {
		Reanudar bool           `json:"reanudar"`
		Ronda    bingoRespuesta `json:"ronda"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &salida); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	if !salida.Reanudar || salida.Ronda.RondaID != res.RondaID {
		t.Errorf("no se retomo la ronda: %+v", salida)
	}

	w = env.close(res.RondaID)
	if w.Code != http.StatusOK {
		t.Fatalf("close respondio %d", w.Code)
	}
	w = env.reanudar()
	if w.Code != http.StatusOK {
		t.Fatalf("reanudar sin ronda abierta respondio %d: %s", w.Code, w.Body.String())
	}
	var vacia struct {
		Reanudar bool           `json:"reanudar"`
		Ronda    bingoRespuesta `json:"ronda"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &vacia); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	if vacia.Reanudar || vacia.Ronda.RondaID != 0 {
		t.Errorf("reanudar sin ronda abierta devolvio una ronda: %+v", vacia)
	}
}

// nombreBase vuelve el nombre de la prueba seguro para usarlo como nombre de
// una base SQLite en memoria (sin barras ni espacios).
func nombreBase(nombre string) string {
	out := make([]rune, 0, len(nombre))
	for _, r := range nombre {
		switch {
		case r == '/' || r == ' ' || r == '\\':
			out = append(out, '_')
		default:
			out = append(out, r)
		}
	}
	return string(out)
}
