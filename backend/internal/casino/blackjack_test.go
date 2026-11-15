package casino

import "testing"

func TestBlackjackValor(t *testing.T) {
	// Ojo: el indice de carta no es el valor. i%13 da la posicion y el valor es
	// v+1 para v<10 (10=J, 11=Q, 12=K), con el as (v=0) valiendo 11.
	casos := []struct {
		mano   []int
		valor  int
		soft   bool
		pasado bool
	}{
		{[]int{0, 10}, 21, true, false},    // A + J (as aun valiendo 11)
		{[]int{11, 5}, 16, false, false},   // Q + 6
		{[]int{12, 6}, 17, false, false},   // K + 7
		{[]int{0, 0}, 12, true, false},     // A + A
		{[]int{10, 9, 5}, 26, false, true}, // J + 10 + 6
		{[]int{0, 9, 8}, 20, false, false}, // A + 10 + 9 (el as baja a 1)
		{[]int{9, 9, 9}, 30, false, true},  // 10 + 10 + 10
	}
	for i, c := range casos {
		v, soft, pasado := ValorBlackjack(c.mano)
		if v != c.valor || soft != c.soft || pasado != c.pasado {
			t.Errorf("caso %d: mano %v dio (%d,%v,%v), esperaba (%d,%v,%v)",
				i, c.mano, v, soft, pasado, c.valor, c.soft, c.pasado)
		}
	}
}

func TestBlackjackNatural(t *testing.T) {
	if !BlackjackNatural([]int{0, 10}) {
		t.Error("A+J deberia ser natural")
	}
	if BlackjackNatural([]int{9, 10, 0}) {
		t.Error("tres cartas no son natural")
	}
}

func TestBlackjackResultado(t *testing.T) {
	casos := []struct {
		jugador  []int
		crupier  []int
		apostado int64
		estado   string
		premio   int64
	}{
		{[]int{0, 10}, []int{9, 10}, 1000, "blackjack", 2500}, // 3:2
		{[]int{9, 10}, []int{5, 10}, 1000, "ganada", 2000},
		{[]int{9, 10}, []int{9, 10}, 1000, "empate", 1000},
		{[]int{5, 10}, []int{9, 10}, 1000, "perdida", 0},
		{[]int{10, 9, 5}, []int{9, 10}, 1000, "perdida", 0}, // el jugador se paso
	}
	for i, c := range casos {
		estado, premio, _ := BlackjackResultado(c.jugador, c.crupier, c.apostado)
		if estado != c.estado || premio != c.premio {
			t.Errorf("caso %d: dio (%s,%d), esperaba (%s,%d)", i, estado, premio, c.estado, c.premio)
		}
	}
}

func TestBlackjackCrupierHasta17(t *testing.T) {
	// mazo donde la tercera carta del crupier es un 10 (indice 9)
	mazo := make([]int, 52)
	for i := range mazo {
		mazo[i] = i
	}
	crupier := []int{9, 5} // 10 + 4 = 14
	final, pos := CrupierHasta17(crupier, mazo, 0)
	v, _, _ := ValorBlackjack(final)
	if v < 17 {
		t.Errorf("el crupier deberia llegar a 17+, quedo en %d", v)
	}
	if pos == 0 {
		t.Error("la posicion del mazo no avanzo")
	}
}

func TestBarajaDeterminista(t *testing.T) {
	a := Baraja("cli-1", "servidor", 7, 64)
	b := Baraja("cli-1", "servidor", 7, 64)
	if len(a) != 52 {
		t.Fatalf("la baraja tiene %d cartas", len(a))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("la baraja no es determinista en %d: %d != %d", i, a[i], b[i])
		}
	}
	visto := map[int]bool{}
	for _, c := range a {
		if visto[c] {
			t.Fatalf("carta repetida %d", c)
		}
		visto[c] = true
	}
}

func TestBlackjackRendicion(t *testing.T) {
	casos := []struct {
		apostado int64
		premio   int64
	}{
		{1000, 500},
		{500, 250},
		{700, 350}, // impar: el premio es la mitad entera
		{1, 0},
	}
	for i, c := range casos {
		if p := BlackjackRendicion(c.apostado); p != c.premio {
			t.Errorf("caso %d: rendicion de %d dio %d, esperaba %d", i, c.apostado, p, c.premio)
		}
	}
}

func TestSeguroAplica(t *testing.T) {
	if !SeguroAplica([]int{0, 9}) { // As visible
		t.Error("con As del crupier el seguro deberia aplicar")
	}
	if SeguroAplica([]int{12, 9}) { // K visible
		t.Error("sin As visible el seguro no aplica")
	}
	if SeguroAplica(nil) {
		t.Error("sin cartas del crupier el seguro no aplica")
	}
}

func TestSeguroCostoYPaga(t *testing.T) {
	if c := SeguroCosto(1000); c != 500 {
		t.Errorf("costo del seguro de 1000 deberia ser 500, fue %d", c)
	}
	if p := SeguroPaga(1000); p != 1000 {
		t.Errorf("el seguro deberia pagar 1000 (2:1 sobre 500), pago %d", p)
	}
}
