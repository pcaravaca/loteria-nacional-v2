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
	minuto int
}

var calendario = map[string]struct {
	dias   []time.Weekday
	turnos []turnoPlan
	titulo string
}{
	"LN": {dias: []time.Weekday{time.Sunday}, turnos: []turnoPlan{{"", 19, 0}}, titulo: "Lotería Nacional"},
	"CA": {dias: []time.Weekday{time.Tuesday, time.Friday}, turnos: []turnoPlan{{"", 19, 0}}, titulo: "Chances"},
	// Los tres turnos de NT son a las 12:55, 16:30 y 19:30 (hora CR).
	"NT": {dias: nil, turnos: []turnoPlan{
		{"la tarde", 16, 30}, {"el medio día", 12, 55}, {"la noche", 19, 30},
	}, titulo: "Nuevos Tiempos"},
	// Lotto y su revancha se cantan los lunes, miércoles y sábados.
	"LOTTO":    {dias: []time.Weekday{time.Monday, time.Wednesday, time.Saturday}, turnos: []turnoPlan{{"", 20, 0}}, titulo: "Lotto"},
	"REVANCHA": {dias: []time.Weekday{time.Monday, time.Wednesday, time.Saturday}, turnos: []turnoPlan{{"", 20, 0}}, titulo: "Lotto Revancha"},
	"MONAZOS": {dias: nil, turnos: []turnoPlan{
		{"el medio día", 12, 55}, {"la tarde", 16, 30}, {"la noche", 19, 30},
	}, titulo: "3 Monazos"},
}

// GenerarSorteos crea los sorteos futuros según el calendario, sin duplicar los
// que ya existen. Devuelve cuántos creó.
func GenerarSorteos(db *gorm.DB, zona *time.Location, diasAvantel int) (int, error) {
	return GenerarSorteosVentana(db, zona, 0, diasAvantel)
}

// GenerarSorteosVentana crea los sorteos que faltan en una ventana de diasAtras
// días anteriores y diasAvantel días siguientes, sin duplicar los que ya
// existen. Incluir el pasado es lo que permite recuperar los sorteos que
// quedaron sin crear (si el importador se cayó varios días, o si el calendario
// se generó por primera vez a mitad de mes) y por tanto importar sus
// resultados. Devuelve cuántos creó.
func GenerarSorteosVentana(db *gorm.DB, zona *time.Location, diasAtras, diasAvantel int) (int, error) {
	if diasAtras < 0 {
		diasAtras = 0
	}
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

	hoy := time.Now().In(zona)
	// Medianoche local, no Truncate(24h): Truncate alinea con la medianoche
	// UTC y en Costa Rica deja el día 6 horas antes, lo que corría el sorteo al
	// día siguiente (LN quedaba en lunes en vez de domingo, Chances en sábado).
	hoy = time.Date(hoy.Year(), hoy.Month(), hoy.Day(), 0, 0, 0, 0, zona)
	creados := 0
	for codigo, plan := range calendario {
		l, ok := porCodigo[codigo]
		if !ok {
			continue
		}
		for d := -diasAtras; d <= diasAvantel; d++ {
			fecha := hoy.AddDate(0, 0, d)
			if plan.dias != nil && !diaEsta(fecha, plan.dias) {
				continue
			}
			for _, tp := range plan.turnos {
				titulo := plan.titulo + " · Sorteo"
				if tp.sufijo != "" {
					titulo = fmt.Sprintf("%s · Sorteo de %s", plan.titulo, tp.sufijo)
				}
				f := fecha.Add(time.Duration(tp.hora)*time.Hour + time.Duration(tp.minuto)*time.Minute)
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

// existeSorteo evita duplicar un sorteo por (lotería, título, día). Se compara
// por día y no por hora exacta: los sorteos que ya existen en la base pueden
// tener una hora distinta de la que generaría el calendario hoy, y lo que
// importa es que el sorteo de ese turno y ese día esté una sola vez.
func existeSorteo(db *gorm.DB, loteriaID uint, titulo string, fecha time.Time) (bool, error) {
	var n int64
	inicio := time.Date(fecha.Year(), fecha.Month(), fecha.Day(), 0, 0, 0, 0, fecha.Location())
	fin := inicio.AddDate(0, 0, 1)
	err := db.Model(&models.Sorteo{}).
		Where("loteria_id = ? AND titulo = ? AND fecha >= ? AND fecha < ?", loteriaID, titulo, inicio, fin).
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
