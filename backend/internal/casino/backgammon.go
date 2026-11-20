package casino

import "fmt"

// Backgammon simplificado: carrera de tres fichas por lado en un tablero de 12
// puntos. En cada turno se tiran dos dados y se avanza una ficha por la suma; si
// salen dobles, el avance se duplica. No hay ni golpes ni barredura: gana el
// primero que saca las tres fichas. La casa mueve siempre la ficha mas atrasada.
//
// La tirada de dados sale de las semillas (provably fair) y es reproducible.

const (
	// BackgammonFichas es la cantidad de fichas de cada lado.
	BackgammonFichas = 3
	// BackgammonMeta es el punto al que hay que llegar para sacar la ficha.
	BackgammonMeta = 12
	// BackgammonMultVictoria es lo que paga cuando el jugador gana la carrera.
	BackgammonMultVictoria = 1.85
)

// BackgammonEstado es el estado de una carrera.
type BackgammonEstado struct {
	Jugador [BackgammonFichas]int
	Casa    [BackgammonFichas]int
	Turno   string // "jugador" | "casa"
	Dado1   int
	Dado2   int
	Dobles  bool
	Lance   int
	Fin     bool
	Ganador string // "jugador" | "casa"
}

// BackgammonDados devuelve la tirada de un lance (1..6 cada dado).
func BackgammonDados(clientSeed, serverSeed string, nonce int64, lance int) (int, int) {
	d := Sortear(serverSeed, clientSeed, nonce, 2*lance+2, 6)
	return d[2*lance] + 1, d[2*lance+1] + 1
}

// NuevoBackgammon arma el estado inicial. Quien empieza sale del sorteo, asi que
// la casa no siempre toma la primera jugada.
func NuevoBackgammon(clientSeed, serverSeed string, nonce int64) *BackgammonEstado {
	est := &BackgammonEstado{Turno: "jugador"}
	if Sortear(serverSeed, clientSeed, nonce, 1, 2)[0] == 1 {
		est.Turno = "casa"
	}
	est.Dado1, est.Dado2 = BackgammonDados(clientSeed, serverSeed, nonce, 0)
	est.Dobles = est.Dado1 == est.Dado2
	return est
}

func backgammonAvance(est *BackgammonEstado) int {
	a := est.Dado1 + est.Dado2
	if est.Dobles {
		a *= 2
	}
	return a
}

func backgammonFuera(f [BackgammonFichas]int) bool {
	for _, p := range f {
		if p < BackgammonMeta {
			return false
		}
	}
	return true
}

// BackgammonJugadorJuega avanza una ficha del jugador.
func BackgammonJugadorJuega(est *BackgammonEstado, ficha int) error {
	if est.Fin {
		return fmt.Errorf("la jugada ya termino")
	}
	if est.Turno != "jugador" {
		return fmt.Errorf("no es tu turno")
	}
	if ficha < 0 || ficha >= BackgammonFichas {
		return fmt.Errorf("ficha invalida")
	}
	est.Jugador[ficha] += backgammonAvance(est)
	if est.Jugador[ficha] > BackgammonMeta {
		est.Jugador[ficha] = BackgammonMeta
	}
	if backgammonFuera(est.Jugador) {
		est.Fin = true
		est.Ganador = "jugador"
		return nil
	}
	est.Turno = "casa"
	return nil
}

// BackgammonCasaJuega mueve la ficha mas atrasada de la casa.
func BackgammonCasaJuega(est *BackgammonEstado) {
	if est.Fin || est.Turno != "casa" {
		return
	}
	idx := 0
	for i := 1; i < BackgammonFichas; i++ {
		if est.Casa[i] < est.Casa[idx] {
			idx = i
		}
	}
	est.Casa[idx] += backgammonAvance(est)
	if est.Casa[idx] > BackgammonMeta {
		est.Casa[idx] = BackgammonMeta
	}
	if backgammonFuera(est.Casa) {
		est.Fin = true
		est.Ganador = "casa"
		return
	}
	est.Turno = "jugador"
}

// BackgammonSiguiente pide el proximo lance y tira los dados del turno que sigue.
func BackgammonSiguiente(est *BackgammonEstado, clientSeed, serverSeed string, nonce int64) {
	est.Lance++
	est.Dado1, est.Dado2 = BackgammonDados(clientSeed, serverSeed, nonce, est.Lance)
	est.Dobles = est.Dado1 == est.Dado2
}

// BackgammonMultiplicador devuelve el multiplicador segun el ganador.
func BackgammonMultiplicador(ganador string) float64 {
	if ganador == "jugador" {
		return BackgammonMultVictoria
	}
	return 0
}

// BackgammonPremio calcula lo que se paga segun el ganador.
func BackgammonPremio(monto int64, ganador string) int64 {
	if monto <= 0 {
		return 0
	}
	return int64(float64(monto) * BackgammonMultiplicador(ganador))
}

// BackgammonSimular juega una carrera completa con una estrategia determinista.
// "atrasada" mueve la ficha mas atrasada (como la casa), "adelantada" la mas
// avanzada y "aleatorio" una al azar.
func BackgammonSimular(clientSeed, serverSeed string, nonce int64, estrategia string) float64 {
	est := NuevoBackgammon(clientSeed, serverSeed, nonce)
	paso := 0
	for !est.Fin {
		if est.Turno == "jugador" {
			elegido := 0
			switch estrategia {
			case "adelantada":
				for i := 1; i < BackgammonFichas; i++ {
					if est.Jugador[i] > est.Jugador[elegido] {
						elegido = i
					}
				}
			case "aleatorio":
				d := Sortear(serverSeed, fmt.Sprintf("%s#%d", clientSeed, paso), nonce, 1, 1<<30)[0]
				paso++
				elegido = d % BackgammonFichas
			default:
				for i := 1; i < BackgammonFichas; i++ {
					if est.Jugador[i] < est.Jugador[elegido] {
						elegido = i
					}
				}
			}
			_ = BackgammonJugadorJuega(est, elegido)
			if est.Fin {
				break
			}
			BackgammonSiguiente(est, clientSeed, serverSeed, nonce)
			BackgammonCasaJuega(est)
			if est.Fin {
				break
			}
			BackgammonSiguiente(est, clientSeed, serverSeed, nonce)
			continue
		}
		BackgammonCasaJuega(est)
		if est.Fin {
			break
		}
		BackgammonSiguiente(est, clientSeed, serverSeed, nonce)
	}
	return BackgammonMultiplicador(est.Ganador)
}

// BackgammonRepetir reproduce una carrera a partir de las jugadas del jugador.
func BackgammonRepetir(clientSeed, serverSeed string, nonce int64, jugadas []int) (*BackgammonEstado, error) {
	est := NuevoBackgammon(clientSeed, serverSeed, nonce)
	paso := 0
	for !est.Fin {
		if est.Turno == "jugador" {
			if paso >= len(jugadas) {
				return nil, fmt.Errorf("faltan jugadas")
			}
			if err := BackgammonJugadorJuega(est, jugadas[paso]); err != nil {
				return nil, err
			}
			paso++
			if est.Fin {
				break
			}
			BackgammonSiguiente(est, clientSeed, serverSeed, nonce)
			BackgammonCasaJuega(est)
			if est.Fin {
				break
			}
			BackgammonSiguiente(est, clientSeed, serverSeed, nonce)
			continue
		}
		BackgammonCasaJuega(est)
		if est.Fin {
			break
		}
		BackgammonSiguiente(est, clientSeed, serverSeed, nonce)
	}
	return est, nil
}

// RTPBackgammon mide el retorno al jugador por simulacion.
func RTPBackgammon(iteraciones int, estrategia string) float64 {
	if iteraciones <= 0 {
		iteraciones = 200000
	}
	var suma float64
	for i := 0; i < iteraciones; i++ {
		seed := fmt.Sprintf("sim-%d", i)
		suma += BackgammonSimular("sim", seed, int64(i), estrategia)
	}
	return suma / float64(iteraciones) * 100
}
