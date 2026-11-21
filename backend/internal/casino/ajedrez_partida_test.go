package casino

import (
	"strings"
	"testing"
)

// TestAjedrezMotorPuzzles valida el motor contra las posiciones curadas: la
// jugada correcta debe ser legal y dar mate, y las distractoras legales y no
// mate. Es la verificacion cruzada entre el motor nuevo y los puzzles.
func TestAjedrezMotorPuzzles(t *testing.T) {
	for _, p := range AjedrezPuzzles {
		pos, ok := NuevaAjedrezPosicion(p.FEN)
		if !ok {
			t.Fatalf("%s: FEN invalido", p.FEN)
		}
		if pos.AFEN() != p.FEN {
			t.Errorf("%s: FEN no round-trip -> %s", p.FEN, pos.AFEN())
		}
		if !pos.EnJaque(pos.Turno == 'w') == false {
			// la posicion inicial del puzzle no debe estar en jaque
		}
		correcta, ok := ajedrezMovimientoUCI(&pos, p.Correcta)
		if !ok {
			t.Errorf("%s: la jugada correcta %s no es legal", p.FEN, p.Correcta)
			continue
		}
		cp := pos
		cp.aplicar(correcta)
		if !cp.EnJaque(cp.Turno == 'w') {
			t.Errorf("%s: %s no da jaque", p.FEN, p.Correcta)
		}
		if n := len(cp.JugadasLegales()); n != 0 {
			t.Errorf("%s: %s no es mate (quedan %d jugadas)", p.FEN, p.Correcta, n)
		}
		for _, d := range p.Decoys {
			m, ok := ajedrezMovimientoUCI(&pos, d)
			if !ok {
				t.Errorf("%s: distractoria %s no es legal", p.FEN, d)
				continue
			}
			cd := pos
			cd.aplicar(m)
			if cd.EnJaque(cd.Turno == 'w') && len(cd.JugadasLegales()) == 0 {
				t.Errorf("%s: la distractioria %s tambien es mate", p.FEN, d)
			}
		}
	}
}

// TestAjedrezMotorReglasBasicas verifica el conteo de jugadas del inicio, el
// enroque, el al paso y la promocion.
func TestAjedrezMotorReglasBasicas(t *testing.T) {
	inicio := NuevoAjedrezJuego()
	if n := len(inicio.Posicion.JugadasLegales()); n != 20 {
		t.Errorf("jugadas legales desde el inicio = %d, esperaba 20", n)
	}
	// Enroque blanco disponible en el inicio del enroque.
	enroque, ok := NuevaAjedrezPosicion("r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1")
	if !ok {
		t.Fatal("FEN de enroque invalida")
	}
	var vioEnroque bool
	for _, m := range enroque.JugadasLegales() {
		if m.Enroque {
			vioEnroque = true
		}
	}
	if !vioEnroque {
		t.Error("el enroque no aparece entre las jugadas legales")
	}
	// Al paso: peon blanco en e5, negro acaba de jugar d7d5.
	alpaso, ok := NuevaAjedrezPosicion("8/8/8/3pP3/8/8/8/4K2k w - d6 0 2")
	if !ok {
		t.Fatal("FEN de al paso invalida")
	}
	var vioAlPaso bool
	for _, m := range alpaso.JugadasLegales() {
		if m.AlPaso && m.Hasta == 19 { // d6
			vioAlPaso = true
		}
	}
	if !vioAlPaso {
		t.Error("la captura al paso no aparece entre las jugadas legales")
	}
	// Promocion: peon blanco en a7.
	promo, ok := NuevaAjedrezPosicion("8/P7/8/8/8/8/8/k6K w - - 0 1")
	if !ok {
		t.Fatal("FEN de promocion invalida")
	}
	var promociones int
	for _, m := range promo.JugadasLegales() {
		if m.Promocion != 0 {
			promociones++
		}
	}
	if promociones != 4 {
		t.Errorf("promociones = %d, esperaba 4", promociones)
	}
	// Ahogado: rey negro en h8, dama blanca en f7, rey blanco en g6... uso una
	// posicion clasica de ahogado.
	ahogado, ok := NuevaAjedrezPosicion("7k/5Q2/6K1/8/8/8/8/8 b - - 0 1")
	if !ok {
		t.Fatal("FEN de ahogado invalida")
	}
	if len(ahogado.JugadasLegales()) != 0 {
		t.Fatal("la posicion de ahogado tiene jugadas")
	}
	if ahogado.EnJaque(false) {
		t.Fatal("la posicion de ahogado esta en jaque")
	}
	j := AjedrezJuego{Posicion: ahogado, Historial: map[string]int{}}
	j.actualizar()
	if j.Resultado != "tablas" || j.Motivo != "tablas por ahogado" {
		t.Errorf("ahogado = %s/%s, esperaba tablas", j.Resultado, j.Motivo)
	}
}

// TestAjedrezJugadaCasaDeterminista verifica que la jugada de la casa depende
// solo de la posicion, el ply y las semillas.
func TestAjedrezJugadaCasaDeterminista(t *testing.T) {
	j := NuevoAjedrezJuego()
	for i := 0; i < 6; i++ {
		m1, ok1 := AjedrezJugadaCasa("cli", "srv", 7, &j.Posicion, len(j.Jugadas))
		m2, ok2 := AjedrezJugadaCasa("cli", "srv", 7, &j.Posicion, len(j.Jugadas))
		if !ok1 || !ok2 || m1.UCI() != m2.UCI() {
			t.Fatalf("jugada de la casa no determinista: %s vs %s", m1.UCI(), m2.UCI())
		}
		if !j.AplicarUCI(m1.UCI()) {
			t.Fatalf("no se pudo aplicar %s", m1.UCI())
		}
	}
}

// TestAjedrezPartidaCompletaHeadless juega una partida completo entre el motor
// de la casa y el motor del jugador, y verifica que termina, que el resultado
// es coherente y que la reconstruccion reproduce la misma posicion.
func TestAjedrezPartidaCompletaHeadless(t *testing.T) {
	j := NuevoAjedrezJuego()
	sem := "srv-headless"
	cli := "cli-headless"
	for j.Resultado == "" {
		if j.Posicion.Turno == 'w' {
			m, ok := ajedrezMejorJugada(&j.Posicion, AjedrezProfundidadCasa)
			if !ok {
				t.Fatal("el jugador no tiene jugadas pero la partida sigue")
			}
			if !j.AplicarUCI(m.UCI()) {
				t.Fatalf("jugada del jugador ilegal: %s", m.UCI())
			}
			continue
		}
		m, ok := AjedrezJugadaCasa(cli, sem, 3, &j.Posicion, len(j.Jugadas))
		if !ok {
			t.Fatal("la casa no tiene jugadas pero la partida sigue")
		}
		if !j.AplicarUCI(m.UCI()) {
			t.Fatalf("jugada de la casa ilegal: %s", m.UCI())
		}
	}
	if j.Resultado == "" {
		t.Fatal("la partida no termino")
	}
	t.Logf("resultado=%s motivo=%s jugadas=%d fen=%s", j.Resultado, j.Motivo, len(j.Jugadas), j.Posicion.AFEN())
	if len(j.Jugadas) < 2 {
		t.Fatal("una partida completa tiene al menos dos jugadas")
	}
	rep, ok := AjedrezReconstruir(j.Jugadas)
	if !ok {
		t.Fatal("no se pudo reconstruir la partida")
	}
	if rep.Posicion.AFEN() != j.Posicion.AFEN() {
		t.Errorf("la reconstruccion diverge: %s != %s", rep.Posicion.AFEN(), j.Posicion.AFEN())
	}
	if rep.Resultado != j.Resultado {
		t.Errorf("resultado reconstruido = %s, esperaba %s", rep.Resultado, j.Resultado)
	}
	// Cada jugada de la casa debe reproducirse con las semillas.
	pos := NuevoAjedrezJuego()
	for ply, uci := range j.Jugadas {
		esCasa := pos.Posicion.Turno != 'w' // el jugador juega blancas
		if esCasa {
			m, ok := AjedrezJugadaCasa(cli, sem, 3, &pos.Posicion, ply)
			if !ok || m.UCI() != uci {
				t.Fatalf("ply %d: la casa %s no reproduce %s", ply, m.UCI(), uci)
			}
		}
		if !pos.AplicarUCI(uci) {
			t.Fatalf("ply %d: jugada ilegal %s", ply, uci)
		}
	}
}

// TestAjedrezVerificacionReproduceLaCasa juega una partida donde el jugador
// elige jugadas al azar y verifica que la casa se reproduce jugada a jugada.
func TestAjedrezVerificacionReproduceLaCasa(t *testing.T) {
	j := NuevoAjedrezJuego()
	sem := "srv-verifica"
	cli := "cli-verifica"
	for j.Resultado == "" {
		if j.Posicion.Turno == 'w' {
			movs := j.Posicion.JugadasLegales()
			if len(movs) == 0 {
				break
			}
			r := Sortear(sem, cli, 11, 1, 1<<30)[0]
			if !j.AplicarUCI(movs[r%len(movs)].UCI()) {
				t.Fatal("jugada del jugador ilegal")
			}
			continue
		}
		m, ok := AjedrezJugadaCasa(cli, sem, 11, &j.Posicion, len(j.Jugadas))
		if !ok {
			break
		}
		if !j.AplicarUCI(m.UCI()) {
			t.Fatal("jugada de la casa ilegal")
		}
	}
	rep, ok := AjedrezReconstruir(j.Jugadas)
	if !ok {
		t.Fatal("no se reconstruyo la partida")
	}
	pos := NuevoAjedrezJuego()
	for ply, uci := range j.Jugadas {
		if pos.Posicion.Turno != 'w' {
			m, ok := AjedrezJugadaCasa(cli, sem, 11, &pos.Posicion, ply)
			if !ok || m.UCI() != uci {
				t.Fatalf("ply %d: la casa no reprodujo %s", ply, uci)
			}
		}
		if !pos.AplicarUCI(uci) {
			t.Fatalf("ply %d: ilegal %s", ply, uci)
		}
	}
	if rep.Resultado != j.Resultado {
		t.Errorf("resultado = %s, esperaba %s", rep.Resultado, j.Resultado)
	}
}

// TestAjedrezPartidaPagos verifica la tabla de pagos de la partida completa.
func TestAjedrezPartidaPagos(t *testing.T) {
	if got := AjedrezPartidaPremio(1000, "blancas", "blancas"); got != 1700 {
		t.Errorf("victoria = %d, esperaba 1700", got)
	}
	if got := AjedrezPartidaPremio(1000, "tablas", "blancas"); got != 1000 {
		t.Errorf("empate = %d, esperaba 1000", got)
	}
	if got := AjedrezPartidaPremio(1000, "negras", "blancas"); got != 0 {
		t.Errorf("derrota = %d, esperaba 0", got)
	}
	if AjedrezPartidaMultiplicador("tablas", "negras") != AjedrezPartidaMultEmpate {
		t.Error("multiplicador de empate incorrecto")
	}
}

// TestRTPAjedrezPartidaEnBanda mide el RTP de la partida completa y lo deja en
// la banda 78-92 % para el juego hábil; el casual se declara aunque quede fuera.
func TestRTPAjedrezPartidaEnBanda(t *testing.T) {
	habil, casual := RTPAjedrezPartida(240)
	t.Logf("RTP partida completa: hábil %.2f%% · casual %.2f%%", habil, casual)
	if habil < 78 || habil > 92 {
		t.Errorf("RTP hábil = %.2f, fuera de banda 78-92", habil)
	}
	if casual <= 0 {
		t.Errorf("RTP casual = %.2f, debe estar declarado", casual)
	}
}

// TestAjedrezConteoSimulacion deja registradas las victorias, empates y
// derrotas de cada perfil: es la base de la calibracion de pagos.
func TestAjedrezConteoSimulacion(t *testing.T) {
	const n = 120
	contar := func(habil bool) (v, e, d int) {
		for i := 0; i < n; i++ {
			color := "blancas"
			if i%2 == 1 {
				color = "negras"
			}
			res := AjedrezResultadoSimulacion(int64(i), habil, i%2 == 0)
			switch res {
			case color:
				v++
			case "tablas":
				e++
			default:
				d++
			}
		}
		return
	}
	vh, eh, dh := contar(true)
	vc, ec, dc := contar(false)
	t.Logf("hábil: V=%d E=%d D=%d (%.1f%% %.1f%% %.1f%%)", vh, eh, dh, pct(vh, n), pct(eh, n), pct(dh, n))
	t.Logf("casual: V=%d E=%d D=%d (%.1f%% %.1f%% %.1f%%)", vc, ec, dc, pct(vc, n), pct(ec, n), pct(dc, n))
}

func pct(p, n int) float64 { return float64(p) / float64(n) * 100 }

// TestAjedrezFENConocidos verifica algunas posiciones y jugadas de referencia.
func TestAjedrezFENConocidos(t *testing.T) {
	casos := []struct {
		fen    string
		legales int
	}{
		{"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", 20},
		{"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", 48}, // posición de prueba clásica
		{"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1", 14},
	}
	for _, c := range casos {
		pos, ok := NuevaAjedrezPosicion(c.fen)
		if !ok {
			t.Errorf("%s: FEN invalido", c.fen)
			continue
		}
		if n := len(pos.JugadasLegales()); n != c.legales {
			t.Errorf("%s: jugadas = %d, esperaba %d", c.fen, n, c.legales)
		}
		if pos.AFEN() != c.fen {
			t.Errorf("%s: round-trip -> %s", c.fen, pos.AFEN())
		}
	}
}

// TestAjedrezNotacionUCI abarca las cuatro esquinas del tablero.
func TestAjedrezNotacionUCI(t *testing.T) {
	for _, c := range []struct {
		idx  int
		nombre string
	}{
		{0, "a8"}, {7, "h8"}, {56, "a1"}, {63, "h1"}, {60, "e1"}, {4, "e8"}, {36, "e4"},
	} {
		if got := ajedrezCasilla(c.idx); got != c.nombre {
			t.Errorf("casilla %d = %s, esperaba %s", c.idx, got, c.nombre)
		}
		idx, ok := ajedrezIndice(c.nombre)
		if !ok || idx != c.idx {
			t.Errorf("indice de %s = %d (%v), esperaba %d", c.nombre, idx, ok, c.idx)
		}
	}
	if strings.TrimSpace(AjedrezInicial) == "" {
		t.Error("FEN inicial vacio")
	}
}
