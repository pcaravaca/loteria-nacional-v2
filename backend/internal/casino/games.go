package casino

import (
	"fmt"
)

// Params son los parametros de la jugada que envia el cliente (su eleccion).
type Params map[string]any

func (p Params) Int(key string) (int, error) {
	v, ok := p[key]
	if !ok {
		return 0, fmt.Errorf("falta el parametro %s", key)
	}
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	case int64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("parametro %s invalido", key)
	}
}

func (p Params) String(key string) (string, error) {
	v, ok := p[key]
	if !ok {
		return "", fmt.Errorf("falta el parametro %s", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("parametro %s invalido", key)
	}
	return s, nil
}

func (p Params) Ints(key string) ([]int, error) {
	v, ok := p[key]
	if !ok {
		return nil, fmt.Errorf("falta el parametro %s", key)
	}
	arr, ok := v.([]any)
	if !ok {
		if directo, ok2 := v.([]int); ok2 {
			return directo, nil
		}
		return nil, fmt.Errorf("parametro %s invalido", key)
	}
	out := make([]int, 0, len(arr))
	for _, item := range arr {
		f, ok := item.(float64)
		if !ok {
			return nil, fmt.Errorf("parametro %s invalido", key)
		}
		out = append(out, int(f))
	}
	return out, nil
}

// Outcome es lo que hay que pagar y lo que hay que animar en el cliente.
type Outcome struct {
	Multiplicador int64         `json:"multiplicador"`
	Data          map[string]any `json:"data"`
}

// ---------------------------------------------------------------- dados

var dadosMultipl = map[int]int64{
	2: 30, 3: 15, 4: 10, 5: 8, 6: 6, 7: 5,
	8: 6, 9: 8, 10: 10, 11: 15, 12: 30,
}

func jugarDados(monto int64, draws []int, p Params) (Outcome, error) {
	elegida, err := p.Int("suma")
	if err != nil {
		return Outcome{}, err
	}
	if elegida < 2 || elegida > 12 {
		return Outcome{}, fmt.Errorf("la suma debe estar entre 2 y 12")
	}
	if len(draws) < 2 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	d1 := draws[0]%6 + 1
	d2 := draws[1]%6 + 1
	suma := d1 + d2
	mult := int64(0)
	if suma == elegida {
		mult = dadosMultipl[suma]
	}
	return Outcome{
		Multiplicador: mult,
		Data: map[string]any{
			"dado1":   d1,
			"dado2":   d2,
			"suma":    suma,
			"elegida": elegida,
		},
	}, nil
}

// ---------------------------------------------------------------- slots

type slotSimbolo struct {
	Nombre string
	Peso   int
	Triple int64
	Par    int64 // con los dos primeros rodillos
}

var slotsSimbolos = []slotSimbolo{
	{"cereza", 30, 8, 1},
	{"campana", 24, 18, 3},
	{"herradura", 20, 5, 5},
	{"barra", 16, 2, 2},
	{"diamante", 10, 25, 8},
	{"siete", 6, 50, 20},
}

var slotsPesoTotal = func() int {
	t := 0
	for _, s := range slotsSimbolos {
		t += s.Peso
	}
	return t
}()

func slotPorDraw(draw int) slotSimbolo {
	r := draw % slotsPesoTotal
	acum := 0
	for _, s := range slotsSimbolos {
		acum += s.Peso
		if r < acum {
			return s
		}
	}
	return slotsSimbolos[len(slotsSimbolos)-1]
}

func jugarSlots(monto int64, draws []int, p Params) (Outcome, error) {
	if len(draws) < 3 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	a := slotPorDraw(draws[0])
	b := slotPorDraw(draws[1])
	c := slotPorDraw(draws[2])

	var mult int64
	switch {
	case a.Nombre == b.Nombre && b.Nombre == c.Nombre:
		mult = a.Triple
	case a.Nombre == b.Nombre:
		mult = a.Par
	}
	return Outcome{
		Multiplicador: mult,
		Data: map[string]any{
			"rodillos": []string{a.Nombre, b.Nombre, c.Nombre},
		},
	}, nil
}

// ---------------------------------------------------------------- ruleta

var ruletaRojos = map[int]bool{
	1: true, 3: true, 5: true, 7: true, 9: true, 12: true, 14: true, 16: true,
	18: true, 19: true, 21: true, 23: true, 25: true, 27: true, 30: true,
	32: true, 34: true, 36: true,
}

func ruletaColor(n int) string {
	switch {
	case n == 0:
		return "verde"
	case ruletaRojos[n]:
		return "rojo"
	default:
		return "negro"
	}
}

func jugarRuleta(monto int64, draws []int, p Params) (Outcome, error) {
	if len(draws) < 1 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	numero := draws[0] % 37
	tipo, err := p.String("tipo")
	if err != nil {
		return Outcome{}, err
	}
	color := ruletaColor(numero)

	var mult int64
	switch tipo {
	case "pleno":
		n, err := p.Int("numero")
		if err != nil {
			return Outcome{}, err
		}
		if n < 0 || n > 36 {
			return Outcome{}, fmt.Errorf("el numero debe estar entre 0 y 36")
		}
		if n == numero {
			mult = 35
		}
	case "color":
		c, err := p.String("color")
		if err != nil {
			return Outcome{}, err
		}
		if c != "rojo" && c != "negro" {
			return Outcome{}, fmt.Errorf("el color debe ser rojo o negro")
		}
		if c == color {
			mult = 2
		}
	default:
		return Outcome{}, fmt.Errorf("tipo de apuesta invalido")
	}

	return Outcome{
		Multiplicador: mult,
		Data: map[string]any{
			"numero": numero,
			"color":  color,
			"tipo":   tipo,
		},
	}, nil
}

// ---------------------------------------------------------------- raspa

var raspaSimbolos = []string{"cereza", "campana", "siete", "diamante", "herradura", "barra"}

func jugarRaspa(monto int64, draws []int, p Params) (Outcome, error) {
	if len(draws) < 3 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	a := raspaSimbolos[draws[0]%len(raspaSimbolos)]
	b := raspaSimbolos[draws[1]%len(raspaSimbolos)]
	c := raspaSimbolos[draws[2]%len(raspaSimbolos)]

	var mult int64
	switch {
	case a == b && b == c:
		mult = 18
	case a == b || b == c || a == c:
		mult = 1
	}
	return Outcome{
		Multiplicador: mult,
		Data: map[string]any{
			"simbolos": []string{a, b, c},
		},
	}, nil
}

// ---------------------------------------------------------------- keno

func kenoBolillas(draws []int) []int {
	vistos := map[int]bool{}
	out := make([]int, 0, 15)
	for _, d := range draws {
		n := (d % 60) + 1
		for vistos[n] {
			n = n%60 + 1
		}
		vistos[n] = true
		out = append(out, n)
		if len(out) == 15 {
			return out
		}
	}
	for n := 1; len(out) < 15 && n <= 60; n++ {
		if !vistos[n] {
			vistos[n] = true
			out = append(out, n)
		}
	}
	return out
}

var kenoMultipl = map[int]int64{2: 1, 3: 3, 4: 8, 5: 65}

func jugarKeno(monto int64, draws []int, p Params) (Outcome, error) {
	tuyos, err := p.Ints("numeros")
	if err != nil {
		return Outcome{}, err
	}
	if len(tuyos) != 5 {
		return Outcome{}, fmt.Errorf("debemos elegir exactamente 5 numeros")
	}
	vistos := map[int]bool{}
	for _, n := range tuyos {
		if n < 1 || n > 60 {
			return Outcome{}, fmt.Errorf("los numeros deben estar entre 1 y 60")
		}
		if vistos[n] {
			return Outcome{}, fmt.Errorf("los numeros no pueden repetirse")
		}
		vistos[n] = true
	}

	bolillas := kenoBolillas(draws)
	aciertos := 0
	for _, b := range bolillas {
		if vistos[b] {
			aciertos++
		}
	}
	mult := int64(0)
	if m, ok := kenoMultipl[aciertos]; ok {
		mult = m
	}
	return Outcome{
		Multiplicador: mult,
		Data: map[string]any{
			"bolillas":  bolillas,
			"tus_numeros": tuyos,
			"aciertos":  aciertos,
		},
	}, nil
}

// ---------------------------------------------------------------- despacho

// Jugar resuelve una jugada. El resultado depende unicamente de los draws y
// de la eleccion del jugador; el pago lo aplica el handler.
func Jugar(codigo string, monto int64, draws []int, p Params) (Outcome, error) {
	if p == nil {
		p = Params{}
	}
	switch codigo {
	case "dados":
		return jugarDados(monto, draws, p)
	case "slots":
		return jugarSlots(monto, draws, p)
	case "ruleta":
		return jugarRuleta(monto, draws, p)
	case "raspa":
		return jugarRaspa(monto, draws, p)
	case "keno":
		return jugarKeno(monto, draws, p)
	default:
		return Outcome{}, fmt.Errorf("juego no soportado: %s", codigo)
	}
}

// Draws indica cuantos numeros necesita cada juego para resolver una jugada.
func Draws(codigo string) int {
	switch codigo {
	case "dados", "slots", "raspa":
		return 3
	case "ruleta":
		return 1
	case "keno":
		return 24
	default:
		return 1
	}
}

// RTP aproximado por simulacion (util para afinar multiplicadores).
func RTP(codigo string, iteraciones int) float64 {
	if iteraciones <= 0 {
		iteraciones = 200000
	}
	var suma float64
	for i := 0; i < iteraciones; i++ {
		seed := fmt.Sprintf("sim-%d", i)
		draws := Sortear(seed, "sim", int64(i), Draws(codigo), 1<<31-1)
		var p Params
		switch codigo {
		case "dados":
			p = Params{"suma": 7}
		case "ruleta":
			p = Params{"tipo": "pleno", "numero": 17}
		case "keno":
			p = Params{"numeros": []int{1, 2, 3, 4, 5}}
		}
		out, err := Jugar(codigo, 1000, draws, p)
		if err != nil {
			continue
		}
		suma += float64(out.Multiplicador)
	}
	return suma / float64(iteraciones) * 100
}
