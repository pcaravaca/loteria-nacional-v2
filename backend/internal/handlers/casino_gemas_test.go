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

type gemasTestEnv struct {
	h    *CasinoHandler
	db   *gorm.DB
	user models.User
}

func nuevoGemasTestEnv(t *testing.T) *gemasTestEnv {
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
	user := models.User{Nombre: "Jugador", Email: "pgemas@test", Rol: models.RolCliente, Saldo: 50000, CasinoSeed: semilla}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("usuario: %v", err)
	}
	return &gemasTestEnv{h: NewCasinoHandler(db, &config.Config{}), db: db, user: user}
}

func (e *gemasTestEnv) contexto(metodo, ruta string, cuerpo any, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
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

func (e *gemasTestEnv) start(cuerpo any) *httptest.ResponseRecorder {
	c, w := e.contexto("POST", "/casino/gemas/start", cuerpo, gin.Params{})
	e.h.GemasStart(c)
	return w
}

func (e *gemasTestEnv) move(id uint, a, b int) *httptest.ResponseRecorder {
	params := gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c, w := e.contexto("POST", "/casino/gemas/x/move", map[string]any{"a": a, "b": b}, params)
	e.h.GemasMove(c)
	return w
}

func (e *gemasTestEnv) reventar(id uint) *httptest.ResponseRecorder {
	params := gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c, w := e.contexto("POST", "/casino/gemas/x/reventar", nil, params)
	e.h.GemasReventar(c)
	return w
}

func (e *gemasTestEnv) verificar(id uint) *httptest.ResponseRecorder {
	params := gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c, w := e.contexto("GET", "/casino/gemas/x/verificar", nil, params)
	e.h.GemasVerificar(c)
	return w
}

func (e *gemasTestEnv) saldo() int64 {
	var u models.User
	e.db.First(&u, e.user.ID)
	return u.Saldo
}

type gemasRespuesta struct {
	RondaID        uint     `json:"ronda_id"`
	Monto          int64    `json:"monto"`
	Tablero        []int    `json:"tablero"`
	Jugadas        [][2]int `json:"jugadas"`
	Movimientos    int      `json:"movimientos"`
	MovimientosMax int      `json:"movimientos_max"`
	Puntos         int64    `json:"puntos"`
	Multiplicador  float64  `json:"multiplicador"`
	Premio         int64    `json:"premio"`
	PremioPagado   int64    `json:"premio_pagado"`
	Estado         string   `json:"estado"`
	Saldo          int64    `json:"saldo"`
	ServerSeedHash string   `json:"server_seed_hash"`
	Nonce          int64    `json:"nonce"`
}

func decodearGemas(t *testing.T, w *httptest.ResponseRecorder) gemasRespuesta {
	t.Helper()
	var out gemasRespuesta
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	return out
}

// primerMovimientoValido devuelve un intercambio que forma combinación.
func primerMovimientoValido(tablero []int) [2]int {
	movs := casino.GemasMovimientosValidos(tablero)
	if len(movs) == 0 {
		return [2]int{-1, -1}
	}
	return movs[0]
}

func TestGemasStartCobraYAbre(t *testing.T) {
	env := nuevoGemasTestEnv(t)
	w := env.start(map[string]any{"monto": 1000, "client_seed": "cli-gemas"})
	if w.Code != http.StatusCreated {
		t.Fatalf("start respondio %d: %s", w.Code, w.Body.String())
	}
	res := decodearGemas(t, w)
	if res.Monto != 1000 {
		t.Errorf("monto %d", res.Monto)
	}
	if len(res.Tablero) != casino.GemasFilas*casino.GemasColumnas {
		t.Fatalf("tablero de %d celdas", len(res.Tablero))
	}
	for i, v := range res.Tablero {
		if v < 0 || v >= casino.GemasTiposBase {
			t.Fatalf("tipo %d en la celda %d", v, i)
		}
	}
	if res.Estado != "abierta" || res.Movimientos != 0 || res.Puntos != 0 {
		t.Errorf("estado %q movimientos %d puntos %d", res.Estado, res.Movimientos, res.Puntos)
	}
	if res.MovimientosMax != casino.GemasMovimientos {
		t.Errorf("movimientos max %d", res.MovimientosMax)
	}
	if env.saldo() != 49000 {
		t.Errorf("saldo tras abrir: %d", env.saldo())
	}
}

func TestGemasMoveYReventar(t *testing.T) {
	env := nuevoGemasTestEnv(t)
	w := env.start(map[string]any{"monto": 1000})
	if w.Code != http.StatusCreated {
		t.Fatalf("start respondio %d: %s", w.Code, w.Body.String())
	}
	res := decodearGemas(t, w)
	mov := primerMovimientoValido(res.Tablero)
	if mov[0] < 0 {
		t.Fatal("el tablero no tiene movimientos validos")
	}

	w = env.move(res.RondaID, mov[0], mov[1])
	if w.Code != http.StatusOK {
		t.Fatalf("move respondio %d: %s", w.Code, w.Body.String())
	}
	res = decodearGemas(t, w)
	if res.Movimientos != 1 {
		t.Errorf("movimientos %d", res.Movimientos)
	}
	if res.Puntos <= 0 {
		t.Errorf("puntos %d, deberia ser mayor a cero", res.Puntos)
	}
	if res.Premio != casino.GemasPremio(res.Monto, res.Puntos) {
		t.Errorf("premio %d, esperaba %d", res.Premio, casino.GemasPremio(res.Monto, res.Puntos))
	}
	if res.Estado != "abierta" {
		t.Errorf("estado %q", res.Estado)
	}
	if env.saldo() != 49000 {
		t.Errorf("saldo tras mover: %d", env.saldo())
	}

	w = env.reventar(res.RondaID)
	if w.Code != http.StatusOK {
		t.Fatalf("reventar respondio %d: %s", w.Code, w.Body.String())
	}
	res = decodearGemas(t, w)
	if res.Estado != "pagada" {
		t.Errorf("estado %q", res.Estado)
	}
	if res.PremioPagado != res.Premio {
		t.Errorf("premio pagado %d, premio %d", res.PremioPagado, res.Premio)
	}
	if env.saldo() != 49000+res.PremioPagado {
		t.Errorf("saldo final %d, esperaba %d", env.saldo(), 49000+res.PremioPagado)
	}

	// Una ronda cerrada no se puede seguir.
	if w := env.move(res.RondaID, mov[0], mov[1]); w.Code != http.StatusBadRequest {
		t.Errorf("mover en cerrada responde %d, esperaba 400", w.Code)
	}
	if w := env.reventar(res.RondaID); w.Code != http.StatusBadRequest {
		t.Errorf("reventar en cerrada responde %d, esperaba 400", w.Code)
	}
}

func TestGemasMovimientoInvalidoNoGasta(t *testing.T) {
	env := nuevoGemasTestEnv(t)
	w := env.start(map[string]any{"monto": 500})
	if w.Code != http.StatusCreated {
		t.Fatalf("start respondio %d: %s", w.Code, w.Body.String())
	}
	res := decodearGemas(t, w)

	// Celdas no vecinas.
	if w := env.move(res.RondaID, 0, 2); w.Code != http.StatusBadRequest {
		t.Errorf("move no vecino responde %d, esperaba 400", w.Code)
	}
	// Fuera de rango.
	if w := env.move(res.RondaID, -1, 0); w.Code != http.StatusBadRequest {
		t.Errorf("move fuera de rango responde %d, esperaba 400", w.Code)
	}
	// Un intercambio que no forma combinación (buscamos uno).
	valido := [2]int{-1, -1}
	for r := 0; r < casino.GemasFilas && valido[0] < 0; r++ {
		for c := 0; c < casino.GemasColumnas-1; c++ {
			a := r*casino.GemasColumnas + c
			b := a + 1
			if casino.GemasValorVisible(res.Tablero, a, b) <= 0 {
				valido = [2]int{a, b}
				break
			}
		}
	}
	if valido[0] >= 0 {
		if w := env.move(res.RondaID, valido[0], valido[1]); w.Code != http.StatusBadRequest {
			t.Errorf("move sin combinacion responde %d, esperaba 400", w.Code)
		}
	}

	// Verificamos el estado: no se gastó ningún movimiento.
	w = env.reventar(res.RondaID)
	res = decodearGemas(t, w)
	if res.Movimientos != 0 || res.Puntos != 0 {
		t.Errorf("movimientos %d puntos %d, deberia ser 0/0", res.Movimientos, res.Puntos)
	}
}

func TestGemasSeCierraALos25(t *testing.T) {
	env := nuevoGemasTestEnv(t)
	w := env.start(map[string]any{"monto": 500})
	if w.Code != http.StatusCreated {
		t.Fatalf("start respondio %d: %s", w.Code, w.Body.String())
	}
	res := decodearGemas(t, w)

	var ult gemasRespuesta
	for i := 0; i < casino.GemasMovimientos; i++ {
		mov := primerMovimientoValido(res.Tablero)
		if mov[0] < 0 {
			t.Fatalf("sin movimientos en el movimiento %d", i)
		}
		w := env.move(res.RondaID, mov[0], mov[1])
		if w.Code != http.StatusOK {
			t.Fatalf("move %d respondio %d: %s", i+1, w.Code, w.Body.String())
		}
		ult = decodearGemas(t, w)
		res = ult
	}
	if ult.Estado != "pagada" {
		t.Errorf("estado tras %d movimientos: %q", casino.GemasMovimientos, ult.Estado)
	}
	if ult.Movimientos != casino.GemasMovimientos {
		t.Errorf("movimientos %d", ult.Movimientos)
	}
	if ult.PremioPagado != ult.Premio {
		t.Errorf("premio pagado %d, premio %d", ult.PremioPagado, ult.Premio)
	}
	if env.saldo() != 49500+ult.PremioPagado {
		t.Errorf("saldo final %d, esperaba %d", env.saldo(), 49500+ult.PremioPagado)
	}
	if w := env.move(res.RondaID, 0, 1); w.Code != http.StatusBadRequest {
		t.Errorf("move 26 responde %d, esperaba 400", w.Code)
	}
}

func TestGemasLimites(t *testing.T) {
	env := nuevoGemasTestEnv(t)
	if w := env.start(map[string]any{"monto": 100}); w.Code != http.StatusBadRequest {
		t.Errorf("monto bajo el minimo responde %d, esperaba 400", w.Code)
	}
	// El techo por jugada (MaxPremio) recorta la apuesta maxima efectiva:
	// gemas tiene MultiplicadorMaximo 100 y MaxPremio 100000 -> tope efectivo 1000.
	if w := env.start(map[string]any{"monto": 2000}); w.Code != http.StatusBadRequest {
		t.Errorf("monto sobre el maximo efectivo responde %d, esperaba 400", w.Code)
	}
	if w := env.start(map[string]any{"monto": 1000}); w.Code != http.StatusCreated {
		t.Errorf("maximo aceptado responde %d: %s", w.Code, w.Body.String())
	}
}

func TestGemasSemillasYVerificar(t *testing.T) {
	env := nuevoGemasTestEnv(t)
	w := env.start(map[string]any{"monto": 1000, "client_seed": "cli-audit"})
	if w.Code != http.StatusCreated {
		t.Fatalf("start respondio %d: %s", w.Code, w.Body.String())
	}
	res := decodearGemas(t, w)
	mov := primerMovimientoValido(res.Tablero)
	w = env.move(res.RondaID, mov[0], mov[1])
	if w.Code != http.StatusOK {
		t.Fatalf("move respondio %d: %s", w.Code, w.Body.String())
	}
	res = decodearGemas(t, w)

	w = env.verificar(res.RondaID)
	if w.Code != http.StatusOK {
		t.Fatalf("verificar respondio %d: %s", w.Code, w.Body.String())
	}
	var out struct {
		RondaID     uint   `json:"ronda_id"`
		Puntos      int64  `json:"puntos"`
		ServerSeed  string `json:"server_seed"`
		ClientSeed  string `json:"client_seed"`
		Verificado  bool   `json:"verificado"`
		Movimientos int    `json:"movimientos"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !out.Verificado {
		t.Error("la verificacion no coincide")
	}
	if out.ServerSeed == "" || out.ClientSeed != "cli-audit" {
		t.Errorf("semillas: server %q client %q", out.ServerSeed, out.ClientSeed)
	}
	if out.Puntos != res.Puntos || out.Movimientos != res.Movimientos {
		t.Errorf("puntos %d/%d movimientos %d/%d", out.Puntos, res.Puntos, out.Movimientos, res.Movimientos)
	}
}

func TestGemasSeedActivoEnCatalogo(t *testing.T) {
	env := nuevoGemasTestEnv(t)
	var juego models.CasinoGame
	if err := env.db.Where("codigo = ?", models.CasinoGemas).First(&juego).Error; err != nil {
		t.Fatalf("gemas no esta en el catalogo: %v", err)
	}
	if !juego.Activo {
		t.Error("gemas esta desactivado")
	}
	if juego.MinApuesta != 200 || juego.MaxApuesta != 50000 {
		t.Errorf("limites %d/%d", juego.MinApuesta, juego.MaxApuesta)
	}
	if juego.MaxMultipl != casino.MultiplicadorMaximo(models.CasinoGemas, juego.MaxMultipl) {
		t.Errorf("multiplicador max %d", juego.MaxMultipl)
	}
	if juego.RTP <= 0 || juego.RTP > 100 {
		t.Errorf("rtp %v", juego.RTP)
	}
	t.Logf("gemas: RTP declarado %.2f%%", juego.RTP)
}
