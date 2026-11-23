package models

import "time"

// AlertaCS es un caso que el servicio al cliente debería mirar: una apuesta
// que quedó retenida, un reintento que delata un corte de conexión o un pago
// que se resolvió dos veces. Las genera un worker, no el jugador.
type AlertaCS struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Tipo      string    `gorm:"size:40;not null;index" json:"tipo"`
	Severidad string    `gorm:"size:16;not null;default:media" json:"severidad"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Entidad   string    `gorm:"size:40" json:"entidad"`
	EntidadID string    `gorm:"size:40" json:"entidad_id"`
	Detalle   string    `gorm:"type:text" json:"detalle"`
	Revisada  bool      `gorm:"not null;default:false;index" json:"revisada"`
	CreatedAt time.Time `json:"created_at"`
}

func (AlertaCS) TableName() string { return "alertas_cs" }

// Tipos de alerta conocidos.
const (
	AlertaBingoRetenida  = "bingo_retenida" // ronda abierta demasiado tiempo
	AlertaReintento      = "reintento_red"  // el jugador reintentó tras un corte
	AlertaDobleCobro     = "doble_cobro"    // dos cobros del mismo request
	AlertaRondaHuertfana = "ronda_huerfana"   // start cobró pero el jugador nunca siguió
	AlertaRifaPendiente  = "rifa_pendiente"   // vencio y falta el resultado de la lotería
)
