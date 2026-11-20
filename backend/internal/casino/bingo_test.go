package casino

import (
	"os"
	"strconv"
	"testing"
)

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

// TestBingo90SorteoDeterminista revisa el orden de 90 bolillas y que los tickets
// 3x9 sean válidos: 5 números por fila, 1 a 3 por columna, sin repetidos y con
// los números en la columna que les corresponde.
func TestBingo90SorteoDeterminista(t *testing.T) {
	orden, grids := BingoSorteoVariante("cli-90", "servidor", 11, 4, "bingo90")
	if len(orden) != 90 {
		t.Fatalf("el orden tiene %d bolillas", len(orden))
	}
	visto := map[int]bool{}
	for _, b := range orden {
		if b < 1 || b > 90 || visto[b] {
			t.Fatalf("bolilla invalida o repetida: %d", b)
		}
		visto[b] = true
	}
	if len(grids) != 4 {
		t.Fatalf("se generaron %d tickets", len(grids))
	}
	for n, g := range grids {
		if len(g) != 3 {
			t.Fatalf("ticket %d: %d filas", n, len(g))
		}
		vistos := map[int]bool{}
		for r := 0; r < 3; r++ {
			if len(g[r]) != 9 {
				t.Fatalf("ticket %d: fila %d con %d columnas", n, r, len(g[r]))
			}
			fila := 0
			for c := 0; c < 9; c++ {
				v := g[r][c]
				if v == 0 {
					continue
				}
				fila++
				if vistos[v] {
					t.Fatalf("ticket %d: número repetido %d", n, v)
				}
				vistos[v] = true
				inicio, fin := rangoColumna90Extremos(c)
				if v < inicio || v > fin {
					t.Fatalf("ticket %d: %d no pertenece a la columna %d", n, v, c)
				}
			}
			if fila != 5 {
				t.Fatalf("ticket %d: la fila %d tiene %d números", n, r, fila)
			}
		}
		for c := 0; c < 9; c++ {
			n := 0
			for r := 0; r < 3; r++ {
				if g[r][c] != 0 {
					n++
				}
			}
			if n < 1 || n > 3 {
				t.Fatalf("ticket %d: la columna %d tiene %d números", n, c, n)
			}
		}
	}

	otra, otros := BingoSorteoVariante("cli-90", "servidor", 11, 4, "bingo90")
	for i := range orden {
		if orden[i] != otra[i] {
			t.Fatal("el orden 90 no es determinista")
		}
	}
	for n := range grids {
		for r := 0; r < 3; r++ {
			for c := 0; c < 9; c++ {
				if grids[n][r][c] != otros[n][r][c] {
					t.Fatal("los tickets 90 no son deterministas")
				}
			}
		}
	}
}

func rangoColumna90Extremos(c int) (int, int) {
	switch {
	case c == 0:
		return 1, 9
	case c == 8:
		return 80, 90
	default:
		return c * 10, c*10 + 9
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

// TestBingoPatronesVariantes cubre Cuatro Esquinas y Patrón Diamante.
func TestBingoPatronesVariantes(t *testing.T) {
	grid := [][]int{
		{1, 16, 31, 46, 61},
		{2, 17, 32, 47, 62},
		{3, 18, 0, 48, 63},
		{4, 19, 33, 49, 64},
		{5, 20, 34, 50, 65},
	}
	grids := [][][]int{grid}

	// Cuatro esquinas: 1, 61, 5, 65.
	premio, _, lleno, tiers, patron := BingoPremioVariante(1000, grids, []int{1, 61, 5, 65}, "bingo-esquinas")
	if premio != 9000 || lleno || tiers[0] != 9 || patron != "Cuatro esquinas" {
		t.Errorf("esquinas: premio %d lleno %v tiers %v patron %q", premio, lleno, tiers, patron)
	}
	// Sin esquinas no paga.
	premio, _, _, tiers, _ = BingoPremioVariante(1000, grids, []int{1, 61, 5}, "bingo-esquinas")
	if premio != 0 || tiers[0] != 0 {
		t.Errorf("esquinas incompletas: premio %d tiers %v", premio, tiers)
	}

	// Diamante: centro libre + 32, 18, 48, 33.
	premio, _, lleno, tiers, patron = BingoPremioVariante(1000, grids, []int{32, 18, 48, 33}, "bingo-diamante")
	if premio != 9000 || lleno || tiers[0] != 9 || patron != "Diamante" {
		t.Errorf("diamante: premio %d lleno %v tiers %v patron %q", premio, lleno, tiers, patron)
	}
	premio, _, _, tiers, _ = BingoPremioVariante(1000, grids, []int{32, 18, 48}, "bingo-diamante")
	if premio != 0 || tiers[0] != 0 {
		t.Errorf("diamante incompleto: premio %d tiers %v", premio, tiers)
	}

	// Cartón lleno en las dos variantes: x25.
	todas := []int{}
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			if grid[r][c] != 0 {
				todas = append(todas, grid[r][c])
			}
		}
	}
	for _, codigo := range []string{"bingo-esquinas", "bingo-diamante"} {
		premio, _, lleno, tiers, _ := BingoPremioVariante(1000, grids, todas, codigo)
		if premio != 25000 || !lleno || tiers[0] != 25 {
			t.Errorf("%s lleno: premio %d lleno %v tiers %v", codigo, premio, lleno, tiers)
		}
	}
}

// TestBingo90Premios cubre una línea, dos líneas y cartón lleno del ticket 3x9.
func TestBingo90Premios(t *testing.T) {
	grid := [][]int{
		{1, 10, 20, 30, 0, 0, 60, 0, 0},
		{2, 11, 21, 31, 0, 0, 0, 70, 0},
		{3, 12, 0, 0, 40, 50, 0, 0, 80},
	}
	grids := [][][]int{grid}
	fila0 := []int{1, 10, 20, 30, 60}
	fila1 := []int{2, 11, 21, 31, 70}
	fila2 := []int{3, 12, 40, 50, 80}

	premio, lineas, lleno, tiers, patron := BingoPremioVariante(1000, grids, fila0, "bingo90")
	if premio != 2000 || lineas != 1 || lleno || tiers[0] != 2 || patron != "Una linea" {
		t.Errorf("1 linea: premio %d lineas %d lleno %v tiers %v patron %q", premio, lineas, lleno, tiers, patron)
	}
	dos := append(append([]int{}, fila0...), fila1...)
	premio, lineas, lleno, tiers, patron = BingoPremioVariante(1000, grids, dos, "bingo90")
	if premio != 5000 || lineas != 2 || lleno || tiers[0] != 5 || patron != "Dos lineas" {
		t.Errorf("2 lineas: premio %d lineas %d lleno %v tiers %v patron %q", premio, lineas, lleno, tiers, patron)
	}
	tres := append(append([]int{}, dos...), fila2...)
	premio, lineas, lleno, tiers, patron = BingoPremioVariante(1000, grids, tres, "bingo90")
	if premio != 40000 || lineas != 3 || !lleno || tiers[0] != 40 || patron != "Carton lleno" {
		t.Errorf("lleno: premio %d lineas %d lleno %v tiers %v patron %q", premio, lineas, lleno, tiers, patron)
	}
}

// TestBingoRelampagoPremios cubre la tabla de aciertos.
func TestBingoRelampagoPremios(t *testing.T) {
	grid := [][]int{
		{1, 16, 31, 46, 61},
		{2, 17, 32, 47, 62},
		{3, 18, 0, 48, 63},
		{4, 19, 33, 49, 64},
		{5, 20, 34, 50, 65},
	}
	grids := [][][]int{grid}
	casos := []struct {
		nombre   string
		cantadas []int
		mult     int64
	}{
		{"4 aciertos", []int{1, 16, 31, 46}, 0},
		{"5 aciertos", []int{1, 16, 31, 46, 61}, 0},
		{"6 aciertos", []int{1, 16, 31, 46, 61, 2}, 1},
		{"8 aciertos", []int{1, 16, 31, 46, 61, 2, 17, 32}, 1},
		{"9 aciertos", []int{1, 16, 31, 46, 61, 2, 17, 32, 47}, 2},
		{"10 aciertos", []int{1, 16, 31, 46, 61, 2, 17, 32, 47, 62}, 3},
		{"11 aciertos", []int{1, 16, 31, 46, 61, 2, 17, 32, 47, 62, 3}, 4},
		{"12 aciertos", []int{1, 16, 31, 46, 61, 2, 17, 32, 47, 62, 3, 18}, 5},
	}
	for _, c := range casos {
		premio, _, _, tiers, _ := BingoPremioVariante(1000, grids, c.cantadas, "bingo-relampago")
		if tiers[0] != c.mult || premio != 1000*c.mult {
			t.Errorf("%s: tiers %v premio %d, esperaba mult %d", c.nombre, tiers, premio, c.mult)
		}
	}
}

// TestBingoMultMaximo verifica el tope por variante.
func TestBingoMultMaximo(t *testing.T) {
	casos := map[string]int64{
		"bingo":           100,
		"bingo90":         160,
		"bingo-esquinas":  100,
		"bingo-diamante":  100,
		"bingo-relampago": 20,
	}
	for codigo, esperado := range casos {
		if got := BingoMultMaximo(codigo); got != esperado {
			t.Errorf("%s: mult maximo %d, esperaba %d", codigo, got, esperado)
		}
	}
	if got := BingoMultMaximo("no-existe"); got != 0 {
		t.Errorf("variante inexistente: %d", got)
	}
}

// TestBingoVariantesDeclaradas verifica que el catálogo del motor tenga las
// variantes activas con la geometría esperada.
func TestBingoVariantesDeclaradas(t *testing.T) {
	esperadas := map[string]BingoVariante{
		"bingo":           {Bolillas: 75, Filas: 5, Columnas: 5, TopeCantadas: 42},
		"bingo90":         {Bolillas: 90, Filas: 3, Columnas: 9, TopeCantadas: 60},
		"bingo-esquinas":  {Bolillas: 75, Filas: 5, Columnas: 5, TopeCantadas: 42},
		"bingo-diamante":  {Bolillas: 75, Filas: 5, Columnas: 5, TopeCantadas: 42},
		"bingo-relampago": {Bolillas: 75, Filas: 5, Columnas: 5, TopeCantadas: 20},
	}
	if len(bingoVariantes) != len(esperadas) {
		t.Fatalf("hay %d variantes, esperaba %d", len(bingoVariantes), len(esperadas))
	}
	for codigo, esp := range esperadas {
		v, ok := BingoVariantePorCodigo(codigo)
		if !ok {
			t.Fatalf("falta la variante %s", codigo)
		}
		if v.Bolillas != esp.Bolillas || v.Filas != esp.Filas || v.Columnas != esp.Columnas || v.TopeCantadas != esp.TopeCantadas {
			t.Errorf("%s: geometria %+v, esperaba %+v", codigo, v, esp)
		}
	}
}

// TestRTPBingo reporta el retorno medido por simulación (no es un assert).
func TestRTPBingo(t *testing.T) {
	t.Logf("bingo: RTP %.2f%%", RTPBingo(iteracionesRTP()))
}

// iteracionesRTP permite medir con mas iteraciones sin arrastrar el suite:
//
//	BINGO_RTP_ITER=100000 go test ./internal/casino -run TestRTPBingoVariantes -v
func iteracionesRTP() int {
	if v := os.Getenv("BINGO_RTP_ITER"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 20000
}

// TestRTPBingoVariantes mide el RTP de cada variante y verifica que esté dentro
// de la regla del proyecto (78% a 92%).
func TestRTPBingoVariantes(t *testing.T) {
	for _, v := range BingoVariantes() {
		rtp := RTPBingoVariante(v.Codigo, iteracionesRTP())
		t.Logf("%s (%s): RTP %.2f%%", v.Codigo, v.Nombre, rtp)
		if rtp < 78 || rtp > 92 {
			t.Errorf("%s: RTP %.2f%% fuera de rango [78, 92]", v.Codigo, rtp)
		}
	}
}
