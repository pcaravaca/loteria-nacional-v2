package resultados

import (
	"os"
	"testing"
	"time"
)

// Los números esperados son los que la fuente publica para el 2026-11-20 y
// coinciden con los que ya había importado la fuente principal (07/31/33).
func TestExtraerYeluNuevosTiempos(t *testing.T) {
	doc, err := os.ReadFile("testdata/yelu-nt.html")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	fecha := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	rs := extraerYelu(JuegoNuevosTiempos, string(doc), fecha, "test")
	if len(rs) != 3 {
		t.Fatalf("esperaba 3 turnos, dio %d", len(rs))
	}
	porTurno := map[string][]int{}
	for _, r := range rs {
		porTurno[r.Turno] = r.Numeros
	}
	casos := []struct {
		turno string
		num   int
	}{
		{TurnoMedioDia, 7},
		{TurnoTarde, 31},
		{TurnoNoche, 33},
	}
	for _, c := range casos {
		got := porTurno[c.turno]
		if len(got) != 1 || got[0] != c.num {
			t.Errorf("turno %s: esperaba [%d], dio %v", c.turno, c.num, got)
		}
	}
}

// Una fecha que la fuente no publica devuelve vacío, sin error.
func TestExtraerYeluFechaSinDatos(t *testing.T) {
	doc, err := os.ReadFile("testdata/yelu-nt.html")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	fecha := time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)
	if rs := extraerYelu(JuegoNuevosTiempos, string(doc), fecha, "test"); len(rs) != 0 {
		t.Fatalf("esperaba 0 resultados, dio %d", len(rs))
	}
}

// Los turnos que no reconozco no se usan: graduar mal es peor que no graduar.
func TestTurnoYeluDesconocido(t *testing.T) {
	if turnoYelu("MARATÓN") != "" {
		t.Fatal("un turno desconocido no debe mapear")
	}
	if turnoYelu("NOCHE") != TurnoNoche {
		t.Fatal("NOCHE debe mapear a noche")
	}
}

// "SERIE: 245" no es un número ganador y no debe colarse.
func TestNumerosYeluIgnoraSerie(t *testing.T) {
	h := `<div class="lotto_no_r bbb1">76</div><div class="lotto_no_r">SERIE: 245</div>`
	got := numerosYelu(h)
	if len(got) != 1 || got[0] != 76 {
		t.Fatalf("esperaba [76], dio %v", got)
	}
}
