package models

import "testing"

func TestPremioAciertoTotal(t *testing.T) {
	l := Loteria{CostoNumero: 1000, LongitudNum: 2, Multiplicador: 50}
	if got := l.Premio(1000, 2, 2); got != 50_000 {
		t.Fatalf("acierto total: esperaba 50000, dio %d", got)
	}
}

func TestPremioParcialConTabla(t *testing.T) {
	l := Loteria{CostoNumero: 1000, LongitudNum: 3, Multiplicador: 400,
		TablaPremios: `{"1":2,"2":15}`}
	casos := []struct {
		aciertos int
		espera   int64
	}{{1, 2000}, {2, 15000}, {0, 0}}
	for _, c := range casos {
		if got := l.Premio(1000, c.aciertos, 3); got != c.espera {
			t.Fatalf("aciertos=%d: esperaba %d, dio %d", c.aciertos, c.espera, got)
		}
	}
}

// Sin tabla declarada se mantiene la regla historica (parcial = x2) para no
// cambiar el comportamiento de las loterias ya sembradas.
func TestPremioParcialSinTabla(t *testing.T) {
	l := Loteria{CostoNumero: 1000, LongitudNum: 3, Multiplicador: 400}
	if got := l.Premio(1000, 1, 3); got != 2000 {
		t.Fatalf("parcial sin tabla: esperaba 2000, dio %d", got)
	}
}

func TestTablaPremiosPorAciertosInvalida(t *testing.T) {
	l := Loteria{TablaPremios: "{no-json"}
	if len(l.TablaPremiosPorAciertos()) != 0 {
		t.Fatal("una tabla invalida debe devolver mapa vacio")
	}
}