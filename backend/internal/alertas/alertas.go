package alertas

import (
	"log"
	"time"

	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// Umbrales de detección. Una ronda de bingo abierta 15 minutos sin actividad
// significa que el jugador se fue (corte, app cerrada) y la apuesta quedó
// retenida.
const (
	bingoRetenidaMin     = 15 * time.Minute
	reintentoVentana     = 6 * time.Hour
	dedupVentana         = 1 * time.Hour
	severidadAlta        = "alta"
	severidadMedia       = "media"
	severidadInformativa = "informativa"
)

// Nueva deja una alerta si no hay una igual sin revisar hace menos de una hora:
// un corte de red no debe inundar el panel.
func Nueva(db *gorm.DB, tipo, severidad string, userID uint, entidad, entidadID, detalle string) bool {
	if db == nil {
		return false
	}
	var previas int64
	db.Model(&models.AlertaCS{}).
		Where("tipo = ? AND entidad_id = ? AND revisada = ? AND created_at > ?",
			tipo, entidadID, false, time.Now().Add(-dedupVentana)).
		Count(&previas)
	if previas > 0 {
		return false
	}
	if err := db.Create(&models.AlertaCS{
		Tipo: tipo, Severidad: severidad, UserID: userID,
		Entidad: entidad, EntidadID: entidadID, Detalle: detalle,
	}).Error; err != nil {
		return false
	}
	return true
}

// Detectar recorre las fuentes de verdad (rondas y auditoria) y deja las
// alertas. Devuelve cuantas creó.
func Detectar(db *gorm.DB) int {
	if db == nil {
		return 0
	}
	creadas := 0

	// 1) Apuesta retenida: ronda de bingo abierta demasiado tiempo.
	var retenidas []models.BingoRonda
	db.Where("estado = ? AND created_at < ?", "abierta", time.Now().Add(-bingoRetenidaMin)).
		Limit(50).Find(&retenidas)
	for _, r := range retenidas {
		if Nueva(db, models.AlertaBingoRetenida, severidadAlta, r.UserID, "bingo_ronda",
			itostr(uint64(r.ID)),
			"ronda abierta desde las "+r.CreatedAt.Format("15:04")+" con "+itostr(uint64(r.Monto))+
				" retenidos y "+itostr(uint64(len(r.Cantadas)))+" bolillas cantadas") {
			creadas++
		}
	}

	// 2) Reintentos: delatan un corte de conexion. Se avisa, no se cobra dos veces.
	var reintentos []models.AuditLog
	db.Where("accion IN ? AND created_at > ?",
		[]string{"bingo_hit_sincroniza", "bingo_close_repetida", "bingo_hit_cerrada", "bingo_start_repetida"},
		time.Now().Add(-reintentoVentana)).Limit(100).Find(&reintentos)
	for _, a := range reintentos {
		if Nueva(db, models.AlertaReintento, severidadMedia, a.ActorID, "auditoria",
			a.RequestID,
			"el jugador reintentó "+a.Accion+" ("+a.Entidad+" "+a.EntidadID+") desde "+a.IP) {
			creadas++
		}
	}

	// 3) Doble cobro: dos cierres con el mismo request id.
	var dobles []models.AuditLog
	db.Where("accion IN ? AND request_id <> '' AND created_at > ?",
		[]string{"bingo_close", "bingo_close_repetida", "casino_play"},
		time.Now().Add(-reintentoVentana)).
		Order("request_id, created_at").Limit(500).Find(&dobles)
	vistos := map[string]bool{}
	for _, a := range dobles {
		if vistos[a.RequestID] {
			if Nueva(db, models.AlertaDobleCobro, severidadAlta, a.ActorID, "auditoria",
				a.RequestID, "dos cobros con el mismo request_id ("+a.RequestID+")") {
				creadas++
			}
			continue
		}
		vistos[a.RequestID] = true
	}

	return creadas
}

// Trabajador corre la deteccion cada tanto mientras la API viva.
func Trabajador(db *gorm.DB) {
	const cada = 2 * time.Minute
	for {
		time.Sleep(cada)
		n := Detectar(db)
		if n > 0 {
			log.Printf("alertas CS: %d nueva(s)", n)
		}
	}
}

func itostr(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
