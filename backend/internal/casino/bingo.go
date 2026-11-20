package casino

import (
	"fmt"
	"sort"
)

// Bingo: motor parametrizado por variante. Todas las variantes comparten el
// mismo esquema provably fair (HMAC-SHA256 con server_seed + client_seed y
// nonce) y difieren en la geometria del carton, la cantidad de bolillas, el
// tope de cantadas y los patrones que pagan.
//
// El orden de cantadas es finito (TopeCantadas) para que la ronda cierre sola:
// el jugador no puede estirarla hasta llenar el carton, que siempre ganaria.

const (
	// BingoCantadas es el tope de la variante clasica de 75 bolillas.
	BingoCantadas = 42
	// BingoMaxCartones es la cantidad maxima de cartones por jugada.
	BingoMaxCartones = 4
)

// BingoPago describe una linea de la tabla de pagos: se usa para mostrarla en
// el frontend y para documentar las reglas de cada variante.
type BingoPago struct {
	Nombre string
	Mult   int64
}

// BingoVariante define la geometria y las reglas de una modalidad de bingo.
type BingoVariante struct {
	Codigo       string
	Nombre       string
	Descripcion  string
	Bolillas     int
	Filas        int
	Columnas     int
	CentroLibre  bool
	Diagonal     bool // las diagonales cuentan como linea (solo tablero cuadrado)
	TopeCantadas int
	MaxCartones  int
	Pagos        []BingoPago
	// Evalua devuelve el multiplicador del carton y el nombre del patron que
	// pago. `m` son las casillas marcadas, `lineas` las lineas completas y
	// `lleno` indica el carton completo.
	Evalua func(m [][]bool, lineas int, lleno bool) (int64, string)
}

// bingoVariantePorCodigo devuelve la variante declarada o false si el codigo no
// existe. El handler usa esto para validar lo que pide el cliente.
func bingoVariantePorCodigo(codigo string) (BingoVariante, bool) {
	v, ok := bingoVariantes[codigo]
	return v, ok
}

// BingoVariantePorCodigo es la version exportada (catalogo, handlers, tests).
func BingoVariantePorCodigo(codigo string) (BingoVariante, bool) {
	return bingoVariantePorCodigo(codigo)
}

// BingoVariantes devuelve las variantes declaradas, en orden de catalogo.
func BingoVariantes() []BingoVariante {
	codigos := make([]string, 0, len(bingoVariantes))
	for c := range bingoVariantes {
		codigos = append(codigos, c)
	}
	sort.Strings(codigos)
	out := make([]BingoVariante, 0, len(codigos))
	for _, c := range codigos {
		out = append(out, bingoVariantes[c])
	}
	return out
}

var bingoVariantes = map[string]BingoVariante{
	"bingo": {
		Codigo:       "bingo",
		Nombre:       "Bingo 75",
		Descripcion:  "Carton 5x5 de 75 bolillas: una linea devuelve lo apostado, dos o mas lineas pagan x3 y el carton lleno x25.",
		Bolillas:     75,
		Filas:        5,
		Columnas:     5,
		CentroLibre:  true,
		Diagonal:     true,
		TopeCantadas: BingoCantadas,
		MaxCartones:  BingoMaxCartones,
		Pagos: []BingoPago{
			{Nombre: "Carton lleno", Mult: 25},
			{Nombre: "Dos o mas lineas", Mult: 3},
			{Nombre: "Una linea", Mult: 1},
		},
		Evalua: func(_ [][]bool, lineas int, lleno bool) (int64, string) {
			switch {
			case lleno:
				return 25, "Carton lleno"
			case lineas >= 2:
				return 3, "Dos o mas lineas"
			case lineas == 1:
				return 1, "Una linea"
			}
			return 0, ""
		},
	},
	"bingo90": {
		Codigo:       "bingo90",
		Nombre:       "Bingo 90",
		Descripcion:  "Ticket 3x9 de 90 bolillas: una linea x2, dos lineas x5 y carton lleno x40.",
		Bolillas:     90,
		Filas:        3,
		Columnas:     9,
		CentroLibre:  false,
		Diagonal:     false,
		TopeCantadas: 60,
		MaxCartones:  BingoMaxCartones,
		Pagos: []BingoPago{
			{Nombre: "Carton lleno", Mult: 40},
			{Nombre: "Dos lineas", Mult: 5},
			{Nombre: "Una linea", Mult: 2},
		},
		Evalua: func(_ [][]bool, lineas int, lleno bool) (int64, string) {
			switch {
			case lleno:
				return 40, "Carton lleno"
			case lineas >= 2:
				return 5, "Dos lineas"
			case lineas == 1:
				return 2, "Una linea"
			}
			return 0, ""
		},
	},
	"bingo-esquinas": {
		Codigo:       "bingo-esquinas",
		Nombre:       "Cuatro Esquinas",
		Descripcion:  "Carton 5x5 de 75 bolillas: completa las cuatro esquinas para ganar x9; el carton lleno paga x25.",
		Bolillas:     75,
		Filas:        5,
		Columnas:     5,
		CentroLibre:  true,
		Diagonal:     true,
		TopeCantadas: BingoCantadas,
		MaxCartones:  BingoMaxCartones,
		Pagos: []BingoPago{
			{Nombre: "Carton lleno", Mult: 25},
			{Nombre: "Cuatro esquinas", Mult: 9},
		},
		Evalua: func(m [][]bool, _ int, lleno bool) (int64, string) {
			if lleno {
				return 25, "Carton lleno"
			}
			if bingoEsquinas(m) {
				return 9, "Cuatro esquinas"
			}
			return 0, ""
		},
	},
	"bingo-diamante": {
		Codigo:       "bingo-diamante",
		Nombre:       "Patron Diamante",
		Descripcion:  "Carton 5x5 de 75 bolillas: completa el diamante (centro libre y sus cuatro vecinos) para ganar x9; el carton lleno paga x25.",
		Bolillas:     75,
		Filas:        5,
		Columnas:     5,
		CentroLibre:  true,
		Diagonal:     true,
		TopeCantadas: BingoCantadas,
		MaxCartones:  BingoMaxCartones,
		Pagos: []BingoPago{
			{Nombre: "Carton lleno", Mult: 25},
			{Nombre: "Diamante", Mult: 9},
		},
		Evalua: func(m [][]bool, _ int, lleno bool) (int64, string) {
			if lleno {
				return 25, "Carton lleno"
			}
			if bingoDiamante(m) {
				return 9, "Diamante"
			}
			return 0, ""
		},
	},
	"bingo-relampago": {
		Codigo:       "bingo-relampago",
		Nombre:       "Bingo Relampago",
		Descripcion:  "Carton 5x5 de 75 bolillas con solo 20 cantadas: paga un fijo segun tus aciertos (6-8 x1, 9 x2, 10 x3, 11 x4, 12+ x5).",
		Bolillas:     75,
		Filas:        5,
		Columnas:     5,
		CentroLibre:  true,
		Diagonal:     true,
		TopeCantadas: 20,
		MaxCartones:  BingoMaxCartones,
		Pagos: []BingoPago{
			{Nombre: "12 o mas aciertos", Mult: 5},
			{Nombre: "11 aciertos", Mult: 4},
			{Nombre: "10 aciertos", Mult: 3},
			{Nombre: "9 aciertos", Mult: 2},
			{Nombre: "6 a 8 aciertos", Mult: 1},
		},
		Evalua: func(m [][]bool, _ int, lleno bool) (int64, string) {
			if lleno {
				return 5, "Carton lleno"
			}
			aciertos := bingoAciertos(m) - 1 // el centro libre no cuenta
			switch {
			case aciertos >= 12:
				return 5, "12 o mas aciertos"
			case aciertos == 11:
				return 4, "11 aciertos"
			case aciertos == 10:
				return 3, "10 aciertos"
			case aciertos == 9:
				return 2, "9 aciertos"
			case aciertos >= 6:
				return 1, "6 a 8 aciertos"
			}
			return 0, ""
		},
	},
}

// bingoEsquinas indica si las cuatro esquinas estan marcadas.
func bingoEsquinas(m [][]bool) bool {
	filas := len(m)
	cols := len(m[0])
	return m[0][0] && m[0][cols-1] && m[filas-1][0] && m[filas-1][cols-1]
}

// bingoDiamante indica si el centro y sus cuatro vecinos ortogonales estan
// marcados. En el carton 5x5 el centro es libre, asi que alcanza con los cuatro
// vecinos.
func bingoDiamante(m [][]bool) bool {
	filas := len(m)
	cols := len(m[0])
	cm := filas / 2
	cc := cols / 2
	if cm == 0 || cc == 0 || cm+1 >= filas || cc+1 >= cols {
		return false
	}
	return m[cm][cc] && m[cm-1][cc] && m[cm][cc-1] && m[cm][cc+1] && m[cm+1][cc]
}

// bingoAciertos cuenta las casillas marcadas.
func bingoAciertos(m [][]bool) int {
	n := 0
	for _, fila := range m {
		for _, v := range fila {
			if v {
				n++
			}
		}
	}
	return n
}

// BingoSorteo devuelve el orden de las 75 bolillas y los cartones del jugador
// para la variante clasica. Todo se deriva de las semillas y el nonce: misma
// semilla y nonce producen el mismo orden y los mismos cartones.
func BingoSorteo(clientSeed, serverSeed string, nonce int64, cartones int) ([]int, [][][]int) {
	return BingoSorteoVariante(clientSeed, serverSeed, nonce, cartones, "bingo")
}

// BingoSorteoVariante es BingoSorteo para una variante cualquiera. Si el codigo
// no existe cae a la variante clasica.
func BingoSorteoVariante(clientSeed, serverSeed string, nonce int64, cartones int, codigo string) ([]int, [][][]int) {
	v, ok := bingoVariantePorCodigo(codigo)
	if !ok {
		v = bingoVariantes["bingo"]
	}
	if cartones < 1 {
		cartones = 1
	}
	if cartones > v.MaxCartones {
		cartones = v.MaxCartones
	}
	// Los draws salen de un generador HMAC perezoso: el orden usa los primeros
	// draws del stream y los cartones consumen lo que necesitan, sin pedir de mas.
	draws := nuevoBingoDraws(serverSeed, clientSeed, nonce)
	next := func() int { return draws.next() }

	orden := make([]int, v.Bolillas)
	for i := range orden {
		orden[i] = i + 1
	}
	for i := len(orden) - 1; i > 0; i-- {
		j := next() % (i + 1)
		orden[i], orden[j] = orden[j], orden[i]
	}

	grids := make([][][]int, cartones)
	for n := 0; n < cartones; n++ {
		if v.Bolillas == 90 {
			grids[n] = bingoCarton90(next)
		} else {
			grids[n] = bingoCarton(next)
		}
	}
	return orden, grids
}

// bingoShuffle mezcla un slice de enteros usando los draws.
func bingoShuffle(n int, next func() int) []int {
	a := make([]int, n)
	for i := range a {
		a[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := next() % (i + 1)
		a[i], a[j] = a[j], a[i]
	}
	return a
}

// rangoColumna90 devuelve los nmeros de la columna del ticket 3x9: 1-9,
// 10-19, ... , 80-90.
func rangoColumna90(c int) []int {
	var inicio, fin int
	switch {
	case c == 0:
		inicio, fin = 1, 9
	case c == 8:
		inicio, fin = 80, 90
	default:
		inicio, fin = c*10, c*10+9
	}
	out := make([]int, 0, fin-inicio+1)
	for n := inicio; n <= fin; n++ {
		out = append(out, n)
	}
	return out
}

// bingoDraws entrega los draws HMAC de a trozos. El primer trozo conserva el
// stream de Sortear con el nonce de la ronda; si el carton necesita mas, el
// trozo siguiente usa nonce+1 y asi, siempre de forma determinista.
type bingoDraws struct {
	servidor string
	cliente  string
	nonce    int64
	extra    int64
	chunk    int
	buf      []int
	k        int
}

func nuevoBingoDraws(servidor, cliente string, nonce int64) *bingoDraws {
	return &bingoDraws{servidor: servidor, cliente: cliente, nonce: nonce, chunk: 256}
}

func (d *bingoDraws) next() int {
	if d.k >= len(d.buf) {
		d.buf = Sortear(d.servidor, d.cliente, d.nonce+d.extra, d.chunk, 1<<31-1)
		d.extra++
		d.k = 0
		if d.chunk < 4096 {
			d.chunk *= 2
		}
	}
	v := d.buf[d.k]
	d.k++
	return v
}

// bingoCarton90 arma un ticket 3x9: cada columna tiene entre 1 y 3 nmeros, cada
// fila tiene exactamente 5 y no se repite ninguno.
func bingoCarton90(next func() int) [][]int {
	// 1) cuantas casillas ocupa cada columna (1..3, suma 15).
	cuentas := make([]int, 9)
	for i := range cuentas {
		cuentas[i] = 1
	}
	inicio := next() % 9
	extra := 6
	for intentos := 0; extra > 0 && intentos < 64; intentos++ {
		c := (inicio + intentos) % 9
		if cuentas[c] < 3 {
			cuentas[c]++
			extra--
		}
	}
	// 2) matriz 3x9 con sumas de columna cuentas y sumas de fila 5: se reparten
	// las casillas de cada columna en las filas menos cargadas.
	cols := bingoShuffle(9, next)
	bin := make([][]bool, 3)
	for r := range bin {
		bin[r] = make([]bool, 9)
	}
	carga := []int{0, 0, 0}
	for _, c := range cols {
		filas := []int{0, 1, 2}
		for i := 0; i < 3; i++ {
			for j := i + 1; j < 3; j++ {
				if carga[filas[j]] < carga[filas[i]] {
					filas[i], filas[j] = filas[j], filas[i]
				}
			}
		}
		puestas := 0
		for _, r := range filas {
			if puestas >= cuentas[c] {
				break
			}
			bin[r][c] = true
			carga[r]++
			puestas++
		}
	}
	if !bingoTicket90Valido(bin) {
		bin = bingoTicket90Fallback()
	}
	// 3) numeros por columna, sin repetir.
	grid := make([][]int, 3)
	for r := range grid {
		grid[r] = make([]int, 9)
	}
	for c := 0; c < 9; c++ {
		rango := rangoColumna90(c)
		for i := len(rango) - 1; i > 0; i-- {
			j := next() % (i + 1)
			rango[i], rango[j] = rango[j], rango[i]
		}
		k := 0
		for r := 0; r < 3; r++ {
			if bin[r][c] {
				grid[r][c] = rango[k]
				k++
			}
		}
	}
	return grid
}

// bingoTicket90Valido revisa que el ticket 3x9 tenga 5 numeros por fila y entre
// 1 y 3 por columna.
func bingoTicket90Valido(bin [][]bool) bool {
	if len(bin) != 3 {
		return false
	}
	for _, fila := range bin {
		if len(fila) != 9 {
			return false
		}
		n := 0
		for _, v := range fila {
			if v {
				n++
			}
		}
		if n != 5 {
			return false
		}
	}
	for c := 0; c < 9; c++ {
		n := 0
		for r := 0; r < 3; r++ {
			if bin[r][c] {
				n++
			}
		}
		if n < 1 || n > 3 {
			return false
		}
	}
	return true
}

// bingoTicket90Fallback es un ticket valido fijo para el caso (no deberia
// ocurrir) en que el reparto aleatorio no cierre las sumas.
func bingoTicket90Fallback() [][]bool {
	// Columnas: 3,3,2,2,1,1,1,1,1 (suma 15); filas: 5 cada una.
	plan := [][]int{
		{0, 1, 2},
		{0, 1, 2},
		{0, 1},
		{0, 1},
		{2},
		{2},
		{0},
		{1},
		{2},
	}
	bin := make([][]bool, 3)
	for r := range bin {
		bin[r] = make([]bool, 9)
	}
	for c, filas := range plan {
		for _, r := range filas {
			bin[r][c] = true
		}
	}
	return bin
}

// bingoCarton arma un carton 5x5: cada columna elige sus nmeros de su decena y
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

// bingoMarcadas devuelve las casillas marcadas del carton segun las bolillas
// cantadas. El centro libre siempre queda marcado.
func bingoMarcadas(grid [][]int, cantadas []int, v BingoVariante) [][]bool {
	m := make([][]bool, v.Filas)
	for r := range m {
		m[r] = make([]bool, v.Columnas)
	}
	vistos := make(map[int]bool, len(cantadas))
	for _, b := range cantadas {
		vistos[b] = true
	}
	cm := v.Filas / 2
	cc := v.Columnas / 2
	for r := 0; r < v.Filas; r++ {
		for c := 0; c < v.Columnas; c++ {
			if v.CentroLibre && r == cm && c == cc {
				m[r][c] = true
				continue
			}
			m[r][c] = vistos[grid[r][c]]
		}
	}
	return m
}

// BingoMarcadas es la version 5x5 clasica.
func BingoMarcadas(grid [][]int, cantadas []int) [5][5]bool {
	var out [5][5]bool
	m := bingoMarcadas(grid, cantadas, bingoVariantes["bingo"])
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			out[r][c] = m[r][c]
		}
	}
	return out
}

// bingoLineas cuenta las lineas completas. En el tablero cuadrado se suman
// filas, columnas y diagonales; en el ticket 3x9 solo las filas (las columnas
// tienen 1 a 3 casillas y no son linea). Las celdas vacias del ticket no cuentan.
func bingoLineas(m [][]bool, v BingoVariante, grid [][]int) int {
	lineas := 0
	for r := 0; r < v.Filas; r++ {
		ok := true
		alguna := false
		for c := 0; c < v.Columnas; c++ {
			if grid != nil && grid[r][c] == 0 {
				continue
			}
			alguna = true
			if !m[r][c] {
				ok = false
				break
			}
		}
		if alguna && ok {
			lineas++
		}
	}
	if v.Filas != v.Columnas {
		return lineas
	}
	for c := 0; c < v.Columnas; c++ {
		ok := true
		alguna := false
		for r := 0; r < v.Filas; r++ {
			if grid != nil && grid[r][c] == 0 {
				continue
			}
			alguna = true
			if !m[r][c] {
				ok = false
				break
			}
		}
		if alguna && ok {
			lineas++
		}
	}
	if v.Diagonal {
		d1, d2 := true, true
		for i := 0; i < v.Filas; i++ {
			if !(grid != nil && grid[i][i] == 0) && !m[i][i] {
				d1 = false
			}
			if !(grid != nil && grid[i][v.Columnas-1-i] == 0) && !m[i][v.Columnas-1-i] {
				d2 = false
			}
		}
		if d1 {
			lineas++
		}
		if d2 {
			lineas++
		}
	}
	return lineas
}

// BingoLineas cuenta las lneas completas del carton 5x5: las 5 filas, las 5
// columnas y las dos diagonales (el centro libre cuenta como marcado).
func BingoLineas(m [5][5]bool) int {
	b := make([][]bool, 5)
	for r := 0; r < 5; r++ {
		b[r] = make([]bool, 5)
		for c := 0; c < 5; c++ {
			b[r][c] = m[r][c]
		}
	}
	return bingoLineas(b, bingoVariantes["bingo"], nil)
}

// bingoLleno indica si el carton quedó completo (bingo). Las celdas vacías del
// ticket 90 no cuentan.
func bingoLleno(m [][]bool, grid [][]int) bool {
	for r, fila := range m {
		for c, v := range fila {
			if grid != nil && grid[r][c] == 0 {
				continue
			}
			if !v {
				return false
			}
		}
	}
	return true
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

// BingoPremio calcula el premio de la ronda para la variante clasica.
func BingoPremio(monto int64, grids [][][]int, cantadas []int) (premio int64, lineas int, lleno bool, tiers []int64) {
	premio, lineas, lleno, tiers, _ = BingoPremioVariante(monto, grids, cantadas, "bingo")
	return
}

// BingoPremioVariante calcula el premio de la ronda para una variante. Cada
// carton es una apuesta independiente de `monto`: paga su propio multiplicador
// y los premios se suman. Devuelve el premio total, la mayor cantidad de lineas
// en un carton, si algun carton se lleno, el multiplicador de cada carton y el
// nombre del mejor patron que pago.
func BingoPremioVariante(monto int64, grids [][][]int, cantadas []int, codigo string) (premio int64, lineas int, lleno bool, tiers []int64, patron string) {
	v, ok := bingoVariantePorCodigo(codigo)
	if !ok {
		v = bingoVariantes["bingo"]
	}
	mejor := int64(0)
	for _, g := range grids {
		m := bingoMarcadas(g, cantadas, v)
		l := bingoLineas(m, v, g)
		full := bingoLleno(m, g)
		mult, nombre := v.Evalua(m, l, full)
		if l > lineas {
			lineas = l
		}
		if full {
			lleno = true
		}
		if mult > mejor {
			mejor = mult
			patron = nombre
		}
		tiers = append(tiers, mult)
		premio += monto * mult
	}
	return
}

// BingoMultMaximo devuelve el peor multiplicador posible de una variante: el
// mayor pago por carton por la cantidad maxima de cartones.
func BingoMultMaximo(codigo string) int64 {
	v, ok := bingoVariantePorCodigo(codigo)
	if !ok {
		return 0
	}
	var max int64
	for _, p := range v.Pagos {
		if p.Mult > max {
			max = p.Mult
		}
	}
	if max == 0 {
		return 0
	}
	return max * int64(v.MaxCartones)
}

// RTPBingo mide el retorno al jugador de la variante clasica por simulacion.
func RTPBingo(iteraciones int) float64 {
	return RTPBingoVariante("bingo", iteraciones)
}

// RTPBingoVariante mide el retorno al jugador por simulacion, cantando el tope
// de bolillas en un solo carton (el RTP no depende de la cantidad de cartones:
// cada uno es una apuesta independiente).
func RTPBingoVariante(codigo string, iteraciones int) float64 {
	v, ok := bingoVariantePorCodigo(codigo)
	if !ok {
		return 0
	}
	if iteraciones <= 0 {
		iteraciones = 200000
	}
	var suma float64
	for i := 0; i < iteraciones; i++ {
		seed := fmt.Sprintf("sim-%s-%d", codigo, i)
		orden, grids := BingoSorteoVariante("sim", seed, int64(i), 1, codigo)
		premio, _, _, _, _ := BingoPremioVariante(1000, grids, orden[:v.TopeCantadas], codigo)
		suma += float64(premio) / 1000
	}
	return suma / float64(iteraciones) * 100
}
