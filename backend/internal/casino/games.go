package casino

import (
	"fmt"
	"sort"
	"strconv"
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
	Multiplicador int64          `json:"multiplicador"`
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
	2: 27, 3: 14, 4: 9, 5: 7, 6: 6, 7: 5,
	8: 6, 9: 7, 10: 9, 11: 14, 12: 27,
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

// ruletaRueda describe una modalidad de la ruleta: las casillas de la rueda en
// orden y los multiplicadores del pleno y del color. El sorteo escoge un indice
// de esa lista, asi que la rueda europea (del 0 al 36) sigue mandando el valor
// tal cual y la rapida usa una rueda corta con 0 y 00.
type ruletaRueda struct {
	Casillas []string
	Pleno    int64
	Color    int64
}

var ruletaCasillasEuropeas = func() []string {
	out := make([]string, 37)
	for i := range out {
		out[i] = strconv.Itoa(i)
	}
	return out
}()

var ruletaRuedas = map[string]ruletaRueda{
	"europea": {
		Casillas: ruletaCasillasEuropeas,
		Pleno:    35,
		Color:    2,
	},
	"rapida": {
		Casillas: []string{"0", "00", "1", "2", "3", "4", "5", "6", "7", "8"},
		Pleno:    8,
		Color:    2,
	},
}

// ruletaColorCasilla devuelve el color de una casilla de la modalidad elegida.
// En la rapida los dos ceros son verdes y los pares son negros.
func ruletaColorCasilla(modalidad, casilla string) string {
	if modalidad == "rapida" {
		switch casilla {
		case "0", "00":
			return "verde"
		case "1", "3", "5", "7":
			return "rojo"
		default:
			return "negro"
		}
	}
	n, _ := strconv.Atoi(casilla)
	return ruletaColor(n)
}

func jugarRuleta(monto int64, draws []int, p Params) (Outcome, error) {
	if len(draws) < 1 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	modCod, err := normalizarModalidad("ruleta", p)
	if err != nil {
		return Outcome{}, err
	}
	rueda := ruletaRuedas[modCod]
	casilla := rueda.Casillas[draws[0]%len(rueda.Casillas)]
	numero, _ := strconv.Atoi(casilla) // "00" no es numero: vale 0
	color := ruletaColorCasilla(modCod, casilla)

	tipo, err := p.String("tipo")
	if err != nil {
		return Outcome{}, err
	}

	var mult int64
	switch tipo {
	case "pleno":
		n, err := p.Int("numero")
		if err != nil {
			return Outcome{}, err
		}
		if n < 0 || n >= len(rueda.Casillas) {
			return Outcome{}, fmt.Errorf("el numero debe estar entre 0 y %d", len(rueda.Casillas)-1)
		}
		if rueda.Casillas[n] == casilla {
			mult = rueda.Pleno
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
			mult = rueda.Color
		}
	default:
		return Outcome{}, fmt.Errorf("tipo de apuesta invalido")
	}

	data := DataModalidad("ruleta", modCod)
	data["numero"] = numero
	data["casilla"] = casilla
	data["color"] = color
	data["tipo"] = tipo
	data["casillas"] = len(rueda.Casillas)
	return Outcome{
		Multiplicador: mult,
		Data:          data,
	}, nil
}

// ---------------------------------------------------------------- raspa

var raspaSimbolos = []string{"cereza", "campana", "siete", "diamante", "herradura", "barra"}

// raspaCfg describe una modalidad de la raspa: cuantos simbolos entran en el
// rodillo y cuanto paga el triple y el par.
type raspaCfg struct {
	Simbolos int
	Triple   int64
	Par      int64
}

// El par paga x2: antes pagaba x1 y el jugador "ganaba" lo mismo que habia
// apostado, lo que no se entendia. Los triple se ajustaron para conservar el RTP.
var raspaCfgs = map[string]raspaCfg{
	"clasica": {Simbolos: 6, Triple: 3, Par: 2},
	"expres":  {Simbolos: 4, Triple: 5, Par: 1},
}

func jugarRaspa(monto int64, draws []int, p Params) (Outcome, error) {
	if len(draws) < 3 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	modCod, err := normalizarModalidad("raspa", p)
	if err != nil {
		return Outcome{}, err
	}
	cfg := raspaCfgs[modCod]
	simbolos := raspaSimbolos[:cfg.Simbolos]
	a := simbolos[draws[0]%len(simbolos)]
	b := simbolos[draws[1]%len(simbolos)]
	c := simbolos[draws[2]%len(simbolos)]

	var mult int64
	switch {
	case a == b && b == c:
		mult = cfg.Triple
	case a == b || b == c || a == c:
		mult = cfg.Par
	}
	data := DataModalidad("raspa", modCod)
	data["simbolos"] = []string{a, b, c}
	data["simbolos_totales"] = cfg.Simbolos
	return Outcome{
		Multiplicador: mult,
		Data:          data,
	}, nil
}

// ---------------------------------------------------------------- keno

// kenoCfg describe una modalidad del keno: de donde salen las bolillas, cuantas
// se cantan, cuantos numeros elige el jugador y cuanto paga cada acierto.
type kenoCfg struct {
	Bolillas int
	Elegidos int
	Cantidad int
	Tabla    map[int]int64
}

var kenoCfgs = map[string]kenoCfg{
	"clasico": {
		Bolillas: 60,
		Elegidos: 5,
		Cantidad: 15,
		Tabla:    map[int]int64{2: 1, 3: 5, 4: 15, 5: 65},
	},
	"relampago": {
		Bolillas: 40,
		Elegidos: 4,
		Cantidad: 8,
		Tabla:    map[int]int64{1: 1, 2: 2, 3: 5, 4: 8},
	},
}

// kenoBolillas devuelve `cantidad` bolillas distintas entre 1 y `bolillas`.
func kenoBolillas(draws []int, bolillas, cantidad int) []int {
	vistos := map[int]bool{}
	out := make([]int, 0, cantidad)
	for _, d := range draws {
		n := (d % bolillas) + 1
		if vistos[n] {
			// Sin probing: saltear al proximo valor. Sumar 1 corria la bolilla
			// y sesgaba el sorteo hacia numeros consecutivos.
			continue
		}
		vistos[n] = true
		out = append(out, n)
		if len(out) == cantidad {
			return out
		}
	}
	for n := 1; len(out) < cantidad && n <= bolillas; n++ {
		if !vistos[n] {
			vistos[n] = true
			out = append(out, n)
		}
	}
	return out
}

var kenoMultipl = map[int]int64{2: 1, 3: 3, 4: 8, 5: 65}

func jugarKeno(monto int64, draws []int, p Params) (Outcome, error) {
	modCod, err := normalizarModalidad("keno", p)
	if err != nil {
		return Outcome{}, err
	}
	cfg := kenoCfgs[modCod]
	tuyos, err := p.Ints("numeros")
	if err != nil {
		return Outcome{}, err
	}
	if len(tuyos) != cfg.Elegidos {
		return Outcome{}, fmt.Errorf("debemos elegir exactamente %d numeros", cfg.Elegidos)
	}
	vistos := map[int]bool{}
	for _, n := range tuyos {
		if n < 1 || n > cfg.Bolillas {
			return Outcome{}, fmt.Errorf("los numeros deben estar entre 1 y %d", cfg.Bolillas)
		}
		if vistos[n] {
			return Outcome{}, fmt.Errorf("los numeros no pueden repetirse")
		}
		vistos[n] = true
	}

	bolillas := kenoBolillas(draws, cfg.Bolillas, cfg.Cantidad)
	aciertos := 0
	for _, b := range bolillas {
		if vistos[b] {
			aciertos++
		}
	}
	mult := int64(0)
	if m, ok := cfg.Tabla[aciertos]; ok {
		mult = m
	}
	data := DataModalidad("keno", modCod)
	data["bolillas"] = bolillas
	data["tus_numeros"] = tuyos
	data["aciertos"] = aciertos
	data["tabla"] = cfg.Tabla
	data["bolillas_totales"] = cfg.Bolillas
	data["elegidos"] = cfg.Elegidos
	return Outcome{
		Multiplicador: mult,
		Data:          data,
	}, nil
}

// ---------------------------------------------------------------- cascada (plinko)

// cascadaFilas es la cantidad de filas de pinchos; la bola termina en una de
// cascadaFilas+1 casillas. La tabla paga poco en el centro y mucho en los
// bordes, como el plinko clasico (RTP ~96.5%).
const cascadaFilas = 9

var cascadaTabla = []int64{10, 5, 3, 1, 0, 0, 1, 3, 5, 10}

// cascadaCfg describe una modalidad de la cascada: filas de pinchos y la tabla
// de la fila de casillas (filas+1 valores). La clasica es la de 9 filas.
type cascadaCfg struct {
	Filas int
	Tabla []int64
}

var cascadaCfgs = map[string]cascadaCfg{
	"clasica":  {Filas: cascadaFilas, Tabla: cascadaTabla},
	"rapida":   {Filas: 5, Tabla: []int64{4, 2, 0, 0, 2, 4}},
	"profunda": {Filas: 12, Tabla: []int64{10, 9, 6, 3, 1, 0, 0, 0, 1, 3, 6, 9, 10}},
}

// cascadaFilasMax es la mayor cantidad de filas entre las modalidades: define
// cuantos draws pide el juego (las modalidades cortas ignoran los de sobra).
func cascadaFilasMax() int {
	max := 0
	for _, c := range cascadaCfgs {
		if c.Filas > max {
			max = c.Filas
		}
	}
	return max
}

func jugarCascada(monto int64, draws []int, p Params) (Outcome, error) {
	modCod, err := normalizarModalidad("cascada", p)
	if err != nil {
		return Outcome{}, err
	}
	cfg := cascadaCfgs[modCod]
	if len(draws) < cfg.Filas {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	camino := make([]int, 0, cfg.Filas)
	pos := 0
	for i := 0; i < cfg.Filas; i++ {
		d := draws[i] % 2
		camino = append(camino, d)
		pos += d
	}
	data := DataModalidad("cascada", modCod)
	data["camino"] = camino
	data["posicion"] = pos
	data["tabla"] = cfg.Tabla
	data["filas"] = cfg.Filas
	return Outcome{
		Multiplicador: cfg.Tabla[pos],
		Data:          data,
	}, nil
}

// ---------------------------------------------------------------- mayor o menor

var hiloFiguras = []string{"", "A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}

func hiloCarta(draw int) int { return draw%13 + 1 }

// hiloCfg describe una modalidad del hilo: cuantas cartas se reparten y el
// multiplicador de la racha (0 significa el clasico, que paga x2 por un acierto).
type hiloCfg struct {
	Cartas int
	Mult   int64
}

var hiloCfgs = map[string]hiloCfg{
	"clasico": {Cartas: 2, Mult: 0},
	"racha":   {Cartas: 3, Mult: 6},
}

func jugarHilo(monto int64, draws []int, p Params) (Outcome, error) {
	modCod, err := normalizarModalidad("hilo", p)
	if err != nil {
		return Outcome{}, err
	}
	cfg := hiloCfgs[modCod]
	if len(draws) < cfg.Cartas {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	dir, err := p.String("direccion")
	if err != nil {
		return Outcome{}, err
	}
	if dir != "mayor" && dir != "menor" {
		return Outcome{}, fmt.Errorf("la direccion debe ser mayor o menor")
	}
	cartas := make([]int, 0, cfg.Cartas)
	for i := 0; i < cfg.Cartas; i++ {
		cartas = append(cartas, hiloCarta(draws[i]))
	}
	// En la racha hay que acertar todas las predicciones seguidas.
	gano := true
	for i := 1; i < len(cartas); i++ {
		if dir == "mayor" && cartas[i] <= cartas[i-1] {
			gano = false
		}
		if dir == "menor" && cartas[i] >= cartas[i-1] {
			gano = false
		}
	}
	mult := int64(0)
	if gano {
		if cfg.Mult > 0 {
			mult = cfg.Mult
		} else {
			mult = 2
		}
	}
	figuras := make([]string, 0, len(cartas))
	for _, c := range cartas {
		figuras = append(figuras, hiloFiguras[c])
	}
	data := DataModalidad("hilo", modCod)
	data["base"] = cartas[0]
	data["siguiente"] = cartas[1]
	data["figura_base"] = hiloFiguras[cartas[0]]
	data["figura_siguiente"] = hiloFiguras[cartas[1]]
	data["cartas"] = cartas
	data["figuras"] = figuras
	data["direccion"] = dir
	data["tiros"] = cfg.Cartas - 1
	return Outcome{
		Multiplicador: mult,
		Data:          data,
	}, nil
}

// ---------------------------------------------------------------- bajo o sobre (dado 100)

// altoTopes mapea el tope elegido (porcentaje 1-100) con su multiplicador.
// El RTP depende del riesgo elegido (~96% en los extremos, menor en el medio).
var altoTopes = map[int]int64{2: 48, 5: 19, 10: 9, 25: 3, 48: 2, 75: 1, 90: 1, 95: 1}

// altoCfg describe una modalidad del alto/menor: cuantos tiros hay que acertar y
// el multiplicador de la racha (0 significa el clasico, que paga segun el tope).
type altoCfg struct {
	Tiros int
	Mult  int64
}

var altoCfgs = map[string]altoCfg{
	"clasico": {Tiros: 1, Mult: 0},
	"racha":   {Tiros: 2, Mult: 4},
}

func jugarAlto(monto int64, draws []int, p Params) (Outcome, error) {
	modCod, err := normalizarModalidad("alto", p)
	if err != nil {
		return Outcome{}, err
	}
	cfg := altoCfgs[modCod]
	if len(draws) < cfg.Tiros {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
	}
	tope, err := p.Int("tope")
	if err != nil {
		return Outcome{}, err
	}
	multTope, ok := altoTopes[tope]
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
	tiradas := make([]int, 0, cfg.Tiros)
	gano := true
	for i := 0; i < cfg.Tiros; i++ {
		tirada := draws[i]%100 + 1
		tiradas = append(tiradas, tirada)
		if direccion == "bajo" {
			if tirada > tope {
				gano = false
			}
		} else if tirada <= 100-tope {
			gano = false
		}
	}
	mult := int64(0)
	if gano {
		if cfg.Mult > 0 {
			mult = cfg.Mult
		} else {
			mult = multTope
		}
	}
	data := DataModalidad("alto", modCod)
	data["tirada"] = tiradas[0]
	data["tiradas"] = tiradas
	data["tope"] = tope
	data["direccion"] = direccion
	data["tabla"] = altoTopes
	data["tiros"] = cfg.Tiros
	return Outcome{
		Multiplicador: mult,
		Data:          data,
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
		if bolillaSet[n] {
			// Sin probing: saltear al proximo valor. Sumar 1 corria la bolilla
			// y inflaba el sorteo (siempre 4 bolillas, mas aciertos).
			continue
		}
		bolillaSet[n] = true
		if len(bolillaSet) == 4 {
			break
		}
	}
	if len(bolillaSet) < 4 {
		return Outcome{}, fmt.Errorf("sorteo incompleto")
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
			"bolillas":    bolillas,
			"tus_numeros": tuyos,
			"aciertos":    aciertos,
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

// baccaratCfg describe una modalidad del baccarat: si la banca cobra comision y
// cuanto paga cuando la banca gana con 6 (0 = paga el resto igual).
type baccaratCfg struct {
	Comision  bool
	BancaSeis int64
}

var baccaratCfgs = map[string]baccaratCfg{
	"estandar":     {Comision: true, BancaSeis: 0},
	"sin-comision": {Comision: false, BancaSeis: 1},
}

// jugarBaccarat reparte dos cartas por mano y aplica la regla de la tercera
// carta (el jugador pide con 0-5 y se planta con 6-7; la banca sigue la tabla
// oficial segun la tercera carta del jugador). El empate paga x8 y las apuestas
// a jugador o banca quedan en empate (se devuelve la apuesta).
func jugarBaccarat(monto int64, draws []int, p Params) (Outcome, error) {
	modCod, err := normalizarModalidad("baccarat", p)
	if err != nil {
		return Outcome{}, err
	}
	cfg := baccaratCfgs[modCod]
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
		// de la ganancia al momento de pagar (ver casino.Premio). En la
		// modalidad sin comision no se descuenta nada, pero la banca que gana
		// con 6 paga la mitad (la regla del "super 6").
		if cfg.Comision {
			comision = monto / 20
		} else if cfg.BancaSeis > 0 && tb == 6 {
			mult = cfg.BancaSeis
		}
	case tj == tb && tipo == "empate":
		mult = 8
	case tj == tb:
		// Las manos en empate devuelven la apuesta en la modalidad estandar;
		// en la sin comision el empate no devuelve nada.
		if cfg.Comision {
			mult = 1
		}
	}

	data := DataModalidad("baccarat", modCod)
	data["cartas_jugador"] = jugador
	data["cartas_banca"] = banca
	data["total_jugador"] = tj
	data["total_banca"] = tb
	data["tipo"] = tipo
	data["resultado"] = resultadoBaccarat(tj, tb)
	data["comision"] = comision
	return Outcome{
		Multiplicador: mult,
		Data:          data,
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
	case "pinball-neon", "pinball-clasico", "plinko-premium":
		return jugarPinball(codigo, monto, draws, p)
	case minaCodigoBase:
		return minaJugar(codigo, monto, draws, p)
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

// multMaximaModalidad devuelve el mayor multiplicador que puede pagar una
// modalidad concreta del juego. Lo usa MultMaximoJuego para recortar la
// apuesta maxima sin tocar el multiplicador que audita el jugador.
func multMaximaModalidad(codigoJuego, codigoModalidad string) int64 {
	switch codigoJuego {
	case "ruleta":
		if r, ok := ruletaRuedas[codigoModalidad]; ok {
			return max(r.Pleno, r.Color)
		}
	case "keno":
		if c, ok := kenoCfgs[codigoModalidad]; ok {
			return maxMapInt64(c.Tabla)
		}
	case "raspa":
		if c, ok := raspaCfgs[codigoModalidad]; ok {
			return max(c.Triple, c.Par)
		}
	case "baccarat":
		if _, ok := baccaratCfgs[codigoModalidad]; ok {
			// El empate siempre paga x8; la banca paga x2 (x1 con 6 en la
			// modalidad sin comision).
			return 8
		}
	case "alto":
		if c, ok := altoCfgs[codigoModalidad]; ok {
			if c.Mult > 0 {
				return c.Mult
			}
			return maxMapInt64(altoTopes)
		}
	case "hilo":
		if c, ok := hiloCfgs[codigoModalidad]; ok {
			if c.Mult > 0 {
				return c.Mult
			}
			return 2
		}
	case "cascada":
		if c, ok := cascadaCfgs[codigoModalidad]; ok {
			var max int64
			for _, v := range c.Tabla {
				if v > max {
					max = v
				}
			}
			return max
		}
	}
	return 0
}

// maxMapInt64 devuelve el mayor valor de un mapa de multiplicadores.
func maxMapInt64(m map[int]int64) int64 {
	var max int64
	for _, v := range m {
		if v > max {
			max = v
		}
	}
	return max
}

// MultiplicadorMaximo devuelve el peor multiplicador posible de un juego. El
// blackjack puede doblar y ganar, asi que paga hasta 4 veces la apuesta; el
// natural (3:2) solo se da antes de doblar.
func MultiplicadorMaximo(codigo string, maxMultipl int64) int64 {
	if codigo == "blackjack" {
		return 4
	}
	// El bingo paga hasta x25 por cartón y admite hasta 4 cartones: el peor
	// caso absoluto es x100 sobre la apuesta por cartón. Cada variante tiene su
	// propio tope (el 90 paga x40 por cartón).
	if codigo == "bingo" {
		return 25 * 4
	}
	if mult := BingoMultMaximo(codigo); mult > 0 {
		return mult
	}
	// Gemas: 25 intercambios con multiplicador de cascada creciente. El tope
	// declarado deja margen por encima de lo que la simulacion alguna vez midio.
	if codigo == "gemas" {
		return 100
	}
	if m := MaxMultiplSlot(codigo); m > 0 {
		return m
	}
	if m := MaxMultiplPoker(codigo); m > 0 {
		return m
	}
	// Pinball: el multiplicador lo pone la tronera, la fuerza y los bumpers; el
	// tope de la mesa cubre los tres sin tocar el catálogo.
	if m := pinballMaxMult(codigo); m > 0 {
		return m
	}
	// Mina de Oro: el multiplicador lo marca la tabla del nivel (hasta x3).
	if codigo == minaCodigoBase {
		return minaMultiplicadorMaximo()
	}
	// Los juegos que tienen modalidades pueden pagar mas que el multiplicador
	// declarado en el catalogo (el hilo en racha paga x6). El tope cubre todas
	// las modalidades para que ninguna apuesta supere el techo de premio.
	if maxMultipl > 0 {
		if m := MultMaximoJuego(codigo); m > maxMultipl {
			return m
		}
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
		// 15 bolillas de 60 sin repetir: se piden de sobra porque los duplicados
		// se descartan.
		return 40
	case "cascada":
		// La modalidad profunda pide 12 draws; las cortas ignoran los de sobra.
		return cascadaFilasMax()
	case "hilo":
		// La racha reparte tres cartas; el clasico usa solo las dos primeras.
		return 3
	case "alto":
		// La racha tira dos veces; el clasico usa solo la primera.
		return 2
	case "fortuna", "moneda", "suertudo":
		return 1
	case "parley":
		// Cuatro bolillas de 20 sin repetir: se piden de sobra porque los
		// duplicados se descartan.
		return 12
	case "baccarat":
		return 6
	case "pinball-neon", "pinball-clasico", "plinko-premium":
		return pinballDraws(codigo)
	case minaCodigoBase:
		// 6 filas (la dificultad mas larga) por 3 draws por fila.
		return minaMaxFilas * (minaPorFila + 1)
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

// RTP aproximado por simulacion de la modalidad por defecto (util para afinar
// multiplicadores).
func RTP(codigo string, iteraciones int) float64 {
	return RTPModalidad(codigo, "", iteraciones)
}

// RTPModalidad mide por simulacion el retorno al jugador de una modalidad
// concreta. Con modalidad vacia usa la clasica. La apuesta de referencia es la
// misma que usa RTP para el resto de los juegos.
func RTPModalidad(codigo, modalidad string, iteraciones int) float64 {
	if iteraciones <= 0 {
		iteraciones = 200000
	}
	var suma float64
	for i := 0; i < iteraciones; i++ {
		seed := fmt.Sprintf("sim-%d", i)
		draws := Sortear(seed, "sim", int64(i), Draws(codigo), 1<<31-1)
		out, err := Jugar(codigo, 1000, draws, paramsRTP(codigo, modalidad))
		if err != nil {
			continue
		}
		suma += float64(Premio(1000, out)) / 1000
	}
	return suma / float64(iteraciones) * 100
}

// paramsRTP es la apuesta de referencia de la simulacion: una por juego, elegida
// para que el RTP sea representativo de cada modalidad.
func paramsRTP(codigo, modalidad string) Params {
	p := Params{}
	if modalidad != "" {
		p["modalidad"] = modalidad
	}
	switch codigo {
	case "dados":
		p["suma"] = 7
	case "ruleta":
		p["tipo"] = "pleno"
		if modalidad == "rapida" {
			p["numero"] = 5
		} else {
			p["numero"] = 17
		}
	case "keno":
		if modalidad == "relampago" {
			p["numeros"] = []int{1, 2, 3, 4}
		} else {
			p["numeros"] = []int{1, 2, 3, 4, 5}
		}
	case "cascada":
	case "hilo":
		p["direccion"] = "mayor"
	case "alto":
		p["tope"] = 48
		p["direccion"] = "bajo"
	case "fortuna":
	case "baccarat":
		p["tipo"] = "banca"
	case "raspa":
	case "pinball-neon", "pinball-clasico", "plinko-premium":
		p["fuerza"] = 1
	case minaCodigoBase:
		p["dificultad"] = 1
	}
	return p
}
