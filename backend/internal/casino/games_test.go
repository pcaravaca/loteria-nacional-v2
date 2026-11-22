package casino

import "testing"

// TestRTP simula jugadas para afinar los multiplicadores. No es un test de
// assert: solo reporta el retorno al jugador de cada juego.
func TestRTP(t *testing.T) {
	for _, codigo := range []string{"dados", "slots", "ruleta", "raspa", "keno", "moneda", "suertudo", "parley", "cascada", "hilo", "alto", "fortuna", "baccarat"} {
		t.Logf("%s: RTP %.2f%%", codigo, RTP(codigo, 200000))
	}
}

// TestApuestaMaxima verifica que el techo de premio acote la apuesta maxima sin
// tocar el multiplicador del sorteo (que es lo que audita el jugador).
func TestApuestaMaxima(t *testing.T) {
	casos := []struct {
		nombre     string
		maxApuesta int64
		maxMultipl int64
		maxPremio  int64
		codigo     string
		esperado   int64
	}{
		{"sin techo respeta max apuesta", 50000, 30, 0, "dados", 50000},
		{"techo holgado no recorta", 50000, 30, 9999999, "dados", 50000},
		{"techo recorta a un tercio", 50000, 30, 1000000, "dados", 33333},
		{"keno recortado por su multiplicador", 50000, 65, 1000000, "keno", 15384},
		{"blackjack usa doblar y ganar (x4)", 1000000, 3, 1000000, "blackjack", 250000},
		{"multiplicador cero no rompe", 50000, 0, 1000000, "ruleta", 50000},
	}
	for _, c := range casos {
		if got := ApuestaMaxima(c.maxApuesta, c.maxMultipl, c.maxPremio, c.codigo); got != c.esperado {
			t.Errorf("%s: ApuestaMaxima(%d,%d,%d,%s) = %d, esperaba %d",
				c.nombre, c.maxApuesta, c.maxMultipl, c.maxPremio, c.codigo, got, c.esperado)
		}
	}
}

// TestMultiplicadorMaximoBlackjack fija el peor caso del blackjack: doblar la
// apuesta y ganar paga 4 veces lo apostado originalmente.
func TestMultiplicadorMaximoBlackjack(t *testing.T) {
	if got := MultiplicadorMaximo("blackjack", 3); got != 4 {
		t.Errorf("blackjack: MultiplicadorMaximo = %d, esperaba 4", got)
	}
	if got := MultiplicadorMaximo("dados", 30); got != 30 {
		t.Errorf("dados: MultiplicadorMaximo = %d, esperaba 30", got)
	}
}

// TestBaccarat cubre las manos naturales, el empate, la regla de la tercera
// carta y las validaciones de la apuesta.
func TestBaccarat(t *testing.T) {
	casos := []struct {
		nombre     string
		draws      []int
		tipo       string
		esperado   int64
		resultado  string
		wantErr    bool
	}{
		{"natural gana jugador", []int{4, 5, 4, 4}, "jugador", 2, "jugador", false},
		{"natural banca", []int{3, 4, 4, 4}, "banca", 2, "banca", false},
		{"empate paga x8", []int{3, 4, 3, 4}, "empate", 8, "empate", false},
		{"empate devuelve lo apostado", []int{3, 4, 3, 4}, "jugador", 1, "empate", false},
		{"jugador pide y gana", []int{0, 0, 1, 2, 8, 0}, "jugador", 2, "jugador", false},
		{"banca se planta con 6", []int{1, 1, 3, 3, 1, 9}, "banca", 2, "banca", false},
		{"tipo invalido", []int{1, 1, 1, 1}, "empate-total", 0, "", true},
		{"sorteo incompleto", []int{1, 1, 1, 1, 1}, "jugador", 0, "", true},
	}
	for _, c := range casos {
		out, err := jugarBaccarat(1000, c.draws, Params{"tipo": c.tipo})
		if c.wantErr {
			if err == nil {
				t.Errorf("%s: esperaba error y no hubo", c.nombre)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: error inesperado: %v", c.nombre, err)
		}
		if out.Multiplicador != c.esperado {
			t.Errorf("%s: multiplicador = %d, esperaba %d", c.nombre, out.Multiplicador, c.esperado)
		}
		if got := out.Data["resultado"]; got != c.resultado {
			t.Errorf("%s: resultado = %v, esperaba %s", c.nombre, got, c.resultado)
		}
	}
}

// TestPremio verifica el pago: el multiplicador manda, pero la comision que
// indica el motor (baccarat a la banca) se descuenta de la ganancia.
func TestPremio(t *testing.T) {
	casos := []struct {
		nombre   string
		monto    int64
		out      Outcome
		esperado int64
	}{
		{"sin comision", 1000, Outcome{Multiplicador: 5}, 5000},
		{"banca con 5%", 1000, Outcome{Multiplicador: 2, Data: map[string]any{"comision": int64(50)}}, 1950},
		{"comision no deja negativo", 1000, Outcome{Multiplicador: 1, Data: map[string]any{"comision": int64(5000)}}, 0},
	}
	for _, c := range casos {
		if got := Premio(c.monto, c.out); got != c.esperado {
			t.Errorf("%s: Premio = %d, esperaba %d", c.nombre, got, c.esperado)
		}
	}
}

// TestBaccaratComision comprueba que acertar a la banca cobra la comision del 5%.
func TestBaccaratComision(t *testing.T) {
	out, err := jugarBaccarat(1000, []int{3, 4, 4, 4}, Params{"tipo": "banca"})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if out.Multiplicador != 2 {
		t.Errorf("multiplicador = %d, esperaba 2", out.Multiplicador)
	}
	if got := out.Data["comision"]; got != int64(50) {
		t.Errorf("comision = %v, esperaba 50", got)
	}
	if got := Premio(1000, out); got != 1950 {
		t.Errorf("premio = %d, esperaba 1950", got)
	}
}

// TestSorteoDeterminista verifica que el mismo insumo produce el mismo sorteo,
// que es la base del esquema provably fair.
func TestSorteoDeterminista(t *testing.T) {
	a := Sortear("server-seed", "client-seed", 7, 5, 100)
	b := Sortear("server-seed", "client-seed", 7, 5, 100)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("sorteo no determinista en %d: %d != %d", i, a[i], b[i])
		}
	}
	c := Sortear("otra-semilla", "client-seed", 7, 5, 100)
	iguales := true
	for i := range a {
		if a[i] != c[i] {
			iguales = false
		}
	}
	if iguales {
		t.Fatal("semillas distintas produjeron el mismo sorteo")
	}
}

// TestDadosMotorCoincideConLaTabla: paramsRTP juega siempre a la suma 7, que
// sale 6 de 36 veces y paga x5 -> 83,33 %. Este test fija el techo que el
// catalogo puede prometer: el RTP declarado de Dados del Sur no puede pasarse
// de lo que la tabla paga (antes declaba 88 %, imposible).
func TestDadosMotorCoincideConLaTabla(t *testing.T) {
	if got := RTP("dados", 100000); got < 81.8 || got > 84.8 {
		t.Errorf("RTP(dados) = %.2f, la tabla dice 83,33 (el declarado no puede pasarse)", got)
	}
}
