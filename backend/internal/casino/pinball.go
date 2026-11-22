package casino

import "fmt"

// Pinball: tres mesas que comparten el mismo motor.
//
// La bola cae por la mesa rebotando en los bumpers; cada rebote sale de un draw
// (par = izquierda, impar = derecha), así que la tronera en la que cae sigue una
// distribución binomial. La fuerza del lanzamiento elige cuántos rebotes da la
// bola y, con eso, la tabla de multiplicadores: a más fuerza, mayor dispersión y
// mayor premio máximo. Las palancas son parte de la animación de la mesa.
type pinballMesa struct {
	N       int
	Tabla   []int64
	MaxMult int64
}

// pinballMesas: cada mesa tiene tres tablas, una por fuerza (0 suave, 1 media,
// 2 fuerte). Los multiplicadores están calibrados para un RTP de entre 81% y 88%.
var pinballMesas = map[string]map[int]pinballMesa{
	"pinball-neon": {
		0: {N: 4, Tabla: []int64{4, 0, 1, 0, 4}, MaxMult: 4},
		1: {N: 7, Tabla: []int64{17, 0, 0, 1, 1, 0, 0, 17}, MaxMult: 17},
		2: {N: 10, Tabla: []int64{10, 2, 1, 3, 0, 0, 0, 3, 1, 2, 10}, MaxMult: 10},
	},
	"pinball-clasico": {
		0: {N: 4, Tabla: []int64{3, 1, 0, 1, 3}, MaxMult: 3},
		1: {N: 7, Tabla: []int64{12, 1, 0, 1, 1, 0, 1, 12}, MaxMult: 12},
		2: {N: 10, Tabla: []int64{12, 2, 1, 3, 0, 0, 0, 3, 1, 2, 12}, MaxMult: 12},
	},
	"plinko-premium": {
		0: {N: 6, Tabla: []int64{6, 1, 1, 0, 1, 1, 6}, MaxMult: 6},
		1: {N: 9, Tabla: []int64{20, 0, 2, 0, 1, 1, 0, 2, 0, 20}, MaxMult: 20},
		2: {N: 12, Tabla: []int64{11, 3, 0, 2, 0, 1, 1, 1, 0, 2, 0, 3, 11}, MaxMult: 11},
	},
}

func pinballMesaDe(codigo string, fuerza int) (pinballMesa, bool) {
	filas, ok := pinballMesas[codigo]
	if !ok {
		return pinballMesa{}, false
	}
	m, ok := filas[fuerza]
	return m, ok
}

// pinballMaxMult es el multiplicador máximo que la mesa puede pagar.
func pinballMaxMult(codigo string) int64 {
	filas, ok := pinballMesas[codigo]
	if !ok {
		return 0
	}
	var max int64
	for _, m := range filas {
		if m.MaxMult > max {
			max = m.MaxMult
		}
	}
	return max
}

// pinballDraws: los rebotes de la tabla más larga.
func pinballDraws(codigo string) int {
	filas, ok := pinballMesas[codigo]
	if !ok {
		return 1
	}
	max := 0
	for _, m := range filas {
		if m.N > max {
			max = m.N
		}
	}
	return max
}

func clampEntero(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func jugarPinball(codigo string, monto int64, draws []int, p Params) (Outcome, error) {
	fuerza, err := p.Int("fuerza")
	if err != nil {
		return Outcome{}, err
	}
	fuerza = clampEntero(fuerza, 0, 2)
	mesa, ok := pinballMesaDe(codigo, fuerza)
	if !ok {
		return Outcome{}, fmt.Errorf("mesa de pinball no valida: %s", codigo)
	}
	if len(draws) < mesa.N {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}

	camino := make([]int, 0, mesa.N)
	pos := 0
	for i := 0; i < mesa.N; i++ {
		d := draws[i] % 2
		camino = append(camino, d)
		if d == 1 {
			pos++
		}
	}
	mult := mesa.Tabla[pos]
	data := map[string]any{
		"camino": camino,
		"pos":    pos,
		"fuerza": fuerza,
		"tabla":  mesa.Tabla,
	}
	return Outcome{Multiplicador: mult, Data: data}, nil
}
