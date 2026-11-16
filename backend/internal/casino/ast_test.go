package casino

import (
	"math"
	"testing"
)

func TestCatalogoAST(t *testing.T) {
	if len(SlotsAST()) != 12 {
		t.Errorf("slots: esperaba 12 maquinas, hay %d", len(SlotsAST()))
	}
	if len(PokersAST()) != 8 {
		t.Errorf("poker: esperaba 8 mesas, hay %d", len(PokersAST()))
	}
	for _, s := range SlotsAST() {
		if s.Rodillos < 3 || s.Filas < 1 || len(s.Lineas) == 0 {
			t.Errorf("%s: geometria invalida (%d rodillos, %d filas, %d lineas)", s.Codigo, s.Rodillos, s.Filas, len(s.Lineas))
		}
		b, ok := s.Bandas[s.BandaPorDefecto]
		if !ok {
			t.Fatalf("%s: no tiene banda por defecto", s.Codigo)
		}
		if len(b.Pesos) != s.Rodillos {
			t.Errorf("%s: pesos con %d rodillos", s.Codigo, len(b.Pesos))
		}
		for _, linea := range s.Lineas {
			if len(linea) != s.Rodillos {
				t.Errorf("%s: linea con %d posiciones", s.Codigo, len(linea))
			}
			for _, fila := range linea {
				if fila < 1 || fila > s.Filas {
					t.Errorf("%s: fila %d fuera de rango", s.Codigo, fila)
				}
			}
		}
	}
}

func TestSlotDeterministaYRango(t *testing.T) {
	for _, s := range SlotsAST() {
		draws := Sortear("semilla-fija", "cli", 7, Draws(s.Codigo), 1<<31-1)
		primera, err := Jugar(s.Codigo, 1000, draws, Params{})
		if err != nil {
			t.Fatalf("%s: %v", s.Codigo, err)
		}
		segunda, err := Jugar(s.Codigo, 1000, draws, Params{})
		if err != nil {
			t.Fatalf("%s: %v", s.Codigo, err)
		}
		if primera.Multiplicador != segunda.Multiplicador {
			t.Errorf("%s: no es determinista", s.Codigo)
		}
		if primera.Multiplicador < 0 {
			t.Errorf("%s: multiplicador negativo", s.Codigo)
		}
		rodillos, ok := primera.Data["rodillos"].([][]string)
		if !ok || len(rodillos) != s.Rodillos {
			t.Errorf("%s: la ventana no llega completa", s.Codigo)
		}
		tabla, ok := primera.Data["tabla"].([]map[string]any)
		if !ok || len(tabla) == 0 {
			t.Errorf("%s: no trae la tabla de pagos", s.Codigo)
		}
		maximo := MaxMultiplSlot(s.Codigo)
		if maxima := primera.Multiplicador; maxima > maximo {
			t.Errorf("%s: pago x%d por encima del maximo x%d", s.Codigo, maxima, maximo)
		}
	}
}

func TestSlotRTPObjetivo(t *testing.T) {
	for _, s := range SlotsAST() {
		b := s.Bandas[s.BandaPorDefecto]
		ev := evLineaSlot(b) * float64(len(s.Lineas)) * 100
		objetivo := RTPObjetivo(s.BandaPorDefecto)
		if math.Abs(ev-objetivo) > 3.0 {
			t.Errorf("%s: RTP analitico %.2f lejos del objetivo %.2f", s.Codigo, ev, objetivo)
		}
	}
}

func TestPokerManos(t *testing.T) {
	cfg, _ := BuscarPoker("jacksorbetter")
	cp := func(r, p int) cartaPoker { return cartaPoker{Rango: r, Palo: p} }
	casos := []struct {
		cartas []cartaPoker
		cat    string
	}{
		{[]cartaPoker{cp(14, 0), cp(13, 0), cp(12, 0), cp(11, 0), cp(10, 0)}, "RoyalFlush"},
		{[]cartaPoker{cp(9, 1), cp(8, 1), cp(7, 1), cp(6, 1), cp(5, 1)}, "StraightFlush"},
		{[]cartaPoker{cp(2, 0), cp(2, 1), cp(2, 2), cp(2, 3), cp(14, 0)}, "FourOfAKind"},
		{[]cartaPoker{cp(7, 0), cp(7, 1), cp(7, 2), cp(11, 0), cp(11, 1)}, "FullHouse"},
		{[]cartaPoker{cp(2, 0), cp(5, 0), cp(9, 0), cp(11, 0), cp(13, 0)}, "Flush"},
		{[]cartaPoker{cp(14, 0), cp(13, 1), cp(12, 2), cp(11, 3), cp(10, 0)}, "Straight"},
		{[]cartaPoker{cp(6, 0), cp(6, 1), cp(6, 2), cp(9, 3), cp(11, 0)}, "ThreeOfAKind"},
		{[]cartaPoker{cp(6, 0), cp(6, 1), cp(9, 2), cp(9, 3), cp(13, 0)}, "TwoPair"},
		{[]cartaPoker{cp(14, 0), cp(14, 1), cp(5, 2), cp(7, 3), cp(9, 0)}, "Pair"},
		{[]cartaPoker{cp(14, 0), cp(11, 1), cp(8, 2), cp(6, 3), cp(4, 0)}, ""},
	}
	for i, c := range casos {
		cat, _, _ := evaluarPoker(cfg, c.cartas)
		if cat != c.cat {
			t.Errorf("caso %d: llego %q y esperaba %q", i, cat, c.cat)
		}
	}
}

func TestPokerPagos(t *testing.T) {
	cfg, _ := BuscarPoker("jacksorbetter")
	if pago := pagoPoker(cfg, "RoyalFlush", 0, 0); pago != cfg.RealMaxima {
		t.Errorf("real: pago %d, esperaba %d", pago, cfg.RealMaxima)
	}
	if pago := pagoPoker(cfg, "Pair", 14, 0); pago != 1 {
		t.Errorf("par alto: pago %d, esperaba 1", pago)
	}
	if pago := pagoPoker(cfg, "Pair", 5, 0); pago != 0 {
		t.Errorf("par bajo: pago %d, esperaba 0", pago)
	}
}

func TestDrawsAST(t *testing.T) {
	for _, s := range SlotsAST() {
		if Draws(s.Codigo) != s.Rodillos*s.Filas {
			t.Errorf("%s: Draws() = %d", s.Codigo, Draws(s.Codigo))
		}
	}
	for _, p := range PokersAST() {
		if Draws(p.Codigo) < 10 {
			t.Errorf("%s: Draws() = %d, alcanzan cortos", p.Codigo, Draws(p.Codigo))
		}
	}
	if MultiplicadorMaximo("spy", 0) <= 0 {
		t.Error("spy: MultiplicadorMaximo vacio")
	}
	if MultiplicadorMaximo("jacksorbetter", 0) != 800 {
		t.Errorf("jacksorbetter: maximo %d", MultiplicadorMaximo("jacksorbetter", 0))
	}
}
