package casino

import "testing"

// TestBingoSorteoDeterminista verifica que el orden y los cartones dependen solo
// de las semillas y el nonce, y que el orden es una permutación de 1..75.
func TestBingoSorteoDeterminista(t *testing.T) {
	orden, grids := BingoSorteo("cli-1", "servidor", 7, 2)
	if len(orden) != 75 {
		t.Fatalf("el orden tiene %d bolillas", len(orden))
	}
	visto := map[int]bool{}
	for _, b := range orden {
		if b < 1 || b > 75 {
			t.Fatalf("bolilla fuera de rango: %d", b)
		}
		if visto[b] {
			t.Fatalf("bolilla repetida: %d", b)
		}
		visto[b] = true
	}
	if len(grids) != 2 {
		t.Fatalf("se generaron %d cartones", len(grids))
	}

	otra, otros := BingoSorteo("cli-1", "servidor", 7, 2)
	for i := range orden {
		if orden[i] != otra[i] {
			t.Fatalf("el orden no es determinista en %d", i)
		}
	}
	for n := range grids {
		for r := 0; r < 5; r++ {
			for c := 0; c < 5; c++ {
				if grids[n][r][c] != otros[n][r][c] {
					t.Fatalf("los cartones no son deterministas")
				}
			}
		}
	}

	cambio, _ := BingoSorteo("cli-2", "servidor", 7, 1)
	iguales := true
	for i := range orden {
		if orden[i] != cambio[i] {
			iguales = false
			break
		}
	}
	if iguales {
		t.Fatal("distintas semillas produjeron el mismo orden")
	}
}

// TestBingoCartonValido revisa las columnas, el centro libre y la ausencia de
// repetidos dentro de un cartón.
func TestBingoCartonValido(t *testing.T) {
	_, grids := BingoSorteo("cli", "servidor", 3, 4)
	for n, g := range grids {
		if len(g) != 5 {
			t.Fatalf("cartón %d: %d filas", n, len(g))
		}
		vistos := map[int]bool{}
		for c := 0; c < 5; c++ {
			columna := []int{}
			for r := 0; r < 5; r++ {
				if r == 2 && c == 2 {
					if g[r][c] != 0 {
						t.Fatalf("cartón %d: el centro no está libre", n)
					}
					continue
				}
				v := g[r][c]
				if v < 15*c+1 || v > 15*c+15 {
					t.Fatalf("cartón %d: %d no pertenece a la columna %d", n, v, c)
				}
				if vistos[v] {
					t.Fatalf("cartón %d: número repetido %d", n, v)
				}
				vistos[v] = true
				columna = append(columna, v)
			}
			for i := 1; i < len(columna); i++ {
				if columna[i-1] > columna[i] {
					t.Fatalf("cartón %d: la columna %d no está ordenada", n, c)
				}
			}
			if c == 2 && len(columna) != 4 {
				t.Fatalf("cartón %d: la columna N tiene %d números", n, len(columna))
			}
			if c != 2 && len(columna) != 5 {
				t.Fatalf("cartón %d: la columna %d tiene %d números", n, c, len(columna))
			}
		}
	}
}

// TestBingoLineas cubre filas, columnas y diagonales con el centro libre.
func TestBingoLineas(t *testing.T) {
	grid := [][]int{
		{1, 16, 31, 46, 61},
		{2, 17, 32, 47, 62},
		{3, 18, 0, 48, 63},
		{4, 19, 33, 49, 64},
		{5, 20, 34, 50, 65},
	}
	casos := []struct {
		nombre   string
		cantadas []int
		lineas   int
		lleno    bool
	}{
		{"sin bolillas", nil, 0, false},
		{"fila completa", []int{1, 16, 31, 46, 61}, 1, false},
		{"columna B", []int{1, 2, 3, 4, 5}, 1, false},
		{"columna N con libre", []int{31, 32, 33, 34}, 1, false},
		{"diagonal principal", []int{1, 17, 49, 65}, 1, false},
		{"diagonal secundaria", []int{61, 47, 19, 5}, 1, false},
		{"dos líneas", []int{1, 16, 31, 46, 61, 1, 2, 3, 4, 5}, 2, false},
	}
	for _, c := range casos {
		m := BingoMarcadas(grid, c.cantadas)
		if got := BingoLineas(m); got != c.lineas {
			t.Errorf("%s: líneas = %d, esperaba %d", c.nombre, got, c.lineas)
		}
		if got := BingoLleno(m); got != c.lleno {
			t.Errorf("%s: lleno = %v, esperaba %v", c.nombre, got, c.lleno)
		}
	}

	// Cartón lleno: las 24 bolillas del cartón.
	todas := []int{}
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			if grid[r][c] != 0 {
				todas = append(todas, grid[r][c])
			}
		}
	}
	m := BingoMarcadas(grid, todas)
	if !BingoLleno(m) {
		t.Error("el cartón completo debería contar como lleno")
	}
	if got := BingoLineas(m); got != 12 {
		t.Errorf("cartón lleno: líneas = %d, esperaba 12", got)
	}
}

// TestBingoTier fija la tabla de multiplicadores.
func TestBingoTier(t *testing.T) {
	casos := []struct {
		lineas int
		lleno  bool
		mult   int64
	}{
		{0, false, 0},
		{1, false, 1},
		{2, false, 3},
		{3, false, 3},
		{8, false, 3},
		{0, true, 25},
		{12, true, 25},
	}
	for _, c := range casos {
		if got := BingoTier(c.lineas, c.lleno); got != c.mult {
			t.Errorf("líneas %d lleno %v: multiplicador = %d, esperaba %d", c.lineas, c.lleno, got, c.mult)
		}
	}
}

// TestBingoPremioPorCarton verifica que cada cartón paga su propio multiplicador
// sobre la apuesta por cartón.
func TestBingoPremioPorCarton(t *testing.T) {
	grid := [][]int{
		{1, 16, 31, 46, 61},
		{2, 17, 32, 47, 62},
		{3, 18, 0, 48, 63},
		{4, 19, 33, 49, 64},
		{5, 20, 34, 50, 65},
	}
	grids := [][][]int{grid, grid}
	// Una línea en cada cartón: paga el doble de la apuesta por cartón.
	premio, lineas, lleno, tiers := BingoPremio(1000, grids, []int{1, 16, 31, 46, 61})
	if premio != 2000 {
		t.Errorf("premio = %d, esperaba 2000", premio)
	}
	if lineas != 1 || lleno {
		t.Errorf("lineas = %d lleno = %v", lineas, lleno)
	}
	if len(tiers) != 2 || tiers[0] != 1 || tiers[1] != 1 {
		t.Errorf("tiers = %v", tiers)
	}
}

// TestRTPBingo reporta el retorno medido por simulación (no es un assert).
func TestRTPBingo(t *testing.T) {
	t.Logf("bingo: RTP %.2f%%", RTPBingo(200000))
}
