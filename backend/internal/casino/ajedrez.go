package casino

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
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

// ============================================================================
// Ajedrez completo: partida real contra el adversario de la casa.
// ============================================================================
//
// Motor sin dependencias: generacion de jugadas legales (peón, captura, al
// paso, enroque y promocion), jaque y jaque mate, y tablas por ahogado, regla
// de 50 movimientos, material insuficiente y triple repeticion. El adversario
// es una busqueda negamax con poda alfa-beta a profundidad 2 que valora
// material y control del centro; los empates se desempatan con las semillas,
// asi que la jugada de la casa se puede reproducir en la verificacion.
//
// Las casillas se numeran 0 = a8 ... 7 = h8, 56 = a1 ... 63 = h1 (orden FEN).

const (
	// AjedrezInicial es la posicion de apertura estandar.
	AjedrezInicial = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	// AjedrezProfundidadCasa es la profundidad de busqueda del adversario.
	AjedrezProfundidadCasa = 2
	// AjedrezPartidaMultVictoria y AjedrezPartidaMultEmpate son los pagos de la
	// partida completa. Calibrados por simulacion (RTPAjedrezPartida): el
	// jugador que usa el motor gana ~37 %, empata ~21 % y pierde ~41 %, asi que
	// x1.7 por victoria y devolver lo apostado en empate cierran el hábil en
	// ~85 % (banda 78-92 %); el casual queda declarado (~5 %).
	AjedrezPartidaMultVictoria = 1.7
	AjedrezPartidaMultEmpate   = 1.0
)

// AjedrezPosicion es el estado completo de un tablero.
type AjedrezPosicion struct {
	Casillas [64]byte // 0 vacio; mayusculas blancas, minusculas negras
	Turno    byte     // 'w' o 'b'
	Enroques byte     // 1 K blanco, 2 Q blanco, 4 K negro, 8 Q negro
	AlPaso   int      // casilla al paso (0..63) o -1
	Medios   int      // medio movimientos para la regla de 50
	Numero   int      // numero de jugada completa
}

// AjedrezMovimiento es una jugada pseudo-legal con sus banderas.
type AjedrezMovimiento struct {
	Desde     int
	Hasta     int
	Promocion byte // 0 o 'q','r','b','n'
	AlPaso    bool
	Enroque   bool
	Doble     bool
}

// AjedrezJuego es una partida completa: posicion, jugadas UCI y historial de
// repeticiones. Se puede reconstruir desde cero y reproducir jugada a jugada.
type AjedrezJuego struct {
	Posicion  AjedrezPosicion
	Jugadas   []string
	Historial map[string]int
	Resultado string // "", "blancas", "negras" o "tablas"
	Motivo    string
}

type ajedrezUndo struct {
	casillas [64]byte
	turno    byte
	enroques byte
	alPaso   int
	medios   int
	numero   int
}

var ajedrezCaballo = [8][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}}
var ajedrezRey = [8][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}
var ajedrezDiag = [4][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
var ajedrezRecto = [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}

// ajedrezCentro premia las cuatro casillas centrales y las ocho vecinas.
var ajedrezCentro = map[int]int{36: 20, 35: 20, 28: 20, 27: 20, 34: 10, 37: 10, 26: 10, 29: 10}

func ajedrezEnRango(f, c int) bool { return f >= 0 && f < 8 && c >= 0 && c < 8 }

func ajedrezMinuscula(p byte) byte {
	if p >= 'A' && p <= 'Z' {
		return p + ('a' - 'A')
	}
	return p
}

func (t *AjedrezPosicion) esBlanco(p byte) bool { return p >= 'A' && p <= 'Z' }

// ajedrezIndice convierte "e4" en el indice de casilla.
func ajedrezIndice(s string) (int, bool) {
	if len(s) != 2 {
		return 0, false
	}
	c := int(s[0] - 'a')
	f := int(s[1] - '1')
	if c < 0 || c > 7 || f < 0 || f > 7 {
		return 0, false
	}
	return (7-f)*8 + c, true
}

// ajedrezCasilla convierte un indice en notacion algebraica.
func ajedrezCasilla(idx int) string {
	if idx < 0 || idx > 63 {
		return "-"
	}
	return string([]byte{byte('a' + idx%8), byte('1' + 7 - idx/8)})
}

// NuevaAjedrezPosicion arma una posicion desde su FEN.
func NuevaAjedrezPosicion(fen string) (AjedrezPosicion, bool) {
	var t AjedrezPosicion
	t.AlPaso = -1
	campos := strings.Fields(fen)
	if len(campos) < 2 {
		return t, false
	}
	filas := strings.Split(campos[0], "/")
	if len(filas) != 8 {
		return t, false
	}
	for i, fila := range filas {
		pos := 0
		for _, c := range fila {
			if c >= '1' && c <= '8' {
				pos += int(c - '0')
				continue
			}
			if pos >= 8 {
				return t, false
			}
			t.Casillas[i*8+pos] = byte(c)
			pos++
		}
		if pos != 8 {
			return t, false
		}
	}
	if campos[1] != "w" && campos[1] != "b" {
		return t, false
	}
	t.Turno = campos[1][0]
	if len(campos) >= 3 && campos[2] != "-" {
		for _, c := range campos[2] {
			switch c {
			case 'K':
				t.Enroques |= 1
			case 'Q':
				t.Enroques |= 2
			case 'k':
				t.Enroques |= 4
			case 'q':
				t.Enroques |= 8
			}
		}
	}
	if len(campos) >= 4 && campos[3] != "-" {
		sq, ok := ajedrezIndice(campos[3])
		if !ok {
			return t, false
		}
		t.AlPaso = sq
	}
	if len(campos) >= 5 {
		if v, err := strconv.Atoi(campos[4]); err == nil {
			t.Medios = v
		}
	}
	if len(campos) >= 6 {
		if v, err := strconv.Atoi(campos[5]); err == nil {
			t.Numero = v
		}
	}
	return t, true
}

// AFEN serializa la posicion en notacion FEN.
func (t *AjedrezPosicion) AFEN() string {
	var b strings.Builder
	for f := 0; f < 8; f++ {
		if f > 0 {
			b.WriteByte('/')
		}
		vacias := 0
		for c := 0; c < 8; c++ {
			p := t.Casillas[f*8+c]
			if p == 0 {
				vacias++
				continue
			}
			if vacias > 0 {
				fmt.Fprintf(&b, "%d", vacias)
				vacias = 0
			}
			b.WriteByte(p)
		}
		if vacias > 0 {
			fmt.Fprintf(&b, "%d", vacias)
		}
	}
	fmt.Fprintf(&b, " %c ", t.Turno)
	if t.Enroques == 0 {
		b.WriteByte('-')
	} else {
		if t.Enroques&1 != 0 {
			b.WriteByte('K')
		}
		if t.Enroques&2 != 0 {
			b.WriteByte('Q')
		}
		if t.Enroques&4 != 0 {
			b.WriteByte('k')
		}
		if t.Enroques&8 != 0 {
			b.WriteByte('q')
		}
	}
	b.WriteByte(' ')
	if t.AlPaso < 0 {
		b.WriteByte('-')
	} else {
		b.WriteString(ajedrezCasilla(t.AlPaso))
	}
	fmt.Fprintf(&b, " %d %d", t.Medios, t.Numero)
	return b.String()
}

// claveRepeticion identifica la posicion para la triple repeticion: no
// incluye los contadores de la regla de 50.
func (t *AjedrezPosicion) claveRepeticion() string {
	var b strings.Builder
	for i := 0; i < 64; i++ {
		if t.Casillas[i] == 0 {
			b.WriteByte('.')
		} else {
			b.WriteByte(t.Casillas[i])
		}
	}
	fmt.Fprintf(&b, "|%c|%d|", t.Turno, t.Enroques)
	if t.AlPaso >= 0 {
		fmt.Fprintf(&b, "%d", t.AlPaso)
	}
	return b.String()
}

// ajedrezAtacado indica si la casilla sq esta atacada por el bando indicado.
func ajedrezAtacado(t *AjedrezPosicion, sq int, porBlanco bool) bool {
	f, c := sq/8, sq%8
	// Peones: un peon blanco ataca desde abajo (f+1), uno negro desde arriba.
	if porBlanco {
		if f+1 < 8 {
			if c-1 >= 0 && t.Casillas[(f+1)*8+c-1] == 'P' {
				return true
			}
			if c+1 < 8 && t.Casillas[(f+1)*8+c+1] == 'P' {
				return true
			}
		}
	} else {
		if f-1 >= 0 {
			if c-1 >= 0 && t.Casillas[(f-1)*8+c-1] == 'p' {
				return true
			}
			if c+1 < 8 && t.Casillas[(f-1)*8+c+1] == 'p' {
				return true
			}
		}
	}
	// Caballos.
	nb := byte('N')
	if !porBlanco {
		nb = 'n'
	}
	for _, o := range ajedrezCaballo {
		nf, nc := f+o[0], c+o[1]
		if ajedrezEnRango(nf, nc) && t.Casillas[nf*8+nc] == nb {
			return true
		}
	}
	// Rey.
	rb := byte('K')
	if !porBlanco {
		rb = 'k'
	}
	for _, o := range ajedrezRey {
		nf, nc := f+o[0], c+o[1]
		if ajedrezEnRango(nf, nc) && t.Casillas[nf*8+nc] == rb {
			return true
		}
	}
	// Diagonales (alfil o dama).
	ab, qb := byte('B'), byte('Q')
	if !porBlanco {
		ab, qb = 'b', 'q'
	}
	for _, d := range ajedrezDiag {
		for paso := 1; paso < 8; paso++ {
			nf, nc := f+d[0]*paso, c+d[1]*paso
			if !ajedrezEnRango(nf, nc) {
				break
			}
			p := t.Casillas[nf*8+nc]
			if p == 0 {
				continue
			}
			if p == ab || p == qb {
				return true
			}
			break
		}
	}
	// Orthogonales (torre o dama).
	tb := byte('R')
	if !porBlanco {
		tb = 'r'
	}
	for _, d := range ajedrezRecto {
		for paso := 1; paso < 8; paso++ {
			nf, nc := f+d[0]*paso, c+d[1]*paso
			if !ajedrezEnRango(nf, nc) {
				break
			}
			p := t.Casillas[nf*8+nc]
			if p == 0 {
				continue
			}
			if p == tb || p == qb {
				return true
			}
			break
		}
	}
	return false
}

func (t *AjedrezPosicion) reyCasilla(blancas bool) int {
	obj := byte('K')
	if !blancas {
		obj = 'k'
	}
	for i := 0; i < 64; i++ {
		if t.Casillas[i] == obj {
			return i
		}
	}
	return -1
}

// EnJaque indica si el rey del bando esta en jaque.
func (t *AjedrezPosicion) EnJaque(blancas bool) bool {
	sq := t.reyCasilla(blancas)
	if sq < 0 {
		return false
	}
	return ajedrezAtacado(t, sq, !blancas)
}

func ajedrezPromoMovs(desde, hasta int, esPromocion bool) []AjedrezMovimiento {
	if !esPromocion {
		return []AjedrezMovimiento{{Desde: desde, Hasta: hasta}}
	}
	var out []AjedrezMovimiento
	for _, pr := range []byte{'q', 'r', 'b', 'n'} {
		out = append(out, AjedrezMovimiento{Desde: desde, Hasta: hasta, Promocion: pr})
	}
	return out
}

// pseudoMovs genera todas las jugadas del turno sin filtrar legality.
func (t *AjedrezPosicion) pseudoMovs() []AjedrezMovimiento {
	blancas := t.Turno == 'w'
	var out []AjedrezMovimiento
	for idx := 0; idx < 64; idx++ {
		p := t.Casillas[idx]
		if p == 0 || t.esBlanco(p) != blancas {
			continue
		}
		f, c := idx/8, idx%8
		switch ajedrezMinuscula(p) {
		case 'p':
			dir, ini, fin := -1, 6, 0
			if !blancas {
				dir, ini, fin = 1, 1, 7
			}
			nf := f + dir
			if nf >= 0 && nf < 8 && t.Casillas[nf*8+c] == 0 {
				out = append(out, ajedrezPromoMovs(idx, nf*8+c, nf == fin)...)
				if f == ini && t.Casillas[(f+2*dir)*8+c] == 0 {
					out = append(out, AjedrezMovimiento{Desde: idx, Hasta: (f+2*dir)*8 + c, Doble: true})
				}
			}
			for _, dc := range []int{-1, 1} {
				nc := c + dc
				if nc < 0 || nc >= 8 {
					continue
				}
				dest := nf*8 + nc
				q := t.Casillas[dest]
				if q != 0 && t.esBlanco(q) != blancas {
					out = append(out, ajedrezPromoMovs(idx, dest, nf == fin)...)
				} else if q == 0 && dest == t.AlPaso {
					out = append(out, AjedrezMovimiento{Desde: idx, Hasta: dest, AlPaso: true})
				}
			}
		case 'n':
			for _, o := range ajedrezCaballo {
				nf, nc := f+o[0], c+o[1]
				if !ajedrezEnRango(nf, nc) {
					continue
				}
				dest := nf*8 + nc
				if t.Casillas[dest] == 0 || t.esBlanco(t.Casillas[dest]) != blancas {
					out = append(out, AjedrezMovimiento{Desde: idx, Hasta: dest})
				}
			}
		case 'b', 'r', 'q':
			var dirs [8][2]int
			n := 0
			if ajedrezMinuscula(p) != 'r' {
				for _, d := range ajedrezDiag {
					dirs[n] = d
					n++
				}
			}
			if ajedrezMinuscula(p) != 'b' {
				for _, d := range ajedrezRecto {
					dirs[n] = d
					n++
				}
			}
			for i := 0; i < n; i++ {
				d := dirs[i]
				for paso := 1; paso < 8; paso++ {
					nf, nc := f+d[0]*paso, c+d[1]*paso
					if !ajedrezEnRango(nf, nc) {
						break
					}
					dest := nf*8 + nc
					q := t.Casillas[dest]
					if q == 0 {
						out = append(out, AjedrezMovimiento{Desde: idx, Hasta: dest})
						continue
					}
					if t.esBlanco(q) != blancas {
						out = append(out, AjedrezMovimiento{Desde: idx, Hasta: dest})
					}
					break
				}
			}
		case 'k':
			for _, o := range ajedrezRey {
				nf, nc := f+o[0], c+o[1]
				if !ajedrezEnRango(nf, nc) {
					continue
				}
				dest := nf*8 + nc
				if t.Casillas[dest] == 0 || t.esBlanco(t.Casillas[dest]) != blancas {
					out = append(out, AjedrezMovimiento{Desde: idx, Hasta: dest})
				}
			}
			// Enroques: el rey no puede estar en jaque ni cruzar casillas
			// atacadas, y las casillas intermedias deben estar vacias.
			if blancas && idx == 60 {
				if t.Enroques&1 != 0 && t.Casillas[61] == 0 && t.Casillas[62] == 0 && t.Casillas[63] == 'R' &&
					!ajedrezAtacado(t, 60, false) && !ajedrezAtacado(t, 61, false) && !ajedrezAtacado(t, 62, false) {
					out = append(out, AjedrezMovimiento{Desde: 60, Hasta: 62, Enroque: true})
				}
				if t.Enroques&2 != 0 && t.Casillas[57] == 0 && t.Casillas[58] == 0 && t.Casillas[59] == 0 && t.Casillas[56] == 'R' &&
					!ajedrezAtacado(t, 60, false) && !ajedrezAtacado(t, 59, false) && !ajedrezAtacado(t, 58, false) {
					out = append(out, AjedrezMovimiento{Desde: 60, Hasta: 58, Enroque: true})
				}
			}
			if !blancas && idx == 4 {
				if t.Enroques&4 != 0 && t.Casillas[5] == 0 && t.Casillas[6] == 0 && t.Casillas[7] == 'r' &&
					!ajedrezAtacado(t, 4, true) && !ajedrezAtacado(t, 5, true) && !ajedrezAtacado(t, 6, true) {
					out = append(out, AjedrezMovimiento{Desde: 4, Hasta: 6, Enroque: true})
				}
				if t.Enroques&8 != 0 && t.Casillas[1] == 0 && t.Casillas[2] == 0 && t.Casillas[3] == 0 && t.Casillas[0] == 'r' &&
					!ajedrezAtacado(t, 4, true) && !ajedrezAtacado(t, 3, true) && !ajedrezAtacado(t, 2, true) {
					out = append(out, AjedrezMovimiento{Desde: 4, Hasta: 2, Enroque: true})
				}
			}
		}
	}
	return out
}

// aplicar realiza la jugada y devuelve el estado previo para deshacer.
func (t *AjedrezPosicion) aplicar(m AjedrezMovimiento) ajedrezUndo {
	u := ajedrezUndo{t.Casillas, t.Turno, t.Enroques, t.AlPaso, t.Medios, t.Numero}
	p := t.Casillas[m.Desde]
	blancas := t.esBlanco(p)
	captura := t.Casillas[m.Hasta] != 0 || m.AlPaso
	if m.AlPaso {
		cap := m.Hasta + 8
		if !blancas {
			cap = m.Hasta - 8
		}
		t.Casillas[cap] = 0
	}
	if m.Enroque {
		switch m.Hasta {
		case 62:
			t.Casillas[63] = 0
			t.Casillas[61] = 'R'
		case 58:
			t.Casillas[56] = 0
			t.Casillas[59] = 'R'
		case 6:
			t.Casillas[7] = 0
			t.Casillas[5] = 'r'
		case 2:
			t.Casillas[0] = 0
			t.Casillas[3] = 'r'
		}
	}
	t.Casillas[m.Desde] = 0
	if m.Promocion != 0 {
		promo := ajedrezMinuscula(m.Promocion)
		if blancas {
			promo = promo - ('a' - 'A')
		}
		t.Casillas[m.Hasta] = promo
	} else {
		t.Casillas[m.Hasta] = p
	}
	// Derechos de enroque.
	if p == 'K' {
		t.Enroques &^= 3
	}
	if p == 'k' {
		t.Enroques &^= 12
	}
	if m.Desde == 63 || m.Hasta == 63 {
		t.Enroques &^= 1
	}
	if m.Desde == 56 || m.Hasta == 56 {
		t.Enroques &^= 2
	}
	if m.Desde == 7 || m.Hasta == 7 {
		t.Enroques &^= 4
	}
	if m.Desde == 0 || m.Hasta == 0 {
		t.Enroques &^= 8
	}
	// Al paso: solo justo despues del avance doble del peon.
	t.AlPaso = -1
	if m.Doble {
		if blancas {
			t.AlPaso = m.Desde - 8
		} else {
			t.AlPaso = m.Desde + 8
		}
	}
	if ajedrezMinuscula(p) == 'p' || captura {
		t.Medios = 0
	} else {
		t.Medios++
	}
	if !blancas {
		t.Numero++
	}
	if t.Turno == 'w' {
		t.Turno = 'b'
	} else {
		t.Turno = 'w'
	}
	return u
}

func (t *AjedrezPosicion) deshacer(u ajedrezUndo) {
	t.Casillas = u.casillas
	t.Turno = u.turno
	t.Enroques = u.enroques
	t.AlPaso = u.alPaso
	t.Medios = u.medios
	t.Numero = u.numero
}

// JugadasLegales filtra las pseudo-legales dejando solo las que no dejan el
// propio rey en jaque.
func (t *AjedrezPosicion) JugadasLegales() []AjedrezMovimiento {
	blancas := t.Turno == 'w'
	pseudo := t.pseudoMovs()
	out := make([]AjedrezMovimiento, 0, len(pseudo))
	for _, m := range pseudo {
		u := t.aplicar(m)
		if !t.EnJaque(blancas) {
			out = append(out, m)
		}
		t.deshacer(u)
	}
	return out
}

// UCI devuelve la jugada en notacion UCI (e2e4, e7e8q).
func (m AjedrezMovimiento) UCI() string {
	s := ajedrezCasilla(m.Desde) + ajedrezCasilla(m.Hasta)
	if m.Promocion != 0 {
		s += string(ajedrezMinuscula(m.Promocion))
	}
	return s
}

// ajedrezMovimientoUCI busca la jugada legal que coincide con el UCI.
func ajedrezMovimientoUCI(t *AjedrezPosicion, uci string) (AjedrezMovimiento, bool) {
	if len(uci) != 4 && len(uci) != 5 {
		return AjedrezMovimiento{}, false
	}
	desde, ok1 := ajedrezIndice(uci[0:2])
	hasta, ok2 := ajedrezIndice(uci[2:4])
	if !ok1 || !ok2 {
		return AjedrezMovimiento{}, false
	}
	var promo byte
	if len(uci) == 5 {
		promo = ajedrezMinuscula(uci[4])
	}
	for _, m := range t.JugadasLegales() {
		if m.Desde == desde && m.Hasta == hasta && m.Promocion == promo {
			return m, true
		}
	}
	return AjedrezMovimiento{}, false
}

// NuevoAjedrezJuego abre una partida desde la posicion inicial.
func NuevoAjedrezJuego() AjedrezJuego {
	t, _ := NuevaAjedrezPosicion(AjedrezInicial)
	j := AjedrezJuego{Posicion: t, Historial: map[string]int{}}
	j.Historial[t.claveRepeticion()] = 1
	return j
}

// AjedrezReconstruir recrea la partida desde la posicion inicial aplicando las
// jugadas UCI guardadas. Devuelve false si alguna es ilegal.
func AjedrezReconstruir(jugadas []string) (AjedrezJuego, bool) {
	j := NuevoAjedrezJuego()
	for _, u := range jugadas {
		if !j.AplicarUCI(u) {
			return j, false
		}
	}
	return j, true
}

// AplicarUCI aplica una jugada del turno y actualiza el resultado.
func (j *AjedrezJuego) AplicarUCI(uci string) bool {
	if j.Resultado != "" {
		return false
	}
	m, ok := ajedrezMovimientoUCI(&j.Posicion, uci)
	if !ok {
		return false
	}
	j.Posicion.aplicar(m)
	j.Jugadas = append(j.Jugadas, uci)
	clave := j.Posicion.claveRepeticion()
	j.Historial[clave]++
	j.actualizar()
	return true
}

// LegalesUCI devuelve las jugadas legales del turno en notacion UCI, ordenadas.
func (j *AjedrezJuego) LegalesUCI() []string {
	movs := j.Posicion.JugadasLegales()
	out := make([]string, 0, len(movs))
	for _, m := range movs {
		out = append(out, m.UCI())
	}
	sort.Strings(out)
	return out
}

func (j *AjedrezJuego) actualizar() {
	t := &j.Posicion
	movs := t.JugadasLegales()
	if len(movs) == 0 {
		if t.EnJaque(t.Turno == 'w') {
			if t.Turno == 'w' {
				j.Resultado = "negras"
			} else {
				j.Resultado = "blancas"
			}
			j.Motivo = "jaque mate"
		} else {
			j.Resultado = "tablas"
			j.Motivo = "tablas por ahogado"
		}
		return
	}
	if t.Medios >= 100 {
		j.Resultado = "tablas"
		j.Motivo = "regla de 50 movimientos"
		return
	}
	if ajedrezMaterialInsuficiente(t) {
		j.Resultado = "tablas"
		j.Motivo = "material insuficiente"
		return
	}
	if j.Historial[t.claveRepeticion()] >= 3 {
		j.Resultado = "tablas"
		j.Motivo = "triple repeticion"
	}
}

// ajedrezMaterialInsuficiente detecta K vs K, K+B vs K y K+N vs K.
func ajedrezMaterialInsuficiente(t *AjedrezPosicion) bool {
	var piezas []byte
	for i := 0; i < 64; i++ {
		p := t.Casillas[i]
		if p != 0 && ajedrezMinuscula(p) != 'k' {
			piezas = append(piezas, ajedrezMinuscula(p))
		}
	}
	switch len(piezas) {
	case 0:
		return true
	case 1:
		return piezas[0] == 'b' || piezas[0] == 'n'
	}
	return false
}

func ajedrezValorPieza(p byte) int {
	switch ajedrezMinuscula(p) {
	case 'p':
		return 100
	case 'n':
		return 320
	case 'b':
		return 330
	case 'r':
		return 500
	case 'q':
		return 900
	}
	return 0
}

// ajedrezEvalua es la evaluacion material + centro desde el punto de vista de
// las blancas.
func ajedrezEvalua(t *AjedrezPosicion) int {
	var s int
	for i := 0; i < 64; i++ {
		p := t.Casillas[i]
		if p == 0 {
			continue
		}
		v := ajedrezValorPieza(p)
		if ajedrezMinuscula(p) != 'p' && ajedrezMinuscula(p) != 'k' {
			v += ajedrezCentro[i]
		}
		if t.esBlanco(p) {
			s += v
		} else {
			s -= v
		}
	}
	return s
}

func ajedrezEvaluaPara(t *AjedrezPosicion) int {
	if t.Turno == 'w' {
		return ajedrezEvalua(t)
	}
	return -ajedrezEvalua(t)
}

type ajedrezPar struct {
	m AjedrezMovimiento
	s int
}

// ajedrezOrdenar ordena las jugadas con capturas primero (MVV-LVA) y
// promociones, desempatando por UCI para que la busqueda sea determinista.
func ajedrezOrdenar(t *AjedrezPosicion, movs []AjedrezMovimiento) {
	par := make([]ajedrezPar, len(movs))
	for i, m := range movs {
		victima := t.Casillas[m.Hasta]
		if m.AlPaso {
			victima = 'p'
		}
		s := 0
		if victima != 0 {
			s += 10*ajedrezValorPieza(victima) - ajedrezValorPieza(t.Casillas[m.Desde])
		}
		if m.Promocion != 0 {
			s += 900
		}
		par[i] = ajedrezPar{m: m, s: s}
	}
	sort.SliceStable(par, func(a, b int) bool {
		if par[a].s != par[b].s {
			return par[a].s > par[b].s
		}
		return par[a].m.UCI() < par[b].m.UCI()
	})
	for i := range par {
		movs[i] = par[i].m
	}
}

const (
	ajedrezInfinito = 1 << 28
	ajedrezMate     = 100000
)

// ajedrezNegamax busca la evaluacion negamax con poda alfa-beta.
func ajedrezNegamax(t *AjedrezPosicion, depth, alpha, beta, ply int) int {
	if depth == 0 {
		if t.Medios >= 100 {
			return 0
		}
		return ajedrezEvaluaPara(t)
	}
	movs := t.JugadasLegales()
	if len(movs) == 0 {
		if t.EnJaque(t.Turno == 'w') {
			return -ajedrezMate + ply
		}
		return 0
	}
	ajedrezOrdenar(t, movs)
	mejor := -ajedrezInfinito
	for _, m := range movs {
		u := t.aplicar(m)
		s := -ajedrezNegamax(t, depth-1, -beta, -alpha, ply+1)
		t.deshacer(u)
		if s > mejor {
			mejor = s
		}
		if mejor > alpha {
			alpha = mejor
		}
		if alpha >= beta {
			break
		}
	}
	return mejor
}

// AjedrezJugadaCasa elige la jugada del adversario. Es una funcion pura de la
// posicion, el ply y las semillas: la verificacion la reproduce sin state.
func AjedrezJugadaCasa(clientSeed, serverSeed string, nonce int64, t *AjedrezPosicion, ply int) (AjedrezMovimiento, bool) {
	movs := t.JugadasLegales()
	if len(movs) == 0 {
		return AjedrezMovimiento{}, false
	}
	if len(movs) == 1 {
		return movs[0], true
	}
	ajedrezOrdenar(t, movs)
	mejor := -ajedrezInfinito
	var candidatos []AjedrezMovimiento
	for _, m := range movs {
		u := t.aplicar(m)
		s := -ajedrezNegamax(t, AjedrezProfundidadCasa-1, -ajedrezInfinito, ajedrezInfinito, 1)
		t.deshacer(u)
		if s > mejor {
			mejor = s
			candidatos = []AjedrezMovimiento{m}
		} else if s == mejor {
			candidatos = append(candidatos, m)
		}
	}
	if len(candidatos) == 1 {
		return candidatos[0], true
	}
	r := Sortear(serverSeed, clientSeed, nonce+int64(ply)*7919, 1, 1<<30)[0]
	return candidatos[r%len(candidatos)], true
}

// ajedrezMejorJugada devuelve la mejor jugada segun el mismo motor (jugador
// hábil de la simulacion).
func ajedrezMejorJugada(t *AjedrezPosicion, depth int) (AjedrezMovimiento, bool) {
	movs := t.JugadasLegales()
	if len(movs) == 0 {
		return AjedrezMovimiento{}, false
	}
	ajedrezOrdenar(t, movs)
	mejor := -ajedrezInfinito
	var elegida AjedrezMovimiento
	for _, m := range movs {
		u := t.aplicar(m)
		s := -ajedrezNegamax(t, depth-1, -ajedrezInfinito, ajedrezInfinito, 1)
		t.deshacer(u)
		if s > mejor {
			mejor = s
			elegida = m
		}
	}
	return elegida, true
}

// AjedrezPartidaPremio devuelve el premio de una partida completa: gana quien
// da mate; las tablas devuelven una fraccion de lo apostado.
func AjedrezPartidaPremio(monto int64, resultado, colorJugador string) int64 {
	if monto <= 0 {
		return 0
	}
	switch {
	case resultado == colorJugador:
		return int64(float64(monto) * AjedrezPartidaMultVictoria)
	case resultado == "tablas":
		return int64(float64(monto) * AjedrezPartidaMultEmpate)
	}
	return 0
}

// AjedrezPartidaMultiplicador devuelve el multiplicador segun el resultado.
func AjedrezPartidaMultiplicador(resultado, colorJugador string) float64 {
	if resultado == colorJugador {
		return AjedrezPartidaMultVictoria
	}
	if resultado == "tablas" {
		return AjedrezPartidaMultEmpate
	}
	return 0
}

// AjedrezResultadoSimulacion juega una partida completo y devuelve el resultado
// ("blancas", "negras" o "tablas"). El jugador hábil usa el mismo motor que la
// casa (espejo); el casual juega al azar.
func AjedrezResultadoSimulacion(semilla int64, habil, jugadorBlancas bool) string {
	sem := fmt.Sprintf("sim-%d", semilla)
	constante := fmt.Sprintf("cli-%d", semilla)
	j := NuevoAjedrezJuego()
	for j.Resultado == "" {
		esJugador := (j.Posicion.Turno == 'w') == jugadorBlancas
		if esJugador {
			var m AjedrezMovimiento
			if habil {
				// mismo motor que la casa, con su propia semilla de desempate
				m, _ = AjedrezJugadaCasa(constante+"-jugador", sem, semilla*2+1, &j.Posicion, len(j.Jugadas))
			} else {
				movs := j.Posicion.JugadasLegales()
				if len(movs) == 0 {
					break
				}
				r := Sortear(sem, constante, semilla, 1, 1<<30)[0]
				m = movs[r%len(movs)]
			}
			j.AplicarUCI(m.UCI())
			continue
		}
		m, ok := AjedrezJugadaCasa(constante, sem, semilla*2, &j.Posicion, len(j.Jugadas))
		if !ok {
			break
		}
		j.AplicarUCI(m.UCI())
	}
	return j.Resultado
}

// AjedrezSimularPartida juega una partida completo: el jugador usa el motor
// (habil) o jugadas al azar (casual) y la casa responde con AjedrezJugadaCasa.
// Devuelve el multiplicador obtenido.
func AjedrezSimularPartida(semilla int64, habil, jugadorBlancas bool) float64 {
	color := "blancas"
	if !jugadorBlancas {
		color = "negras"
	}
	return AjedrezPartidaMultiplicador(AjedrezResultadoSimulacion(semilla, habil, jugadorBlancas), color)
}

// RTPAjedrezPartida mide el retorno de la partida completa simulando dos
// perfiles de jugador: hábil (el mismo motor) y casual (jugadas al azar),
// alternando el color para que la declaracion no dependa de un solo lado.
func RTPAjedrezPartida(iteraciones int) (habil, casual float64) {
	if iteraciones <= 0 {
		iteraciones = 200
	}
	var sumaH, sumaC float64
	for i := 0; i < iteraciones; i++ {
		blancas := i%2 == 0
		sumaH += AjedrezSimularPartida(int64(i), true, blancas)
		sumaC += AjedrezSimularPartida(int64(i), false, blancas)
	}
	return sumaH / float64(iteraciones) * 100, sumaC / float64(iteraciones) * 100
}
