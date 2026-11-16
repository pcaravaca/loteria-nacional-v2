package casino

import (
	"fmt"
	"sort"
)

// ---------------------------------------------------------------- slots AST

// simboloSlot convierte el indice del catalogo en la clave que ve el cliente.
func simboloSlot(i int) string { return fmt.Sprintf("s%d", i) }

// ventanaSlot arma la matriz visible (rodillos x filas) a partir de los draws.
// Cada celda escoge un simbolo segun los pesos del rodillo.
func ventanaSlot(cfg AstSlot, banda AstBand, draws []int) ([][]int, error) {
	necesarios := cfg.Rodillos * cfg.Filas
	if len(draws) < necesarios {
		return nil, fmt.Errorf("sorteo incompleto")
	}
	ventana := make([][]int, cfg.Rodillos)
	for r := 0; r < cfg.Rodillos; r++ {
		pesos := banda.Pesos[r]
		total := 0
		for _, p := range pesos {
			total += p
		}
		if total <= 0 {
			return nil, fmt.Errorf("rodillo %d sin pesos", r)
		}
		columna := make([]int, cfg.Filas)
		for f := 0; f < cfg.Filas; f++ {
			pos := draws[r*cfg.Filas+f] % total
			acum := 0
			elegido := len(pesos) - 1
			for s, p := range pesos {
				acum += p
				if pos < acum {
					elegido = s
					break
				}
			}
			columna[f] = elegido
		}
		ventana[r] = columna
	}
	return ventana, nil
}

// evaluaLineas recorre las lineas de la maquina y suma los pagos. Devuelve el
// multiplicador total y el detalle de cada linea ganadora para animarlas.
func evaluaLineas(cfg AstSlot, banda AstBand, ventana [][]int) (int64, []map[string]any) {
	var total int64
	ganadoras := []map[string]any{}
	for indice, linea := range cfg.Lineas {
		if len(linea) > cfg.Rodillos {
			continue
		}
		simbolos := make([]int, len(linea))
		for r, fila := range linea {
			if fila < 1 || fila > cfg.Filas {
				simbolos = nil
				break
			}
			simbolos[r] = ventana[r][fila-1]
		}
		if len(simbolos) == 0 {
			continue
		}
		n := 1
		for n < len(simbolos) && simbolos[n] == simbolos[0] {
			n++
		}
		if n-1 >= len(banda.Pagos) || simbolos[0] >= len(banda.Pagos[n-1]) {
			continue
		}
		pago := int64(banda.Pagos[n-1][simbolos[0]])
		if pago <= 0 {
			continue
		}
		total += pago
		celdas := make([][]int, 0, n)
		for r := 0; r < n; r++ {
			celdas = append(celdas, []int{r, linea[r] - 1})
		}
		ganadoras = append(ganadoras, map[string]any{
			"linea":      indice,
			"simbolo":    simboloSlot(simbolos[0]),
			"repeticion": n,
			"multipl":    pago,
			"celdas":     celdas,
		})
	}
	return total, ganadoras
}

func bandaSlot(cfg AstSlot, nombre string) (AstBand, string, error) {
	if nombre != "" {
		if b, ok := cfg.Bandas[nombre]; ok {
			return b, nombre, nil
		}
		return AstBand{}, "", fmt.Errorf("banda invalida: %s", nombre)
	}
	nombre = cfg.BandaPorDefecto
	b, ok := cfg.Bandas[nombre]
	if !ok {
		return AstBand{}, "", fmt.Errorf("la maquina no tiene banda por defecto")
	}
	return b, nombre, nil
}

// jugarAstSlot resuelve una jugada de una maquina del catalogo AST.
func jugarAstSlot(codigo string, monto int64, draws []int, p Params) (Outcome, error) {
	cfg, ok := BuscarSlot(codigo)
	if !ok {
		return Outcome{}, fmt.Errorf("juego no soportado: %s", codigo)
	}
	bandaNombre := ""
	if p != nil {
		bandaNombre, _ = p.String("banda")
	}
	banda, bandaUsada, err := bandaSlot(cfg, bandaNombre)
	if err != nil {
		return Outcome{}, err
	}
	ventana, err := ventanaSlot(cfg, banda, draws)
	if err != nil {
		return Outcome{}, err
	}
	mult, ganadoras := evaluaLineas(cfg, banda, ventana)

	rodillos := make([][]string, cfg.Rodillos)
	for r, col := range ventana {
		filas := make([]string, cfg.Filas)
		for f, s := range col {
			filas[f] = simboloSlot(s)
		}
		rodillos[r] = filas
	}
	tabla := tablaPagos(banda)
	return Outcome{
		Multiplicador: mult,
		Data: map[string]any{
			"rodillos":     rodillos,
			"lineas":       ganadoras,
			"banda":        bandaUsada,
			"bonus":        cfg.Bonus,
			"giros_libres": cfg.GirosLibres,
			"jackpot":      cfg.Jackpot,
			"tabla":        tabla,
		},
	}, nil
}

// tablaPagos devuelve los pagos de la banda ordenados de mayor a menor, para
// que el cliente muestre la tabla sin duplicar los numeros del servidor.
func tablaPagos(banda AstBand) []map[string]any {
	type fila struct {
		simbolo    int
		repeticion int
		multipl    int
	}
	var filas []fila
	for k, columna := range banda.Pagos {
		for s, v := range columna {
			if v > 0 {
				filas = append(filas, fila{simbolo: s, repeticion: k + 1, multipl: v})
			}
		}
	}
	sort.Slice(filas, func(i, j int) bool {
		if filas[i].multipl != filas[j].multipl {
			return filas[i].multipl > filas[j].multipl
		}
		return filas[i].repeticion > filas[j].repeticion
	})
	salida := make([]map[string]any, 0, len(filas))
	for _, f := range filas {
		salida = append(salida, map[string]any{
			"simbolo":    simboloSlot(f.simbolo),
			"repeticion": f.repeticion,
			"multipl":    f.multipl,
		})
	}
	return salida
}

// MaxMultiplSlot devuelve el mayor multiplicador posible de una maquina (la
// suma de todas las lineas pagando lo maximo), para el techo de apuesta.
func MaxMultiplSlot(codigo string) int64 {
	cfg, ok := BuscarSlot(codigo)
	if !ok {
		return 0
	}
	banda := cfg.Bandas[cfg.BandaPorDefecto]
	maximo := int64(0)
	for _, fila := range banda.Pagos {
		for _, v := range fila {
			if int64(v) > maximo {
				maximo = int64(v)
			}
		}
	}
	return maximo * int64(len(cfg.Lineas))
}

// RTPASTSlots simula cada maquina del catalogo y devuelve su RTP medido.
func RTPASTSlots(iteraciones int) map[string]float64 {
	if iteraciones <= 0 {
		iteraciones = 50000
	}
	salida := map[string]float64{}
	for _, cfg := range SlotsAST() {
		var suma float64
		for i := 0; i < iteraciones; i++ {
			draws := Sortear(fmt.Sprintf("rtp-%s", cfg.Codigo), "sim", int64(i), cfg.Rodillos*cfg.Filas, 1<<31-1)
			out, err := jugarAstSlot(cfg.Codigo, 1000, draws, Params{})
			if err != nil {
				continue
			}
			suma += float64(Premio(1000, out)) / 1000
		}
		salida[cfg.Codigo] = suma / float64(iteraciones) * 100
	}
	return salida
}
