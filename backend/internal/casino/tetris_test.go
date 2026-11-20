package casino

import (
	"fmt"
	"testing"
)

func TestTetrisSecuenciaDeterminista(t *testing.T) {
	a := TetrisSecuencia("cli", "srv", 1)
	b := TetrisSecuencia("cli", "srv", 1)
	if len(a) != TetrisPiezas {
		t.Fatalf("piezas = %d", len(a))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("la secuencia no es determinista")
		}
		if a[i] < 0 || a[i] >= TetrisTipos {
			t.Errorf("pieza fuera de rango: %d", a[i])
		}
	}
}

func TestTetrisSoltarYLimpiar(t *testing.T) {
	seq := []int{1, 1, 1, 1} // piezas O
	tab := make([]int, TetrisFilas*TetrisColumnas)
	total := 0
	for p := 0; p < 4; p++ {
		lim, ok := TetrisSoltar(tab, seq, p, 0, 0)
		if !ok {
			t.Fatalf("la pieza %d no entro", p)
		}
		total += lim
	}
	if total != 0 {
		t.Errorf("no se esperaban lineas, llegaron %d", total)
	}
	// Cuatro piezas O en la columna 0 dejan las dos primeras columnas llenas.
	if tab[(TetrisFilas-1)*TetrisColumnas+1] == 0 || tab[(TetrisFilas-1)*TetrisColumnas+2] == 0 {
		t.Errorf("la base del pozo deberia estar ocupada")
	}
}

func TestTetrisLineasSePagAN(t *testing.T) {
	if got := TetrisPremio(1000, 0); got != 0 {
		t.Errorf("0 lineas paga %d", got)
	}
	if got := TetrisPremio(1000, 1); got != 280 {
		t.Errorf("1 linea paga %d, esperaba 280", got)
	}
	if got := TetrisPremio(1000, 3); got != 850 {
		t.Errorf("3 lineas paga %d, esperaba 850", got)
	}
}

func TestTetrisRepetirCoincide(t *testing.T) {
	seq := TetrisSecuencia("cli", "srv", 6)
	tab := make([]int, TetrisFilas*TetrisColumnas)
	lineas := 0
	var jugadas [][2]int
	for p := 0; p < 5; p++ {
		lim, ok := TetrisSoltar(tab, seq, p, 0, 0)
		if !ok {
			t.Fatalf("la pieza %d no entro", p)
		}
		jugadas = append(jugadas, [2]int{0, 0})
		lineas += lim
	}
	rep, repLineas, err := TetrisRepetir("cli", "srv", 6, jugadas)
	if err != nil {
		t.Fatalf("repetir: %v", err)
	}
	if repLineas != lineas {
		t.Errorf("lineas = %d, esperaba %d", repLineas, lineas)
	}
	for i := range rep {
		if rep[i] != tab[i] {
			t.Errorf("el tablero no coincide en la celda %d", i)
		}
	}
}

func TestTetrisColumnaInvalida(t *testing.T) {
	seq := []int{0} // pieza I
	tab := make([]int, TetrisFilas*TetrisColumnas)
	if _, ok := TetrisSoltar(tab, seq, 0, TetrisColumnas-1, 0); ok {
		t.Errorf("una columna fuera de rango no deberia entrar")
	}
}

func TestRTPTetrisEnBanda(t *testing.T) {
	rtp := RTPTetris(5000, "baja")
	t.Logf("RTP tetris (baja): %.2f%%", rtp)
	if rtp < 78 || rtp > 92 {
		t.Errorf("RTP tetris = %.2f, fuera de banda 78-92", rtp)
	}
}

func TestTetrisDistribucionLineas(t *testing.T) {
	// Referencia: la jugada hábil se mantiene cerca de las 3 líneas en 8 piezas.
	promedio := 0.0
	n := 500
	for i := 0; i < n; i++ {
		promedio += float64(TetrisSimular("sim", fmt.Sprintf("sim-%d", i), int64(i), "baja"))
	}
	promedio /= float64(n)
	t.Logf("tetris lineas promedio (baja): %.2f", promedio)
	if promedio < 1.5 || promedio > 5 {
		t.Errorf("promedio de lineas = %.2f, fuera de lo esperado", promedio)
	}
}
