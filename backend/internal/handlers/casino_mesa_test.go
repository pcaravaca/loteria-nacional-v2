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

// Pruebas de punta a punta de los cuatro juegos de mesa: se abre la ronda
// cobrando la apuesta, se juegan los turnos y se verifica con las semillas.

type mesaTestEnv struct {
	h    *CasinoHandler
	db   *gorm.DB
	user models.User
}

func nuevoMesaTestEnv(t *testing.T) *mesaTestEnv {
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
	user := models.User{Nombre: "Jugador", Email: "pmesa@test", Rol: models.RolCliente, Saldo: 50000, CasinoSeed: semilla}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("usuario: %v", err)
	}
	return &mesaTestEnv{h: NewCasinoHandler(db, &config.Config{}), db: db, user: user}
}

func (e *mesaTestEnv) ctx(metodo, ruta string, cuerpo any, id uint) (*gin.Context, *httptest.ResponseRecorder) {
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

func (e *mesaTestEnv) start(juego string, monto int64) *httptest.ResponseRecorder {
	c, w := e.ctx("POST", "/casino/"+juego+"/start", map[string]any{"monto": monto, "client_seed": "cli-mesa"}, 0)
	switch juego {
	case "domino":
		e.h.DominoStart(c)
	case "ajedrez":
		e.h.AjedrezStart(c)
	case "backgammon":
		e.h.BackgammonStart(c)
	case "tetris":
		e.h.TetrisStart(c)
	}
	return w
}

func (e *mesaTestEnv) move(juego string, id uint, cuerpo any) *httptest.ResponseRecorder {
	c, w := e.ctx("POST", "/casino/"+juego+"/x/move", cuerpo, id)
	switch juego {
	case "domino":
		e.h.DominoMove(c)
	case "ajedrez":
		e.h.AjedrezMove(c)
	case "backgammon":
		e.h.BackgammonMove(c)
	case "tetris":
		e.h.TetrisMove(c)
	}
	return w
}

func (e *mesaTestEnv) verificar(juego string, id uint) *httptest.ResponseRecorder {
	c, w := e.ctx("GET", "/casino/"+juego+"/x/verificar", nil, id)
	switch juego {
	case "domino":
		e.h.DominoVerificar(c)
	case "ajedrez":
		e.h.AjedrezVerificar(c)
	case "backgammon":
		e.h.BackgammonVerificar(c)
	case "tetris":
		e.h.TetrisVerificar(c)
	}
	return w
}

func (e *mesaTestEnv) saldo() int64 {
	var u models.User
	e.db.First(&u, e.user.ID)
	return u.Saldo
}

type mesaEstado struct {
	RondaID        uint     `json:"ronda_id"`
	Monto          int64    `json:"monto"`
	Estado         string   `json:"estado"`
	Mano           [][2]int `json:"mano"`
	Casa           json.RawMessage `json:"casa"`
	CasaFichas     int      `json:"casa_fichas"`
	Tablero        json.RawMessage `json:"tablero"`
	Extremos       [2]int   `json:"extremos"`
	Turno          string   `json:"turno"`
	FEN            string   `json:"fen"`
	Opciones       json.RawMessage `json:"opciones"`
	Jugador        [3]int   `json:"jugador"`
	Dado1          int      `json:"dado1"`
	Dado2          int      `json:"dado2"`
	Piezas         []int    `json:"piezas"`
	Pieza          int      `json:"pieza"`
	PiezasTotales  int      `json:"piezas_totales"`
	Lineas         int      `json:"lineas"`
	Multiplicador  float64  `json:"multiplicador"`
	Premio         int64    `json:"premio"`
	PremioPagado   int64    `json:"premio_pagado"`
	Saldo          int64    `json:"saldo"`
	Verificado     bool     `json:"verificado"`
}

func decodearMesa(t *testing.T, w *httptest.ResponseRecorder) mesaEstado {
	t.Helper()
	var out mesaEstado
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	return out
}

func TestMesaStartCobraYAbre(t *testing.T) {
	for _, juego := range []string{"domino", "ajedrez", "backgammon", "tetris"} {
		env := nuevoMesaTestEnv(t)
		w := env.start(juego, 1000)
		if w.Code != http.StatusCreated {
			t.Fatalf("%s: start respondio %d: %s", juego, w.Code, w.Body.String())
		}
		st := decodearMesa(t, w)
		if st.Estado != "abierta" {
			t.Errorf("%s: estado = %s", juego, st.Estado)
		}
		if env.saldo() != 49000 {
			t.Errorf("%s: saldo = %d, esperaba 49000", juego, env.saldo())
		}
	}
}

func TestMesaApuestaMinimaYMaxima(t *testing.T) {
	env := nuevoMesaTestEnv(t)
	if w := env.start("domino", 1); w.Code != http.StatusBadRequest {
		t.Errorf("apuesta minima: %d", w.Code)
	}
	// El techo de premio recorta la apuesta maxima.
	juego := models.CasinoGame{}
	if err := env.db.Where("codigo = ?", models.CasinoDomino).First(&juego).Error; err != nil {
		t.Fatalf("juego: %v", err)
	}
	maximo := casino.ApuestaMaxima(juego.MaxApuesta, juego.MaxMultipl, juego.MaxPremio, juego.Codigo)
	if w := env.start("domino", maximo+1); w.Code != http.StatusBadRequest {
		t.Errorf("apuesta maxima: %d", w.Code)
	}
}

func TestMesaLimiteDiario(t *testing.T) {
	env := nuevoMesaTestEnv(t)
	if err := env.db.Model(&models.CasinoGame{}).Where("codigo = ?", models.CasinoTetris).
		Update("limite_diario", 1500).Error; err != nil {
		t.Fatalf("update: %v", err)
	}
	if w := env.start("tetris", 1000); w.Code != http.StatusCreated {
		t.Fatalf("primer jugada: %d", w.Code)
	}
	if w := env.start("tetris", 1000); w.Code != http.StatusTooManyRequests {
		t.Errorf("tope diario: %d", w.Code)
	}
}

func TestMesaDominoFlujoYVerificacion(t *testing.T) {
	env := nuevoMesaTestEnv(t)
	w := env.start("domino", 1000)
	if w.Code != http.StatusCreated {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	st := decodearMesa(t, w)
	id := st.RondaID
	for st.Estado == "abierta" {
		var tab [][2]int
		_ = json.Unmarshal(st.Tablero, &tab)
		vacio := len(tab) == 0
		ficha := -1
		for i, f := range st.Mano {
			if vacio || f[0] == st.Extremos[0] || f[1] == st.Extremos[0] ||
				f[0] == st.Extremos[1] || f[1] == st.Extremos[1] {
				ficha = i
				break
			}
		}
		w = env.move("domino", id, map[string]any{"ficha": ficha})
		if w.Code != http.StatusOK {
			t.Fatalf("move respondio %d: %s", w.Code, w.Body.String())
		}
		st = decodearMesa(t, w)
		if st.Estado == "abierta" && st.Turno != "jugador" {
			t.Fatalf("turno inesperado: %s", st.Turno)
		}
	}
	if st.PremioPagado != st.Premio {
		t.Errorf("premio pagado = %d, premio = %d", st.PremioPagado, st.Premio)
	}
	v := env.verificar("domino", id)
	if v.Code != http.StatusOK {
		t.Fatalf("verificar: %d", v.Code)
	}
	if !decodearMesa(t, v).Verificado {
		t.Errorf("la verificacion no paso: %s", v.Body.String())
	}
}

func TestMesaAjedrezFlujoYVerificacion(t *testing.T) {
	env := nuevoMesaTestEnv(t)
	w := env.start("ajedrez", 1000)
	if w.Code != http.StatusCreated {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	st := decodearMesa(t, w)
	var opciones []gin.H
	if err := json.Unmarshal(st.Opciones, &opciones); err != nil {
		t.Fatalf("opciones: %v", err)
	}
	if len(opciones) != 4 {
		t.Fatalf("opciones = %d", len(opciones))
	}
	w = env.move("ajedrez", st.RondaID, map[string]any{"opcion": 0})
	if w.Code != http.StatusOK {
		t.Fatalf("move: %d %s", w.Code, w.Body.String())
	}
	st = decodearMesa(t, w)
	if st.Estado != "pagada" {
		t.Errorf("estado = %s", st.Estado)
	}
	v := env.verificar("ajedrez", st.RondaID)
	if !decodearMesa(t, v).Verificado {
		t.Errorf("la verificacion no paso: %s", v.Body.String())
	}
}

func TestMesaBackgammonFlujoYVerificacion(t *testing.T) {
	env := nuevoMesaTestEnv(t)
	w := env.start("backgammon", 1000)
	if w.Code != http.StatusCreated {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	st := decodearMesa(t, w)
	id := st.RondaID
	for st.Estado == "abierta" {
		if st.Turno != "jugador" {
			t.Fatalf("turno inesperado: %s", st.Turno)
		}
		elegido := 0
		for i := 1; i < 3; i++ {
			if st.Jugador[i] < st.Jugador[elegido] {
				elegido = i
			}
		}
		w = env.move("backgammon", id, map[string]any{"ficha": elegido})
		if w.Code != http.StatusOK {
			t.Fatalf("move: %d %s", w.Code, w.Body.String())
		}
		st = decodearMesa(t, w)
	}
	v := env.verificar("backgammon", id)
	if !decodearMesa(t, v).Verificado {
		t.Errorf("la verificacion no paso: %s", v.Body.String())
	}
}

func TestMesaTetrisFlujoYVerificacion(t *testing.T) {
	env := nuevoMesaTestEnv(t)
	w := env.start("tetris", 1000)
	if w.Code != http.StatusCreated {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	st := decodearMesa(t, w)
	id := st.RondaID
	for st.Estado == "abierta" {
		colocada := false
		for col := 0; col < 6; col++ {
			w = env.move("tetris", id, map[string]any{"columna": col, "rotacion": 0})
			if w.Code == http.StatusOK {
				st = decodearMesa(t, w)
				colocada = true
				break
			}
		}
		if !colocada {
			t.Fatalf("no se pudo colocar la pieza: %s", w.Body.String())
		}
	}
	if st.Pieza == 0 {
		t.Errorf("no se avanzo la pieza")
	}
	v := env.verificar("tetris", id)
	if !decodearMesa(t, v).Verificado {
		t.Errorf("la verificacion no paso: %s", v.Body.String())
	}
}
