// Package rifasauto gradúa las rifas que se cuelgan del resultado de la
// Lotería Nacional: al vencer la fecha, busca el sorteo de LN del mismo día y,
// si ya tiene números, reparte el premio. Si el resultado no está cargado, la
// rifa queda "pendiente" y el operador la gradúa cuando cargue los números.
package rifasauto

import (
	"encoding/json"
	"log"
	"time"

	"gorm.io/gorm"

	"loteria-backend/internal/models"
	"loteria-backend/internal/tiempo"
)

// loteriaClave es el código de la Lotería Nacional. Si el operador lo cambia,
// este worker deja de encontrar sorteo y las rifas quedan pendientes (nunca
// gradúa con un resultado ajeno).
const loteriaClave = "LN"

// GradaResultado repasa las rifas vencidas y devuelve (graduadas, pendientes).
func GradaResultado(db *gorm.DB) (int, int) {
	if db == nil {
		return 0, 0
	}
	graduadas, pendientes := 0, 0

	var loteria models.Loteria
	if err := db.Where("codigo = ?", loteriaClave).First(&loteria).Error; err != nil {
		return 0, 0
	}

	var rifas []models.Rifa
	db.Where("estado IN ?", []string{models.RifaAbierta, models.RifaPendiente}).
		Order("fecha_sorteo asc").Limit(100).Find(&rifas)

	for _, r := range rifas {
		if r.FechaSorteo.After(time.Now()) {
			continue
		}
		sorteo, ok := sorteoDelDia(db, loteria.ID, r.FechaSorteo)
		if !ok {
			if r.Estado != models.RifaPendiente {
				_ = db.Model(&models.Rifa{}).Where("id = ?", r.ID).
					Update("estado", models.RifaPendiente).Error
				nuevaAlerta(db, r, "sin sorteo de Lotería Nacional el "+
					r.FechaSorteo.In(tiempo.CR).Format("2006-01-02"))
			}
			pendientes++
			continue
		}
		numeros := numerosDelSorteo(sorteo)
		if len(numeros) == 0 {
			if r.Estado != models.RifaPendiente {
				_ = db.Model(&models.Rifa{}).Where("id = ?", r.ID).
					Update("estado", models.RifaPendiente).Error
			}
			nuevaAlerta(db, r, "el sorteo de Lotería Nacional del "+
				sorteo.Fecha.In(tiempo.CR).Format("2006-01-02")+" todavía no tiene resultado cargado")
			pendientes++
			continue
		}
		if err := Graduar(db, &r, numeros); err != nil {
			log.Printf("rifa %d: no se pudo graduar: %v", r.ID, err)
			continue
		}
		graduadas++
	}
	return graduadas, pendientes
}

// sorteoDelDia busca el sorteo de la lotería que cae en el mismo día calendario
// (en hora de Costa Rica) que la fecha de la rifa.
func sorteoDelDia(db *gorm.DB, loteriaID uint, fecha time.Time) (models.Sorteo, bool) {
	var sorteo models.Sorteo
	dia := fecha.In(tiempo.CR)
	desde := time.Date(dia.Year(), dia.Month(), dia.Day(), 0, 0, 0, 0, tiempo.CR)
	hasta := desde.Add(24 * time.Hour)
	err := db.Where("loteria_id = ? AND fecha >= ? AND fecha < ?", loteriaID, desde, hasta).
		Order("fecha asc").First(&sorteo).Error
	if err != nil {
		return sorteo, false
	}
	return sorteo, true
}

// numerosDelSorteo parsea los números ganadores guardados por el importador.
func numerosDelSorteo(s models.Sorteo) []int {
	if s.NumerosRaw == "" || s.NumerosRaw == "[]" {
		return nil
	}
	var out []int
	if err := json.Unmarshal([]byte(s.NumerosRaw), &out); err != nil {
		return nil
	}
	return out
}

// Graduar cierra la rifa con los números ganadores y paga el premio a quien
// tenga los números agraciados.
func Graduar(db *gorm.DB, rifa *models.Rifa, numeros []int) error {
	var vendidos []models.RifaNumero
	if err := db.Where("rifa_id = ? AND user_id IS NOT NULL", rifa.ID).
		Order("numero asc").Find(&vendidos).Error; err != nil {
		return err
	}
	if len(vendidos) == 0 {
		return nil
	}
	// El rango de la rifa puede ser menor que el número de la lotería: se toma
	// el último dígito (módulo el ancho del rango). Así una rifa de 0 a 49 se
	// gradúa con los dos últimos dígitos del número agraciado.
	ancho := rifa.HastaNumero - rifa.DesdeNumero + 1
	ganadores := make([]int, 0, len(numeros))
	vistos := map[int]bool{}
	for _, n := range numeros {
		normalizado := rifa.DesdeNumero + (n % ancho)
		if vistos[normalizado] {
			continue
		}
		vistos[normalizado] = true
		ganadores = append(ganadores, normalizado)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, num := range ganadores {
			var ganador models.RifaNumero
			if err := tx.Where("rifa_id = ? AND numero = ?", rifa.ID, num).First(&ganador).Error; err != nil {
				continue // no vendido: no hay a quien pagarle
			}
			if rifa.PremioMonto > 0 && ganador.UserID != nil {
				if err := tx.Model(&models.User{}).Where("id = ?", *ganador.UserID).
					Update("saldo", gorm.Expr("saldo + ?", rifa.PremioMonto)).Error; err != nil {
					return err
				}
				if err := tx.Create(&models.Movimiento{
					UserID: *ganador.UserID, Tipo: models.MovRifaPremio, Monto: rifa.PremioMonto,
					Descripcion: "Premio rifa " + rifa.Titulo,
				}).Error; err != nil {
					return err
				}
			}
		}
		rifa.Estado = models.RifaSorteada
		rifa.NumerosGanadores = ganadores
		if err := tx.Save(rifa).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuditLog{
			Accion: "rifa.graduada_loteria", Entidad: "rifa",
			EntidadID: itoa(uint64(rifa.ID)),
			Detalle:   "numeros=" + joinInts(ganadores),
		}).Error
	})
}

// marcarPendiente deja la rifa esperando el resultado de la lotería.
func marcarPendiente(db *gorm.DB, rifa *models.Rifa) {
	if rifa.Estado == models.RifaPendiente {
		return
	}
	_ = db.Model(&models.Rifa{}).Where("id = ?", rifa.ID).
		Update("estado", models.RifaPendiente).Error
	rifa.Estado = models.RifaPendiente
}

func nuevaAlerta(db *gorm.DB, r models.Rifa, motivo string) {
	var previas int64
	db.Model(&models.AlertaCS{}).
		Where("tipo = ? AND entidad_id = ? AND revisada = ? AND created_at > ?",
			models.AlertaRifaPendiente, itoa(uint64(r.ID)), false, time.Now().Add(-time.Hour)).
		Count(&previas)
	if previas > 0 {
		return
	}
	_ = db.Create(&models.AlertaCS{
		Tipo: models.AlertaRifaPendiente, Severidad: "alta", UserID: r.CreadoPor,
		Entidad: "rifa", EntidadID: itoa(uint64(r.ID)),
		Detalle: "«" + r.Titulo + "» venció y quedó pendiente: " + motivo,
	}).Error
}

// GraduarUna gradúa una rifa concreta con el sorteo de Lotería Nacional del
// mismo día. Devuelve el motivo cuando no se pudo (sin sorteo o sin resultado).
func GraduarUna(db *gorm.DB, rifaID uint) (bool, string) {
	var rifa models.Rifa
	if err := db.First(&rifa, rifaID).Error; err != nil {
		return false, "la rifa no existe"
	}
	if rifa.Estado == models.RifaSorteada {
		return false, "la rifa ya fue sorteada"
	}
	var loteria models.Loteria
	if err := db.Where("codigo = ?", loteriaClave).First(&loteria).Error; err != nil {
		return false, "no hay Lotería Nacional configurada"
	}
	sorteo, ok := sorteoDelDia(db, loteria.ID, rifa.FechaSorteo)
	if !ok {
		marcarPendiente(db, &rifa)
		nuevaAlerta(db, rifa, "no hay sorteo de Lotería Nacional el "+
			rifa.FechaSorteo.In(tiempo.CR).Format("2006-01-02"))
		return false, "no hay sorteo de Lotería Nacional el " +
			rifa.FechaSorteo.In(tiempo.CR).Format("2006-01-02")
	}
	numeros := numerosDelSorteo(sorteo)
	if len(numeros) == 0 {
		marcarPendiente(db, &rifa)
		nuevaAlerta(db, rifa, "el sorteo del "+
			sorteo.Fecha.In(tiempo.CR).Format("2006-01-02")+" no tiene resultado cargado")
		return false, "el sorteo de Lotería Nacional del " +
			sorteo.Fecha.In(tiempo.CR).Format("2006-01-02") + " no tiene resultado cargado"
	}
	if err := Graduar(db, &rifa, numeros); err != nil {
		return false, "no se pudo graduar: " + err.Error()
	}
	return true, ""
}

// Trabajador revisa las rifas vencidas cada 5 minutos.
func Trabajador(db *gorm.DB) {
	const cada = 5 * time.Minute
	for {
		time.Sleep(cada)
		g, p := GradaResultado(db)
		if g > 0 || p > 0 {
			log.Printf("rifas: %d graduada(s), %d pendiente(s)", g, p)
		}
	}
}

func itoa(n uint64) string {
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

func joinInts(ns []int) string {
	out := ""
	for i, n := range ns {
		if i > 0 {
			out += ","
		}
		out += itoa(uint64(n))
	}
	return out
}
