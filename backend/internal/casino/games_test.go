package casino

import "testing"

// TestRTP simula jugadas para afinar los multiplicadores. No es un test de
// assert: solo reporta el retorno al jugador de cada juego.
func TestRTP(t *testing.T) {
	for _, codigo := range []string{"dados", "slots", "ruleta", "raspa", "keno"} {
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
