// Package tiempo centraliza el huso horario de la plataforma (Costa Rica,
// UTC-6) y los cortes de periodo que usan los topes de juego responsable.
//
// Los limites diarios/semanales se comparan contra created_at guardado en
// base. El driver de SQLite compara esas marcas como texto, asi que el
// proceso debe escribir y consultar siempre en el mismo huso: Fijar() se
// llama al abrir la base para que time.Now() coincida con CR sin depender
// del TZ del host (sin esto, un servidor con TZ distinto dejaba fuera de la
// ventana las jugadas del dia y los topes no se aplicaban).
package tiempo

import "time"

// CR es la hora de Costa Rica (UTC-6). Costa Rica no usa horario de verano.
var CR = time.FixedZone("CR", -6*60*60)

// Fijar deja time.Local en CR. Es idempotente y cubre servidor, CLI y tests.
func Fijar() {
	time.Local = CR
}

// InicioDia devuelve el medianoche CR del dia en curso.
func InicioDia() time.Time {
	ahora := time.Now().In(CR)
	return time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, CR)
}

// InicioSemana devuelve el lunes 00:00 CR de la semana en curso.
func InicioSemana() time.Time {
	t := InicioDia()
	wd := int(t.Weekday()) // domingo = 0
	if wd == 0 {
		wd = 7
	}
	return t.AddDate(0, 0, -(wd - 1))
}

// InicioMes devuelve el primer dia del mes en curso a las 00:00 CR.
func InicioMes() time.Time {
	t := InicioDia()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, CR)
}
