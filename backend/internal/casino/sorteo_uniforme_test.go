package casino

import (
	"fmt"
	"testing"
)

// TestRepartoPokerUniforme fija el bug del probing: antes, cuando dos cartas
// caian en el mismo indice, la segunda se corria a idx+1, que comparte rango
// con la primera (rango = idx/4+2). El reparto quedaba sesgado hacia manos con
// rangos repetidos: 56,1 % contra 49,9 % uniforme, y el RTP del video póker
// quedaba ~12 puntos por encima del que dice la tabla.
func TestRepartoPokerUniforme(t *testing.T) {
	const n = 50000
	repetidos := 0
	for i := 0; i < n; i++ {
		draws := Sortear(fmt.Sprintf("u-%d", i), "u", int64(i), Draws("jacksorbetter"), 1<<31-1)
		cs, err := repartirPoker(AstPoker{Codigo: "jacksorbetter"}, draws)
		if err != nil {
			t.Fatalf("mano %d: %v", i, err)
		}
		seen := map[int]bool{}
		for _, c := range cs {
			if seen[c.Rango] {
				repetidos++
				break
			}
			seen[c.Rango] = true
		}
	}
	frac := 100 * float64(repetidos) / n
	// Teórico 49,88 %. Con el probing llego a 56,1 %.
	if frac < 48.5 || frac > 51.5 {
		t.Errorf("manos con rangos repetidos %.2f%% (teorico 49,88%%)", frac)
	}
}

// TestKenoBolillasUniforme: el mismo probing en kenoBolillas corria la bolilla
// y llenaba el sorteo de numeros consecutivos (32,2 % contra 23,3 % uniforme).
func TestKenoBolillasUniforme(t *testing.T) {
	const n = 50000
	consecutivas := 0
	for i := 0; i < n; i++ {
		draws := Sortear(fmt.Sprintf("v-%d", i), "v", int64(i), Draws("keno"), 1<<31-1)
		b := kenoBolillas(draws, 80, 5)
		set := map[int]bool{}
		for _, x := range b {
			set[x] = true
		}
		for x := range set {
			if set[x+1] {
				consecutivas++
				break
			}
		}
	}
	frac := 100 * float64(consecutivas) / n
	// Teórico 23,27 %. Con el probing llego a 32,2 %.
	if frac < 21.5 || frac > 25.5 {
		t.Errorf("combinaciones con consecutivos %.2f%% (teorico 23,27%%)", frac)
	}
}

// TestParleyCuatroBolillas: el parley promete 4 bolillas de 20; con el probing
// siempre salian 4 (infladas), sin el fix podrian salir menos.
func TestParleyCuatroBolillas(t *testing.T) {
	const n = 20000
	for i := 0; i < n; i++ {
		draws := Sortear(fmt.Sprintf("w-%d", i), "w", int64(i), Draws("parley"), 1<<31-1)
		out, err := jugarParley(1000, draws, Params{"numeros": []int{3, 17}})
		if err != nil {
			t.Fatalf("parley %d: %v", i, err)
		}
		bolillas, _ := out.Data["bolillas"].([]int)
		if len(bolillas) != 4 {
			t.Fatalf("parley %d: salieron %d bolillas", i, len(bolillas))
		}
	}
}
