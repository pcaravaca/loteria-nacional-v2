package casino

import (
	"fmt"
	"sort"
)

// Tetris por lineas: un pozo de 6 columnas por 12 filas y una secuencia de 12
// piezas (dos bolsas de 7 mezcladas con el sorteo). El jugador elige la columna
// y la rotacion de cada pieza; las lineas completadas definen el multiplicador.
//
// La secuencia de piezas sale de las semillas (provably fair) y el tablero se
// puede reproducir completo a partir de las jugadas guardadas.

const (
	// TetrisColumnas y TetrisFilas son las dimensiones del pozo.
	TetrisColumnas = 6
	TetrisFilas    = 12
	// TetrisPiezas es la cantidad de piezas por ronda.
	TetrisPiezas = 8
	// TetrisTipos es la cantidad de tetrominós (I, O, T, S, Z, L, J).
	TetrisTipos = 7
	// TetrisMaxMult es el multiplicador maximo declarado (8+ lineas).
	TetrisMaxMult = 3
)

// TetrisCelda es una posicion relativa dentro de la pieza.
type TetrisCelda [2]int

// tetrisFormas son las siete piezas en su orientacion base.
var tetrisFormas = [TetrisTipos][]TetrisCelda{
	0: {{0, 1}, {1, 1}, {2, 1}, {3, 1}}, // I
	1: {{1, 0}, {2, 0}, {1, 1}, {2, 1}}, // O
	2: {{1, 0}, {0, 1}, {1, 1}, {2, 1}}, // T
	3: {{1, 0}, {2, 0}, {0, 1}, {1, 1}}, // S
	4: {{0, 0}, {1, 0}, {1, 1}, {2, 1}}, // Z
	5: {{2, 0}, {0, 1}, {1, 1}, {2, 1}}, // L
	6: {{0, 0}, {0, 1}, {1, 1}, {2, 1}}, // J
}

// tetrisRotar rota la pieza 90 grados a la derecha y la normaliza.
func tetrisRotar(celdas []TetrisCelda) []TetrisCelda {
	out := make([]TetrisCelda, len(celdas))
	minx, miny := 99, 99
	for i, c := range celdas {
		out[i] = TetrisCelda{-c[1], c[0]}
		if out[i][0] < minx {
			minx = out[i][0]
		}
		if out[i][1] < miny {
			miny = out[i][1]
		}
	}
	for i := range out {
		out[i][0] -= minx
		out[i][1] -= miny
	}
	return out
}

// tetrisRotada devuelve la pieza en la rotacion pedida (0..3).
func tetrisRotada(tipo, rot int) []TetrisCelda {
	c := append([]TetrisCelda{}, tetrisFormas[tipo]...)
	for i := 0; i < rot; i++ {
		c = tetrisRotar(c)
	}
	return c
}

// TetrisSecuencia devuelve las 12 piezas de la ronda: dos bolsas de 7 mezcladas
// con el sorteo. La misma semilla y nonce producen la misma secuencia.
func TetrisSecuencia(clientSeed, serverSeed string, nonce int64) []int {
	draws := Sortear(serverSeed, clientSeed, nonce, TetrisTipos*2, 1<<30)
	seq := make([]int, 0, TetrisPiezas)
	for bolsa := 0; bolsa < 2; bolsa++ {
		base := bolsa * TetrisTipos
		orden := []int{0, 1, 2, 3, 4, 5, 6}
		sort.Slice(orden, func(a, b int) bool {
			ca, cb := draws[base+orden[a]], draws[base+orden[b]]
			if ca != cb {
				return ca < cb
			}
			return orden[a] < orden[b]
		})
		seq = append(seq, orden...)
	}
	if len(seq) > TetrisPiezas {
		seq = seq[:TetrisPiezas]
	}
	return seq
}

// TetrisEstado es el estado de una ronda.
type TetrisEstado struct {
	Tablero []int // 72 celdas: 0 vacia, 1..7 tipo de pieza
	Pieza   int   // indice en la secuencia
	Lineas  int
	Fin     bool
}

// TetrisSoltar baja la pieza en la columna y rotacion indicadas y devuelve las
// lineas limpias. ok es false si la pieza no entra.
func TetrisSoltar(tablero []int, secuencia []int, pieza, col, rot int) (int, bool) {
	if pieza < 0 || pieza >= len(secuencia) {
		return 0, false
	}
	celdas := tetrisRotada(secuencia[pieza], rot)
	maxx := 0
	for _, c := range celdas {
		if c[0] > maxx {
			maxx = c[0]
		}
	}
	if col < 0 || col+maxx >= TetrisColumnas {
		return 0, false
	}
	fits := func(y int) bool {
		for _, c := range celdas {
			x, yy := col+c[0], y+c[1]
			if x < 0 || x >= TetrisColumnas || yy < 0 || yy >= TetrisFilas {
				return false
			}
			if tablero[yy*TetrisColumnas+x] != 0 {
				return false
			}
		}
		return true
	}
	if !fits(0) {
		return 0, false
	}
	y := 0
	for fits(y+1) {
		y++
	}
	valor := secuencia[pieza] + 1
	for _, c := range celdas {
		tablero[(y+c[1])*TetrisColumnas+col+c[0]] = valor
	}
	limpias := 0
	nuevo := make([]int, TetrisFilas*TetrisColumnas)
	dest := TetrisFilas - 1
	for r := TetrisFilas - 1; r >= 0; r-- {
		full := true
		for c := 0; c < TetrisColumnas; c++ {
			if tablero[r*TetrisColumnas+c] == 0 {
				full = false
				break
			}
		}
		if full {
			limpias++
			continue
		}
		copy(nuevo[dest*TetrisColumnas:(dest+1)*TetrisColumnas], tablero[r*TetrisColumnas:(r+1)*TetrisColumnas])
		dest--
	}
	copy(tablero, nuevo)
	return limpias, true
}

// TetrisMult devuelve el multiplicador segun las lineas acumuladas.
func TetrisMult(lineas int) float64 {
	switch {
	case lineas <= 0:
		return 0
	case lineas == 1:
		return 0.28
	case lineas == 2:
		return 0.55
	case lineas == 3:
		return 0.85
	case lineas == 4:
		return 1.15
	case lineas == 5:
		return 1.45
	case lineas == 6:
		return 1.75
	case lineas == 7:
		return 2.05
	default:
		return float64(TetrisMaxMult)
	}
}

// TetrisPremio calcula lo que se paga segun las lineas.
func TetrisPremio(monto int64, lineas int) int64 {
	if monto <= 0 {
		return 0
	}
	return int64(float64(monto) * TetrisMult(lineas))
}

func tetrisAltura(tablero []int) int {
	for r := 0; r < TetrisFilas; r++ {
		for c := 0; c < TetrisColumnas; c++ {
			if tablero[r*TetrisColumnas+c] != 0 {
				return TetrisFilas - r
			}
		}
	}
	return 0
}

func tetrisHuecos(tablero []int) int {
	huecos := 0
	for c := 0; c < TetrisColumnas; c++ {
		encima := false
		for r := 0; r < TetrisFilas; r++ {
			if tablero[r*TetrisColumnas+c] != 0 {
				encima = true
			} else if encima {
				huecos++
			}
		}
	}
	return huecos
}

// TetrisSimular juega una ronda completa con una estrategia determinista.
// "baja" elige la jugada que mantiene el pozo mas bajo (la jugada habil) y
// "aleatorio" coloca cada pieza en un lugar al azar.
func TetrisSimular(clientSeed, serverSeed string, nonce int64, estrategia string) int {
	seq := TetrisSecuencia(clientSeed, serverSeed, nonce)
	tab := make([]int, TetrisFilas*TetrisColumnas)
	lineas := 0
	for p := 0; p < TetrisPiezas; p++ {
		if estrategia == "aleatorio" {
			var validas [][2]int
			for col := 0; col < TetrisColumnas; col++ {
				for rot := 0; rot < 4; rot++ {
					copia := append([]int{}, tab...)
					if _, ok := TetrisSoltar(copia, seq, p, col, rot); ok {
						validas = append(validas, [2]int{col, rot})
					}
				}
			}
			if len(validas) == 0 {
				break
			}
			d := Sortear(serverSeed, fmt.Sprintf("%s#%d", clientSeed, p), nonce, 1, 1<<30)[0]
			ele := validas[d%len(validas)]
			lim, _ := TetrisSoltar(tab, seq, p, ele[0], ele[1])
			lineas += lim
			continue
		}
		mejorCol, mejorRot, mejorScore := 0, 0, -1<<30
		encontro := false
		for col := 0; col < TetrisColumnas; col++ {
			for rot := 0; rot < 4; rot++ {
				copia := append([]int{}, tab...)
				lim, ok := TetrisSoltar(copia, seq, p, col, rot)
				if !ok {
					continue
				}
				score := lim*100 - tetrisAltura(copia)*10 - tetrisHuecos(copia)*5
				if score > mejorScore {
					mejorScore, mejorCol, mejorRot = score, col, rot
					encontro = true
				}
			}
		}
		if !encontro {
			break
		}
		lim, _ := TetrisSoltar(tab, seq, p, mejorCol, mejorRot)
		lineas += lim
	}
	return lineas
}

// TetrisRepetir reproduce el tablero y las lineas a partir de las jugadas.
func TetrisRepetir(clientSeed, serverSeed string, nonce int64, jugadas [][2]int) ([]int, int, error) {
	seq := TetrisSecuencia(clientSeed, serverSeed, nonce)
	tab := make([]int, TetrisFilas*TetrisColumnas)
	lineas := 0
	for i, j := range jugadas {
		lim, ok := TetrisSoltar(tab, seq, i, j[0], j[1])
		if !ok {
			return nil, 0, fmt.Errorf("jugada invalida")
		}
		lineas += lim
	}
	return tab, lineas, nil
}

// RTPTetris mide el retorno al jugador por simulacion.
func RTPTetris(iteraciones int, estrategia string) float64 {
	if iteraciones <= 0 {
		iteraciones = 200000
	}
	var suma float64
	for i := 0; i < iteraciones; i++ {
		seed := fmt.Sprintf("sim-%d", i)
		lineas := TetrisSimular("sim", seed, int64(i), estrategia)
		suma += TetrisMult(lineas)
	}
	return suma / float64(iteraciones) * 100
}
