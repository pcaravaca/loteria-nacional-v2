package casino

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// Prueba cruzada del motor gemelo: las trazas y sus resultados los genera el
// motor TypeScript (pinballFisica.ts) con gen_trazas.mjs, y acá el motor Go
// tiene que reproducir el mismo puntaje. Si este test falla, los dos motores
// divergen y la verificación del servidor no sirve.

type trazaFixture struct {
	Nombre   string          `json:"nombre"`
	Eventos  []EventoPinball `json:"eventos"`
	Esperado struct {
		Puntaje      int   `json:"puntaje"`
		PorBola      []int `json:"porBola"`
		Frames       int   `json:"frames"`
		Lanzamientos int   `json:"lanzamientos"`
	} `json:"esperado"`
}

func TestMotorGemelo(t *testing.T) {
	raw, err := os.ReadFile("testdata/pinball_trazas.json")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	var casos []trazaFixture
	if err := json.Unmarshal(raw, &casos); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(casos) == 0 {
		t.Fatal("fixture vacía")
	}
	fallan := 0
	for _, c := range casos {
		got := SimularPinball(c.Eventos)
		if got.Puntaje != c.Esperado.Puntaje || !reflect.DeepEqual(got.PorBola, c.Esperado.PorBola) ||
			got.Frames != c.Esperado.Frames || got.Lanzamientos != c.Esperado.Lanzamientos {
			fallan++
			t.Errorf("%s: TS=%+v GO=%+v", c.Nombre, c.Esperado, got)
		}
	}
	t.Logf("casos reproducidos idéntico: %d/%d", len(casos)-fallan, len(casos))
	if fallan > 0 {
		t.Fatalf("%d casos divergieron entre TS y GO", fallan)
	}
}
