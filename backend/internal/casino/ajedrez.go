package casino

import (
	"fmt"
	"sort"
)

// Ajedrez relampago: tactica de "mate en una". La casa propone una posicion
// real (FEN verificado con un motor de ajedrez) y el jugador elige el movimiento
// ganador entre cuatro opciones. Si acierta, cobra AjedrezMultVictoria; si no,
// pierde lo apostado.
//
// Es la version simplificada que pide el plan (no se integra Stockfish): en vez
// de una partida completa, cada jugada es un puzzle de jaque mate en un movimiento
// con piezas y movimientos reales. La posicion y el orden de las opciones salen
// de las semillas (provably fair), asi que la casa no puede cambiar la respuesta
// despues de la apuesta.

const (
	// AjedrezMultVictoria es lo que paga un mate acertado.
	AjedrezMultVictoria = 3.5
	// AjedrezOpciones es la cantidad de movimientos entre los que se elige.
	AjedrezOpciones = 4
)

// AjedrezPuzzle es una posicion de mate en una con tres jugadas distractoras.
type AjedrezPuzzle struct {
	FEN      string
	Correcta string
	Decoys   []string
	Tema     string
}

// AjedrezPuzzles son las posiciones curadas. Todas verificadas: la jugada
// Correcta es jaque mate y las distractoras son legales pero no mate.
var AjedrezPuzzles = []AjedrezPuzzle{
	{FEN: "8/2K5/k7/8/7Q/8/8/8 w - - 0 1", Correcta: "h4a4", Decoys: []string{"h4h5", "h4b4", "c7c8"}, Tema: "KQ"},
	{FEN: "7k/8/7K/8/8/8/8/5Q2 w - - 0 1", Correcta: "f1f8", Decoys: []string{"f1f3", "h6g6", "f1h3"}, Tema: "KQ"},
	{FEN: "8/8/8/3Q4/8/3K4/8/3k4 w - - 0 1", Correcta: "d5h1", Decoys: []string{"d5b5", "d5f7", "d5d7"}, Tema: "KQ"},
	{FEN: "8/8/4R3/8/8/8/k1K5/8 w - - 0 1", Correcta: "e6a6", Decoys: []string{"e6e3", "c2c3", "e6f6"}, Tema: "KR"},
	{FEN: "7k/5K2/8/8/8/8/8/5R2 w - - 0 1", Correcta: "f1h1", Decoys: []string{"f1e1", "f1a1", "f1f4"}, Tema: "KR"},
	{FEN: "8/8/8/5R2/8/8/8/5K1k w - - 0 1", Correcta: "f5h5", Decoys: []string{"f5c5", "f1e2", "f5f8"}, Tema: "KR"},
	{FEN: "8/8/2K5/1R6/8/8/4Q3/k7 w - - 0 1", Correcta: "e2b2", Decoys: []string{"e2g2", "e2h5", "b5b7"}, Tema: "KQR"},
	{FEN: "8/1Q6/8/8/k4K2/8/8/6R1 w - - 0 1", Correcta: "g1a1", Decoys: []string{"f4f3", "b7d7", "b7b6"}, Tema: "KQR"},
	{FEN: "8/2R5/8/8/K7/8/7Q/4k3 w - - 0 1", Correcta: "c7c1", Decoys: []string{"c7c2", "h2a2", "h2e2"}, Tema: "KQR"},
	{FEN: "8/8/8/8/2B5/8/7B/2K1k3 w - - 0 1", Correcta: "h2g3", Decoys: []string{"h2b8", "h2f4", "c1b1"}, Tema: "KBB"},
	{FEN: "3k4/1K6/2B5/8/8/6B1/8/8 w - - 0 1", Correcta: "g3h4", Decoys: []string{"c6e8", "b7b6", "c6h1"}, Tema: "KBB"},
	{FEN: "4k1K1/8/5B2/8/8/8/8/3B4 w - - 0 1", Correcta: "d1a4", Decoys: []string{"d1h5", "d1g4", "g8g7"}, Tema: "KBB"},
	{FEN: "2K5/k7/8/8/4Q3/8/6r1/8 w - - 0 1", Correcta: "e4b7", Decoys: []string{"e4a8", "e4d5", "e4h7"}, Tema: "KQvKR"},
	{FEN: "8/2r5/8/3Q4/8/7K/8/6k1 w - - 0 1", Correcta: "d5g2", Decoys: []string{"d5f3", "d5c6", "h3g4"}, Tema: "KQvKR"},
	{FEN: "4r3/6Q1/8/5K2/7k/8/8/8 w - - 0 1", Correcta: "g7g4", Decoys: []string{"g7g6", "g7f8", "g7g8"}, Tema: "KQvKR"},
	{FEN: "8/R7/8/8/8/4K3/8/4kr2 w - - 0 1", Correcta: "a7a1", Decoys: []string{"a7a2", "a7a8", "a7g7"}, Tema: "KRvKR"},
	{FEN: "7k/5K2/8/R7/8/6r1/8/8 w - - 0 1", Correcta: "a5h5", Decoys: []string{"a5f5", "a5a4", "f7f8"}, Tema: "KRvKR"},
	{FEN: "8/4R3/8/8/5K1k/8/8/3r4 w - - 0 1", Correcta: "e7h7", Decoys: []string{"e7e1", "e7d7", "e7e4"}, Tema: "KRvKR"},
	{FEN: "8/8/8/8/4N3/6K1/4N3/7k w - - 0 1", Correcta: "e4f2", Decoys: []string{"g3h4", "g3h3", "e2c1"}, Tema: "KNN"},
	{FEN: "K1k5/8/4N3/8/2N5/8/8/8 w - - 0 1", Correcta: "c4b6", Decoys: []string{"e6g5", "c4e3", "c4d2"}, Tema: "KNN"},
	{FEN: "8/5N2/8/3N3k/8/7K/8/8 w - - 0 1", Correcta: "d5f4", Decoys: []string{"f7d6", "d5f6", "d5e7"}, Tema: "KNN"},
	{FEN: "3k4/1P6/4K3/8/8/8/8/8 w - - 0 1", Correcta: "b7b8q", Decoys: []string{"e6e5", "b7b8n", "e6d6"}, Tema: "KP"},
	{FEN: "8/7P/7k/5K2/8/8/8/8 w - - 0 1", Correcta: "h7h8q", Decoys: []string{"f5f4", "f5g4", "h7h8b"}, Tema: "KP"},
	{FEN: "1k6/3P4/K7/8/8/8/8/8 w - - 0 1", Correcta: "d7d8q", Decoys: []string{"a6b6", "a6b5", "a6a5"}, Tema: "KP"},
}

// AjedrezElegir elige la posicion y mezcla las cuatro opciones a partir de las
// semillas. Devuelve el puzzle y las opciones en orden de boton.
func AjedrezElegir(clientSeed, serverSeed string, nonce int64) (AjedrezPuzzle, []string) {
	claves := Sortear(serverSeed, clientSeed, nonce, AjedrezOpciones+1, 1<<30)
	p := AjedrezPuzzles[claves[0]%len(AjedrezPuzzles)]
	opciones := make([]string, 0, AjedrezOpciones)
	opciones = append(opciones, p.Correcta)
	opciones = append(opciones, p.Decoys[:AjedrezOpciones-1]...)
	orden := make([]int, len(opciones))
	for i := range orden {
		orden[i] = i
	}
	sort.Slice(orden, func(a, b int) bool {
		ca, cb := claves[1+orden[a]], claves[1+orden[b]]
		if ca != cb {
			return ca < cb
		}
		return orden[a] < orden[b]
	})
	mezcladas := make([]string, len(opciones))
	for i, o := range orden {
		mezcladas[i] = opciones[o]
	}
	return p, mezcladas
}

// AjedrezEtiqueta arma la etiqueta de una opcion: pieza y destino.
func AjedrezEtiqueta(fen, uci string) string {
	if len(uci) < 4 {
		return uci
	}
	pieza := ajedrezPiezaEn(fen, uci[:2])
	nombre := ajedrezNombrePieza(pieza)
	return fmt.Sprintf("%s %s-%s", nombre, uci[:2], uci[2:4])
}

func ajedrezNombrePieza(p byte) string {
	switch p {
	case 'K', 'k':
		return "Rey"
	case 'Q', 'q':
		return "Dama"
	case 'R', 'r':
		return "Torre"
	case 'B', 'b':
		return "Alfil"
	case 'N', 'n':
		return "Caballo"
	case 'P', 'p':
		return "Peon"
	}
	return "Pieza"
}

// ajedrezPiezaEn devuelve la letra FEN de la pieza en la casilla (ej. "d1").
func ajedrezPiezaEn(fen, sq string) byte {
	if len(sq) < 2 {
		return 0
	}
	file := int(sq[0] - 'a')
	rank := int(sq[1] - '1')
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return 0
	}
	filas := 7 - rank
	pos := 0
	for i := 0; i < len(fen); i++ {
		c := fen[i]
		if c == ' ' {
			break
		}
		if c == '/' {
			continue
		}
		if c >= '1' && c <= '8' {
			pos += int(c - '0')
			continue
		}
		if pos == filas*8+file {
			return c
		}
		pos++
	}
	return 0
}

// AjedrezPremio paga si la opcion elegida es la correcta.
func AjedrezPremio(monto int64, acierto bool) int64 {
	if monto <= 0 || !acierto {
		return 0
	}
	return int64(float64(monto) * AjedrezMultVictoria)
}

// AjedrezMultiplicador devuelve el multiplicador segun el acierto.
func AjedrezMultiplicador(acierto bool) float64 {
	if acierto {
		return AjedrezMultVictoria
	}
	return 0
}

// AjedrezRepetir reproduce la eleccion de la ronda: devuelve el puzzle, las
// opciones mezcladas y si la jugada guardada era la correcta.
func AjedrezRepetir(clientSeed, serverSeed string, nonce int64, jugadas []string) (AjedrezPuzzle, []string, bool) {
	p, opciones := AjedrezElegir(clientSeed, serverSeed, nonce)
	if len(jugadas) != 1 {
		return p, opciones, false
	}
	return p, opciones, jugadas[0] == p.Correcta
}

// RTPAjedrez mide el retorno suponiendo que el jugador elige al azar entre las
// cuatro opciones (la linea de base de la ventaja de la casa).
func RTPAjedrez(iteraciones int) float64 {
	if iteraciones <= 0 {
		iteraciones = 200000
	}
	var suma float64
	for i := 0; i < iteraciones; i++ {
		seed := fmt.Sprintf("sim-%d", i)
		p, opciones := AjedrezElegir("sim", seed, int64(i))
		d := Sortear(seed, "sim", int64(i), 1, 1<<30)[0]
		elegida := opciones[d%len(opciones)]
		if elegida == p.Correcta {
			suma += AjedrezMultVictoria
		}
	}
	return suma / float64(iteraciones) * 100
}
