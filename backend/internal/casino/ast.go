package casino

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"sync"
)

// Catalogo de juegos inspirado en la plataforma AST Casino (Casino Diamond).
// Los numeros (rodillos, pesos, pagos, lineas y palms) salen de los XMLFiles
// originales; el arte y los nombres son propios.

//go:embed data/ast_slots.json
var astSlotsJSON []byte

//go:embed data/ast_poker.json
var astPokerJSON []byte

// AstBand es una banda de RTP: la misma máquina con pesos y pagos ajustados.
type AstBand struct {
	Pesos [][]int `json:"pesos"` // por rodillo, por simbolo
	Pagos [][]int `json:"pagos"` // [simbolos consecutivos desde la izquierda - 1][simbolo]
}

// AstSlot es una maquina tragamonedas del catalogo.
type AstSlot struct {
	Codigo          string           `json:"codigo"`
	Nombre          string           `json:"nombre"`
	Rodillos         int              `json:"rodillos"`
	Filas           int              `json:"filas"`
	Lineas          [][]int          `json:"lineas"`
	Bonus           bool             `json:"bonus"`
	GirosLibres     bool             `json:"giros_libres"`
	Jackpot         bool             `json:"jackpot"`
	BandaPorDefecto string           `json:"banda_por_defecto"`
	Bandas          map[string]AstBand `json:"bandas"`
}

// AstPoker es una mesa de video poker del catalogo.
type AstPoker struct {
	Codigo      string `json:"codigo"`
	Nombre      string `json:"nombre"`
	Pagos       []struct {
		Mano    string `json:"mano"`
		Multipl int64  `json:"multipl"`
	} `json:"pagos"`
	RealMaxima int64   `json:"real_maxima"`
	MaxFichas  float64 `json:"max_fichas"`
}

type astCatalogo struct {
	Slots  []AstSlot  `json:"slots"`
	Pokers []AstPoker `json:"pokers"`
}

var (
	astOnce sync.Once
	astCat  astCatalogo
)

// RTPObjetivo deja el RTP configurado dentro de un rango sensato para una sala
// de apuestas. El catalogo original maneja bandas de 88 a 101; el 101 se recorta.
func RTPObjetivo(banda string) float64 {
	medio := map[string]float64{
		"88-92": 90, "92-96": 94, "95-98": 96.5, "97-99": 98, "99-101": 98,
	}
	if v, ok := medio[banda]; ok {
		return v
	}
	return 95
}

// evLineaSlot calcula la esperanza matematica por linea de una banda. La linea
// paga una sola vez, por la mayor cantidad de simbolos consecutivos desde la
// izquierda (no acumula los pagos intermedios).
func evLineaSlot(banda AstBand) float64 {
	if len(banda.Pesos) == 0 {
		return 0
	}
	var ev float64
	for s := range banda.Pesos[0] {
		probables := make([]float64, 0, len(banda.Pesos))
		prob := 1.0
		for k := range banda.Pesos {
			total := 0
			for _, w := range banda.Pesos[k] {
				total += w
			}
			if total == 0 || s >= len(banda.Pesos[k]) {
				prob = 0
				break
			}
			prob *= float64(banda.Pesos[k][s]) / float64(total)
			probables = append(probables, prob)
		}
		for n := 1; n <= len(probables); n++ {
			if n-1 >= len(banda.Pagos) || s >= len(banda.Pagos[n-1]) {
				break
			}
			exito := probables[n-1]
			if n < len(probables) {
				exito -= probables[n]
			}
			ev += exito * float64(banda.Pagos[n-1][s])
		}
	}
	return ev
}

// calibrar ajusta los pagos de cada banda para que el RTP total (la suma de las
// lineas sobre la apuesta completa) caiga en el objetivo de la banda. Los XML
// originales son configuraciones del cliente de AST; aca se normalizan para que
// la sala cierre sus numeros. Se busca el factor por biseccion y despues se
// ajusta celda a celda, porque los pagos son enteros y la curva es una escalera.
func calibrar(cfg *AstSlot) {
	for nombre, original := range cfg.Bandas {
		objetivo := RTPObjetivo(nombre) / 100
		factor := buscarFactor(original, objetivo, len(cfg.Lineas))
		banda := escalarBandas(original, factor)
		banda = ajustarFino(len(cfg.Lineas), banda, objetivo)
		cfg.Bandas[nombre] = banda
	}
}

// buscarFactor ubica el factor donde el RTP cruza el objetivo (biseccion
// logaritmica, robusta contra la escalera del redondeo).
func buscarFactor(original AstBand, objetivo float64, lineas int) float64 {
	lo, hi := 1e-4, 10.0
	mejor := 1.0
	mejorDiff := math.Abs(evLineaSlot(original)*float64(lineas) - objetivo)
	for i := 0; i < 80; i++ {
		mid := math.Sqrt(lo * hi)
		b := escalarBandas(original, mid)
		ev := evLineaSlot(b) * float64(lineas)
		if dif := math.Abs(ev - objetivo); dif < mejorDiff {
			mejorDiff = dif
			mejor = mid
		}
		if ev > objetivo {
			hi = mid
		} else {
			lo = mid
		}
	}
	return mejor
}

// ajustarFino mueve pagos de a una unidad hasta acercar el RTP al objetivo.
func ajustarFino(lineas int, banda AstBand, objetivo float64) AstBand {
	n := float64(lineas)
	ev := evLineaSlot(banda) * n
	for paso := 0; paso < 400; paso++ {
		dif := objetivo - ev
		if math.Abs(dif) < 0.0005 {
			break
		}
		mejorK, mejorS := -1, -1
		mejorCoste := math.Abs(dif)
		encontrado := false
		for k := range banda.Pagos {
			for s := range banda.Pagos[k] {
				if dif < 0 && banda.Pagos[k][s] == 0 {
					continue
				}
				delta := contribucionCelda(banda, k, s) * n
				if delta <= 0 {
					continue
				}
				coste := math.Abs(dif - delta)
				if coste < mejorCoste {
					mejorCoste = coste
					mejorK, mejorS = k, s
					encontrado = true
				}
			}
		}
		if !encontrado {
			break
		}
		delta := contribucionCelda(banda, mejorK, mejorS) * n
		if dif > 0 {
			banda.Pagos[mejorK][mejorS]++
			ev += delta
		} else {
			banda.Pagos[mejorK][mejorS]--
			ev -= delta
		}
	}
	return banda
}

// contribucionCelda devuelve cuanto aporta una unidad mas en el pago [k][s].
func contribucionCelda(banda AstBand, k, s int) float64 {
	if len(banda.Pesos) == 0 || s >= len(banda.Pesos[0]) {
		return 0
	}
	probables := make([]float64, 0, len(banda.Pesos))
	prob := 1.0
	for r := range banda.Pesos {
		total := 0
		for _, w := range banda.Pesos[r] {
			total += w
		}
		if total == 0 || s >= len(banda.Pesos[r]) {
			return 0
		}
		prob *= float64(banda.Pesos[r][s]) / float64(total)
		probables = append(probables, prob)
	}
	if k >= len(probables) {
		return 0
	}
	exito := probables[k]
	if k+1 < len(probables) {
		exito -= probables[k+1]
	}
	return exito
}

// escalarBandas devuelve una copia de la banda con los pagos multiplicados por
// el factor y redondeados (nunca modifica la banda original).
func escalarBandas(original AstBand, factor float64) AstBand {
	pagos := make([][]int, len(original.Pagos))
	for k := range original.Pagos {
		pagos[k] = make([]int, len(original.Pagos[k]))
		for s, v := range original.Pagos[k] {
			pagos[k][s] = int(math.Round(float64(v) * factor))
		}
	}
	return AstBand{Pesos: original.Pesos, Pagos: pagos}
}

func catalogoAST() *astCatalogo {
	astOnce.Do(func() {
		if err := json.Unmarshal(astSlotsJSON, &struct {
			Slots *[]AstSlot `json:"slots"`
		}{Slots: &astCat.Slots}); err != nil {
			panic("ast_slots.json invalido: " + err.Error())
		}
		if err := json.Unmarshal(astPokerJSON, &struct {
			Pokers *[]AstPoker `json:"pokers"`
		}{Pokers: &astCat.Pokers}); err != nil {
			panic("ast_poker.json invalido: " + err.Error())
		}
		for i := range astCat.Slots {
			calibrar(&astCat.Slots[i])
		}
	})
	return &astCat
}

// BuscarSlot devuelve la maquina del catalogo por codigo.
func BuscarSlot(codigo string) (AstSlot, bool) {
	for _, s := range catalogoAST().Slots {
		if s.Codigo == codigo {
			return s, true
		}
	}
	return AstSlot{}, false
}

// BuscarPoker devuelve la mesa de video poker por codigo.
func BuscarPoker(codigo string) (AstPoker, bool) {
	for _, p := range catalogoAST().Pokers {
		if p.Codigo == codigo {
			return p, true
		}
	}
	return AstPoker{}, false
}

// SlotsAST y PokersAST listan el catalogo (para el panel y los seeds).
func SlotsAST() []AstSlot { return catalogoAST().Slots }

// PokersAST lista las mesas de video poker.
func PokersAST() []AstPoker { return catalogoAST().Pokers }

// SpecJuego describe un juego del catalogo AST para el seed de la base.
type SpecJuego struct {
	Codigo      string
	Nombre      string
	Descripcion string
	RTP         float64
	MaxMultipl  int64
	Tipo        string // "slots" o "video poker"
}

// SpecsAST devuelve los juegos del catalogo listos para sembrar.
func SpecsAST() []SpecJuego {
	var out []SpecJuego
	for _, s := range SlotsAST() {
		b := s.Bandas[s.BandaPorDefecto]
		// el RTP declarado es el que queda despues de calibrar (no la banda
		// nominal), asi la tabla y lo que se anuncia coinciden.
		rtp := math.Round(evLineaSlot(b)*float64(len(s.Lineas))*1000) / 10
		out = append(out, SpecJuego{
			Codigo:      s.Codigo,
			Nombre:      s.Nombre,
			Descripcion: fmt.Sprintf("Tragamonedas de %d rodillos y %d lineas.", s.Rodillos, len(s.Lineas)),
			RTP:         rtp,
			MaxMultipl:  MaxMultiplSlot(s.Codigo),
			Tipo:        "Slots",
		})
	}
	for _, p := range PokersAST() {
		out = append(out, SpecJuego{
			Codigo:      p.Codigo,
			Nombre:      p.Nombre,
			Descripcion: "Video poker de cinco cartas con reparto automatico.",
			RTP:         rtpPoker(p.Codigo),
			MaxMultipl:  MaxMultiplPoker(p.Codigo),
			Tipo:        "Video Poker",
		})
	}
	return out
}

// rtpPoker declara el RTP medido con la estrategia automatica del sistema y un
// reparto sin sesgo (ver sorteo_uniforme_test.go). Valores de RTP() con
// 200.000 iteraciones por mesa.
func rtpPoker(codigo string) float64 {
	valores := map[string]float64{
		"jacksorbetter": 78.4,
		"allamerican":   79.6,
		"tensorbetter":  80.7,
		"acesandfaces":  78.8,
		"eightsandaces": 79.8,
		"twowaysroyal":  80.1,
		"deucewild":     63.5,
		"jokerpoker":    60.9,
	}
	if v, ok := valores[codigo]; ok {
		return v
	}
	return 85
}
