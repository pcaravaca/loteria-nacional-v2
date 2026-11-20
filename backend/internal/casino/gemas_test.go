package casino

import "testing"

// tableroPrueba arma un tablero sin combinaciones armadas pero con una jugada
// válida: al intercambiar (0,2) con (1,2) se forma un grupo de tres amatistas.
func tableroPrueba() []int {
	return []int{
		2, 2, 0, 1, 0, 1, 0, 1,
		0, 1, 2, 0, 1, 0, 1, 0,
		1, 0, 1, 0, 1, 0, 1, 0,
		0, 1, 0, 1, 0, 1, 0, 1,
		1, 0, 1, 0, 1, 0, 1, 0,
		0, 1, 0, 1, 0, 1, 0, 1,
		1, 0, 1, 0, 1, 0, 1, 0,
		0, 1, 0, 1, 0, 1, 0, 1,
	}
}

// chorroDePrueba devuelve un chorro determinista para pruebas.
func chorroDePrueba() *GemasChorro { return NuevoGemasChorro("cli", "servidor", 1) }

// TestGemasSorteoDeterminista verifica que el tablero y el offset dependen solo
// de las semillas y el nonce, y que el tablero inicial no tiene combinaciones.
func TestGemasSorteoDeterminista(t *testing.T) {
	t1, o1 := GemasSorteo("cli-1", "servidor", 7)
	t2, o2 := GemasSorteo("cli-1", "servidor", 7)
	if o1 != o2 {
		t.Fatalf("offset %d != %d", o1, o2)
	}
	if len(t1) != GemasFilas*GemasColumnas {
		t.Fatalf("tablero de %d celdas", len(t1))
	}
	for i := range t1 {
		if t1[i] != t2[i] {
			t.Fatalf("tablero no determinista en %d", i)
		}
		if t1[i] < 0 || t1[i] >= GemasTiposBase {
			t.Fatalf("tipo invalido %d en %d", t1[i], i)
		}
	}
	if len(GemasPartidos(t1)) != 0 {
		t.Fatal("el tablero inicial ya tiene combinaciones")
	}
	if !GemasHayMovimiento(t1) {
		t.Fatal("el tablero inicial no tiene movimientos validos")
	}

	otro, _ := GemasSorteo("cli-2", "servidor", 7)
	iguales := true
	for i := range t1 {
		if t1[i] != otro[i] {
			iguales = false
			break
		}
	}
	if iguales {
		t.Fatal("distintas semillas produjeron el mismo tablero")
	}
}

// TestGemasPartidos cubre grupos horizontales y verticales.
func TestGemasPartidos(t *testing.T) {
	tab := tableroPrueba()
	if len(GemasPartidos(tab)) != 0 {
		t.Fatal("el tablero de prueba deberia estar limpio")
	}
	// Formamos una fila de tres y una columna de cuatro.
	for i := 0; i < 3; i++ {
		tab[gemasIdx(0, i)] = 4
	}
	for r := 0; r < 4; r++ {
		tab[gemasIdx(r, 5)] = 6
	}
	partidos := GemasPartidos(tab)
	if len(partidos) != 2 {
		t.Fatalf("partidos = %d, esperaba 2", len(partidos))
	}
	total := 0
	for _, p := range partidos {
		total += len(p)
		if len(p) < 3 {
			t.Fatalf("grupo de %d gemas", len(p))
		}
	}
	if total != 7 {
		t.Errorf("celdas en grupos = %d, esperaba 7", total)
	}
}

// TestGemasAplicar verifica un intercambio simple: puntos, limpieza y relleno.
func TestGemasAplicar(t *testing.T) {
	tab := tableroPrueba()
	chorro := chorroDePrueba()
	res, err := GemasAplicar(tab, chorro, 2, 10) // (0,2) <-> (1,2)
	if err != nil {
		t.Fatalf("aplicar: %v", err)
	}
	if res.Puntos != GemasTipos[2].Puntos {
		t.Errorf("puntos = %d, esperaba %d", res.Puntos, GemasTipos[2].Puntos)
	}
	if len(res.Pasos) != 1 {
		t.Fatalf("pasos = %d, esperaba 1", len(res.Pasos))
	}
	paso := res.Pasos[0]
	if len(paso.Eliminadas) != 3 {
		t.Errorf("eliminadas = %d, esperaba 3", len(paso.Eliminadas))
	}
	if len(paso.Especiales) != 0 {
		t.Errorf("no deberia crear gema rayo, creo %v", paso.Especiales)
	}
	for _, i := range paso.Eliminadas {
		if paso.Antes[i] != 2 {
			t.Errorf("la celda %d no era amatista", i)
		}
	}
	// El tablero resultante sigue siendo valido y sin combinaciones.
	if len(GemasPartidos(res.Tablero)) != 0 {
		t.Fatal("el tablero quedo con combinaciones sin resolver")
	}
	// El original no se muta.
	if tab[2] != 0 || tab[10] != 2 {
		t.Fatal("el tablero original se muto")
	}
}

// TestGemasIntercambioInvalido verifica que un intercambio que no forma grupo no
// gasta el movimiento ni toca el tablero.
func TestGemasIntercambioInvalido(t *testing.T) {
	tab := tableroPrueba()
	if _, err := GemasAplicar(tab, chorroDePrueba(), 0, 1); err == nil {
		t.Fatal("un intercambio sin combinacion deberia fallar")
	}
	if _, err := GemasAplicar(tab, chorroDePrueba(), 0, 8); err == nil {
		t.Fatal("un intercambio no adyacente deberia fallar")
	}
	if _, err := GemasAplicar(tab, chorroDePrueba(), -1, 0); err == nil {
		t.Fatal("una celda fuera de rango deberia fallar")
	}
}

// TestGemasEspecialYExplosion verifica que un grupo de cuatro deja una gema rayo
// y que al reventar limpia su fila y su columna.
func TestGemasEspecialYExplosion(t *testing.T) {
	tab := tableroPrueba()
	// Dejamos cuatro amatistas en la fila 0 al intercambiar (0,3) con (1,3).
	tab[gemasIdx(0, 2)] = 2
	tab[gemasIdx(0, 3)] = 0
	tab[gemasIdx(1, 3)] = 2
	res, err := GemasAplicar(tab, chorroDePrueba(), 3, 11)
	if err != nil {
		t.Fatalf("aplicar: %v", err)
	}
	if len(res.Pasos) != 1 {
		t.Fatalf("pasos = %d", len(res.Pasos))
	}
	paso := res.Pasos[0]
	if len(paso.Especiales) != 1 || paso.Especiales[0] != gemasIdx(0, 3) {
		t.Fatalf("especiales = %v, esperaba la celda movida", paso.Especiales)
	}
	if res.Puntos != GemasTipos[2].Puntos*2 {
		t.Errorf("puntos = %d, esperaba %d", res.Puntos, GemasTipos[2].Puntos*2)
	}
	if res.Tablero[gemasIdx(0, 3)] != GemasTipoRayo {
		t.Errorf("la celda especial quedo %d", res.Tablero[gemasIdx(0, 3)])
	}

	// Tres gemas rayo juntas revientan en cruz.
	rayos := make([]int, GemasFilas*GemasColumnas)
	for i := range rayos {
		rayos[i] = (i + 1) % GemasTiposBase
	}
	rayos[gemasIdx(3, 0)] = GemasTipoRayo
	rayos[gemasIdx(3, 1)] = GemasTipoRayo
	rayos[gemasIdx(3, 2)] = GemasTipoRayo
	// Nos aseguro que no haya otros grupos armados.
	for r := 0; r < GemasFilas; r++ {
		for c := 0; c < GemasColumnas; c++ {
			i := gemasIdx(r, c)
			if rayos[i] != GemasTipoRayo && r > 0 && rayos[gemasIdx(r-1, c)] == rayos[i] && r > 1 && rayos[gemasIdx(r-2, c)] == rayos[i] {
				rayos[i] = (rayos[i] + 1) % GemasTiposBase
			}
		}
	}
	if len(GemasPartidos(rayos)) == 0 {
		t.Fatal("el tablero de rayos deberia tener un grupo")
	}
	res2, err := GemasAplicar(rayos, chorroDePrueba(), gemasIdx(3, 0), gemasIdx(3, 1))
	if err != nil {
		t.Fatalf("aplicar rayos: %v", err)
	}
	// El grupo de tres rayos paga x40 y limpia la fila 3 y las columnas 0,1,2.
	if res2.Puntos < GemasPuntosRayo {
		t.Errorf("puntos = %d, esperaba al menos %d", res2.Puntos, GemasPuntosRayo)
	}
	limpiadas := map[int]bool{}
	for _, p := range res2.Pasos {
		for _, i := range p.Eliminadas {
			limpiadas[i] = true
		}
	}
	for c := 0; c < GemasColumnas; c++ {
		if !limpiadas[gemasIdx(3, c)] {
			t.Errorf("la fila 3 no se limpio en la columna %d", c)
		}
	}
	for r := 0; r < GemasFilas; r++ {
		if !limpiadas[gemasIdx(r, 0)] {
			t.Errorf("la columna 0 no se limpio en la fila %d", r)
		}
	}
}

// TestGemasRepetir verifica que la auditoria reproduce tablero y puntos.
func TestGemasRepetir(t *testing.T) {
	client, server := "cli", "servidor"
	tab, offset := GemasSorteo(client, server, 3)
	chorro := NuevoGemasChorro(client, server, 3)
	chorro.Saltar(offset)

	jugadas := [][2]int{}
	actual := append([]int{}, tab...)
	for m := 0; m < 5; m++ {
		movs := GemasMovimientosValidos(actual)
		if len(movs) == 0 {
			break
		}
		r, err := GemasAplicar(actual, chorro, movs[0][0], movs[0][1])
		if err != nil {
			t.Fatalf("movimiento %d: %v", m, err)
		}
		actual = r.Tablero
		jugadas = append(jugadas, movs[0])
	}
	chorro2 := NuevoGemasChorro(client, server, 3)
	chorro2.Saltar(offset)
	repetido, err := GemasRepetir(tab, chorro2, jugadas)
	if err != nil {
		t.Fatalf("repetir: %v", err)
	}
	if !igualarInts(repetido.Tablero, actual) {
		t.Fatal("el tablero reproducido no coincide")
	}
}

func igualarInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestGemasPremio fija la conversion de puntos a premio.
func TestGemasPremio(t *testing.T) {
	casos := []struct {
		monto, puntos, premio int64
	}{
		{1000, 0, 0},
		{1000, 300, 1000},
		{1000, 750, 2500},
		{1000, 37, 123},
		{0, 300, 0},
	}
	for _, c := range casos {
		if got := GemasPremio(c.monto, c.puntos); got != c.premio {
			t.Errorf("GemasPremio(%d,%d) = %d, esperaba %d", c.monto, c.puntos, got, c.premio)
		}
	}
}

// TestGemasRondaCompleta juega los 25 movimientos con la estrategia "primer
// movimiento valido" y verifica que el tablero nunca se traba.
func TestGemasRondaCompleta(t *testing.T) {
	client, server := "cli", "servidor"
	tab, offset := GemasSorteo(client, server, 11)
	chorro := NuevoGemasChorro(client, server, 11)
	chorro.Saltar(offset)
	var puntos int64
	for m := 0; m < GemasMovimientos; m++ {
		movs := GemasMovimientosValidos(tab)
		if len(movs) == 0 {
			t.Fatalf("sin movimientos en el movimiento %d", m)
		}
		r, err := GemasAplicar(tab, chorro, movs[0][0], movs[0][1])
		if err != nil {
			t.Fatalf("movimiento %d: %v", m, err)
		}
		tab = r.Tablero
		puntos += r.Puntos
		if len(GemasPartidos(tab)) != 0 {
			t.Fatalf("quedaron combinaciones sin resolver en el movimiento %d", m)
		}
	}
	if puntos <= 0 {
		t.Fatal("una ronda completa debería sumar puntos")
	}
	t.Logf("ronda completa: %d puntos (x%.2f), draws %d", puntos, float64(puntos)/100, chorro.Posicion())
}

// TestRTPGemas reporta el retorno medido por simulacion (no es un assert).
func TestRTPGemas(t *testing.T) {
	t.Logf("gemas (primer movimiento valido): RTP %.2f%%", RTPGemas(5000, "primer"))
	t.Logf("gemas (jugada hábil): RTP %.2f%%", RTPGemas(3000, "mayor"))
}
