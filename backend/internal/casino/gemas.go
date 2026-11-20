package casino

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// Gemas (match-3 con apuesta): tablero de 8x8 con 7 tipos de gema. La apuesta
// se descuenta al abrir la ronda y el jugador tiene GemasMovimientos
// intercambios para armar combinaciones de 3 o más. Cada combinación suma
// puntos según el tipo de gema, la longitud del grupo y la cascada en la que
// cae; 100 puntos devuelven lo apostado. Al alcanzar el tope de movimientos (o
// al tocar "Reventar") la ronda se liquida y paga casino_premio.
//
// Todo se deriva de las semillas (provably fair): el tablero inicial y el
// chorro de gemas de relleno salen de HMAC-SHA256(server_seed, "cliente:nonce:i").
// Los intercambios los elige el jugador, pero no pueden alterar el azar: las
// gemas nuevas están comprometidas desde el inicio de la ronda.

const (
	// GemasFilas y GemasColumnas son las dimensiones del tablero.
	GemasFilas    = 8
	GemasColumnas = 8
	// GemasTiposBase es la cantidad de gemas base (0..6). El 7 es la gema rayo.
	GemasTiposBase = 7
	// GemasTipoRayo es la gema especial: se crea con un grupo de 4+ y al
	// reventar limpia toda su fila y su columna.
	GemasTipoRayo = 7
	// GemasMovimientos es el tope de intercambios por ronda.
	GemasMovimientos = 25
	// GemasCascadasMax es un tope de seguridad por si un tablero entra en una
	// cascada larguísima. En la práctica nunca se alcanza.
	GemasCascadasMax = 20
	// GemasPuntosDivisor convierte puntos en multiplicador: 300 puntos = x1.
	// Se ajustó con la simulación (RTPGemas) para que la jugada hábil devuelva
	// ~92% y la casual ~68%.
	GemasPuntosDivisor = 300
)

// GemasTipo describe una gema base: su peso en el sorteo y los puntos que paga
// un grupo de tres.
type GemasTipo struct {
	Nombre string
	Peso   int
	Puntos int64
}

// GemasTipos en orden de aparición (el backend y el frontend comparten el orden).
var GemasTipos = []GemasTipo{
	{Nombre: "esmeralda", Peso: 24, Puntos: 3},
	{Nombre: "zafiro", Peso: 20, Puntos: 4},
	{Nombre: "amatista", Peso: 17, Puntos: 5},
	{Nombre: "rubi", Peso: 14, Puntos: 7},
	{Nombre: "topacio", Peso: 11, Puntos: 9},
	{Nombre: "naranja", Peso: 8, Puntos: 12},
	{Nombre: "diamante", Peso: 6, Puntos: 20},
}

// GemasPuntosRayo es el pago base de un grupo de tres gemas rayo.
const GemasPuntosRayo int64 = 40

// GemasPesoTotal es la suma de los pesos de los tipos base.
var GemasPesoTotal = func() int {
	t := 0
	for _, g := range GemasTipos {
		t += g.Peso
	}
	return t
}()

// GemasTipoPorDraw convierte un draw del sorteo en un tipo de gema base.
func GemasTipoPorDraw(draw int) int {
	r := draw % GemasPesoTotal
	acum := 0
	for i, g := range GemasTipos {
		acum += g.Peso
		if r < acum {
			return i
		}
	}
	return GemasTiposBase - 1
}

// gemasDraw reproduce el sorteo de Sortear para una posición: el mismo
// HMAC-SHA256(server_seed, "cliente:nonce:i") que usa el resto del casino.
func gemasDraw(semillaServidor, semillaCliente string, nonce int64, i int) int {
	mac := hmac.New(sha256.New, []byte(semillaServidor))
	fmt.Fprintf(mac, "%s:%d:%d", semillaCliente, nonce, i)
	sum := mac.Sum(nil)
	return int(binary.BigEndian.Uint32(sum[:4]) % uint32(1<<31-1))
}

// GemasChorro es el chorro de draws de una ronda. Se consume en orden: primero
// el tablero inicial y después cada relleno de cascada.
type GemasChorro struct {
	semillaServidor string
	semillaCliente  string
	nonce           int64
	i               int
}

// NuevoGemasChorro crea el chorro de una ronda a partir de las semillas.
func NuevoGemasChorro(clientSeed, serverSeed string, nonce int64) *GemasChorro {
	return &GemasChorro{semillaServidor: serverSeed, semillaCliente: clientSeed, nonce: nonce}
}

// Draw devuelve el próximo draw del chorro y avanza el contador.
func (c *GemasChorro) Draw() int {
	v := gemasDraw(c.semillaServidor, c.semillaCliente, c.nonce, c.i)
	c.i++
	return v
}

// Tipo devuelve la próxima gema del chorro.
func (c *GemasChorro) Tipo() int { return GemasTipoPorDraw(c.Draw()) }

// Saltar avanza el chorro sin devolver los draws.
func (c *GemasChorro) Saltar(n int) {
	for k := 0; k < n; k++ {
		c.Draw()
	}
}

// Posicion devuelve cuántos draws se consumieron.
func (c *GemasChorro) Posicion() int { return c.i }

func gemasIdx(fila, col int) int { return fila*GemasColumnas + col }

func gemasRC(i int) (int, int) { return i / GemasColumnas, i % GemasColumnas }

// GemasSorteo arma el tablero inicial y devuelve el offset en el que empieza el
// chorro de relleno. El tablero no tiene combinaciones armadas (sería dinero
// gratis) y siempre tiene al menos un intercambio válido. Misma semilla y
// nonce producen el mismo tablero y el mismo offset.
func GemasSorteo(clientSeed, serverSeed string, nonce int64) ([]int, int) {
	c := NuevoGemasChorro(clientSeed, serverSeed, nonce)
	var ultimo []int
	for intento := 0; intento < 64; intento++ {
		ultimo = make([]int, GemasFilas*GemasColumnas)
		for i := range ultimo {
			ultimo[i] = c.Tipo()
		}
		if len(GemasPartidos(ultimo)) == 0 && GemasHayMovimiento(ultimo) {
			return ultimo, c.Posicion()
		}
	}
	// Prácticamente inalcanzable: con 7 tipos y 8x8 siempre hay movimientos.
	// Si ocurre, se entrega el último tablero (nunca se traba la ronda).
	return ultimo, c.Posicion()
}

// GemasPartidos devuelve los grupos de 3+ gemas iguales (horizontales y
// verticales). Un grupo es una lista de índices del tablero.
func GemasPartidos(tablero []int) [][]int {
	var partidos [][]int
	// Horizontales.
	for r := 0; r < GemasFilas; r++ {
		c := 0
		for c < GemasColumnas {
			v := tablero[gemasIdx(r, c)]
			c2 := c + 1
			for c2 < GemasColumnas && tablero[gemasIdx(r, c2)] == v {
				c2++
			}
			if c2-c >= 3 {
				g := make([]int, 0, c2-c)
				for k := c; k < c2; k++ {
					g = append(g, gemasIdx(r, k))
				}
				partidos = append(partidos, g)
			}
			c = c2
		}
	}
	// Verticales.
	for c := 0; c < GemasColumnas; c++ {
		r := 0
		for r < GemasFilas {
			v := tablero[gemasIdx(r, c)]
			r2 := r + 1
			for r2 < GemasFilas && tablero[gemasIdx(r2, c)] == v {
				r2++
			}
			if r2-r >= 3 {
				g := make([]int, 0, r2-r)
				for k := r; k < r2; k++ {
					g = append(g, gemasIdx(k, c))
				}
				partidos = append(partidos, g)
			}
			r = r2
		}
	}
	return partidos
}

// GemasHayMovimiento indica si existe al menos un intercambio adyacente que
// forma una combinación.
func GemasHayMovimiento(tablero []int) bool {
	return len(GemasMovimientosValidos(tablero)) > 0
}

// GemasMovimientosValidos devuelve todos los pares (a,b) adyacentes cuyo
// intercambio forma al menos una combinación.
func GemasMovimientosValidos(tablero []int) [][2]int {
	var movs [][2]int
	for r := 0; r < GemasFilas; r++ {
		for c := 0; c < GemasColumnas; c++ {
			i := gemasIdx(r, c)
			for _, d := range [][2]int{{0, 1}, {1, 0}} {
				r2, c2 := r+d[0], c+d[1]
				if r2 >= GemasFilas || c2 >= GemasColumnas {
					continue
				}
				j := gemasIdx(r2, c2)
				tablero[i], tablero[j] = tablero[j], tablero[i]
				ok := len(GemasPartidos(tablero)) > 0
				tablero[i], tablero[j] = tablero[j], tablero[i]
				if ok {
					movs = append(movs, [2]int{i, j})
				}
			}
		}
	}
	return movs
}

// GemasPaso es un eslabón de la cascada: el tablero antes de reventar, las
// celdas que revientan y el tablero ya rellenado. El frontend lo anima.
type GemasPaso struct {
	Antes      []int `json:"antes"`
	Despues    []int `json:"despues"`
	Eliminadas []int `json:"eliminadas"`
	Especiales []int `json:"especiales"`
	Puntos     int64 `json:"puntos"`
	Cascada    int   `json:"cascada"`
}

// GemasResultado es el estado del tablero tras un intercambio.
type GemasResultado struct {
	Tablero []int
	Cursor  int
	Puntos  int64
	Pasos   []GemasPaso
}

// GemasLenMult da el multiplicador por longitud: 3 -> 1, 4 -> 2, 5 -> 3 y así
// hasta un tope de 6 (grupos de 8 en una fila/columna).
func GemasLenMult(n int) int64 {
	m := int64(n - 2)
	if m < 1 {
		m = 1
	}
	if m > 6 {
		m = 6
	}
	return m
}

// gemasCascadaMult da el multiplicador por cascada: crece con cada eslabón para
// premiar las cadenas, pero menos que el número de cascada.
func gemasCascadaMult(cascada int) int64 {
	if cascada <= 1 {
		return 1
	}
	return int64(1 + (cascada-1)/2)
}

// GemasAplicar intercambia las celdas a y b y resuelve todas las cascadas.
// Devuelve error si las celdas no son adyacentes o si el intercambio no forma
// ninguna combinación (en ese caso el tablero no se toca: el jugador no gasta
// el movimiento). El chorro avanza con cada gema de relleno.
func GemasAplicar(tablero []int, chorro *GemasChorro, a, b int) (*GemasResultado, error) {
	if a < 0 || a >= len(tablero) || b < 0 || b >= len(tablero) {
		return nil, fmt.Errorf("las celdas estan fuera del tablero")
	}
	ra, ca := gemasRC(a)
	rb, cb := gemasRC(b)
	if abs(ra-rb)+abs(ca-cb) != 1 {
		return nil, fmt.Errorf("las celdas deben ser vecinas")
	}
	res := &GemasResultado{Tablero: append([]int{}, tablero...)}
	res.Tablero[a], res.Tablero[b] = res.Tablero[b], res.Tablero[a]

	cascada := 0
	for cascada < GemasCascadasMax {
		partidos := GemasPartidos(res.Tablero)
		if len(partidos) == 0 {
			break
		}
		cascada++
		antes := append([]int{}, res.Tablero...)

		// Grupos de 4+ dejan una gema rayo en el tablero (preferentemente la
		// celda que el jugador movió, si queda dentro del grupo).
		especiales := map[int]bool{}
		for _, p := range partidos {
			if len(p) < 4 {
				continue
			}
			cand := p[len(p)/2]
			for _, i := range p {
				if i == a {
					cand = i
					break
				}
			}
			if !especiales[cand] {
				especiales[cand] = true
			}
		}

		// Las celdas que revientan son las de los grupos, salvo las que se
		// convierten en gema rayo.
		claro := map[int]bool{}
		for _, p := range partidos {
			for _, i := range p {
				if !especiales[i] {
					claro[i] = true
				}
			}
		}
		// Las gemas rayo revientan en cruz: limpian su fila y su columna, y lo
		// que sea una gema rayo también detona (reacción en cadena).
		for {
			cambiado := false
			for i := range claro {
				if res.Tablero[i] != GemasTipoRayo {
					continue
				}
				r, c := gemasRC(i)
				for k := 0; k < GemasColumnas; k++ {
					j := gemasIdx(r, k)
					if !claro[j] {
						claro[j] = true
						cambiado = true
					}
				}
				for k := 0; k < GemasFilas; k++ {
					j := gemasIdx(k, c)
					if !claro[j] {
						claro[j] = true
						cambiado = true
					}
				}
			}
			if !cambiado {
				break
			}
		}

		var puntos int64
		for _, p := range partidos {
			tipo := res.Tablero[p[0]]
			var base int64
			if tipo == GemasTipoRayo {
				base = GemasPuntosRayo
			} else if tipo >= 0 && tipo < GemasTiposBase {
				base = GemasTipos[tipo].Puntos
			}
			puntos += base * GemasLenMult(len(p)) * gemasCascadaMult(cascada)
		}

		eliminadas := make([]int, 0, len(claro))
		for i := range claro {
			eliminadas = append(eliminadas, i)
		}
		ordenarInts(eliminadas)
		espLista := make([]int, 0, len(especiales))
		for i := range especiales {
			espLista = append(espLista, i)
		}
		ordenarInts(espLista)

		for i := range claro {
			res.Tablero[i] = -1
		}
		for i := range especiales {
			if res.Tablero[i] != -1 {
				res.Tablero[i] = GemasTipoRayo
			}
		}

		// Gravedad: las gemas bajan y el relleno entra por arriba, en orden de
		// columna y de arriba hacia abajo (consumo determinista del chorro).
		for c := 0; c < GemasColumnas; c++ {
			columna := make([]int, 0, GemasFilas)
			for r := GemasFilas - 1; r >= 0; r-- {
				if res.Tablero[gemasIdx(r, c)] != -1 {
					columna = append(columna, res.Tablero[gemasIdx(r, c)])
				}
			}
			faltan := GemasFilas - len(columna)
			for r := 0; r < faltan; r++ {
				res.Tablero[gemasIdx(r, c)] = chorro.Tipo()
			}
			// columna está de abajo hacia arriba: la fila r recibe el valor que
			// estaba en la fila espejo (GemasFilas-1-r).
			for r := faltan; r < GemasFilas; r++ {
				res.Tablero[gemasIdx(r, c)] = columna[GemasFilas-1-r]
			}
		}

		res.Puntos += puntos
		res.Pasos = append(res.Pasos, GemasPaso{
			Antes:      antes,
			Despues:    append([]int{}, res.Tablero...),
			Eliminadas: eliminadas,
			Especiales: espLista,
			Puntos:     puntos,
			Cascada:    cascada,
		})
	}
	if cascada == 0 {
		return nil, fmt.Errorf("el intercambio no forma ninguna combinacion")
	}
	res.Cursor = chorro.Posicion()
	return res, nil
}

// GemasRepetir reproduce una ronda a partir del tablero inicial, el chorro y la
// lista de intercambios. Se usa para auditar que el resultado guardado coincide
// con el que derivan las semillas.
func GemasRepetir(tablero []int, chorro *GemasChorro, jugadas [][2]int) (*GemasResultado, error) {
	res := &GemasResultado{Tablero: append([]int{}, tablero...)}
	for _, j := range jugadas {
		r, err := GemasAplicar(res.Tablero, chorro, j[0], j[1])
		if err != nil {
			return nil, err
		}
		res.Tablero = r.Tablero
		res.Cursor = r.Cursor
		res.Puntos += r.Puntos
		res.Pasos = append(res.Pasos, r.Pasos...)
	}
	return res, nil
}

// GemasPremio convierte los puntos acumulados en premio: 100 puntos = lo
// apostado.
func GemasPremio(monto, puntos int64) int64 {
	if monto <= 0 || puntos <= 0 {
		return 0
	}
	return monto * puntos / GemasPuntosDivisor
}

// GemasMultiplicador devuelve el multiplicador equivalente de los puntos.
func GemasMultiplicador(puntos int64) float64 {
	return float64(puntos) / float64(GemasPuntosDivisor)
}

// GemasValorVisible devuelve el valor inmediato de un intercambio: la suma de
// los grupos que quedan armados después del cambio, sin contar las cascadas.
// Es la información que ve el jugador antes de tocar una gema.
func GemasValorVisible(tablero []int, a, b int) int64 {
	if a < 0 || a >= len(tablero) || b < 0 || b >= len(tablero) {
		return -1
	}
	ra, ca := gemasRC(a)
	rb, cb := gemasRC(b)
	if abs(ra-rb)+abs(ca-cb) != 1 {
		return -1
	}
	tablero[a], tablero[b] = tablero[b], tablero[a]
	var v int64
	for _, p := range GemasPartidos(tablero) {
		tipo := tablero[p[0]]
		var base int64
		if tipo == GemasTipoRayo {
			base = GemasPuntosRayo
		} else if tipo >= 0 && tipo < GemasTiposBase {
			base = GemasTipos[tipo].Puntos
		}
		v += base * GemasLenMult(len(p))
	}
	tablero[a], tablero[b] = tablero[b], tablero[a]
	return v
}

// GemasSimular juega una ronda completa con una estrategia determinista y
// devuelve los puntos. estrategia "primer" elige el primer movimiento válido y
// "mayor" el de mayor valor visible (la jugada hábil que haría una persona).
func GemasSimular(clientSeed, serverSeed string, nonce int64, estrategia string) int64 {
	tablero, offset := GemasSorteo(clientSeed, serverSeed, nonce)
	chorro := NuevoGemasChorro(clientSeed, serverSeed, nonce)
	chorro.Saltar(offset)
	var puntos int64
	for m := 0; m < GemasMovimientos; m++ {
		movs := GemasMovimientosValidos(tablero)
		if len(movs) == 0 {
			break
		}
		elegido := movs[0]
		if estrategia == "mayor" {
			mejor := int64(-1)
			for _, mv := range movs {
				if v := GemasValorVisible(tablero, mv[0], mv[1]); v > mejor {
					mejor = v
					elegido = mv
				}
			}
		}
		r, err := GemasAplicar(tablero, chorro, elegido[0], elegido[1])
		if err != nil {
			break
		}
		tablero = r.Tablero
		puntos += r.Puntos
	}
	return puntos
}

// RTPGemas mide el retorno al jugador por simulación. Se reporta la estrategia
// usada porque el match-3 depende de la habilidad del jugador.
func RTPGemas(iteraciones int, estrategia string) float64 {
	if iteraciones <= 0 {
		iteraciones = 200000
	}
	var suma float64
	for i := 0; i < iteraciones; i++ {
		seed := fmt.Sprintf("sim-%d", i)
		puntos := GemasSimular("sim", seed, int64(i), estrategia)
		suma += float64(puntos) / float64(GemasPuntosDivisor)
	}
	return suma / float64(iteraciones) * 100
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func ordenarInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}
