package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Juegos del mini casino. El resultado y el pago se calculan siempre en el
// servidor; el cliente solo anima lo que este le devuelve.
const (
	CasinoDados          = "dados"
	CasinoSlots          = "slots"
	CasinoRuleta         = "ruleta"
	CasinoRaspa          = "raspa"
	CasinoKeno           = "keno"
	CasinoBlackjack      = "blackjack"
	CasinoCascada        = "cascada"
	CasinoHilo           = "hilo"
	CasinoAlto           = "alto"
	CasinoFortuna        = "fortuna"
	CasinoMoneda         = "moneda"
	CasinoSuertudo       = "suertudo"
	CasinoParley         = "parley"
	CasinoBaccarat       = "baccarat"
	CasinoBingo          = "bingo"
	CasinoBingo90        = "bingo90"
	CasinoBingoEsquinas  = "bingo-esquinas"
	CasinoBingoDiamante  = "bingo-diamante"
	CasinoBingoRelampago = "bingo-relampago"
	CasinoGemas          = "gemas"
	CasinoDomino         = "domino"
	CasinoAjedrez        = "ajedrez"
	CasinoBackgammon     = "backgammon"
	CasinoTetris         = "tetris"
	CasinoPinballNeon    = "pinball-neon"
	CasinoPinballClasico = "pinball-clasico"
	CasinoPlinkoPremium  = "plinko-premium"
	CasinoMinaDeOro      = "mina-de-oro"
)

// CasinoGame es un juego del mini casino con sus limites y su RTP declarado.
type CasinoGame struct {
	Codigo       string  `gorm:"primaryKey;size:32" json:"codigo"`
	Nombre       string  `gorm:"not null" json:"nombre"`
	Descripcion  string  `json:"descripcion"`
	MinApuesta   int64   `gorm:"not null;default:200" json:"min_apuesta"`
	MaxApuesta   int64   `gorm:"not null;default:50000" json:"max_apuesta"`
	RTP          float64 `gorm:"not null;default:96" json:"rtp"`
	RTPCasual    float64 `gorm:"not null;default:0" json:"rtp_casual"` // RTP jugando al azar; 0 = no aplica (juego de azar puro)
	MaxMultipl   int64   `gorm:"not null;default:10" json:"max_multiplicador"`
	MaxPremio    int64   `gorm:"not null;default:0" json:"max_premio"`         // techo absoluto por jugada; 0 = sin techo
	LimiteDiario int64   `gorm:"not null;default:200000" json:"limite_diario"` // 0 = sin limite
	Orden        int     `gorm:"not null;default:0" json:"orden"`
	Activo       bool    `gorm:"not null;default:true" json:"activo"`
}

// CasinoRound es una jugada de casino. ServerSeedHash compromete el resultado
// antes de jugar; el server seed se revela al verificar la ronda.
type CasinoRound struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	GameCodigo     string    `gorm:"not null;index;size:32" json:"game"`
	Monto          int64     `gorm:"not null" json:"monto"`
	Multiplicador  int64     `gorm:"not null;default:0" json:"multiplicador"`
	Premio         int64     `gorm:"not null;default:0" json:"premio"`
	ClientSeed     string    `gorm:"size:64" json:"client_seed"`
	ServerSeedHash string    `gorm:"size:64" json:"server_seed_hash"`
	Nonce          int64     `json:"nonce"`
	ResultadoRaw   string    `gorm:"column:resultado_raw" json:"-"`
	Resultado      any       `gorm:"-" json:"resultado"`
	CreatedAt      time.Time `json:"created_at"`
}

func (CasinoGame) TableName() string  { return "casino_juegos" }
func (CasinoRound) TableName() string { return "casino_rondas" }

// CasinoBlackjack es el estado de una mano de blackjack. A diferencia de los
// demas juegos, el blackjack necesita varios turnos (pedir, plantarse, doblar),
// por eso la mano vive en su propia tabla hasta que se resuelve.
type BlackjackMano struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	UserID        uint    `gorm:"not null;index" json:"user_id"`
	Monto         int64   `gorm:"not null" json:"monto"`
	ManoRaw       string  `gorm:"column:mano_raw" json:"-"`
	Mano          []int   `gorm:"-" json:"mano"`
	CrupierRaw    string  `gorm:"column:crupier_raw" json:"-"`
	Crupier       []int   `gorm:"-" json:"crupier"`
	Estado        string  `gorm:"not null;default:abierta" json:"estado"`
	Multiplicador float64 `gorm:"not null;default:0" json:"multiplicador"`
	Premio        int64   `gorm:"not null;default:0" json:"premio"`
	Seguro        int64   `gorm:"not null;default:0" json:"seguro"` // costo del seguro cobrado; 0 = sin seguro
	// GrupoID/Orden: las manos divididas (split) son filas hermanas. Orden 1 y 2;
	// el crupier es comun y el premio del grupo es la suma de las dos.
	GrupoID        uint      `gorm:"index" json:"grupo_id"`
	Orden          int       `gorm:"not null;default:1" json:"orden"`
	ClientSeed     string    `gorm:"size:64" json:"client_seed"`
	ServerSeedHash string    `gorm:"size:64" json:"server_seed_hash"`
	Nonce          int64     `json:"nonce"`
	CreatedAt      time.Time `json:"created_at"`
}

func (BlackjackMano) TableName() string { return "casino_blackjack" }

// PokerMano es una mano de video poker. La apuesta se descuenta al repartir y
// la mano queda abierta hasta que el jugador elige que cartas sostiene y cobra;
// si la abandona, el servidor la cierra solo con el reparto automatico.
type PokerMano struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	RondaID        uint      `gorm:"not null;index" json:"ronda_id"` // fila de casino_rondas
	Juego          string    `gorm:"not null;index;size:32" json:"juego"`
	Monto          int64     `gorm:"not null" json:"monto"`
	InicialesRaw   string    `gorm:"column:iniciales_raw" json:"-"`
	Iniciales      []string  `gorm:"-" json:"iniciales"`
	Estado         string    `gorm:"not null;default:abierta" json:"estado"` // abierta | pagada | caducada
	ClientSeed     string    `gorm:"size:64" json:"client_seed"`
	ServerSeedHash string    `gorm:"size:64" json:"server_seed_hash"`
	Nonce          int64     `json:"nonce"`
	CreatedAt      time.Time `json:"created_at"`
}

func (PokerMano) TableName() string { return "casino_poker_manos" }

func (p *PokerMano) BeforeSave(tx *gorm.DB) error {
	b, err := json.Marshal(p.Iniciales)
	if err != nil {
		return err
	}
	p.InicialesRaw = string(b)
	return nil
}

func (p *PokerMano) AfterFind(tx *gorm.DB) error {
	if p.InicialesRaw == "" {
		return nil
	}
	return json.Unmarshal([]byte(p.InicialesRaw), &p.Iniciales)
}

func (b *BlackjackMano) BeforeSave(tx *gorm.DB) error {
	mb, err := json.Marshal(b.Mano)
	if err != nil {
		return err
	}
	cb, err := json.Marshal(b.Crupier)
	if err != nil {
		return err
	}
	b.ManoRaw = string(mb)
	b.CrupierRaw = string(cb)
	return nil
}

func (b *BlackjackMano) AfterFind(tx *gorm.DB) error {
	if err := json.Unmarshal([]byte(b.ManoRaw), &b.Mano); err != nil {
		return err
	}
	return json.Unmarshal([]byte(b.CrupierRaw), &b.Crupier)
}

func (r *CasinoRound) BeforeSave(tx *gorm.DB) error {
	b, err := json.Marshal(r.Resultado)
	if err != nil {
		return err
	}
	r.ResultadoRaw = string(b)
	return nil
}

func (r *CasinoRound) AfterFind(tx *gorm.DB) error {
	if r.ResultadoRaw == "" {
		return nil
	}
	return json.Unmarshal([]byte(r.ResultadoRaw), &r.Resultado)
}

// BingoRonda es una jugada de bingo. La apuesta se descuenta al abrir y la
// ronda queda abierta mientras se cantan bolillas de a una; el premio se calcula
// segun la variante (patrones y multiplicadores propios). El orden de las
// bolillas y los cartones se derivan de las semillas (provably fair).
type BingoRonda struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	Variante       string    `gorm:"size:32;not null;default:bingo" json:"variante"`
	Monto          int64     `gorm:"not null" json:"monto"` // apuesta total (por cartón x cartones)
	MontoPorCarton int64     `gorm:"not null" json:"monto_por_carton"`
	Cartones       int       `gorm:"not null;default:1" json:"cartones"`
	CartonesRaw    string    `gorm:"column:cartones_raw;type:text" json:"-"`
	Grids          [][][]int `gorm:"-" json:"grids"` // cartón -> fila -> columna
	OrdenRaw       string    `gorm:"column:orden_raw;type:text" json:"-"`
	Orden          []int     `gorm:"-" json:"orden"` // 75 bolillas mezcladas
	CantadasRaw    string    `gorm:"column:cantadas_raw;type:text" json:"-"`
	Cantadas       []int     `gorm:"-" json:"cantadas"` // bolillas ya cantadas
	Estado         string    `gorm:"not null;default:abierta" json:"estado"`
	Multiplicador  float64   `gorm:"not null;default:0" json:"multiplicador"`
	Premio         int64     `gorm:"not null;default:0" json:"premio"`
	Lineas         int       `gorm:"not null;default:0" json:"lineas"` // mayor nº de líneas en un cartón
	ClientSeed     string    `gorm:"size:64" json:"client_seed"`
	ServerSeedHash string    `gorm:"size:64" json:"server_seed_hash"`
	Nonce          int64     `json:"nonce"`
	// IdempotencyKey identifica la jugada desde el cliente: reenviar el mismo
	// start con la misma clave devuelve la ronda ya creada en vez de cobrar
	// dos veces si la respuesta se perdio (sin datos, 502, timeout).
	IdempotencyKey string    `gorm:"size:64;uniqueIndex:uq_bingo_idem,where:idempotency_key <> ''" json:"-"`
	RequestID      string    `gorm:"size:40;index" json:"request_id"`
	CreatedAt      time.Time `json:"created_at"`
}

func (BingoRonda) TableName() string { return "bingo_rondas" }

func (b *BingoRonda) BeforeSave(tx *gorm.DB) error {
	cb, err := json.Marshal(b.Grids)
	if err != nil {
		return err
	}
	ob, err := json.Marshal(b.Orden)
	if err != nil {
		return err
	}
	if b.Cantadas == nil {
		b.Cantadas = []int{}
	}
	db, err := json.Marshal(b.Cantadas)
	if err != nil {
		return err
	}
	b.CartonesRaw = string(cb)
	b.OrdenRaw = string(ob)
	b.CantadasRaw = string(db)
	return nil
}

func (b *BingoRonda) AfterFind(tx *gorm.DB) error {
	if b.CartonesRaw != "" {
		if err := json.Unmarshal([]byte(b.CartonesRaw), &b.Grids); err != nil {
			return err
		}
	}
	if b.OrdenRaw != "" {
		if err := json.Unmarshal([]byte(b.OrdenRaw), &b.Orden); err != nil {
			return err
		}
	}
	if b.CantadasRaw != "" {
		if err := json.Unmarshal([]byte(b.CantadasRaw), &b.Cantadas); err != nil {
			return err
		}
	}
	return nil
}

// GemasRonda es una jugada de Gemas (match-3). La apuesta se descuenta al
// abrir la ronda y el jugador tiene una cantidad limitada de intercambios; el
// tablero inicial y el chorro de relleno se derivan de las semillas (provably
// fair), y cada intercambio se guarda para poder auditar la jugada compelta.
type GemasRonda struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	Monto          int64     `gorm:"not null" json:"monto"`
	TableroRaw     string    `gorm:"column:tablero_raw;type:text" json:"-"`
	Tablero        []int     `gorm:"-" json:"tablero"` // 64 celdas: 0..6 base, 7 gema rayo
	StreamOffset   int       `gorm:"not null;default:0" json:"stream_offset"`
	Cursor         int       `gorm:"not null;default:0" json:"cursor"` // posicion del chorro de relleno
	JugadasRaw     string    `gorm:"column:jugadas_raw;type:text" json:"-"`
	Jugadas        [][2]int  `gorm:"-" json:"jugadas"` // pares de celdas intercambiadas
	Movimientos    int       `gorm:"not null;default:0" json:"movimientos"`
	Puntos         int64     `gorm:"not null;default:0" json:"puntos"`
	Estado         string    `gorm:"not null;default:abierta" json:"estado"` // abierta | pagada
	Multiplicador  float64   `gorm:"not null;default:0" json:"multiplicador"`
	Premio         int64     `gorm:"not null;default:0" json:"premio"`
	ClientSeed     string    `gorm:"size:64" json:"client_seed"`
	ServerSeedHash string    `gorm:"size:64" json:"server_seed_hash"`
	Nonce          int64     `json:"nonce"`
	CreatedAt      time.Time `json:"created_at"`
}

func (GemasRonda) TableName() string { return "gemas_rondas" }

func (g *GemasRonda) BeforeSave(tx *gorm.DB) error {
	tb, err := json.Marshal(g.Tablero)
	if err != nil {
		return err
	}
	if g.Jugadas == nil {
		g.Jugadas = [][2]int{}
	}
	jb, err := json.Marshal(g.Jugadas)
	if err != nil {
		return err
	}
	g.TableroRaw = string(tb)
	g.JugadasRaw = string(jb)
	return nil
}

func (g *GemasRonda) AfterFind(tx *gorm.DB) error {
	if g.TableroRaw != "" {
		if err := json.Unmarshal([]byte(g.TableroRaw), &g.Tablero); err != nil {
			return err
		}
	}
	if g.JugadasRaw != "" {
		if err := json.Unmarshal([]byte(g.JugadasRaw), &g.Jugadas); err != nil {
			return err
		}
	}
	return nil
}

// DominoRonda es una mano de domino contra la casa. La apuesta se descuenta
// al abrir y la mano queda abierta mientras el jugador y la casa juegan fichas.
// El reparto sale de las semillas (provably fair) y la casa responde siempre
// con la ficha valida de mayor puntaje, asi que la mano se puede reproducir.
type DominoRonda struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	Monto          int64     `gorm:"not null" json:"monto"`
	ManoRaw        string    `gorm:"column:mano_raw;type:text" json:"-"`
	Mano           [][2]int  `gorm:"-" json:"mano"`
	CasaRaw        string    `gorm:"column:casa_raw;type:text" json:"-"`
	Casa           [][2]int  `gorm:"-" json:"casa"`
	TableroRaw     string    `gorm:"column:tablero_raw;type:text" json:"-"`
	Tablero        [][2]int  `gorm:"-" json:"tablero"`
	Turno          string    `gorm:"size:16" json:"turno"`
	JugadasRaw     string    `gorm:"column:jugadas_raw;type:text" json:"-"`
	Jugadas        [][2]int  `gorm:"-" json:"jugadas"` // [ficha, lado] por jugada
	Estado         string    `gorm:"not null;default:abierta" json:"estado"`
	Multiplicador  float64   `gorm:"not null;default:0" json:"multiplicador"`
	Premio         int64     `gorm:"not null;default:0" json:"premio"`
	ClientSeed     string    `gorm:"size:64" json:"client_seed"`
	ServerSeedHash string    `gorm:"size:64" json:"server_seed_hash"`
	Nonce          int64     `json:"nonce"`
	CreatedAt      time.Time `json:"created_at"`
}

func (DominoRonda) TableName() string { return "domino_rondas" }

func (d *DominoRonda) BeforeSave(tx *gorm.DB) error {
	mb, _ := json.Marshal(d.Mano)
	cb, _ := json.Marshal(d.Casa)
	tb, _ := json.Marshal(d.Tablero)
	jb, _ := json.Marshal(d.Jugadas)
	d.ManoRaw = string(mb)
	d.CasaRaw = string(cb)
	d.TableroRaw = string(tb)
	d.JugadasRaw = string(jb)
	return nil
}

func (d *DominoRonda) AfterFind(tx *gorm.DB) error {
	if d.ManoRaw != "" {
		json.Unmarshal([]byte(d.ManoRaw), &d.Mano)
	}
	if d.CasaRaw != "" {
		json.Unmarshal([]byte(d.CasaRaw), &d.Casa)
	}
	if d.TableroRaw != "" {
		json.Unmarshal([]byte(d.TableroRaw), &d.Tablero)
	}
	if d.JugadasRaw != "" {
		json.Unmarshal([]byte(d.JugadasRaw), &d.Jugadas)
	}
	return nil
}

// AjedrezRonda es un puzzle de mate en una. Se apuesta al abrir la posicion y
// se liquida al elegir una de las cuatro jugadas propuestas.
type AjedrezRonda struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	Monto          int64     `gorm:"not null" json:"monto"`
	FEN            string    `gorm:"size:128" json:"fen"`
	Correcta       string    `gorm:"size:8" json:"correcta"`
	OpcionesRaw    string    `gorm:"column:opciones_raw;type:text" json:"-"`
	Opciones       []string  `gorm:"-" json:"opciones"`
	JugadasRaw     string    `gorm:"column:jugadas_raw;type:text" json:"-"`
	Jugadas        []string  `gorm:"-" json:"jugadas"`
	Tema           string    `gorm:"size:32" json:"tema"`
	Estado         string    `gorm:"not null;default:abierta" json:"estado"`
	Multiplicador  float64   `gorm:"not null;default:0" json:"multiplicador"`
	Premio         int64     `gorm:"not null;default:0" json:"premio"`
	ClientSeed     string    `gorm:"size:64" json:"client_seed"`
	ServerSeedHash string    `gorm:"size:64" json:"server_seed_hash"`
	Nonce          int64     `json:"nonce"`
	CreatedAt      time.Time `json:"created_at"`
}

func (AjedrezRonda) TableName() string { return "ajedrez_rondas" }

func (a *AjedrezRonda) BeforeSave(tx *gorm.DB) error {
	ob, _ := json.Marshal(a.Opciones)
	jb, _ := json.Marshal(a.Jugadas)
	a.OpcionesRaw = string(ob)
	a.JugadasRaw = string(jb)
	return nil
}

func (a *AjedrezRonda) AfterFind(tx *gorm.DB) error {
	if a.OpcionesRaw != "" {
		json.Unmarshal([]byte(a.OpcionesRaw), &a.Opciones)
	}
	if a.JugadasRaw != "" {
		json.Unmarshal([]byte(a.JugadasRaw), &a.Jugadas)
	}
	return nil
}

// AjedrezPartida es una partida completa de ajedrez contra el adversario de la
// casa. La apuesta se descuenta al abrir y la partida queda abierta hasta el
// mate o las tablas; la posicion (FEN) y la lista de jugadas guardadas permiten
// reconstruirla y reproducir la jugada de la casa desde las semillas.
type AjedrezPartida struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	Monto          int64     `gorm:"not null" json:"monto"`
	ColorJugador   string    `gorm:"size:16;not null;default:blancas" json:"color"`
	FEN            string    `gorm:"size:128;not null" json:"fen"`
	JugadasRaw     string    `gorm:"column:jugadas_raw;type:text" json:"-"`
	Jugadas        []string  `gorm:"-" json:"jugadas"`
	Turno          string    `gorm:"size:8" json:"turno"`
	Resultado      string    `gorm:"size:16" json:"resultado"`
	Motivo         string    `gorm:"size:64" json:"motivo"`
	Estado         string    `gorm:"not null;default:abierta" json:"estado"`
	Multiplicador  float64   `gorm:"not null;default:0" json:"multiplicador"`
	Premio         int64     `gorm:"not null;default:0" json:"premio"`
	ClientSeed     string    `gorm:"size:64" json:"client_seed"`
	ServerSeedHash string    `gorm:"size:64" json:"server_seed_hash"`
	Nonce          int64     `json:"nonce"`
	CreatedAt      time.Time `json:"created_at"`
}

func (AjedrezPartida) TableName() string { return "ajedrez_partidas" }

func (p *AjedrezPartida) BeforeSave(tx *gorm.DB) error {
	if p.Jugadas == nil {
		p.Jugadas = []string{}
	}
	b, err := json.Marshal(p.Jugadas)
	if err != nil {
		return err
	}
	p.JugadasRaw = string(b)
	return nil
}

func (p *AjedrezPartida) AfterFind(tx *gorm.DB) error {
	if p.JugadasRaw == "" {
		return nil
	}
	return json.Unmarshal([]byte(p.JugadasRaw), &p.Jugadas)
}

// BackgammonRonda es una carrera simplificada de tres fichas por lado. La
// tirada de dados sale de las semillas (provably fair) y la casa mueve siempre
// la ficha mas atrasada.
type BackgammonRonda struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	Monto          int64     `gorm:"not null" json:"monto"`
	Jugador        [3]int    `gorm:"-" json:"jugador"`
	Casa           [3]int    `gorm:"-" json:"casa"`
	JugadorRaw     string    `gorm:"column:jugador_raw;type:text" json:"-"`
	CasaRaw        string    `gorm:"column:casa_raw;type:text" json:"-"`
	Turno          string    `gorm:"size:16" json:"turno"`
	Dado1          int       `json:"dado1"`
	Dado2          int       `json:"dado2"`
	Dobles         bool      `json:"dobles"`
	Lance          int       `json:"lance"`
	Fin            bool      `json:"fin"`
	Ganador        string    `gorm:"size:16" json:"ganador"`
	JugadasRaw     string    `gorm:"column:jugadas_raw;type:text" json:"-"`
	Jugadas        []int     `gorm:"-" json:"jugadas"` // ficha elegida por turno
	Estado         string    `gorm:"not null;default:abierta" json:"estado"`
	Multiplicador  float64   `gorm:"not null;default:0" json:"multiplicador"`
	Premio         int64     `gorm:"not null;default:0" json:"premio"`
	ClientSeed     string    `gorm:"size:64" json:"client_seed"`
	ServerSeedHash string    `gorm:"size:64" json:"server_seed_hash"`
	Nonce          int64     `json:"nonce"`
	CreatedAt      time.Time `json:"created_at"`
}

func (BackgammonRonda) TableName() string { return "backgammon_rondas" }

func (b *BackgammonRonda) BeforeSave(tx *gorm.DB) error {
	jb, _ := json.Marshal(b.Jugador)
	cb, _ := json.Marshal(b.Casa)
	pb, _ := json.Marshal(b.Jugadas)
	b.JugadorRaw = string(jb)
	b.CasaRaw = string(cb)
	b.JugadasRaw = string(pb)
	return nil
}

func (b *BackgammonRonda) AfterFind(tx *gorm.DB) error {
	if b.JugadorRaw != "" {
		json.Unmarshal([]byte(b.JugadorRaw), &b.Jugador)
	}
	if b.CasaRaw != "" {
		json.Unmarshal([]byte(b.CasaRaw), &b.Casa)
	}
	if b.JugadasRaw != "" {
		json.Unmarshal([]byte(b.JugadasRaw), &b.Jugadas)
	}
	return nil
}

// TetrisRonda es una ronda de Tetris por lineas. La secuencia de piezas sale de
// las semillas (provably fair) y el jugador elige columna y rotacion de cada
// pieza; el pago depende de las lineas acumuladas.
type TetrisRonda struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	Monto          int64     `gorm:"not null" json:"monto"`
	TableroRaw     string    `gorm:"column:tablero_raw;type:text" json:"-"`
	Tablero        []int     `gorm:"-" json:"tablero"` // 72 celdas
	PiezasRaw      string    `gorm:"column:piezas_raw;type:text" json:"-"`
	Piezas         []int     `gorm:"-" json:"piezas"`
	Pieza          int       `json:"pieza"`
	Lineas         int       `json:"lineas"`
	JugadasRaw     string    `gorm:"column:jugadas_raw;type:text" json:"-"`
	Jugadas        [][2]int  `gorm:"-" json:"jugadas"` // [columna, rotacion]
	Estado         string    `gorm:"not null;default:abierta" json:"estado"`
	Multiplicador  float64   `gorm:"not null;default:0" json:"multiplicador"`
	Premio         int64     `gorm:"not null;default:0" json:"premio"`
	ClientSeed     string    `gorm:"size:64" json:"client_seed"`
	ServerSeedHash string    `gorm:"size:64" json:"server_seed_hash"`
	Nonce          int64     `json:"nonce"`
	CreatedAt      time.Time `json:"created_at"`
}

func (TetrisRonda) TableName() string { return "tetris_rondas" }

func (t *TetrisRonda) BeforeSave(tx *gorm.DB) error {
	tb, _ := json.Marshal(t.Tablero)
	pb, _ := json.Marshal(t.Piezas)
	jb, _ := json.Marshal(t.Jugadas)
	t.TableroRaw = string(tb)
	t.PiezasRaw = string(pb)
	t.JugadasRaw = string(jb)
	return nil
}

func (t *TetrisRonda) AfterFind(tx *gorm.DB) error {
	if t.TableroRaw != "" {
		json.Unmarshal([]byte(t.TableroRaw), &t.Tablero)
	}
	if t.PiezasRaw != "" {
		json.Unmarshal([]byte(t.PiezasRaw), &t.Piezas)
	}
	if t.JugadasRaw != "" {
		json.Unmarshal([]byte(t.JugadasRaw), &t.Jugadas)
	}
	return nil
}
