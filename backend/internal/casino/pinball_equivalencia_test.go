package casino

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

// TestPinballEquivalenciaTS fija la equivalencia entre el motor del frontend
// (pinballFisica.ts) y este. El fixture trae 200 jugadas generadas por el motor
// TypeScript con un RNG propio: si una traza deja de reproducir el mismo
// puntaje en Go, el servidor pagaría distinto de lo que el jugador vio en
// pantalla (premio fantasma o jugada rechazada).

type eqEv struct {
	F int     `json:"f"`
	K string  `json:"k"`
	V float64 `json:"v"`
}

type eqCaso struct {
	N         int    `json:"n"`
	Eventos   []eqEv `json:"eventos"`
	PuntajeTS int    `json:"puntajeTS"`
	Bolas     int    `json:"bolas"`
	Frames    int    `json:"frames"`
}

func leerTrazasTS(t *testing.T) []eqCaso {
	t.Helper()
	f, err := os.Open("testdata/pinball_trazas.json.gz")
	if err != nil {
		t.Skipf("sin fixture de equivalencia: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	var casos []eqCaso
	if err := json.NewDecoder(zr).Decode(&casos); err != nil {
		t.Fatalf("json: %v", err)
	}
	return casos
}

func TestPinballEquivalenciaTS(t *testing.T) {
	casos := leerTrazasTS(t)
	dif := 0
	for _, c := range casos {
		sim := nuevoPinballSim()
		porFrame := make(map[int][]EventoPinball, len(c.Eventos))
		maxF := 0
		for _, e := range c.Eventos {
			porFrame[e.F] = append(porFrame[e.F], EventoPinball{F: e.F, K: e.K, V: e.V})
			if e.F > maxF {
				maxF = e.F
			}
		}
		for f := 0; f <= maxF && sim.bolaN < 3; f++ {
			sim.paso(f, porFrame[f])
		}
		sim.cerrar()
		if sim.puntaje != c.PuntajeTS || sim.bolaN != c.Bolas {
			dif++
			if dif <= 5 {
				t.Errorf("caso %d: TS=%d pts/%d bolas, Go=%d pts/%d bolas",
					c.N, c.PuntajeTS, c.Bolas, sim.puntaje, sim.bolaN)
			}
		}
	}
	if dif > 0 {
		t.Fatalf("%d de %d casos divergen entre TS y Go", dif, len(casos))
	}
	t.Logf("%d trazas reproducidas con el mismo puntaje y número de bolas", len(casos))
}
