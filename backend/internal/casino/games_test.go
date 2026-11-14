package casino

import "testing"

// TestRTP simula jugadas para afinar los multiplicadores. No es un test de
// assert: solo reporta el retorno al jugador de cada juego.
func TestRTP(t *testing.T) {
	for _, codigo := range []string{"dados", "slots", "ruleta", "raspa", "keno"} {
		t.Logf("%s: RTP %.2f%%", codigo, RTP(codigo, 200000))
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
