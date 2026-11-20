package casino

import "testing"

func TestBackgammonPagos(t *testing.T) {
	if got := BackgammonPremio(1000, "jugador"); got != 1850 {
		t.Errorf("premio por ganar = %d, esperaba 1850", got)
	}
	if got := BackgammonPremio(1000, "casa"); got != 0 {
		t.Errorf("premio por perder = %d, esperaba 0", got)
	}
}

func TestBackgammonDadosDeterministas(t *testing.T) {
	a1, a2 := BackgammonDados("cli", "srv", 4, 0)
	b1, b2 := BackgammonDados("cli", "srv", 4, 0)
	if a1 != b1 || a2 != b2 {
		t.Fatalf("los dados no son deterministas")
	}
	if a1 < 1 || a1 > 6 || a2 < 1 || a2 > 6 {
		t.Errorf("dados fuera de rango: %d,%d", a1, a2)
	}
}

func TestBackgammonRepetirCoincide(t *testing.T) {
	est := NuevoBackgammon("cli", "srv", 9)
	var jugadas []int
	for !est.Fin {
		if est.Turno == "jugador" {
			elegido := 0
			for i := 1; i < BackgammonFichas; i++ {
				if est.Jugador[i] < est.Jugador[elegido] {
					elegido = i
				}
			}
			jugadas = append(jugadas, elegido)
			if err := BackgammonJugadorJuega(est, elegido); err != nil {
				t.Fatalf("jugar: %v", err)
			}
			if est.Fin {
				break
			}
			BackgammonSiguiente(est, "cli", "srv", 9)
			BackgammonCasaJuega(est)
			if est.Fin {
				break
			}
			BackgammonSiguiente(est, "cli", "srv", 9)
			continue
		}
		BackgammonCasaJuega(est)
		if est.Fin {
			break
		}
		BackgammonSiguiente(est, "cli", "srv", 9)
	}
	rep, err := BackgammonRepetir("cli", "srv", 9, jugadas)
	if err != nil {
		t.Fatalf("repetir: %v", err)
	}
	if rep.Ganador != est.Ganador || rep.Fin != est.Fin {
		t.Errorf("la repeticion no coincide: %+v vs %+v", rep, est)
	}
	for i := range rep.Jugador {
		if rep.Jugador[i] != est.Jugador[i] || rep.Casa[i] != est.Casa[i] {
			t.Errorf("fichas distintas en la repeticion")
		}
	}
}

func TestBackgammonNoSePuedeJugarDosVeces(t *testing.T) {
	est := NuevoBackgammon("cli", "srv", 2)
	est.Turno = "jugador"
	if err := BackgammonJugadorJuega(est, 0); err != nil {
		t.Fatalf("primer jugada: %v", err)
	}
	if err := BackgammonJugadorJuega(est, 0); err == nil {
		t.Errorf("no se puede jugar fuera de turno")
	}
}

func TestRTPBackgammonEnBanda(t *testing.T) {
	rtp := RTPBackgammon(5000, "atrasada")
	t.Logf("RTP backgammon (atrasada): %.2f%%", rtp)
	if rtp < 78 || rtp > 92 {
		t.Errorf("RTP backgammon = %.2f, fuera de banda 78-92", rtp)
	}
}
