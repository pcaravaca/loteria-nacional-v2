package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Juegos del mini casino. El resultado y el pago se calculan siempre en el
// servidor; el cliente solo anima lo que este le devuelve.
const (
	CasinoDados     = "dados"
	CasinoSlots     = "slots"
	CasinoRuleta    = "ruleta"
	CasinoRaspa     = "raspa"
	CasinoKeno      = "keno"
	CasinoBlackjack = "blackjack"
)

// CasinoGame es un juego del mini casino con sus limites y su RTP declarado.
type CasinoGame struct {
	Codigo      string  `gorm:"primaryKey;size:32" json:"codigo"`
	Nombre      string  `gorm:"not null" json:"nombre"`
	Descripcion string  `json:"descripcion"`
	MinApuesta  int64   `gorm:"not null;default:200" json:"min_apuesta"`
	MaxApuesta  int64   `gorm:"not null;default:50000" json:"max_apuesta"`
	RTP         float64 `gorm:"not null;default:96" json:"rtp"`
	MaxMultipl  int64   `gorm:"not null;default:10" json:"max_multiplicador"`
	Orden       int     `gorm:"not null;default:0" json:"orden"`
	Activo      bool    `gorm:"not null;default:true" json:"activo"`
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
	ClientSeed     string    `gorm:"size:64" json:"client_seed"`
	ServerSeedHash string    `gorm:"size:64" json:"server_seed_hash"`
	Nonce          int64     `json:"nonce"`
	CreatedAt      time.Time `json:"created_at"`
}

func (BlackjackMano) TableName() string { return "casino_blackjack" }

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
