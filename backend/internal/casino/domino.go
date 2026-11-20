package casino

import (
	"fmt"
	"sort"
)

// Dominó de mano contra la casa. Se reparten 7 fichas dobles a cada lado y se
// juega por turnos: el jugador elige una ficha que encaje en los extremos de la
// cadena y la casa responde con la ficha valida de mayor puntaje. Gana quien
// se queda sin fichas; si ambos pasan dos veces, gana la mano de menor suma.
//
// Todo se deriva de las semillas (provably fair): el reparto sale de
// HMAC-SHA256(server_seed, "cliente:nonce:i"). Las jugadas de la casa son
// deterministicas, asi que el resultado se puede reproducir completo.

const (
	// DominoFichasPorMano es la cantidad de fichas que recibe cada lado.
	DominoFichasPorMano = 7
	// DominoMultVictoria es lo que paga la casa cuando el jugador gana. Un
	// empate devuelve lo apostado.
	DominoMultVictoria = 1.52
)

// DominoFicha es una ficha doble: [0] y [1] son los puntos.
type DominoFicha [2]int

// Puntos devuelve la suma de puntos de la ficha.
func (f DominoFicha) Puntos() int { return f[0] + f[1] }

// DominoSorteo reparte las 28 fichas dobles entre el jugador y la casa. La
// misma semilla y nonce producen el mismo reparto.
func DominoSorteo(clientSeed, serverSeed string, nonce int64) (mano, casa []DominoFicha) {
	draws := Sortear(serverSeed, clientSeed, nonce, 28, 1<<30)
	orden := make([]int, 28)
	for i := range orden {
		orden[i] = i
	}
	sort.Slice(orden, func(a, b int) bool {
		if draws[orden[a]] != draws[orden[b]] {
			return draws[orden[a]] < draws[orden[b]]
		}
		return orden[a] < orden[b]
	})
	todas := make([]DominoFicha, 28)
	for i := 0; i < 28; i++ {
		todas[i] = DominoFicha{i / 7, i % 7}
	}
	mano = make([]DominoFicha, 0, DominoFichasPorMano)
	casa = make([]DominoFicha, 0, DominoFichasPorMano)
	for i := 0; i < DominoFichasPorMano*2; i++ {
		if i < DominoFichasPorMano {
			mano = append(mano, todas[orden[i]])
		} else {
			casa = append(casa, todas[orden[i]])
		}
	}
	return mano, casa
}

// DominoEstado es el estado de una mano de dominó.
type DominoEstado struct {
	Mano    []DominoFicha
	Casa    []DominoFicha
	Tablero []DominoFicha // ordenadas de izquierda a derecha
	Izq     int
	Der     int
	Turno   string // "jugador" | "casa"
	Pases   int
	Fin     bool
	Ganador string // "jugador" | "casa" | "empate"
}

// NuevoDominoEstado arma el estado inicial de una mano.
func NuevoDominoEstado(mano, casa []DominoFicha) *DominoEstado {
	return &DominoEstado{
		Mano:  append([]DominoFicha{}, mano...),
		Casa:  append([]DominoFicha{}, casa...),
		Turno: "jugador",
	}
}

// NuevoDominoEstadoDesde reconstruye el estado de una mano guardada.
func NuevoDominoEstadoDesde(mano, casa, tablero []DominoFicha, turno string) *DominoEstado {
	est := &DominoEstado{
		Mano:    append([]DominoFicha{}, mano...),
		Casa:    append([]DominoFicha{}, casa...),
		Tablero: append([]DominoFicha{}, tablero...),
		Turno:   turno,
	}
	if len(est.Tablero) > 0 {
		est.Izq = est.Tablero[0][0]
		est.Der = est.Tablero[len(est.Tablero)-1][1]
	}
	return est
}

// DominoEsValida indica si la ficha encaja en alguno de los extremos.
func DominoEsValida(f DominoFicha, izq, der int, vacio bool) bool {
	if vacio {
		return true
	}
	return f[0] == izq || f[1] == izq || f[0] == der || f[1] == der
}

// DominoValidas devuelve los indices de las fichas jugables de una mano.
func DominoValidas(mano []DominoFicha, izq, der int, vacio bool) []int {
	var out []int
	for i, f := range mano {
		if DominoEsValida(f, izq, der, vacio) {
			out = append(out, i)
		}
	}
	return out
}

// dominoColocar mete la ficha en la cadena, orientada segun el extremo.
func dominoColocar(est *DominoEstado, f DominoFicha) {
	vacio := len(est.Tablero) == 0
	if vacio {
		est.Tablero = append(est.Tablero, f)
		est.Izq, est.Der = f[0], f[1]
		return
	}
	// Si encaja con la izquierda, se antepone.
	if f[0] == est.Izq || f[1] == est.Izq {
		if f[0] == est.Izq {
			est.Tablero = append([]DominoFicha{{f[1], f[0]}}, est.Tablero...)
			est.Izq = f[1]
		} else {
			est.Tablero = append([]DominoFicha{f}, est.Tablero...)
			est.Izq = f[0]
		}
		return
	}
	// Si no, encaja con la derecha.
	if f[0] == est.Der {
		est.Tablero = append(est.Tablero, f)
		est.Der = f[1]
	} else {
		est.Tablero = append(est.Tablero, DominoFicha{f[1], f[0]})
		est.Der = f[0]
	}
}

func dominoSuma(mano []DominoFicha) int {
	t := 0
	for _, f := range mano {
		t += f.Puntos()
	}
	return t
}

// dominoFinSiBloqueado cierra la mano cuando ambos lados pasaron dos veces.
func dominoFinSiBloqueado(est *DominoEstado) {
	if est.Pases < 2 {
		return
	}
	est.Fin = true
	sj, sc := dominoSuma(est.Mano), dominoSuma(est.Casa)
	switch {
	case sj < sc:
		est.Ganador = "jugador"
	case sc < sj:
		est.Ganador = "casa"
	default:
		est.Ganador = "empate"
	}
}

// DominoJugadorJuega aplica la ficha elegida por el jugador. ficha == -1 es
// pasar (solo valido si no tiene ninguna ficha jugable).
func DominoJugadorJuega(est *DominoEstado, ficha int) error {
	if est.Fin {
		return fmt.Errorf("la jugada ya termino")
	}
	if est.Turno != "jugador" {
		return fmt.Errorf("no es tu turno")
	}
	vacio := len(est.Tablero) == 0
	if ficha == -1 {
		if len(DominoValidas(est.Mano, est.Izq, est.Der, vacio)) > 0 {
			return fmt.Errorf("tenes una ficha para jugar")
		}
		est.Pases++
		est.Turno = "casa"
		dominoFinSiBloqueado(est)
		return nil
	}
	if ficha < 0 || ficha >= len(est.Mano) {
		return fmt.Errorf("ficha invalida")
	}
	if !DominoEsValida(est.Mano[ficha], est.Izq, est.Der, vacio) {
		return fmt.Errorf("esa ficha no encaja")
	}
	f := est.Mano[ficha]
	est.Mano = append(est.Mano[:ficha], est.Mano[ficha+1:]...)
	dominoColocar(est, f)
	est.Pases = 0
	est.Turno = "casa"
	if len(est.Mano) == 0 {
		est.Fin = true
		est.Ganador = "jugador"
	}
	return nil
}

// DominoCasaJuega responde con la ficha valida de mayor puntaje. Si no tiene
// ninguna, pasa.
func DominoCasaJuega(est *DominoEstado) {
	if est.Fin || est.Turno != "casa" {
		return
	}
	vacio := len(est.Tablero) == 0
	validas := DominoValidas(est.Casa, est.Izq, est.Der, vacio)
	if len(validas) == 0 {
		est.Pases++
		est.Turno = "jugador"
		dominoFinSiBloqueado(est)
		return
	}
	mejor := validas[0]
	for _, i := range validas[1:] {
		if est.Casa[i].Puntos() > est.Casa[mejor].Puntos() {
			mejor = i
		}
	}
	f := est.Casa[mejor]
	est.Casa = append(est.Casa[:mejor], est.Casa[mejor+1:]...)
	dominoColocar(est, f)
	est.Pases = 0
	est.Turno = "jugador"
	if len(est.Casa) == 0 {
		est.Fin = true
		est.Ganador = "casa"
	}
}

// DominoGanador resuelve el ganador de una mano terminada.
func DominoGanador(mano, casa []DominoFicha) string {
	if len(mano) == 0 {
		return "jugador"
	}
	if len(casa) == 0 {
		return "casa"
	}
	sj, sc := dominoSuma(mano), dominoSuma(casa)
	switch {
	case sj < sc:
		return "jugador"
	case sc < sj:
		return "casa"
	default:
		return "empate"
	}
}

// DominoRepetir reproduce una mano a partir de las jugadas del jugador.
func DominoRepetir(clientSeed, serverSeed string, nonce int64, jugadas [][2]int) (*DominoEstado, error) {
	mano, casa := DominoSorteo(clientSeed, serverSeed, nonce)
	est := NuevoDominoEstado(mano, casa)
	for _, j := range jugadas {
		if err := DominoJugadorJuega(est, j[0]); err != nil {
			return nil, err
		}
		if est.Fin {
			break
		}
		DominoCasaJuega(est)
	}
	return est, nil
}

// DominoMultiplicador devuelve el multiplicador segun el ganador.
func DominoMultiplicador(ganador string) float64 {
	switch ganador {
	case "jugador":
		return DominoMultVictoria
	case "empate":
		return 1
	default:
		return 0
	}
}

// DominoPremio calcula lo que se paga segun el ganador.
func DominoPremio(monto int64, ganador string) int64 {
	if monto <= 0 {
		return 0
	}
	return int64(float64(monto) * DominoMultiplicador(ganador))
}

// DominoSimular juega una mano completa con una estrategia determinista y
// devuelve el multiplicador obtenido. "mayor" juega la ficha de mayor puntajo
// (como la casa) y "aleatorio" una ficha al azar.
func DominoSimular(clientSeed, serverSeed string, nonce int64, estrategia string) float64 {
	i := 0
	mano, casa := DominoSorteo(clientSeed, serverSeed, nonce)
	est := NuevoDominoEstado(mano, casa)
	for !est.Fin {
		if est.Turno == "jugador" {
			vacio := len(est.Tablero) == 0
			validas := DominoValidas(est.Mano, est.Izq, est.Der, vacio)
			if len(validas) == 0 {
				_ = DominoJugadorJuega(est, -1)
				continue
			}
			elegido := validas[0]
			if estrategia == "mayor" {
				for _, i := range validas[1:] {
					if est.Mano[i].Puntos() > est.Mano[elegido].Puntos() {
						elegido = i
					}
				}
			} else {
				d := Sortear(serverSeed, fmt.Sprintf("%s#%d", clientSeed, i), nonce, 1, 1<<30)[0]
				i++
				elegido = validas[d%len(validas)]
			}
			_ = DominoJugadorJuega(est, elegido)
			continue
		}
		DominoCasaJuega(est)
	}
	return DominoMultiplicador(est.Ganador)
}

// RTPDomino mide el retorno al jugador por simulacion.
func RTPDomino(iteraciones int, estrategia string) float64 {
	if iteraciones <= 0 {
		iteraciones = 200000
	}
	var suma float64
	for i := 0; i < iteraciones; i++ {
		seed := fmt.Sprintf("sim-%d", i)
		suma += DominoSimular("sim", seed, int64(i), estrategia)
	}
	return suma / float64(iteraciones) * 100
}
