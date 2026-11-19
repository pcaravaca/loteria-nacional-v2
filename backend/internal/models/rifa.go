package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Estados posibles de una rifa.
const (
	RifaAbierta   = "abierta"
	RifaSorteada  = "sorteada"
	RifaCancelada = "cancelada"
)

// Tipos de sorteo: azar (entre los vendidos) o manual (el operador ingresa el numero).
const (
	RifaSorteoAzar   = "azar"
	RifaSorteoManual = "manual"
)

// Rifa es un sorteo de numeros con un premio definido por el operador.
type Rifa struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Titulo             string    `gorm:"not null" json:"titulo"`
	Descripcion        string    `json:"descripcion"`
	Premio             string    `gorm:"not null" json:"premio"`
	PremioMonto        int64     `gorm:"not null;default:0" json:"premio_monto"` // si >0 se acredita al ganador
	PrecioNumero       int64     `gorm:"not null" json:"precio_numero"`
	DesdeNumero        int       `gorm:"not null;default:0" json:"desde_numero"`
	HastaNumero        int       `gorm:"not null;default:99" json:"hasta_numero"`
	FechaSorteo        time.Time `json:"fecha_sorteo"`
	Estado             string    `gorm:"not null;default:abierta" json:"estado"`
	TipoSorteo         string    `gorm:"not null;default:azar" json:"tipo_sorteo"`
	CreadoPor          uint      `json:"creado_por"`
	NumerosGanadoresRaw string    `gorm:"column:numeros_ganadores_raw" json:"-"`
	NumerosGanadores    []int     `gorm:"-" json:"numeros_ganadores"`
	// Sorteo provably fair: la semilla se compromete con su hash y se revela al cerrar.
	SemillaHash         string    `gorm:"column:semilla_hash" json:"semilla_hash"`
	Semilla             string    `gorm:"column:semilla" json:"-"`
}

func (Rifa) TableName() string { return "rifas" }

func (r *Rifa) BeforeSave(tx *gorm.DB) error {
	b, err := json.Marshal(r.NumerosGanadores)
	if err != nil {
		return err
	}
	r.NumerosGanadoresRaw = string(b)
	return nil
}

func (r *Rifa) AfterFind(tx *gorm.DB) error {
	if r.NumerosGanadoresRaw == "" {
		return nil
	}
	return json.Unmarshal([]byte(r.NumerosGanadoresRaw), &r.NumerosGanadores)
}

// RifaNumero es un numero dentro de una rifa. UserID nulo significa disponible.
type RifaNumero struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	RifaID     uint       `gorm:"not null;index;uniqueIndex:rifa_numero" json:"rifa_id"`
	Numero     int        `gorm:"not null;uniqueIndex:rifa_numero" json:"numero"`
	UserID     *uint      `gorm:"index" json:"user_id"`
	Monto      int64      `json:"monto"`
	CompradoEn time.Time  `json:"comprado_en"`
	Rifa       Rifa       `gorm:"foreignKey:RifaID" json:"-"`
}

func (RifaNumero) TableName() string { return "rifa_numeros" }

// GanadorRifa es un ganador con su numero y nombre para reportes publicos.
type GanadorRifa struct {
	Numero int    `json:"numero"`
	Nombre string `json:"nombre"`
}
