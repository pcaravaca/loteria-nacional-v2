package casino

import (
	"fmt"
	"sort"
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

// Premio devuelve lo que se paga por una jugada. El multiplicador es lo que
// audita el jugador (monto * multiplicador); algun juego puede ademas cobrar
// una comision sobre la ganancia, que el motor indica en Data["comision"]
// (el baccarat a la banca, que paga 0.95 por peso).
func Premio(monto int64, out Outcome) int64 {
	premio := monto * out.Multiplicador
	if out.Data != nil {
		if com, ok := out.Data["comision"].(int64); ok && com > 0 {
			premio -= com
		}
	}
	if premio < 0 {
		return 0
	}
	return premio
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

// ---------------------------------------------------------------- cascada (plinko)

// cascadaFilas es la cantidad de filas de pinchos; la bola termina en una de
// cascadaFilas+1 casillas. La tabla paga poco en el centro y mucho en los
// bordes, como el plinko clasico (RTP ~96.5%).
const cascadaFilas = 9

var cascadaTabla = []int64{10, 5, 3, 1, 0, 0, 1, 3, 5, 10}

func jugarCascada(monto int64, draws []int, p Params) (Outcome, error) {
	if len(draws) < cascadaFilas {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	camino := make([]int, 0, cascadaFilas)
	pos := 0
	for i := 0; i < cascadaFilas; i++ {
		d := draws[i] % 2
		camino = append(camino, d)
		pos += d
	}
	return Outcome{
		Multiplicador: cascadaTabla[pos],
		Data: map[string]any{
			"camino":   camino,
			"posicion": pos,
			"tabla":    cascadaTabla,
			"filas":    cascadaFilas,
		},
	}, nil
}

// ---------------------------------------------------------------- mayor o menor

var hiloFiguras = []string{"", "A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}

func hiloCarta(draw int) int { return draw%13 + 1 }

func jugarHilo(monto int64, draws []int, p Params) (Outcome, error) {
	if len(draws) < 2 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	dir, err := p.String("direccion")
	if err != nil {
		return Outcome{}, err
	}
	if dir != "mayor" && dir != "menor" {
		return Outcome{}, fmt.Errorf("la direccion debe ser mayor o menor")
	}
	base := hiloCarta(draws[0])
	siguiente := hiloCarta(draws[1])
	var mult int64
	switch {
	case dir == "mayor" && siguiente > base:
		mult = 2
	case dir == "menor" && siguiente < base:
		mult = 2
	}
	return Outcome{
		Multiplicador: mult,
		Data: map[string]any{
			"base":             base,
			"siguiente":        siguiente,
			"figura_base":      hiloFiguras[base],
			"figura_siguiente": hiloFiguras[siguiente],
			"direccion":        dir,
		},
	}, nil
}

// ---------------------------------------------------------------- bajo o sobre (dado 100)

// altoTopes mapea el tope elegido (porcentaje 1-100) con su multiplicador.
// El RTP depende del riesgo elegido (~96% en los extremos, menor en el medio).
var altoTopes = map[int]int64{2: 48, 5: 19, 10: 9, 25: 3, 48: 2, 75: 1, 90: 1, 95: 1}

func jugarAlto(monto int64, draws []int, p Params) (Outcome, error) {
	if len(draws) < 1 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	tope, err := p.Int("tope")
	if err != nil {
		return Outcome{}, err
	}
	mult, ok := altoTopes[tope]
	if !ok {
		return Outcome{}, fmt.Errorf("tope invalido: elegi uno de la lista")
	}
	direccion, _ := p.String("direccion")
	if direccion == "" {
		direccion = "bajo"
	}
	if direccion != "bajo" && direccion != "sobre" {
		return Outcome{}, fmt.Errorf("la direccion debe ser bajo o sobre")
	}
	tirada := draws[0]%100 + 1
	var gano bool
	if direccion == "bajo" {
		gano = tirada <= tope
	} else {
		gano = tirada > 100-tope
	}
	if !gano {
		mult = 0
	}
	return Outcome{
		Multiplicador: mult,
		Data: map[string]any{
			"tirada":     tirada,
			"tope":       tope,
			"direccion":  direccion,
			"tabla":      altoTopes,
		},
	}, nil
}

// ---------------------------------------------------------------- fortuna rapida (ruletta de premios)

var fortunaTabla = []int64{
	0, 0, 0, 0, 0, 0, 0, 0, 0,
	1, 1, 1, 1, 1, 1, 1,
	2, 2,
	3,
	5,
}

func jugarFortuna(monto int64, draws []int, p Params) (Outcome, error) {
	if len(draws) < 1 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	pos := draws[0] % len(fortunaTabla)
	return Outcome{
		Multiplicador: fortunaTabla[pos],
		Data: map[string]any{
			"posicion": pos,
			"tabla":    fortunaTabla,
			"casillas": len(fortunaTabla),
		},
	}, nil
}

// ---------------------------------------------------------------- cara o sello

func jugarMoneda(monto int64, draws []int, p Params) (Outcome, error) {
	if len(draws) < 1 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	lado, err := p.String("lado")
	if err != nil {
		return Outcome{}, err
	}
	if lado != "cara" && lado != "sello" {
		return Outcome{}, fmt.Errorf("el lado debe ser cara o sello")
	}
	salida := "cara"
	if draws[0]%2 == 1 {
		salida = "sello"
	}
	var mult int64
	if salida == lado {
		mult = 2
	}
	return Outcome{
		Multiplicador: mult,
		Data:          map[string]any{"salida": salida, "elegido": lado},
	}, nil
}

// ---------------------------------------------------------------- numero de la suerte (00-99)

func jugarSuertudo(monto int64, draws []int, p Params) (Outcome, error) {
	if len(draws) < 1 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	n, err := p.Int("numero")
	if err != nil {
		return Outcome{}, err
	}
	if n < 0 || n > 99 {
		return Outcome{}, fmt.Errorf("el numero debe estar entre 0 y 99")
	}
	salida := draws[0] % 100
	var mult int64
	if salida == n {
		mult = 75
	}
	return Outcome{
		Multiplicador: mult,
		Data:          map[string]any{"salida": salida, "elegido": n},
	}, nil
}

// ---------------------------------------------------------------- parley de 2 (2 de 20)

func jugarParley(monto int64, draws []int, p Params) (Outcome, error) {
	tuyos, err := p.Ints("numeros")
	if err != nil {
		return Outcome{}, err
	}
	if len(tuyos) != 2 {
		return Outcome{}, fmt.Errorf("debemos elegir exactamente 2 numeros")
	}
	vistos := map[int]bool{}
	for _, n := range tuyos {
		if n < 1 || n > 20 {
			return Outcome{}, fmt.Errorf("los numeros deben estar entre 1 y 20")
		}
		if vistos[n] {
			return Outcome{}, fmt.Errorf("los numeros no pueden repetirse")
		}
		vistos[n] = true
	}
	if len(draws) < 4 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	bolillaSet := map[int]bool{}
	for _, d := range draws {
		n := (d % 20) + 1
		for bolillaSet[n] {
			n = n%20 + 1
		}
		bolillaSet[n] = true
	}
	bolillas := make([]int, 0, len(bolillaSet))
	for n := range bolillaSet {
		bolillas = append(bolillas, n)
	}
	sort.Ints(bolillas)
	aciertos := 0
	for _, b := range bolillas {
		if vistos[b] {
			aciertos++
		}
	}
	var mult int64
	switch aciertos {
	case 2:
		mult = 18
	case 1:
		mult = 1
	}
	return Outcome{
		Multiplicador: mult,
		Data: map[string]any{
			"bolillas": bolillas,
			"tus_numeros": tuyos,
			"aciertos":  aciertos,
		},
	}, nil
}

// ---------------------------------------------------------------- baccarat

// baccaratCarta devuelve el valor de una carta para el baccarat: las figuras y
// los diez valen 0, el as vale 1 y el resto su numero.
func baccaratCarta(draw int) int { return draw % 10 }

func baccaratTotal(cartas []int) int {
	t := 0
	for _, c := range cartas {
		t += c
	}
	return t % 10
}

// jugarBaccarat reparte dos cartas por mano y aplica la regla de la tercera
// carta (el jugador pide con 0-5 y se planta con 6-7; la banca sigue la tabla
// oficial segun la tercera carta del jugador). El empate paga x8 y las apuestas
// a jugador o banca quedan en empate (se devuelve la apuesta).
func jugarBaccarat(monto int64, draws []int, p Params) (Outcome, error) {
	tipo, err := p.String("tipo")
	if err != nil {
		return Outcome{}, err
	}
	if tipo != "jugador" && tipo != "banca" && tipo != "empate" {
		return Outcome{}, fmt.Errorf("tipo de apuesta invalido: elegir jugador, banca o empate")
	}
	if len(draws) < 4 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}

	jugador := []int{baccaratCarta(draws[0]), baccaratCarta(draws[1])}
	banca := []int{baccaratCarta(draws[2]), baccaratCarta(draws[3])}
	tj := baccaratTotal(jugador)
	tb := baccaratTotal(banca)

	// Con 8 o 9 natural no hay tercera carta.
	if tj < 8 && tb < 8 {
		if tj <= 5 {
			if len(draws) < 5 {
				return Outcome{}, fmt.Errorf("sorteo incompleto")
			}
			tercera := baccaratCarta(draws[4])
			jugador = append(jugador, tercera)
			tj = baccaratTotal(jugador)
			pide := false
			switch {
			case tb <= 2:
				pide = true
			case tb == 3:
				pide = tercera != 8
			case tb == 4:
				pide = tercera >= 2 && tercera <= 7
			case tb == 5:
				pide = tercera >= 4 && tercera <= 7
			case tb == 6:
				pide = tercera == 6 || tercera == 7
			}
			if pide {
				if len(draws) < 6 {
					return Outcome{}, fmt.Errorf("sorteo incompleto")
				}
				banca = append(banca, baccaratCarta(draws[5]))
				tb = baccaratTotal(banca)
			}
		} else if tb <= 5 {
			// El jugador se planta con 6 o 7: la banca pide con 0-5.
			if len(draws) < 6 {
				return Outcome{}, fmt.Errorf("sorteo incompleto")
			}
			banca = append(banca, baccaratCarta(draws[5]))
			tb = baccaratTotal(banca)
		}
	}

	var mult int64
	comision := int64(0)
	switch {
	case tj > tb && tipo == "jugador":
		mult = 2
	case tb > tj && tipo == "banca":
		mult = 2
		// La banca paga 0.95 por cada peso: la comision del 5% se descuenta
		// de la ganancia al momento de pagar (ver casino.Premio).
		comision = monto / 20
	case tj == tb && tipo == "empate":
		mult = 8
	case tj == tb:
		mult = 1 // las manos en empate devuelven la apuesta
	}

	return Outcome{
		Multiplicador: mult,
		Data: map[string]any{
			"cartas_jugador": jugador,
			"cartas_banca":   banca,
			"total_jugador":  tj,
			"total_banca":    tb,
			"tipo":           tipo,
			"resultado":      resultadoBaccarat(tj, tb),
			"comision":       comision,
		},
	}, nil
}

func resultadoBaccarat(tj, tb int) string {
	switch {
	case tj > tb:
		return "jugador"
	case tb > tj:
		return "banca"
	default:
		return "empate"
	}
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
	case "cascada":
		return jugarCascada(monto, draws, p)
	case "hilo":
		return jugarHilo(monto, draws, p)
	case "alto":
		return jugarAlto(monto, draws, p)
	case "fortuna":
		return jugarFortuna(monto, draws, p)
	case "moneda":
		return jugarMoneda(monto, draws, p)
	case "suertudo":
		return jugarSuertudo(monto, draws, p)
	case "parley":
		return jugarParley(monto, draws, p)
	case "baccarat":
		return jugarBaccarat(monto, draws, p)
	}
	// Catalogo AST Casino: maquinas tragamonedas y mesas de video poker.
	if _, ok := BuscarSlot(codigo); ok {
		return jugarAstSlot(codigo, monto, draws, p)
	}
	if _, ok := BuscarPoker(codigo); ok {
		return jugarAstPoker(codigo, monto, draws, p)
	}
	return Outcome{}, fmt.Errorf("juego no soportado: %s", codigo)
}

// MultiplicadorMaximo devuelve el peor multiplicador posible de un juego. El
// blackjack puede doblar y ganar, asi que paga hasta 4 veces la apuesta; el
// natural (3:2) solo se da antes de doblar.
func MultiplicadorMaximo(codigo string, maxMultipl int64) int64 {
	if codigo == "blackjack" {
		return 4
	}
	if m := MaxMultiplSlot(codigo); m > 0 {
		return m
	}
	if m := MaxMultiplPoker(codigo); m > 0 {
		return m
	}
	if maxMultipl > 0 {
		return maxMultipl
	}
	return 0
}

// ApuestaMaxima devuelve la apuesta maxima que respeta el techo de premio por
// jugada. maxPremio <= 0 significa sin techo y se respeta maxApuesta tal cual.
// Asi el premio nunca supera el techo sin tocar el multiplicador del sorteo
// (que es lo que verifica el jugador).
func ApuestaMaxima(maxApuesta, maxMultipl, maxPremio int64, codigo string) int64 {
	if maxPremio <= 0 || maxApuesta <= 0 {
		return maxApuesta
	}
	mult := MultiplicadorMaximo(codigo, maxMultipl)
	if mult <= 0 {
		return maxApuesta
	}
	tope := maxPremio / mult
	if tope >= maxApuesta {
		return maxApuesta
	}
	return tope
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
	case "cascada":
		return cascadaFilas
	case "hilo":
		return 2
	case "alto", "fortuna", "moneda", "suertudo":
		return 1
	case "parley":
		return 4
	case "baccarat":
		return 6
	}
	// Catalogo AST: cada rodillo y fila de la ventana consume un draw.
	if s, ok := BuscarSlot(codigo); ok {
		return s.Rodillos * s.Filas
	}
	// Video poker: cinco cartas iniciales y hasta cinco de reemplazo. Se piden
	// de sobra porque los duplicados se descartan al elegir las cartas.
	if _, ok := BuscarPoker(codigo); ok {
		return 25
	}
	return 1
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
		case "cascada":
			p = Params{}
		case "hilo":
			p = Params{"direccion": "mayor"}
		case "alto":
			p = Params{"tope": 48, "direccion": "bajo"}
		case "fortuna":
			p = Params{}
		case "baccarat":
			p = Params{"tipo": "banca"}
		}
		out, err := Jugar(codigo, 1000, draws, p)
		if err != nil {
			continue
		}
		suma += float64(Premio(1000, out)) / 1000
	}
	return suma / float64(iteraciones) * 100
}
