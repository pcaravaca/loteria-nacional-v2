package casino

import (
	"fmt"
	"testing"
)

// TestMinaRTP verifica que los tres niveles de Mina de Oro pagan entre el 80% y
// el 90%. Si una tabla se cambia y el RTP se sale de la banda, este test falla.
func TestMinaRTP(t *testing.T) {
	const iters = 100000
	for dificultad := 0; dificultad <= 2; dificultad++ {
		var suma float64
		for i := 0; i < iters; i++ {
			seed := fmt.Sprintf("mina-%d-%d", dificultad, i)
			draws := Sortear(seed, "sim", int64(i), Draws(minaCodigoBase), 1<<31-1)
			out, err := Jugar(minaCodigoBase, 1000, draws, Params{"dificultad": dificultad})
			if err != nil {
				t.Fatalf("mina-de-oro: %v", err)
			}
			suma += float64(Premio(1000, out)) / 1000
		}
		nivel, _ := minaNivelDe(dificultad)
		rtp := suma / iters * 100
		t.Logf("mina-de-oro (%s, %d filas) -> RTP %.2f", nivel.Nombre, nivel.Filas, rtp)
		if rtp < 80 || rtp > 90 {
			t.Errorf("mina-de-oro dificultad=%d: RTP %.2f fuera de la banda 80-90%%", dificultad, rtp)
		}
	}
}
