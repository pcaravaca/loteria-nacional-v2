package casino

import (
	"math/rand"
	"os"
	"strconv"
	"testing"
)

// El pinball es un juego de habilidad: el resultado sale de la traza de inputs,
// no de los draws. Este test mide el RTP real del motor jugando con dos perfiles
// de bot (casual y hábil) y verifica que las cifras declaradas por la casa
// (pinballRTPCasual / pinballRTPHabil) sigan siendo ciertas. Si se cambia la
// tabla de tramos o la física, este test es el que avisa.

func jugarPinballBot(modo string, rng *rand.Rand) (int, []int) {
	sim := nuevoPinballSim()
	ultL, ultR := -99, -99
	for f := 0; f < maxFrames && sim.bolaN < 3; f++ {
		var evs []EventoPinball
		b := sim.bola
		if b == nil {
			var carga float64
			if modo == "casual" {
				carga = 0.4 + rng.Float64()*0.6
			} else {
				carga = 0.85 + rng.Float64()*0.15
			}
			evs = append(evs, EventoPinball{F: f, K: "P", V: carga})
		} else if modo == "casual" {
			if rng.Float64() < 0.03 {
				v := 0.0
				if rng.Float64() < 0.5 {
					v = 1
				}
				evs = append(evs, EventoPinball{F: f, K: "L", V: v})
			}
			if rng.Float64() < 0.03 {
				v := 0.0
				if rng.Float64() < 0.5 {
					v = 1
				}
				evs = append(evs, EventoPinball{F: f, K: "R", V: v})
			}
		} else {
			// hábil: pulsa el flipper del lado por el que cae la bola
			if b.vy > 0 && b.y > 230 && b.x < 160 && f-ultL > 15 {
				evs = append(evs, EventoPinball{F: f, K: "L", V: 1})
				ultL = f
			}
			if b.vy > 0 && b.y > 230 && b.x >= 160 && f-ultR > 15 {
				evs = append(evs, EventoPinball{F: f, K: "R", V: 1})
				ultR = f
			}
			if f-ultL == 7 {
				evs = append(evs, EventoPinball{F: f, K: "L", V: 0})
			}
			if f-ultR == 7 {
				evs = append(evs, EventoPinball{F: f, K: "R", V: 0})
			}
		}
		sim.paso(f, evs)
	}
	sim.cerrar()
	return sim.puntaje, sim.porBola
}

func medirRTPPinball(t *testing.T, modo string, iters int) (float64, float64) {
	rng := rand.New(rand.NewSource(20261123))
	var sumaMult float64
	var sumaPuntos int
	for i := 0; i < iters; i++ {
		puntos, _ := jugarPinballBot(modo, rng)
		sumaMult += float64(pinballMultiplicador(puntos))
		sumaPuntos += puntos
	}
	return sumaMult / float64(iters) * 100, float64(sumaPuntos) / float64(iters)
}

func TestPinballRTP(t *testing.T) {
	iters := 20000
	if v := os.Getenv("PINBALL_RTP_ITERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			iters = n
		}
	}
	rtpC, mediaC := medirRTPPinball(t, "casual", iters)
	rtpH, mediaH := medirRTPPinball(t, "habil", iters)
	t.Logf("casual -> RTP %.1f%% (media %d pts)", rtpC, int(mediaC))
	t.Logf("habil  -> RTP %.1f%% (media %d pts)", rtpH, int(mediaH))

	if rtpC < pinballRTPCasual-8 || rtpC > pinballRTPCasual+8 {
		t.Errorf("RTP casual %.1f%% se aparta de lo declarado %.1f%%", rtpC, pinballRTPCasual)
	}
	if rtpH < pinballRTPHabil-25 || rtpH > pinballRTPHabil+25 {
		t.Errorf("RTP hábil %.1f%% se aparta de lo declarado %.1f%%", rtpH, pinballRTPHabil)
	}
}

// TestPinballTrazaValidator verifica que una traza adulterada no pase.
func TestPinballTrazaValida(t *testing.T) {
	bien := Params{"eventos": []any{
		map[string]any{"f": 10, "k": "P", "v": 1},
	}}
	if _, err := eventosPinball(bien); err != nil {
		t.Fatalf("traza valida rechazada: %v", err)
	}
	malos := []Params{
		{"eventos": []any{}},
		{"eventos": "no"},
		{},
		{"eventos": []any{map[string]any{"f": -1, "k": "P", "v": 1}}},
		{"eventos": []any{map[string]any{"f": 1, "k": "X", "v": 1}}},
		{"eventos": []any{map[string]any{"f": 1, "k": "L", "v": 2}}},
		{"eventos": []any{map[string]any{"f": 1, "k": "P", "v": 2}}},
	}
	for i, p := range malos {
		if _, err := eventosPinball(p); err == nil {
			t.Errorf("caso %d: traza invalida aceptada", i)
		}
	}
}
