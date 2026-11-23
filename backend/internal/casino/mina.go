package casino

import "fmt"

// Mina de Oro: una grilla de 5 columnas por F filas con 3 minas por fila. El
// camino baja eligiendo una columna por fila; cada fila superada sube el
// multiplicador y si pisa una mina se pierde todo lo avanzado.
//
// La dificultad elige cuántas filas tiene la grilla: a más filas, más lejos
// puede llegar el camino y más alto es el premio máximo.

type minaNivel struct {
	Nombre  string
	Filas   int
	Tabla   []int64 // indice = filas superadas
	MaxMult int64
}

var minaNiveles = map[int]minaNivel{
	0: {Nombre: "Suave", Filas: 4, Tabla: []int64{0, 1, 2, 4, 8}, MaxMult: 8},
	1: {Nombre: "Media", Filas: 5, Tabla: []int64{0, 1, 2, 3, 6, 10}, MaxMult: 10},
	2: {Nombre: "Fuerte", Filas: 6, Tabla: []int64{0, 1, 2, 3, 6, 10, 18}, MaxMult: 18},
}

const (
	minaCodigoBase = "mina-de-oro"
	minaColumnas   = 5
	minaPorFila    = 3
	minaMaxFilas   = 6
)

func minaNivelDe(dificultad int) (minaNivel, bool) {
	n, ok := minaNiveles[dificultad]
	return n, ok
}

func minaMultiplicadorMaximo() int64 {
	return minaNiveles[2].MaxMult
}

// minaJugar resuelve la jugada. `draws` sale de Sortear con el mismo orden.
func minaJugar(codigo string, monto int64, draws []int, p Params) (Outcome, error) {
	dificultad, err := p.Int("dificultad")
	if err != nil {
		return Outcome{}, err
	}
	nivel, ok := minaNivelDe(dificultad)
	if !ok {
		return Outcome{}, fmt.Errorf("dificultad de mina no valida: %d", dificultad)
	}
	if len(draws) < nivel.Filas*(minaPorFila+1) {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}

	filas := nivel.Filas
	minas := make([][]int, filas)
	camino := make([]int, filas)
	pasos := 0
	for f := 0; f < filas; f++ {
		base := f * (minaPorFila + 1)
		// tres minas en columnas distintas
		elegidas := make([]int, 0, minaPorFila)
		for i := 0; i < minaPorFila; i++ {
			c := draws[base+i] % minaColumnas
			duplicada := false
			for _, e := range elegidas {
				if e == c {
					duplicada = true
					break
				}
			}
			if duplicada {
				c = (c + 1) % minaColumnas
			}
			elegidas = append(elegidas, c)
		}
		minas[f] = elegidas
		camino[f] = draws[base+minaPorFila] % minaColumnas
		piso := false
		for _, e := range elegidas {
			if e == camino[f] {
				piso = true
				break
			}
		}
		if piso {
			break
		}
		pasos++
	}

	var mult int64
	if pasos < len(nivel.Tabla) {
		mult = nivel.Tabla[pasos]
	}
	data := map[string]any{
		"filas":      filas,
		"columnas":   minaColumnas,
		"minas":      minas,
		"camino":     camino,
		"pasos":      pasos,
		"dificultad": dificultad,
		"tabla":      nivel.Tabla,
	}
	return Outcome{Multiplicador: mult, Data: data}, nil
}
