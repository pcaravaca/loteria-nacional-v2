package casino

import (
	"fmt"
	"testing"
)

// TestPinballRTP verifica que las tres mesas pagan entre el 80% y el 90% en cada
// fuerza. Si una tabla se cambia y el RTP se sale de la banda, este test falla.
func TestPinballRTP(t *testing.T) {
	const iters = 100000
	for _, cod := range []string{"pinball-neon", "pinball-clasico", "plinko-premium"} {
		for fuerza := 0; fuerza <= 2; fuerza++ {
			var suma float64
			for i := 0; i < iters; i++ {
				seed := fmt.Sprintf("pb-%s-%d-%d", cod, fuerza, i)
				draws := Sortear(seed, "sim", int64(i), Draws(cod), 1<<31-1)
				out, err := Jugar(cod, 1000, draws, Params{"fuerza": fuerza})
				if err != nil {
					t.Fatalf("%s: %v", cod, err)
				}
				suma += float64(Premio(1000, out)) / 1000
			}
			rtp := suma / iters * 100
			t.Logf("%-16s fuerza=%d -> RTP %.2f", cod, fuerza, rtp)
			if rtp < 80 || rtp > 90 {
				t.Errorf("%s fuerza=%d: RTP %.2f fuera de la banda 80-90%%", cod, fuerza, rtp)
			}
		}
	}
}
