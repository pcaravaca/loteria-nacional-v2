package casino

import "fmt"

// PagoModalidad es una linea de la tabla de pagos de una modalidad. Se usa para
// documentar las reglas y para mostrarlas en el frontend.
type PagoModalidad struct {
	Etiqueta string `json:"etiqueta"`
	Mult     string `json:"mult"`
}

// Modalidad es una variante seleccionable de un juego que ya existe en el
// catalogo. No agrega filas a casino_juegos: el juego sigue identificandose por
// su codigo y la variante llega como parametro de la jugada
// ("modalidad": "europea"). La primera modalidad de cada lista es la clasica y
// es la que se juega cuando el cliente no manda el parametro, de modo que el
// comportamiento actual no cambia.
type Modalidad struct {
	Codigo      string          `json:"codigo"`
	Nombre      string          `json:"nombre"`
	Descripcion string          `json:"descripcion"`
	Pagos       []PagoModalidad `json:"pagos"`
	// MultMax es el mayor multiplicador que puede pagar la modalidad. Lo expone
	// el catalogo para que el cliente muestre el tope correcto segun la
	// modalidad elegida (el hilo en racha paga x6, no el x2 del catalogo).
	MultMax int64 `json:"mult_max"`
}

// modalidadesPorJuego declara las variantes de cada juego existente. Las claves
// son los codigos del catalogo (models.CasinoRuleta, etc.).
var modalidadesPorJuego = map[string][]Modalidad{
	"ruleta": {
		{
			Codigo:      "europea",
			Nombre:      "Ruleta Europea",
			Descripcion: "Rueda de 37 casillas (del 0 al 36): pleno x35 y color x2.",
			Pagos:       []PagoModalidad{{Etiqueta: "Pleno", Mult: "x35"}, {Etiqueta: "Color", Mult: "x2"}},
		},
		{
			Codigo:      "rapida",
			Nombre:      "Ruleta Rapida",
			Descripcion: "Rueda corta de 10 casillas (0, 00 y del 1 al 8): pleno x8 y color x2. Gira mas seguido y paga menos por acierto.",
			Pagos:       []PagoModalidad{{Etiqueta: "Pleno", Mult: "x8"}, {Etiqueta: "Color", Mult: "x2"}},
		},
	},
	"keno": {
		{
			Codigo:      "clasico",
			Nombre:      "Keno Clasico",
			Descripcion: "Elegi 5 numeros de 60 y salen 15 bolillas: 2 aciertos x1, 3 x5, 4 x15 y 5 x65.",
			Pagos:       []PagoModalidad{{Etiqueta: "5 aciertos", Mult: "x65"}, {Etiqueta: "4 aciertos", Mult: "x15"}, {Etiqueta: "3 aciertos", Mult: "x5"}, {Etiqueta: "2 aciertos", Mult: "x1"}},
		},
		{
			Codigo:      "relampago",
			Nombre:      "Keno Relampago",
			Descripcion: "Elegi 4 numeros de 40 y salen 8 bolillas: paga desde el primer acierto (x1, x2, x5 y x8).",
			Pagos:       []PagoModalidad{{Etiqueta: "4 aciertos", Mult: "x8"}, {Etiqueta: "3 aciertos", Mult: "x5"}, {Etiqueta: "2 aciertos", Mult: "x2"}, {Etiqueta: "1 acierto", Mult: "x1"}},
		},
	},
	"raspa": {
		{
			Codigo:      "clasica",
			Nombre:      "Raspa Clasica",
			Descripcion: "Tres casillas con 6 simbolos: tres iguales x18 y cualquier par x1.",
			Pagos:       []PagoModalidad{{Etiqueta: "Tres iguales", Mult: "x18"}, {Etiqueta: "Cualquier par", Mult: "x1"}},
		},
		{
			Codigo:      "expres",
			Nombre:      "Raspa Expres",
			Descripcion: "Tres casillas con 4 simbolos: tres iguales x5 y cualquier par x1. Los premios salen mas seguido.",
			Pagos:       []PagoModalidad{{Etiqueta: "Tres iguales", Mult: "x5"}, {Etiqueta: "Cualquier par", Mult: "x1"}},
		},
	},
	"baccarat": {
		{
			Codigo:      "estandar",
			Nombre:      "Baccarat Estandar",
			Descripcion: "Jugador x2, banca x2 con 5% de comision y empate x8. En empate se devuelve lo apostado a jugador o banca.",
			Pagos:       []PagoModalidad{{Etiqueta: "Jugador", Mult: "x2"}, {Etiqueta: "Banca", Mult: "x2 (5% comision)"}, {Etiqueta: "Empate", Mult: "x8"}},
		},
		{
			Codigo:      "sin-comision",
			Nombre:      "Baccarat Sin Comision",
			Descripcion: "La banca paga x2 sin comision, pero si la banca gana con 6 paga x1. Empate x8; en empate se pierde lo apostado a jugador o banca.",
			Pagos:       []PagoModalidad{{Etiqueta: "Jugador", Mult: "x2"}, {Etiqueta: "Banca", Mult: "x2 (x1 si gana con 6)"}, {Etiqueta: "Empate", Mult: "x8"}},
		},
	},
	"alto": {
		{
			Codigo:      "clasico",
			Nombre:      "Bajo o Sobre Clasico",
			Descripcion: "Dado del 1 al 100: elegi el tope y acerta si cae abajo o arriba. Un tiro, un pago.",
			Pagos:       []PagoModalidad{{Etiqueta: "2%", Mult: "x48"}, {Etiqueta: "5%", Mult: "x19"}, {Etiqueta: "10%", Mult: "x9"}, {Etiqueta: "25%", Mult: "x3"}, {Etiqueta: "48%", Mult: "x2"}, {Etiqueta: "75%", Mult: "x1"}, {Etiqueta: "90%", Mult: "x1"}, {Etiqueta: "95%", Mult: "x1"}},
		},
		{
			Codigo:      "racha",
			Nombre:      "Bajo o Sobre en Racha",
			Descripcion: "Dos tiros con el mismo tope: acerta los dos seguidos y cobra x4. Si falla uno solo, pierde la apuesta.",
			Pagos:       []PagoModalidad{{Etiqueta: "Dos aciertos seguidos", Mult: "x4"}},
		},
	},
	"hilo": {
		{
			Codigo:      "clasico",
			Nombre:      "Mayor o Menor Clasico",
			Descripcion: "Adivina si la proxima carta es mas alta o mas baja que la base. Si empata, perdes.",
			Pagos:       []PagoModalidad{{Etiqueta: "Acierto", Mult: "x2"}},
		},
		{
			Codigo:      "racha",
			Nombre:      "Mayor o Menor en Racha",
			Descripcion: "Tres cartas: acerta dos predicciones seguidas con el mismo sentido y cobra x6.",
			Pagos:       []PagoModalidad{{Etiqueta: "Dos aciertos seguidos", Mult: "x6"}},
		},
	},
	"cascada": {
		{
			Codigo:      "clasica",
			Nombre:      "Cascada Clasica",
			Descripcion: "Nueve filas de pinchos y 10 casillas: paga del centro (x0) a los bordes (x10).",
			Pagos:       []PagoModalidad{{Etiqueta: "Bordes", Mult: "x10"}, {Etiqueta: "Centro", Mult: "x0"}},
		},
		{
			Codigo:      "rapida",
			Nombre:      "Cascada Rapida",
			Descripcion: "Cinco filas de pinchos y 6 casillas: cae rapidisimamente y paga x4 en los bordes.",
			Pagos:       []PagoModalidad{{Etiqueta: "Bordes", Mult: "x4"}, {Etiqueta: "Centro", Mult: "x0"}},
		},
		{
			Codigo:      "profunda",
			Nombre:      "Cascada Profunda",
			Descripcion: "Doce filas de pinchos y 13 casillas: recorrido larguisimo con x10 en los bordes.",
			Pagos:       []PagoModalidad{{Etiqueta: "Bordes", Mult: "x10"}, {Etiqueta: "Centro", Mult: "x0"}},
		},
	},
}

// Modalidades devuelve las modalidades declaradas para un juego, en orden. El
// primer elemento es siempre la modalidad por defecto.
func Modalidades(codigoJuego string) []Modalidad {
	origen := modalidadesPorJuego[codigoJuego]
	out := make([]Modalidad, len(origen))
	copy(out, origen)
	for i := range out {
		out[i].MultMax = multMaximaModalidad(codigoJuego, out[i].Codigo)
	}
	return out
}

// ModalidadPorDefecto devuelve el codigo de la modalidad clasica del juego.
func ModalidadPorDefecto(codigoJuego string) string {
	if ms := modalidadesPorJuego[codigoJuego]; len(ms) > 0 {
		return ms[0].Codigo
	}
	return ""
}

// ModalidadPorCodigo devuelve la modalidad pedida. La cadena vacia es valida y
// significa "usar la modalidad por defecto".
func ModalidadPorCodigo(codigoJuego, codigo string) (Modalidad, bool) {
	if codigo == "" {
		codigo = ModalidadPorDefecto(codigoJuego)
	}
	for _, m := range modalidadesPorJuego[codigoJuego] {
		if m.Codigo == codigo {
			return m, true
		}
	}
	return Modalidad{}, false
}

// ModalidadValida indica si el codigo pertenece al juego.
func ModalidadValida(codigoJuego, codigo string) bool {
	_, ok := ModalidadPorCodigo(codigoJuego, codigo)
	return ok
}

// MultMaximoJuego devuelve el mayor multiplicador que puede pagar el juego
// considerando todas sus modalidades. Se usa para recortar la apuesta maxima
// sin tocar el multiplicador que audita el jugador.
func MultMaximoJuego(codigoJuego string) int64 {
	var max int64
	for _, m := range modalidadesPorJuego[codigoJuego] {
		if v := multMaximaModalidad(codigoJuego, m.Codigo); v > max {
			max = v
		}
	}
	return max
}

// MultMaximoModalidad devuelve el mayor multiplicador que puede pagar la
// modalidad; 0 si el codigo no existe.
func MultMaximoModalidad(codigoJuego, codigo string) int64 {
	if !ModalidadValida(codigoJuego, codigo) {
		return 0
	}
	return multMaximaModalidad(codigoJuego, codigo)
}

// DataModalidad arma el bloque de Outcome.Data que el frontend necesita para
// mostrar la modalidad elegida: su codigo, su nombre, su descripcion y su tabla
// de pagos.
func DataModalidad(codigoJuego, codigo string) map[string]any {
	m, ok := ModalidadPorCodigo(codigoJuego, codigo)
	if !ok {
		return map[string]any{"modalidad": ""}
	}
	return map[string]any{
		"modalidad":        m.Codigo,
		"modalidad_nombre": m.Nombre,
		"descripcion":      m.Descripcion,
		"pagos":            m.Pagos,
		"mult_max":         multMaximaModalidad(codigoJuego, m.Codigo),
	}
}

// normalizarModalidad resuelve el parametro "modalidad" contra el juego. El
// parametro es opcional: si no viene (o viene vacio) se usa la modalidad
// clasica, asi que una jugada vieja sigue resolviendose igual.
func normalizarModalidad(codigoJuego string, p Params) (string, error) {
	v, ok := p["modalidad"]
	if !ok || v == nil {
		return ModalidadPorDefecto(codigoJuego), nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("modalidad invalida: debe ser un texto")
	}
	if s == "" {
		return ModalidadPorDefecto(codigoJuego), nil
	}
	if _, ok := ModalidadPorCodigo(codigoJuego, s); !ok {
		return "", fmt.Errorf("modalidad invalida: %s", s)
	}
	return s, nil
}
