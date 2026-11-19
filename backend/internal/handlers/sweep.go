package handlers

import (
	"log"
	"time"

	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// SweepSorteosVencidos reembolsa las apuestas pendientes de los sorteos que ya
// pasaron su fecha y quedaron abiertos, dejandolos en estado "vencido".
//
// Es el cierre automatico (SORTEOS_CIERRE_AUTO=true): evita que quede dinero
// retenido en un sorteo que nadie cerró. Devuelve (sorteos procesados, apuestas
// reembolsadas). Con habilitado=false no hace nada.
func SweepSorteosVencidos(db *gorm.DB, habilitado bool, gracia time.Duration) (int, int) {
	if !habilitado {
		return 0, 0
	}
	corte := time.Now().Add(-gracia)

	var sorteos []models.Sorteo
	if err := db.Where("estado = ? AND fecha < ?", models.SorteoAbierto, corte).Find(&sorteos).Error; err != nil {
		log.Printf("sweep sorteos: %v", err)
		return 0, 0
	}

	procesados, reembolsadas := 0, 0
	for _, s := range sorteos {
		err := db.Transaction(func(tx *gorm.DB) error {
			var apuestas []models.Apuesta
			if err := tx.Where("sorteo_id = ? AND estado = ?", s.ID, models.ApuestaPendiente).
				Find(&apuestas).Error; err != nil {
				return err
			}
			for _, a := range apuestas {
				if err := tx.Model(&models.User{}).Where("id = ?", a.UserID).
					Update("saldo", gorm.Expr("saldo + ?", a.Monto)).Error; err != nil {
					return err
				}
				if err := tx.Create(&models.Movimiento{
					UserID: a.UserID, Tipo: models.MovSorteoReembolso, Monto: a.Monto,
					Descripcion: "Reembolso sorteo vencido " + s.Titulo,
				}).Error; err != nil {
					return err
				}
				reembolsadas++
			}
			s.Estado = models.SorteoVencido
			return tx.Save(&s).Error
		})
		if err != nil {
			log.Printf("sweep sorteo %d: %v", s.ID, err)
			continue
		}
		procesados++
		log.Printf("sorteo %d vencido: %d apuesta(s) reembolsada(s)", s.ID, reembolsadas)
	}
	return procesados, reembolsadas
}

// IniciarSweepSorteos corre el barrido cada intervalo mientras la API viva.
func IniciarSweepSorteos(db *gorm.DB, habilitado bool, intervalo, gracia time.Duration) {
	if !habilitado || intervalo <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(intervalo)
		defer ticker.Stop()
		for range ticker.C {
			SweepSorteosVencidos(db, true, gracia)
		}
	}()
}
