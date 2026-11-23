package casino

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Pinball: tres mesas que comparten el mismo motor físico determinista.
//
// A diferencia del resto de los juegos, el resultado NO sale de los draws: sale
// de la traza de inputs del jugador (flancos de los flippers y lanzamientos).
// El servidor replica la partida con SimularPinball y obtiene el puntaje; el
// multiplicador sale de una tabla de tramos que define la casa.
//
// Provably fair: con el server seed, el client seed, el nonce y la traza
// guardada en Resultado se puede reproducir el puntaje exacto.

// pinballCodigos son las tres mesas que comparten el motor.
var pinballCodigos = map[string]bool{
	"pinball-neon":    true,
	"pinball-clasico": true,
	"plinko-premium":  true,
}

// Retorno al jugador declarado por la casa, medido sobre el motor con bots
// (ver pinball_rtp_test.go): el perfil casual y el perfil hábil.
const (
	pinballRTPCasual = 29.8
	pinballRTPHabil  = 79.8
)

type pinballTramo struct {
	Desde int
	Mult  int64
}

// pinballTramos: puntos de la partida (3 bolas) -> multiplicador entero.
// Calibrado sobre la distribucion medida del motor (TestPinballRTP con 200 000
// partidas por perfil): el jugador casual retiene ~32% y el jugador habil
// cierra en ~83%, dentro de la banda 78-92% del catalogo. El techo de premio
// por jugada y la apuesta maxima son lo que limitan la exposicion.
var pinballTramos = []pinballTramo{
	{0, 0},
	{900, 1},
	{1500, 2},
	{2400, 3},
	{3600, 5},
	{5000, 7},
}

const pinballMaxEventos = 5000

func pinballMultiplicador(puntos int) int64 {
	mult := int64(0)
	for _, t := range pinballTramos {
		if puntos >= t.Desde {
			mult = t.Mult
		}
	}
	return mult
}

func pinballTramoDe(puntos int) pinballTramo {
	tramo := pinballTramos[0]
	for _, t := range pinballTramos {
		if puntos >= t.Desde {
			tramo = t
		}
	}
	return tramo
}

// pinballMaxMult es el multiplicador máximo que paga la mesa.
func pinballMaxMult(codigo string) int64 {
	var max int64
	for _, t := range pinballTramos {
		if t.Mult > max {
			max = t.Mult
		}
	}
	return max
}

// eventosPinball extrae la traza del jugador de Params. El handler manda los
// params tal como llegan del JSON, así que hay que aceptar varias formas.
func eventosPinball(p Params) ([]EventoPinball, error) {
	raw, ok := p["eventos"]
	if !ok {
		return nil, fmt.Errorf("falta la traza de la partida")
	}
	var evs []EventoPinball
	switch v := raw.(type) {
	case []EventoPinball:
		evs = v
	case []map[string]any:
		for _, m := range v {
			ev, err := eventoPinballDeMapa(m)
			if err != nil {
				return nil, err
			}
			evs = append(evs, ev)
		}
	case []any:
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("traza mal formada")
			}
			ev, err := eventoPinballDeMapa(m)
			if err != nil {
				return nil, err
			}
			evs = append(evs, ev)
		}
	default:
		return nil, fmt.Errorf("traza mal formada")
	}
	if len(evs) == 0 {
		return nil, fmt.Errorf("traza vacia")
	}
	if len(evs) > pinballMaxEventos {
		return nil, fmt.Errorf("traza demasiado grande")
	}
	lanzamientos := 0
	for _, ev := range evs {
		if ev.F < 0 || ev.F >= maxFrames {
			return nil, fmt.Errorf("frame fuera de rango")
		}
		switch ev.K {
		case "L", "R":
			if ev.V != 0 && ev.V != 1 {
				return nil, fmt.Errorf("valor de flipper invalido")
			}
		case "P":
			if ev.V < 0 || ev.V > 1 {
				return nil, fmt.Errorf("fuerza de lanzamiento invalida")
			}
			lanzamientos++
		default:
			return nil, fmt.Errorf("evento desconocido: %s", ev.K)
		}
	}
	if lanzamientos == 0 {
		return nil, fmt.Errorf("no se lanzo ninguna bola")
	}
	if lanzamientos > 3 {
		return nil, fmt.Errorf("mas de tres lanzamientos")
	}
	sort.SliceStable(evs, func(i, j int) bool { return evs[i].F < evs[j].F })
	return evs, nil
}

func eventoPinballDeMapa(m map[string]any) (EventoPinball, error) {
	var ev EventoPinball
	f, err := toFloat(m["f"])
	if err != nil {
		return ev, fmt.Errorf("frame invalido")
	}
	ev.F = int(f)
	k, ok := m["k"].(string)
	if !ok {
		return ev, fmt.Errorf("tipo de evento invalido")
	}
	ev.K = k
	v, err := toFloat(m["v"])
	if err != nil {
		return ev, fmt.Errorf("valor invalido")
	}
	ev.V = v
	return ev, nil
}

func toFloat(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case json.Number:
		return n.Float64()
	}
	return 0, fmt.Errorf("no es numero")
}

// EsPinball dice si un juego usa el motor de pinball.
func EsPinball(codigo string) bool { return pinballCodigos[codigo] }

// PinballRTPHabil y PinballRTPCasual exponen los RTP medidos del motor (200 000
// partidas por perfil, ver TestPinballRTP) para que el seed declare el mismo
// número que puede pagar. El hábil es el "rtp" de la ficha y el casual va en
// rtp_casual: es un juego con decisiones.
func PinballRTPHabil() float64  { return pinballRTPHabil }
func PinballRTPCasual() float64 { return pinballRTPCasual }

// EventosDeResultado saca la traza guardada en el Resultado de una ronda.
// Sirve para reproducir la jugada en la verificación (provably fair).
func EventosDeResultado(resultado any) ([]EventoPinball, error) {
	m, ok := resultado.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("resultado sin traza")
	}
	return eventosPinball(Params{"eventos": m["eventos"]})
}

func jugarPinball(codigo string, monto int64, draws []int, p Params) (Outcome, error) {
	if !pinballCodigos[codigo] {
		return Outcome{}, fmt.Errorf("mesa de pinball no valida: %s", codigo)
	}
	eventos, err := eventosPinball(p)
	if err != nil {
		return Outcome{}, err
	}
	res := SimularPinball(eventos)
	mult := pinballMultiplicador(res.Puntaje)
	data := map[string]any{
		"puntaje":      res.Puntaje,
		"porBola":      res.PorBola,
		"frames":       res.Frames,
		"lanzamientos": res.Lanzamientos,
		"tramos":       pinballTramos,
		"tramo":        pinballTramoDe(res.Puntaje),
		"eventos":      eventos,
		"rtpCasual":    pinballRTPCasual,
		"rtpHabil":     pinballRTPHabil,
	}
	return Outcome{Multiplicador: mult, Data: data}, nil
}
