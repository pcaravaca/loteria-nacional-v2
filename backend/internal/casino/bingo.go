package casino

import (
	"fmt"
	"sort"
)

// Bingo 75: cartón de 5x5 con las columnas B(1-15) I(16-30) N(31-45) G(46-60)
// O(61-75) y el centro libre. La ronda canta bolillas de a una desde un orden
// mezclado que deriva de las semillas (provably fair); el pago depende de las
// líneas completadas en los cartones.
//
// El orden de cantadas es finito (BingoCantadas) para que el juego cierre solo:
// el jugador no puede estirar la ronda hasta llenar el cartón, que siempre
// ganaría. Con 40 cantadas una línea sigue siendo un logro y el cartón lleno
// queda como premio mayor.

const (
	// BingoCantadas es la cantidad de bolillas que se cantan por ronda. Con 42
	// cantadas una línea sigue siendo un logro (~52% de las jugadas no completan
	// ninguna) y el RTP de la tabla x1/x3/x25 queda cerca del 88%.
	BingoCantadas = 42
	// BingoMaxCartones es la cantidad máxima de cartones por jugada.
	BingoMaxCartones = 4
)

// BingoSorteo devuelve el orden de las 75 bolillas y los cartones del jugador.
// Todo se deriva de las semillas y el nonce: misma semilla y nonce producen el
// mismo orden y los mismos cartones.
func BingoSorteo(clientSeed, serverSeed string, nonce int64, cartones int) ([]int, [][][]int) {
	if cartones < 1 {
		cartones = 1
	}
	if cartones > BingoMaxCartones {
		cartones = BingoMaxCartones
	}
	// 74 pasos para mezclar las 75 bolillas + 14 por columna (5 columnas) por
	// cartón. Se piden unos más por si acaso.
	draws := Sortear(serverSeed, clientSeed, nonce, 74+5*14*cartones+8, 1<<31-1)
	k := 0
	next := func() int {
		v := draws[k]
		k++
		return v
	}

	orden := make([]int, 75)
	for i := range orden {
		orden[i] = i + 1
	}
	for i := len(orden) - 1; i > 0; i-- {
		j := next() % (i + 1)
		orden[i], orden[j] = orden[j], orden[i]
	}

	grids := make([][][]int, cartones)
	for n := 0; n < cartones; n++ {
		grids[n] = bingoCarton(next)
	}
	return orden, grids
}

// bingoCarton arma un cartón: cada columna elige sus números de su decena y
// quedan ordenados de menor a mayor; el centro se entrega libre (0).
func bingoCarton(next func() int) [][]int {
	grid := make([][]int, 5)
	for r := range grid {
		grid[r] = make([]int, 5)
	}
	for c := 0; c < 5; c++ {
		pool := make([]int, 15)
		for i := range pool {
			pool[i] = 15*c + i + 1
		}
		for i := len(pool) - 1; i > 0; i-- {
			j := next() % (i + 1)
			pool[i], pool[j] = pool[j], pool[i]
		}
		necesita := 5
		filas := []int{0, 1, 2, 3, 4}
		if c == 2 { // columna N: el centro es libre
			necesita = 4
			filas = []int{0, 1, 3, 4}
		}
		elegidos := append([]int{}, pool[:necesita]...)
		sort.Ints(elegidos)
		for i, r := range filas {
			grid[r][c] = elegidos[i]
		}
	}
	grid[2][2] = 0
	return grid
}

// BingoMarcadas devuelve las casillas marcadas del cartón según las bolillas
// cantadas. El centro siempre está marcado.
func BingoMarcadas(grid [][]int, cantadas []int) [5][5]bool {
	var m [5][5]bool
	vistos := make(map[int]bool, len(cantadas))
	for _, b := range cantadas {
		vistos[b] = true
	}
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			if r == 2 && c == 2 {
				m[r][c] = true
				continue
			}
			m[r][c] = vistos[grid[r][c]]
		}
	}
	return m
}

// BingoLineas cuenta las líneas completas: las 5 filas, las 5 columnas y las dos
// diagonales (el centro libre cuenta como marcado).
func BingoLineas(m [5][5]bool) int {
	lineas := 0
	for r := 0; r < 5; r++ {
		if m[r][0] && m[r][1] && m[r][2] && m[r][3] && m[r][4] {
			lineas++
		}
	}
	for c := 0; c < 5; c++ {
		if m[0][c] && m[1][c] && m[2][c] && m[3][c] && m[4][c] {
			lineas++
		}
	}
	if m[0][0] && m[1][1] && m[2][2] && m[3][3] && m[4][4] {
		lineas++
	}
	if m[0][4] && m[1][3] && m[2][2] && m[3][1] && m[4][0] {
		lineas++
	}
	return lineas
}

// BingoLleno indica si el cartón quedó completo (bingo).
func BingoLleno(m [5][5]bool) bool {
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			if !m[r][c] {
				return false
			}
		}
	}
	return true
}

// BingoTier devuelve el multiplicador por cartón según las líneas completadas:
// una línea devuelve lo apostado, dos o más líneas paga x3 y el cartón lleno x25.
func BingoTier(lineas int, lleno bool) int64 {
	if lleno {
		return 25
	}
	switch {
	case lineas >= 2:
		return 3
	case lineas == 1:
		return 1
	}
	return 0
}

// BingoPremio calcula el premio de la ronda. Cada cartón es una apuesta
// independiente de `monto`: paga su propio multiplicador y los premios se suman.
// Devuelve el premio total, la mayor cantidad de líneas en un cartón, si algún
// cartón se llenó y el multiplicador de cada cartón.
func BingoPremio(monto int64, grids [][][]int, cantadas []int) (premio int64, lineas int, lleno bool, tiers []int64) {
	for _, g := range grids {
		m := BingoMarcadas(g, cantadas)
		l := BingoLineas(m)
		full := BingoLleno(m)
		t := BingoTier(l, full)
		if l > lineas {
			lineas = l
		}
		if full {
			lleno = true
		}
		tiers = append(tiers, t)
		premio += monto * t
	}
	return premio, lineas, lleno, tiers
}

// RTPBingo mide el retorno al jugador por simulación, cantando las 40 bolillas
// en un solo cartón (el RTP no depende de la cantidad de cartones: cada uno es
// una apuesta independiente).
func RTPBingo(iteraciones int) float64 {
	if iteraciones <= 0 {
		iteraciones = 200000
	}
	var suma float64
	for i := 0; i < iteraciones; i++ {
		seed := fmt.Sprintf("sim-%d", i)
		orden, grids := BingoSorteo("sim", seed, int64(i), 1)
		premio, _, _, _ := BingoPremio(1000, grids, orden[:BingoCantadas])
		suma += float64(premio) / 1000
	}
	return suma / float64(iteraciones) * 100
}
