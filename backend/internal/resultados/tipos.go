// Package resultados importa los números ganadores de los sorteos desde
// fuentes públicas y los gradúa contra nuestras apuestas.
//
// La JPS (jps.go.cr) solo publica cada resultado como un PDF escaneado, sin
// API ni capa de texto, así que el importador automático usa espejos en texto
// (hoy nacionalloteria.com) y deja la JPS como referencia oficial de verificación.
package resultados

import (
	"context"
	"time"
)

// Codigos de juego que el importador entiende. Deben coincidir con el codigo
// de la lotería en la base (models.Loteria.Codigo).
const (
	JuegoLoteriaNacional = "LN"
	JuegoChances         = "CA"
	JuegoNuevosTiempos   = "NT"
	JuegoLotto           = "LOTTO"
	JuegoLottoRevancha   = "REVANCHA"
	JuegoTresMonazos     = "MONAZOS"
)

// Turnos de los sorteos diarios.
const (
	TurnoMedioDia = "medio_dia"
	TurnoTarde    = "tarde"
	TurnoNoche    = "noche"
)

// Resultado es un sorteo ya cantado, normalizado.
type Resultado struct {
	Juego      string    // codigo de loteria (LN, CA, NT, ...)
	SorteoNum  string    // numero de sorteo segun la fuente (4927, 7087, 24185|...)
	Fecha      time.Time // fecha del sorteo (zona de Costa Rica)
	Turno      string    // medio_dia, tarde, noche; "" si no aplica
	Numeros    []int     // numeros ganadores
	Series     []string  // series, cuando la fuente las publica
	Fuente     string
	URL        string
	Observaciones string
}

// Proveedor sabe cómo traer resultados de una fuente concreta.
type Proveedor interface {
	Nombre() string
	// Soporta indica si la fuente cubre ese juego.
	Soporta(juego string) bool
	// Obtener devuelve los resultados de un juego para una fecha (idealmente uno).
	// Si la fuente no publica esa fecha, devuelve lista vacia sin error.
	Obtener(ctx context.Context, juego string, fecha time.Time) ([]Resultado, error)
}
