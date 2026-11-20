package resultados

import (
	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// uniqCount cuenta valores distintos.
func uniqCount(ns []int) int {
	vistos := map[int]bool{}
	n := 0
	for _, x := range ns {
		if !vistos[x] {
			vistos[x] = true
			n++
		}
	}
	return n
}

func intersect(a, b []int) int {
	set := make(map[int]bool, len(b))
	for _, x := range b {
		set[x] = true
	}
	n := 0
	for _, x := range a {
		if set[x] {
			n++
		}
	}
	return n
}

// GrabarSorteo fija los números ganadores de un sorteo, evalúa las apuestas
// pendientes y paga los premios al instante. Es la misma lógica que usa el
// cierre manual del admin; el importador automático la reutiliza para no tener
// dos caminos de pago distintos.
//
// Un grado inválido (repetidos o fuera del rango de la lotería) se ignora en
// lugar de fallar: una fuente externa no debe poder romper el sorteo.
func GrabarSorteo(db *gorm.DB, sorteo *models.Sorteo, numeros []int) error {
	if len(numeros) == 0 {
		return nil
	}
	if len(numeros) != uniqCount(numeros) {
		return nil
	}
	maxNum := 1
	for i := 0; i < sorteo.Loteria.LongitudNum; i++ {
		maxNum *= 10
	}
	for _, n := range numeros {
		if n < 0 || n >= maxNum {
			return nil
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		sorteo.NumerosGanadores = numeros
		sorteo.Estado = models.SorteoPagado
		if err := tx.Save(sorteo).Error; err != nil {
			return err
		}

		var apuestas []models.Apuesta
		if err := tx.Where("sorteo_id = ? AND estado = ?", sorteo.ID, models.ApuestaPendiente).
			Find(&apuestas).Error; err != nil {
			return err
		}
		for _, a := range apuestas {
			aciertos := intersect(a.Numeros, numeros)
			premio := sorteo.Loteria.Premio(a.Monto, aciertos, len(a.Numeros))
			estado := models.ApuestaPerdedora
			switch {
			case aciertos >= len(a.Numeros) && aciertos > 0:
				estado = models.ApuestaPagada
			case premio > 0:
				estado = models.ApuestaGanadora
			}
			if err := tx.Model(&models.Apuesta{}).Where("id = ?", a.ID).Updates(map[string]any{
				"estado": estado, "aciertos": aciertos, "premio": premio,
			}).Error; err != nil {
				return err
			}
			if premio > 0 {
				if err := tx.Model(&models.User{}).Where("id = ?", a.UserID).
					Update("saldo", gorm.Expr("saldo + ?", premio)).Error; err != nil {
					return err
				}
				if err := tx.Create(&models.Movimiento{
					UserID: a.UserID, Tipo: models.MovPremio, Monto: premio,
					Descripcion: "Premio " + sorteo.Titulo,
				}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
