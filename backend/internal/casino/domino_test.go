package casino

import "testing"

func TestDominoPagos(t *testing.T) {
	if got := DominoPremio(1000, "jugador"); got != 1520 {
		t.Errorf("premio por ganar = %d, esperaba 1520", got)
	}
	if got := DominoPremio(1000, "empate"); got != 1000 {
		t.Errorf("premio por empate = %d, esperaba 1000", got)
	}
	if got := DominoPremio(1000, "casa"); got != 0 {
		t.Errorf("premio por perder = %d, esperaba 0", got)
	}
	if got := DominoMultiplicador("jugador"); got != DominoMultVictoria {
		t.Errorf("multiplicador = %v", got)
	}
}

func TestDominoSorteoDeterminista(t *testing.T) {
	m1, c1 := DominoSorteo("cli", "srv", 7)
	m2, c2 := DominoSorteo("cli", "srv", 7)
	if len(m1) != DominoFichasPorMano || len(c1) != DominoFichasPorMano {
		t.Fatalf("reparto incompleto: %d/%d", len(m1), len(c1))
	}
	for i := range m1 {
		if m1[i] != m2[i] || c1[i] != c2[i] {
			t.Fatalf("el reparto no es determinista")
		}
	}
	// Las 14 fichas repartidas son distintas.
	vistos := map[DominoFicha]bool{}
	for _, f := range append(append([]DominoFicha{}, m1...), c1...) {
		if vistos[f] {
			t.Errorf("ficha repetida %v", f)
		}
		vistos[f] = true
	}
}

func TestDominoRepetirCoincide(t *testing.T) {
	mano, casa := DominoSorteo("cli", "srv", 3)
	est := NuevoDominoEstado(mano, casa)
	var jugadas [][2]int
	for !est.Fin {
		if est.Turno == "jugador" {
			vacio := len(est.Tablero) == 0
			validas := DominoValidas(est.Mano, est.Izq, est.Der, vacio)
			if len(validas) == 0 {
				jugadas = append(jugadas, [2]int{-1, 0})
				if err := DominoJugadorJuega(est, -1); err != nil {
					t.Fatalf("pasar: %v", err)
				}
				continue
			}
			elegido := validas[0]
			jugadas = append(jugadas, [2]int{elegido, 0})
			if err := DominoJugadorJuega(est, elegido); err != nil {
				t.Fatalf("jugar: %v", err)
			}
			continue
		}
		DominoCasaJuega(est)
	}
	rep, err := DominoRepetir("cli", "srv", 3, jugadas)
	if err != nil {
		t.Fatalf("repetir: %v", err)
	}
	if !dominoEstadoIgual(rep, est) {
		t.Errorf("la repeticion no coincide: %+v vs %+v", rep, est)
	}
}

func dominoEstadoIgual(a, b *DominoEstado) bool {
	if a.Fin != b.Fin || a.Ganador != b.Ganador || a.Turno != b.Turno {
		return false
	}
	if len(a.Mano) != len(b.Mano) || len(a.Casa) != len(b.Casa) || len(a.Tablero) != len(b.Tablero) {
		return false
	}
	for i := range a.Mano {
		if a.Mano[i] != b.Mano[i] || a.Casa[i] != b.Casa[i] || a.Tablero[i] != b.Tablero[i] {
			return false
		}
	}
	return true
}

func TestDominoNoSePuedeJugarFichaQueNoEncaja(t *testing.T) {
	mano := []DominoFicha{{0, 0}, {1, 1}}
	casa := []DominoFicha{{2, 2}}
	est := NuevoDominoEstado(mano, casa)
	if err := DominoJugadorJuega(est, 0); err != nil {
		t.Fatalf("la primera ficha no requiere extremo: %v", err)
	}
	// Ahora el extremo es 0 y la ficha {1,1} no encaja.
	est.Turno = "jugador"
	if err := DominoJugadorJuega(est, 0); err == nil {
		t.Errorf("una ficha que no encaja no deberia poder jugarse")
	}
}

func TestRTPDominoEnBanda(t *testing.T) {
	for _, est := range []string{"aleatorio", "mayor"} {
		rtp := RTPDomino(5000, est)
		t.Logf("RTP domino (%s): %.2f%%", est, rtp)
		if rtp < 78 || rtp > 92 {
			t.Errorf("RTP domino (%s) = %.2f, fuera de banda 78-92", est, rtp)
		}
	}
}
