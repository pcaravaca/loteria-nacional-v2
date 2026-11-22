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
// junta casino_rondas, bingo_rondas, gemas_rondas, casino_blackjack y las cuatro
// tablas de mesa (domino, ajedrez, backgammon, tetris), y que el endpoint
// unificado verifica rondas de todas las fuentes, abiertas y cerradas.

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
	var out struct {
		Items []RondaHistorial `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode rounds: %v", err)
	}
	return out.Items
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

func (e *historialTestEnv) blackjackStart(t *testing.T) uint {
	t.Helper()
	c, w := e.contexto("POST", "/casino/blackjack/start", map[string]any{
		"monto": 500, "client_seed": "cli-historial",
	}, gin.Params{})
	e.h.BlackjackStart(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("blackjack start respondio %d: %s", w.Code, w.Body.String())
	}
	var out struct {
		ManoID uint `json:"mano_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode blackjack: %v", err)
	}
	return out.ManoID
}

func TestRoundsUnificadoIncluyeBlackjack(t *testing.T) {
	env := nuevoHistorialTestEnv(t)

	id := env.blackjackStart(t)
	filas := env.rounds(t)
	if len(filas) != 1 {
		t.Fatalf("el historial trae %d filas, esperaba 1: %+v", len(filas), filas)
	}
	f := filas[0]
	if f.Source != FuenteBlackjack || f.ID != id || f.Game != models.CasinoBlackjack {
		t.Errorf("fila blackjack: source %q id %d game %q", f.Source, f.ID, f.Game)
	}
	if f.Monto != 500 {
		t.Errorf("monto %d, esperaba 500", f.Monto)
	}
}

func TestVerifyUnificadoRespondeParaBlackjack(t *testing.T) {
	env := nuevoHistorialTestEnv(t)

	id := env.blackjackStart(t)
	w := env.verify(t, id, FuenteBlackjack)
	if w.Code != http.StatusOK {
		t.Fatalf("verificar blackjack respondio %d: %s", w.Code, w.Body.String())
	}
	res := decodearVerificacion(t, w)
	if !res.Verificado {
		t.Errorf("la verificacion de la mano no coincide: %+v", res)
	}
	if res.Juego != models.CasinoBlackjack || res.Source != FuenteBlackjack {
		t.Errorf("juego %q source %q", res.Juego, res.Source)
	}
	if res.ServerSeed == "" || res.ClientSeed != "cli-historial" {
		t.Errorf("semillas server %q client %q", res.ServerSeed, res.ClientSeed)
	}
	// Sin `source` tambien la encuentra.
	if w := env.verify(t, id, ""); w.Code != http.StatusOK || !decodearVerificacion(t, w).Verificado {
		t.Errorf("verificar sin source: %d %s", w.Code, w.Body.String())
	}
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

// mesaStart abre una ronda de un juego de mesa y devuelve su id y el estado
// inicial. Reutiliza los handlers igual que las pruebas de mesa.
func (e *historialTestEnv) mesaStart(t *testing.T, juego string, monto int64) (uint, mesaEstado) {
	t.Helper()
	c, w := e.contexto("POST", "/casino/"+juego+"/start", map[string]any{
		"monto": monto, "client_seed": "cli-historial",
	}, gin.Params{})
	switch juego {
	case "domino":
		e.h.DominoStart(c)
	case "ajedrez":
		e.h.AjedrezStart(c)
	case "backgammon":
		e.h.BackgammonStart(c)
	case "tetris":
		e.h.TetrisStart(c)
	default:
		t.Fatalf("juego de mesa desconocido: %s", juego)
	}
	if w.Code != http.StatusCreated {
		t.Fatalf("%s start respondio %d: %s", juego, w.Code, w.Body.String())
	}
	var out struct {
		RondaID uint `json:"ronda_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s: %v", juego, err)
	}
	return out.RondaID, decodearMesa(t, w)
}

// mesaMove aplica un turno de un juego de mesa.
func (e *historialTestEnv) mesaMove(t *testing.T, juego string, id uint, cuerpo any) *httptest.ResponseRecorder {
	t.Helper()
	params := gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
	c, w := e.contexto("POST", "/casino/"+juego+"/x/move", cuerpo, params)
	switch juego {
	case "domino":
		e.h.DominoMove(c)
	case "ajedrez":
		e.h.AjedrezMove(c)
	case "backgammon":
		e.h.BackgammonMove(c)
	case "tetris":
		e.h.TetrisMove(c)
	default:
		t.Fatalf("juego de mesa desconocido: %s", juego)
	}
	return w
}

func (e *historialTestEnv) verificarCerrada(t *testing.T, id uint, fuente, juego string) {
	t.Helper()
	w := e.verify(t, id, fuente)
	if w.Code != http.StatusOK {
		t.Fatalf("%s: verificar %d respondio %d: %s", fuente, id, w.Code, w.Body.String())
	}
	res := decodearVerificacion(t, w)
	if !res.Verificado {
		t.Errorf("%s: la verificacion de la ronda cerrada no coincide: %s", fuente, w.Body.String())
	}
	if res.Juego != juego || res.Source != fuente {
		t.Errorf("%s: juego %q source %q", fuente, res.Juego, res.Source)
	}
	if res.Monto != 1000 {
		t.Errorf("%s: monto %d, esperaba 1000", fuente, res.Monto)
	}
	if res.ServerSeed == "" || res.ClientSeed != "cli-historial" {
		t.Errorf("%s: semillas server %q client %q", fuente, res.ServerSeed, res.ClientSeed)
	}
}

var fuentesDeMesa = []struct {
	juego  string
	game   string
	fuente string
}{
	{"domino", models.CasinoDomino, FuenteDomino},
	{"ajedrez", models.CasinoAjedrez, FuenteAjedrez},
	{"backgammon", models.CasinoBackgammon, FuenteBackgammon},
	{"tetris", models.CasinoTetris, FuenteTetris},
}

func TestRoundsUnificadoIncluyeLosCuatroJuegosDeMesa(t *testing.T) {
	env := nuevoHistorialTestEnv(t)

	ids := map[string]uint{}
	for _, c := range fuentesDeMesa {
		id, _ := env.mesaStart(t, c.juego, 1000)
		ids[c.juego] = id
	}

	filas := env.rounds(t)
	if len(filas) != len(fuentesDeMesa) {
		t.Fatalf("el historial trae %d filas, esperaba %d: %+v", len(filas), len(fuentesDeMesa), filas)
	}
	porFuente := map[string]RondaHistorial{}
	for _, f := range filas {
		porFuente[f.Source] = f
	}
	for _, c := range fuentesDeMesa {
		f, ok := porFuente[c.fuente]
		if !ok {
			t.Errorf("falta una jugada de %s en el historial", c.fuente)
			continue
		}
		if f.ID != ids[c.juego] || f.Game != c.game || f.Monto != 1000 {
			t.Errorf("%s: id %d game %q monto %d", c.fuente, f.ID, f.Game, f.Monto)
		}
		if f.Nombre == "" {
			t.Errorf("%s: la fila no trae el nombre del juego", c.fuente)
		}
		if f.Estado != "abierta" {
			t.Errorf("%s: estado %q, esperaba abierta", c.fuente, f.Estado)
		}
		if f.Resultado == nil {
			t.Errorf("%s: la fila no trae un resultado resumido", c.fuente)
		}
	}
}

func TestVerifyUnificadoRespondeParaLosCuatroJuegosDeMesa(t *testing.T) {
	env := nuevoHistorialTestEnv(t)

	for _, c := range fuentesDeMesa {
		id, _ := env.mesaStart(t, c.juego, 1000)
		w := env.verify(t, id, c.fuente)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: verificar %d respondio %d: %s", c.fuente, id, w.Code, w.Body.String())
		}
		res := decodearVerificacion(t, w)
		if !res.Verificado {
			t.Errorf("%s: la verificacion de la ronda abierta no coincide: %s", c.fuente, w.Body.String())
		}
		if res.Juego != c.game || res.Source != c.fuente {
			t.Errorf("%s: juego %q source %q", c.fuente, res.Juego, res.Source)
		}
		if res.ServerSeed == "" || res.ClientSeed != "cli-historial" {
			t.Errorf("%s: semillas server %q client %q", c.fuente, res.ServerSeed, res.ClientSeed)
		}
		// Sin `source` tambien la encuentra.
		if w := env.verify(t, id, ""); w.Code != http.StatusOK || !decodearVerificacion(t, w).Verificado {
			t.Errorf("%s: verificar sin source: %d %s", c.fuente, w.Code, w.Body.String())
		}
	}
}

// TestGamesExponenModalidadesConMultMax verifica que el catalogo que consume el
// frontend traiga el mayor multiplicador por modalidad: el encabezado del modal
// muestra ese tope segun la modalidad elegida (hilo racha x6, no el x2 del
// catalogo).
func TestGamesExponenModalidadesConMultMax(t *testing.T) {
	env := nuevoHistorialTestEnv(t)
	c, w := env.contexto("GET", "/casino/games", nil, gin.Params{})
	c.Set("rol", models.RolCliente)
	env.h.Games(c)
	if w.Code != http.StatusOK {
		t.Fatalf("games respondio %d: %s", w.Code, w.Body.String())
	}
	var juegos []struct {
		Codigo      string `json:"codigo"`
		Modalidades []struct {
			Codigo  string `json:"codigo"`
			MultMax int64  `json:"mult_max"`
		} `json:"modalidades"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &juegos); err != nil {
		t.Fatalf("decode games: %v", err)
	}
	type modalidadVista struct {
		Codigo  string
		MultMax int64
	}
	porCodigo := map[string][]modalidadVista{}
	for _, j := range juegos {
		for _, m := range j.Modalidades {
			porCodigo[j.Codigo] = append(porCodigo[j.Codigo], modalidadVista{Codigo: m.Codigo, MultMax: m.MultMax})
		}
	}
	hilo := porCodigo[models.CasinoHilo]
	if len(hilo) == 0 {
		t.Fatalf("hilo no expone modalidades: %+v", hilo)
	}
	for _, m := range hilo {
		switch m.Codigo {
		case "clasico":
			if m.MultMax != 2 {
				t.Errorf("hilo clasico: mult_max %d, queria 2", m.MultMax)
			}
		case "racha":
			if m.MultMax != 6 {
				t.Errorf("hilo racha: mult_max %d, queria 6", m.MultMax)
			}
		}
	}
	if dados := porCodigo[models.CasinoDados]; len(dados) != 0 {
		t.Errorf("dados no tiene modalidades y trae %d", len(dados))
	}
}

func TestVerifyUnificadoReproduceLasRondasCerradasDeMesa(t *testing.T) {
	env := nuevoHistorialTestEnv(t)

	// Dominó: se juega la primera ficha valida hasta que la mano cierra.
	id, st := env.mesaStart(t, "domino", 1000)
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
		w := env.mesaMove(t, "domino", id, map[string]any{"ficha": ficha})
		if w.Code != http.StatusOK {
			t.Fatalf("domino move respondio %d: %s", w.Code, w.Body.String())
		}
		st = decodearMesa(t, w)
	}
	env.verificarCerrada(t, id, FuenteDomino, models.CasinoDomino)

	// Ajedrez: una jugada cierra la ronda.
	id, st = env.mesaStart(t, "ajedrez", 1000)
	w := env.mesaMove(t, "ajedrez", id, map[string]any{"opcion": 0})
	if w.Code != http.StatusOK {
		t.Fatalf("ajedrez move respondio %d: %s", w.Code, w.Body.String())
	}
	if st = decodearMesa(t, w); st.Estado != "pagada" {
		t.Fatalf("ajedrez: estado %q, esperaba pagada", st.Estado)
	}
	env.verificarCerrada(t, id, FuenteAjedrez, models.CasinoAjedrez)

	// Backgammon: se avanza la ficha mas atrasada hasta el fin de la carrera.
	id, st = env.mesaStart(t, "backgammon", 1000)
	for st.Estado == "abierta" {
		elegido := 0
		for i := 1; i < 3; i++ {
			if st.Jugador[i] < st.Jugador[elegido] {
				elegido = i
			}
		}
		w := env.mesaMove(t, "backgammon", id, map[string]any{"ficha": elegido})
		if w.Code != http.StatusOK {
			t.Fatalf("backgammon move respondio %d: %s", w.Code, w.Body.String())
		}
		st = decodearMesa(t, w)
	}
	env.verificarCerrada(t, id, FuenteBackgammon, models.CasinoBackgammon)

	// Tetris: se colocan las piezas en la primer columna valida hasta cerrar.
	id, st = env.mesaStart(t, "tetris", 1000)
	for st.Estado == "abierta" {
		colocada := false
		for col := 0; col < 6; col++ {
			w := env.mesaMove(t, "tetris", id, map[string]any{"columna": col, "rotacion": 0})
			if w.Code == http.StatusOK {
				st = decodearMesa(t, w)
				colocada = true
				break
			}
		}
		if !colocada {
			t.Fatalf("tetris: no se pudo colocar la pieza: %s", w.Body.String())
		}
	}
	env.verificarCerrada(t, id, FuenteTetris, models.CasinoTetris)
}
