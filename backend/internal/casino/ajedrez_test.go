package casino

import "testing"

func TestAjedrezPagos(t *testing.T) {
	if got := AjedrezPremio(1000, true); got != 3500 {
		t.Errorf("premio por acertar = %d, esperaba 3500", got)
	}
	if got := AjedrezPremio(1000, false); got != 0 {
		t.Errorf("premio por fallar = %d, esperaba 0", got)
	}
}

func TestAjedrezPuzzlesVerificados(t *testing.T) {
	if len(AjedrezPuzzles) < 8 {
		t.Fatalf("pocos puzzles: %d", len(AjedrezPuzzles))
	}
	for _, p := range AjedrezPuzzles {
		if len(p.Decoys) < AjedrezOpciones-1 {
			t.Errorf("%s: faltan distractoras", p.FEN)
		}
		vistos := map[string]bool{p.Correcta: true}
		for _, d := range p.Decoys {
			if vistos[d] {
				t.Errorf("%s: distractoria repetida %s", p.FEN, d)
			}
			vistos[d] = true
		}
	}
}

func TestAjedrezElegirDeterminista(t *testing.T) {
	p1, o1 := AjedrezElegir("cli", "srv", 11)
	p2, o2 := AjedrezElegir("cli", "srv", 11)
	if p1.FEN != p2.FEN {
		t.Fatalf("la posicion no es determinista")
	}
	if len(o1) != AjedrezOpciones {
		t.Fatalf("opciones = %d", len(o1))
	}
	for i := range o1 {
		if o1[i] != o2[i] {
			t.Fatalf("las opciones no son deterministicas")
		}
	}
	encontrada := false
	for _, o := range o1 {
		if o == p1.Correcta {
			encontrada = true
		}
	}
	if !encontrada {
		t.Errorf("la jugada correcta no esta entre las opciones")
	}
}

func TestAjedrezRepetir(t *testing.T) {
	p, opciones := AjedrezElegir("cli", "srv", 5)
	_, _, acierto := AjedrezRepetir("cli", "srv", 5, []string{opciones[0]})
	esperado := opciones[0] == p.Correcta
	if acierto != esperado {
		t.Errorf("acierto = %v, esperaba %v", acierto, esperado)
	}
	if _, _, acierto := AjedrezRepetir("cli", "srv", 5, nil); acierto {
		t.Errorf("sin jugadas no puede ser acierto")
	}
}

func TestRTPAjedrezEnBanda(t *testing.T) {
	rtp := RTPAjedrez(5000)
	t.Logf("RTP ajedrez: %.2f%%", rtp)
	if rtp < 78 || rtp > 92 {
		t.Errorf("RTP ajedrez = %.2f, fuera de banda 78-92", rtp)
	}
}
