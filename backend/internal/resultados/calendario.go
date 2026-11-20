package resultados

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"loteria-backend/internal/models"
)

// Planificación de sorteos. Cada lotería tiene un calendario conocido: la JPS
// canta Lotería Nacional los domingos, Chances los martes y viernes, y Nuevos
// Tiempos tres veces por día. Generar los sorteos por anticipado es lo que
// permite que el importador tenga dónde caerle a cada resultado.
type turnoPlan struct {
	sufijo string // como aparece en el título del sorteo
	hora   int
}

var calendario = map[string]struct {
	dias   []time.Weekday
	turnos []turnoPlan
	titulo string
}{
	"LN": {dias: []time.Weekday{time.Sunday}, turnos: []turnoPlan{{"", 19}}, titulo: "Lotería Nacional"},
	"CA": {dias: []time.Weekday{time.Tuesday, time.Friday}, turnos: []turnoPlan{{"", 19}}, titulo: "Chances"},
	"NT": {dias: nil, turnos: []turnoPlan{
		{"la tarde", 17}, {"el medio día", 12}, {"la noche", 21},
	}, titulo: "Nuevos Tiempos"},
	// Lotto y su revancha se cantan los lunes, miércoles y sábados.
	"LOTTO":    {dias: []time.Weekday{time.Monday, time.Wednesday, time.Saturday}, turnos: []turnoPlan{{"", 20}}, titulo: "Lotto"},
	"REVANCHA": {dias: []time.Weekday{time.Monday, time.Wednesday, time.Saturday}, turnos: []turnoPlan{{"", 20}}, titulo: "Lotto Revancha"},
	"MONAZOS": {dias: nil, turnos: []turnoPlan{
		{"el medio día", 12}, {"la tarde", 17}, {"la noche", 21},
	}, titulo: "3 Monazos"},
}

// GenerarSorteos crea los sorteos futuros según el calendario, sin duplicar los
// que ya existen. Devuelve cuántos creó.
func GenerarSorteos(db *gorm.DB, zona *time.Location, diasAvantel int) (int, error) {
	if diasAvantel <= 0 {
		diasAvantel = 21
	}
	if zona == nil {
		zona = time.FixedZone("CR", -6*3600)
	}
	porCodigo := map[string]models.Loteria{}
	var loterias []models.Loteria
	if err := db.Where("activa = ?", true).Find(&loterias).Error; err != nil {
		return 0, err
	}
	for _, l := range loterias {
		porCodigo[l.Codigo] = l
	}

	hoy := time.Now().In(zona).Truncate(24 * time.Hour)
	creados := 0
	for codigo, plan := range calendario {
		l, ok := porCodigo[codigo]
		if !ok {
			continue
		}
		for d := 0; d <= diasAvantel; d++ {
			fecha := hoy.AddDate(0, 0, d)
			if plan.dias != nil && !diaEsta(fecha, plan.dias) {
				continue
			}
			for _, tp := range plan.turnos {
				titulo := plan.titulo + " · Sorteo"
				if tp.sufijo != "" {
					titulo = fmt.Sprintf("%s · Sorteo de %s", plan.titulo, tp.sufijo)
				}
				f := fecha.Add(time.Duration(tp.hora) * time.Hour)
				existe, err := existeSorteo(db, l.ID, titulo, f)
				if err != nil {
					return creados, err
				}
				if existe {
					continue
				}
				s := models.Sorteo{LoteriaID: l.ID, Titulo: titulo, Fecha: f, Estado: models.SorteoAbierto}
				if err := db.Create(&s).Error; err != nil {
					return creados, err
				}
				creados++
			}
		}
	}
	return creados, nil
}

func diaEsta(f time.Time, dias []time.Weekday) bool {
	for _, d := range dias {
		if f.Weekday() == d {
			return true
		}
	}
	return false
}

func existeSorteo(db *gorm.DB, loteriaID uint, titulo string, fecha time.Time) (bool, error) {
	var n int64
	err := db.Model(&models.Sorteo{}).
		Where("loteria_id = ? AND titulo = ? AND fecha = ?", loteriaID, titulo, fecha).
		Count(&n).Error
	return n > 0, err
}

// NormalizaTurno deja el turno del título en minúsculas y sin tildes, para
// comparar con lo que publica la fuente.
func NormalizaTurno(titulo string) string {
	t := strings.ToLower(titulo)
	repl := strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u")
	return repl.Replace(t)
}
