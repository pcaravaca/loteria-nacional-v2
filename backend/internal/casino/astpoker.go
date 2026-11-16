package casino

import (
	"fmt"
	"sort"
)

// ---------------------------------------------------------------- video poker AST

// cartaPoker es una carta del video poker. rango 2-14 (11=J, 12=Q, 13=K, 14=A),
// palo 0-3; comodin marca los comodines (dos en Deuce Wild, joker en Joker).
type cartaPoker struct {
	Rango   int
	Palo    int
	Comodin bool
}

var palosNombre = []string{"pados", "corazones", "diamantes", "treboles"}
var rangosNombre = map[int]string{11: "J", 12: "Q", 13: "K", 14: "A"}

func (c cartaPoker) Nombre() string {
	if c.Comodin {
		return "comodin"
	}
	r := fmt.Sprintf("%d", c.Rango)
	if n, ok := rangosNombre[c.Rango]; ok {
		r = n
	}
	return r + " de " + palosNombre[c.Palo]
}

// Clave devuelve la identidad unica de la carta para el cliente.
func (c cartaPoker) Clave() string {
	if c.Comodin {
		return "comodin"
	}
	return fmt.Sprintf("%02d-%d", c.Rango, c.Palo)
}

// barajaPoker arma la baraja segun la mesa (Joker Poker incluye comodin).
func barajaPoker(cfg AstPoker) int {
	if cfg.Codigo == "jokerpoker" {
		return 53
	}
	return 52
}

// esComodin define que cartas son salvajes en cada mesa.
func esComodin(cfg AstPoker, idx int) bool {
	if cfg.Codigo == "deucewild" {
		return idx >= 0 && idx < 4 // los cuatro dos
	}
	if cfg.Codigo == "jokerpoker" {
		return idx == 52
	}
	return false
}

func aCarta(cfg AstPoker, idx int) cartaPoker {
	if idx >= 52 {
		return cartaPoker{Comodin: true}
	}
	// orden de baraja: (rango-2)*4 + palo, igual que indiceCarta
	return cartaPoker{Rango: idx/4 + 2, Palo: idx % 4, Comodin: esComodin(cfg, idx)}
}

// repartirPoker reparte cinco cartas sin repetir a partir de los draws.
func repartirPoker(cfg AstPoker, draws []int) ([]cartaPoker, error) {
	if len(draws) < 5 {
		return nil, fmt.Errorf("sorteo incompleto")
	}
	total := barajaPoker(cfg)
	usadas := map[int]bool{}
	cartas := make([]cartaPoker, 0, 5)
	for i := 0; i < len(draws) && len(cartas) < 5; i++ {
		idx := draws[i] % total
		for usadas[idx] {
			idx = (idx + 1) % total
		}
		usadas[idx] = true
		cartas = append(cartas, aCarta(cfg, idx))
	}
	if len(cartas) < 5 {
		return nil, fmt.Errorf("sorteo incompleto")
	}
	return cartas, nil
}

// completarPoker reemplaza las posiciones no retenidas por cartas nuevas.
func completarPoker(cfg AstPoker, draws []int, retener []bool) ([]cartaPoker, error) {
	cartas, err := repartirPoker(cfg, draws)
	if err != nil {
		return nil, err
	}
	total := barajaPoker(cfg)
	usadas := map[int]bool{}
	for i, c := range cartas {
		if retener[i] {
			usadas[indiceCarta(cfg, c)] = true
		}
	}
	siguiente := 5
	for i := range cartas {
		if retener[i] {
			continue
		}
		for siguiente < len(draws) && usadas[draws[siguiente]%total] {
			siguiente++
		}
		if siguiente >= len(draws) {
			return nil, fmt.Errorf("sorteo incompleto")
		}
		idx := draws[siguiente] % total
		for usadas[idx] {
			idx = (idx + 1) % total
		}
		usadas[idx] = true
		cartas[i] = aCarta(cfg, idx)
		siguiente++
	}
	return cartas, nil
}

func indiceCarta(cfg AstPoker, c cartaPoker) int {
	if c.Comodin {
		return 52
	}
	return (c.Rango-2)*4 + c.Palo
}

// categoria devuelve la categoria de la mano concreta (sin comodines):
// "RoyalFlush", "StraightFlush", "FiveOfAKind", "FourOfAKind", "FullHouse",
// "Flush", "Straight", "ThreeOfAKind", "TwoPair", "Pair" o "".
func categoriaConcreta(cartas []cartaPoker) (string, int, int) {
	rangos := make([]int, 5)
	palos := make([]int, 5)
	contador := map[int]int{}
	for i, c := range cartas {
		rangos[i] = c.Rango
		palos[i] = c.Palo
		contador[c.Rango]++
	}
	sort.Ints(rangos)

	esColor := palos[0] == palos[1] && palos[1] == palos[2] && palos[2] == palos[3] && palos[3] == palos[4]
	// escalera: cinco consecutivos o rueda (A-2-3-4-5)
	esEscalera := true
	for i := 1; i < 5; i++ {
		if rangos[i] != rangos[i-1]+1 {
			esEscalera = false
			break
		}
	}
	if !esEscalera {
		rueda := true
		aux := []int{2, 3, 4, 5, 14}
		for i := range aux {
			if rangos[i] != aux[i] {
				rueda = false
				break
			}
		}
		esEscalera = rueda
	}

	mayor := 0
	for _, n := range contador {
		if n > mayor {
			mayor = n
		}
	}
	cuantas := func(n int) int {
		cuantas := 0
		for _, v := range contador {
			if v == n {
				cuantas++
			}
		}
		return cuantas
	}
	rangoDupla := 0
	for r, n := range contador {
		if n == 2 && r > rangoDupla {
			rangoDupla = r
		}
	}

	switch {
	case esColor && esEscalera && rangos[0] == 10:
		return "RoyalFlush", 0, 0
	case esColor && esEscalera:
		return "StraightFlush", 0, 0
	case mayor == 5:
		return "FiveOfAKind", 0, 0
	case mayor == 4:
		return "FourOfAKind", rangoDe(contador, 4), 0
	case mayor == 3 && cuantas(2) == 1:
		return "FullHouse", 0, 0
	case esColor:
		return "Flush", 0, 0
	case esEscalera:
		return "Straight", 0, 0
	case mayor == 3:
		return "ThreeOfAKind", 0, 0
	case cuantas(2) == 2:
		return "TwoPair", 0, 0
	case mayor == 2:
		return "Pair", rangoDupla, 0
	}
	return "", 0, 0
}

func rangoDe(contador map[int]int, n int) int {
	for r, v := range contador {
		if v == n {
			return r
		}
	}
	return 0
}

// evaluarPoker prueba todas las combinaciones posibles de los comodines y se
// queda con la mejor categoria. Devuelve la categoria, el rango de la dupla o
// del poquer (segun corresponda) y cuantos comodines se usaron.
func evaluarPoker(cfg AstPoker, cartas []cartaPoker) (string, int, int) {
	comodines := 0
	for _, c := range cartas {
		if c.Comodin {
			comodines++
		}
	}
	if comodines == 0 {
		cat, rango, _ := categoriaConcreta(cartas)
		return cat, rango, 0
	}
	if comodines > 3 {
		// con cuatro o mas comodines la enumeracion no cierra: se asignan ases
		concretas := make([]cartaPoker, len(cartas))
		for i, c := range cartas {
			if c.Comodin {
				concretas[i] = cartaPoker{Rango: 14, Palo: i % 4}
			} else {
				concretas[i] = c
			}
		}
		cat, rango, _ := categoriaConcreta(concretas)
		return cat, rango, comodines
	}
	concretas := make([]cartaPoker, len(cartas))
	copy(concretas, cartas)
	mejor := ""
	mejorRango := 0
	mejorUsados := 0
	var probar func(pos, usados int)
	probar = func(pos, usados int) {
		if pos == len(concretas) {
			cat, rango, _ := categoriaConcreta(concretas)
			if cat != "" && (mejor == "" || pesoCat[cat] > pesoCat[mejor]) {
				mejor = cat
				mejorRango = rango
				mejorUsados = usados
			}
			return
		}
		if !concretas[pos].Comodin {
			probar(pos+1, usados)
			return
		}
		for r := 14; r >= 2; r-- {
			for p := 0; p < 4; p++ {
				concretas[pos] = cartaPoker{Rango: r, Palo: p}
				probar(pos+1, usados+1)
			}
		}
		concretas[pos] = cartaPoker{Comodin: true}
	}
	probar(0, 0)
	return mejor, mejorRango, mejorUsados
}

var pesoCat = map[string]int{
	"RoyalFlush": 10, "StraightFlush": 9, "FiveOfAKind": 8, "FourOfAKind": 7,
	"FullHouse": 6, "Flush": 5, "Straight": 4, "ThreeOfAKind": 3,
	"TwoPair": 2, "Pair": 1,
}

// pagoPoker busca el multiplicador de la mesa para la categoria lograda.
func pagoPoker(cfg AstPoker, cat string, rangoPar, usados int) int64 {
	tabla := map[string]int64{}
	parMinimo := "JacksOrBetter"
	for _, f := range cfg.Pagos {
		tabla[f.Mano] = f.Multipl
		switch f.Mano {
		case "TensOrBetter":
			parMinimo = "TensOrBetter"
		case "Pair":
			parMinimo = "Pair"
		}
	}
	if cat == "" {
		return 0
	}
	if cat == "RoyalFlush" {
		if usados > 0 {
			if v, ok := tabla["WildRoyalFlush"]; ok {
				return v
			}
			if v, ok := tabla["JokerRoyalFlush"]; ok {
				return v
			}
		}
		if cfg.RealMaxima > 0 && usados == 0 {
			return cfg.RealMaxima
		}
		return tabla["RoyalFlush"]
	}
	if cat == "FourOfAKind" {
		// mesas con pagos especiales para el poquer de ases, sietes o 2-10
		if rangoPar == 14 {
			for _, k := range []string{"FourAcesOrEights", "FourAces"} {
				if v, ok := tabla[k]; ok {
					return v
				}
			}
		}
		if rangoPar == 7 {
			if v, ok := tabla["FourSevens"]; ok {
				return v
			}
		}
		if rangoPar >= 2 && rangoPar <= 10 {
			if v, ok := tabla["Four2Through10"]; ok {
				return v
			}
		}
		if v, ok := tabla["OtherFourOfAKind"]; ok {
			return v
		}
	}
	if cat == "Pair" {
		if parMinimo == "Pair" {
			return tabla["Pair"]
		}
		minimo := 11
		if parMinimo == "TensOrBetter" {
			minimo = 10
		}
		if rangoPar >= minimo {
			return tabla[parMinimo]
		}
		return 0
	}
	return tabla[cat]
}

// optimizarRetencion elige que cartas sostener. Es una estrategia conservadora:
// se planta con lo que ya paga, si no busca color, escalera o figuras.
func optimizarRetencion(cfg AstPoker, cartas []cartaPoker) []bool {
	retener := make([]bool, 5)
	cat, rangoPar, usados := evaluarPoker(cfg, cartas)
	if cat != "" && pagoPoker(cfg, cat, rangoPar, usados) > 0 {
		// ya paga: se planta (si es un par bajo, conviene buscar mejor)
		if cat != "Pair" || rangoPar >= 11 {
			for i := range retener {
				retener[i] = true
			}
			return retener
		}
	}
	// color: cuatro cartas del mismo palo
	porPalo := map[int][]int{}
	for i, c := range cartas {
		if !c.Comodin {
			porPalo[c.Palo] = append(porPalo[c.Palo], i)
		}
	}
	for _, idx := range porPalo {
		if len(idx) >= 4 {
			for _, i := range idx {
				retener[i] = true
			}
			return retener
		}
	}
	// escalera: cuatro cartas consecutivas
	rangos := []int{}
	for _, c := range cartas {
		if c.Comodin {
			rangos = append(rangos, -1)
		} else {
			rangos = append(rangos, c.Rango)
		}
	}
	for desde := 2; desde <= 10; desde++ {
		cuantas := 0
		for _, r := range rangos {
			if r >= desde && r <= desde+4 {
				cuantas++
			}
		}
		if cuantas >= 4 {
			for i, r := range rangos {
				if r >= desde && r <= desde+4 {
					retener[i] = true
				}
			}
			return retener
		}
	}
	// par bajo: se sostiene solo si la tabla paga por par
	if cat == "Pair" && pagoPoker(cfg, "Pair", rangoPar, 0) > 0 {
		for i, c := range cartas {
			if !c.Comodin && c.Rango == rangoPar {
				retener[i] = true
			}
		}
		return retener
	}
	// figuras y comodines
	for i, c := range cartas {
		if c.Comodin || c.Rango >= 11 {
			retener[i] = true
		}
	}
	return retener
}

// jugarAstPoker resuelve una jugada de video poker. Si el cliente no indica
// las cartas a sostener, el sistema retiene la mejor jugada.
func jugarAstPoker(codigo string, monto int64, draws []int, p Params) (Outcome, error) {
	cfg, ok := BuscarPoker(codigo)
	if !ok {
		return Outcome{}, fmt.Errorf("juego no soportado: %s", codigo)
	}
	iniciales, err := repartirPoker(cfg, draws)
	if err != nil {
		return Outcome{}, err
	}
	retener := optimizarRetencion(cfg, iniciales)
	if p != nil {
		if idx, err := p.Ints("retener"); err == nil && len(idx) > 0 {
			retener = make([]bool, 5)
			for _, i := range idx {
				if i >= 0 && i < 5 {
					retener[i] = true
				}
			}
		}
	}
	finales, err := completarPoker(cfg, draws, retener)
	if err != nil {
		return Outcome{}, err
	}
	cat, rangoPar, usados := evaluarPoker(cfg, finales)
	mult := pagoPoker(cfg, cat, rangoPar, usados)

	claves := make([]string, 5)
	for i, c := range finales {
		claves[i] = c.Clave()
	}
	inicial := make([]string, 5)
	for i, c := range iniciales {
		inicial[i] = c.Clave()
	}
	return Outcome{
		Multiplicador: mult,
		Data: map[string]any{
			"cartas":    claves,
			"iniciales": inicial,
			"retencion": retener,
			"mano":      cat,
			"multipl":   mult,
		},
	}, nil
}

// MaxMultiplPoker devuelve el mayor multiplicador de la mesa.
func MaxMultiplPoker(codigo string) int64 {
	cfg, ok := BuscarPoker(codigo)
	if !ok {
		return 0
	}
	maximo := cfg.RealMaxima
	for _, f := range cfg.Pagos {
		if f.Multipl > maximo {
			maximo = f.Multipl
		}
	}
	return maximo
}
