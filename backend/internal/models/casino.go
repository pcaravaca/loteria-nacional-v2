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
)

// CasinoGame es un juego del mini casino con sus limites y su RTP declarado.
type CasinoGame struct {
	Codigo       string  `gorm:"primaryKey;size:32" json:"codigo"`
	Nombre       string  `gorm:"not null" json:"nombre"`
	Descripcion  string  `json:"descripcion"`
	MinApuesta   int64   `gorm:"not null;default:200" json:"min_apuesta"`
	MaxApuesta   int64   `gorm:"not null;default:50000" json:"max_apuesta"`
	RTP          float64 `gorm:"not null;default:96" json:"rtp"`
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
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	Monto          int64     `gorm:"not null" json:"monto"`
	ManoRaw        string    `gorm:"column:mano_raw" json:"-"`
	Mano           []int     `gorm:"-" json:"mano"`
	CrupierRaw     string    `gorm:"column:crupier_raw" json:"-"`
	Crupier        []int     `gorm:"-" json:"crupier"`
	Estado         string    `gorm:"not null;default:abierta" json:"estado"`
	Multiplicador  float64   `gorm:"not null;default:0" json:"multiplicador"`
	Premio         int64     `gorm:"not null;default:0" json:"premio"`
	Seguro         int64     `gorm:"not null;default:0" json:"seguro"` // costo del seguro cobrado; 0 = sin seguro
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
