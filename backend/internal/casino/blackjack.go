package casino

// Blackjack: reglas de la casa. El crupier pide hasta 17, el blackjack natural
// paga 3:2 y el empate devuelve la apuesta. El orden de la baraja es
// determinista (provably fair) a partir de las semillas y el nonce.

// Baraja devuelve los 52 indices de carta barajados.
func Baraja(clientSeed, serverSeed string, nonce int64, n int) []int {
	if n < 52 {
		n = 52
	}
	draws := Sortear(serverSeed, clientSeed, nonce, n, 1<<31-1)
	mazo := make([]int, 52)
	for i := range mazo {
		mazo[i] = i
	}
	for i := len(mazo) - 1; i > 0; i-- {
		j := draws[len(mazo)-1-i] % (i + 1)
		mazo[i], mazo[j] = mazo[j], mazo[i]
	}
	return mazo
}

func blackjackValorCarta(i int) int {
	v := i % 13
	switch {
	case v == 0: // as
		return 11
	case v >= 10: // J, Q, K
		return 10
	default:
		return v + 1
	}
}

func blackjackPalo(i int) string {
	switch i / 13 {
	case 0:
		return "picas"
	case 1:
		return "corazones"
	case 2:
		return "diamantes"
	default:
		return "treboles"
	}
}

// Figura devuelve la letra/numero de una carta (A, 2..10, J, Q, K).
func blackjackFigura(i int) string {
	v := i % 13
	switch v {
	case 0:
		return "A"
	case 10:
		return "J"
	case 11:
		return "Q"
	case 12:
		return "K"
	case 9:
		return "10"
	default:
		return string(rune('1' + v))
	}
}

// ValorBlackjack devuelve el valor de la mano, si es soft y si se paso de 21.
func ValorBlackjack(mano []int) (valor int, soft bool, pasado bool) {
	total := 0
	ases := 0
	for _, c := range mano {
		if blackjackValorCarta(c) == 11 {
			ases++
		}
		total += blackjackValorCarta(c)
	}
	for total > 21 && ases > 0 {
		total -= 10
		ases--
	}
	return total, ases > 0, total > 21
}

func BlackjackNatural(mano []int) bool {
	if len(mano) != 2 {
		return false
	}
	v, _, _ := ValorBlackjack(mano)
	return v == 21
}

// CrupierHasta17 completa la mano del crupier pidiendo hasta 17 o mas.
// Devuelve la mano y la nueva posicion dentro del mazo.
func CrupierHasta17(crupier []int, mazo []int, pos int) ([]int, int) {
	mano := append([]int{}, crupier...)
	for {
		v, _, _ := ValorBlackjack(mano)
		if v >= 17 || pos >= len(mazo) {
			break
		}
		mano = append(mano, mazo[pos])
		pos++
	}
	return mano, pos
}

// BlackjackResultado compara las manos y devuelve estado, premio y multiplicador.
func BlackjackResultado(jugador, crupier []int, apostado int64) (string, int64, float64) {
	vj, _, pasadoJ := ValorBlackjack(jugador)
	if pasadoJ {
		return "perdida", 0, 0
	}
	if BlackjackNatural(jugador) {
		return "blackjack", apostado * 5 / 2, 2.5
	}
	vc, _, pasadoc := ValorBlackjack(crupier)
	switch {
	case pasadoc || vj > vc:
		return "ganada", apostado * 2, 2
	case vj == vc:
		return "empate", apostado, 1
	default:
		return "perdida", 0, 0
	}
}

// Palo y Figura se exponen para dibujar las cartas en el cliente.
func PaloCarta(i int) string { return blackjackPalo(i) }
func FiguraCarta(i int) string { return blackjackFigura(i) }

// Rendirse (surrender): solo con la mano inicial de dos cartas, antes de pedir.
// Devuelve la mitad de lo apostado; el resto lo queda la casa.
func BlackjackRendicion(apostado int64) int64 { return apostado / 2 }

// SeguroAplica indica si se puede asegurar: la carta visible del crupier es un As.
func SeguroAplica(crupier []int) bool {
	return len(crupier) > 0 && crupier[0]%13 == 0
}

// SeguroCosto es la mitad de la apuesta; paga 2:1 si el crupier tiene blackjack.
func SeguroCosto(apostado int64) int64 { return apostado / 2 }

// SeguroPaga devuelve lo que acredita el seguro cuando el crupier tiene natural.
func SeguroPaga(apostado int64) int64 { return SeguroCosto(apostado) * 2 }
