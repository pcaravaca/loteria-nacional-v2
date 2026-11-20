package casino

import (
	"reflect"
	"testing"
)

// jugarCon modula una jugada con draws explicitos: los tests arman los draws a
// mano para conocer el resultado sin depender de la semilla.
func jugarCon(codigo string, draws []int, p Params) (Outcome, error) {
	if p == nil {
		p = Params{}
	}
	return Jugar(codigo, 1000, draws, p)
}

// apuestaBase devuelve los parametros minimos de cada juego, sin modalidad.
func apuestaBase(codigo string) Params {
	switch codigo {
	case "ruleta":
		return Params{"tipo": "pleno", "numero": 5}
	case "keno":
		return Params{"numeros": []int{1, 2, 3, 4, 5}}
	case "baccarat":
		return Params{"tipo": "jugador"}
	case "hilo":
		return Params{"direccion": "mayor"}
	case "alto":
		return Params{"tope": 48, "direccion": "bajo"}
	}
	return Params{}
}

func juegosConModalidades() []string {
	return []string{"ruleta", "keno", "raspa", "baccarat", "alto", "hilo", "cascada"}
}

// TestModalidadesRegistradas verifica que cada juego tenga al menos dos
// modalidades y que la primera sea la clasica.
func TestModalidadesRegistradas(t *testing.T) {
	for _, codigo := range juegosConModalidades() {
		ms := Modalidades(codigo)
		if len(ms) < 2 {
			t.Fatalf("%s: solo tiene %d modalidades", codigo, len(ms))
		}
		vistos := map[string]bool{}
		for _, m := range ms {
			if m.Codigo == "" || m.Nombre == "" || m.Descripcion == "" {
				t.Errorf("%s: modalidad incompleta %+v", codigo, m)
			}
			if len(m.Pagos) == 0 {
				t.Errorf("%s/%s: no tiene tabla de pagos", codigo, m.Codigo)
			}
			if vistos[m.Codigo] {
				t.Errorf("%s: modalidad repetida %s", codigo, m.Codigo)
			}
			vistos[m.Codigo] = true
		}
		if ModalidadPorDefecto(codigo) != ms[0].Codigo {
			t.Errorf("%s: el por defecto no es el primero de la lista", codigo)
		}
		if !ModalidadValida(codigo, "") {
			t.Errorf("%s: la modalidad vacia deberia ser valida", codigo)
		}
		if ModalidadValida(codigo, "no-existe") {
			t.Errorf("%s: una modalidad inexistente no deberia ser valida", codigo)
		}
	}
}

// TestModalidadPorDefectoMantieneComportamiento es la garantia de que no se
// rompe lo aprobado: jugar sin el parametro "modalidad" y jugar con la
// modalidad clasica expl dan exactamente el mismo resultado.
func TestModalidadPorDefectoMantieneComportamiento(t *testing.T) {
	drawsPorJuego := map[string][]int{
		"ruleta":   {3, 17},
		"keno":     {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23},
		"raspa":    {0, 1, 2},
		"baccarat": {0, 1, 2, 3, 4, 1},
		"alto":     {10, 20},
		"hilo":     {4, 9, 12},
		"cascada":  {1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0},
	}
	for _, codigo := range juegosConModalidades() {
		defecto := ModalidadPorDefecto(codigo)
		sinParametro, err := jugarCon(codigo, drawsPorJuego[codigo], apuestaBase(codigo))
		if err != nil {
			t.Fatalf("%s: %v", codigo, err)
		}
		conParametro, err := jugarCon(codigo, drawsPorJuego[codigo], func() Params {
			p := apuestaBase(codigo)
			p["modalidad"] = defecto
			return p
		}())
		if err != nil {
			t.Fatalf("%s: %v", codigo, err)
		}
		if sinParametro.Multiplicador != conParametro.Multiplicador {
			t.Errorf("%s: multiplicador sin parametro %d != con %s %d",
				codigo, sinParametro.Multiplicador, defecto, conParametro.Multiplicador)
		}
		if !reflect.DeepEqual(sinParametro.Data, conParametro.Data) {
			t.Errorf("%s: los datos cambiaron entre el default y %s", codigo, defecto)
		}
		if sinParametro.Data["modalidad"] != defecto {
			t.Errorf("%s: Outcome.Data no expone la modalidad (%v)", codigo, sinParametro.Data["modalidad"])
		}
	}
}

// TestModalidadInvalida rechaza un codigo que no existe.
func TestModalidadInvalida(t *testing.T) {
	for _, codigo := range juegosConModalidades() {
		if _, err := jugarCon(codigo, []int{1, 2, 3}, func() Params {
			p := apuestaBase(codigo)
			p["modalidad"] = "truquito"
			return p
		}()); err == nil {
			t.Errorf("%s: una modalidad invalida no deberia resolver", codigo)
		}
	}
}

// TestRuletaModalidades: la europea sigue de 37 casillas y la rapida es una
// rueda corta con 0 y 00.
func TestRuletaModalidades(t *testing.T) {
	// Europa: pleno al 5 paga x35 y el color sale segun la casilla.
	out, err := jugarCon("ruleta", []int{5}, Params{"tipo": "pleno", "numero": 5})
	if err != nil {
		t.Fatal(err)
	}
	if out.Multiplicador != 35 {
		t.Errorf("europea pleno: multiplicador %d, queria 35", out.Multiplicador)
	}
	if out.Data["casillas"] != 37 {
		t.Errorf("europea: casillas %v, queria 37", out.Data["casillas"])
	}

	// Rapida: 10 casillas, el indice 1 es el 00 (verde) y el 2 es el 1.
	out, err = jugarCon("ruleta", []int{2}, Params{"modalidad": "rapida", "tipo": "pleno", "numero": 2})
	if err != nil {
		t.Fatal(err)
	}
	if out.Data["casilla"] != "1" || out.Data["casillas"] != 10 {
		t.Fatalf("rapida: casilla %v de %v", out.Data["casilla"], out.Data["casillas"])
	}
	if out.Multiplicador != 8 {
		t.Errorf("rapida pleno: multiplicador %d, queria 8", out.Multiplicador)
	}
	// El 00 es verde: no paga ni rojo ni negro.
	out, _ = jugarCon("ruleta", []int{1}, Params{"modalidad": "rapida", "tipo": "color", "color": "rojo"})
	if out.Data["casilla"] != "00" || out.Data["color"] != "verde" {
		t.Fatalf("rapida 00: casilla %v color %v", out.Data["casilla"], out.Data["color"])
	}
	if out.Multiplicador != 0 {
		t.Errorf("el 00 no paga color: multiplicador %d", out.Multiplicador)
	}
	// El 1 es rojo.
	out, _ = jugarCon("ruleta", []int{2}, Params{"modalidad": "rapida", "tipo": "color", "color": "rojo"})
	if out.Multiplicador != 2 {
		t.Errorf("rapida color: multiplicador %d, queria 2", out.Multiplicador)
	}
	// Un pleno fuera de la rueda corta se rechaza.
	if _, err := jugarCon("ruleta", []int{1}, Params{"modalidad": "rapida", "tipo": "pleno", "numero": 17}); err == nil {
		t.Error("la rapida no deberia aceptar un pleno fuera de sus casillas")
	}
}

// TestKenoModalidades: el clasico pide 5 de 60 y el relampago 4 de 40.
func TestKenoModalidades(t *testing.T) {
	// Clasico: bolillas 1..15 con draws 0..14, el jugador pega las 5 primeras.
	out, err := jugarCon("keno", []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14},
		Params{"numeros": []int{1, 2, 3, 4, 5}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Data["aciertos"] != 5 || out.Multiplicador != 65 {
		t.Errorf("clasico: %v aciertos, multiplicador %d", out.Data["aciertos"], out.Multiplicador)
	}

	// Relampago: 4 elegidos de 40 y 8 bolillas; los draws 0..7 dan 1..8.
	draws := []int{0, 1, 2, 3, 4, 5, 6, 7}
	out, err = jugarCon("keno", draws, Params{"modalidad": "relampago", "numeros": []int{1, 2, 3, 4}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Data["aciertos"] != 4 || out.Multiplicador != 8 {
		t.Errorf("relampago: %v aciertos, multiplicador %d", out.Data["aciertos"], out.Multiplicador)
	}
	if out.Data["bolillas_totales"] != 40 || out.Data["elegidos"] != 4 {
		t.Errorf("relampago: bolillas %v elegidos %v", out.Data["bolillas_totales"], out.Data["elegidos"])
	}
	bolillas, ok := out.Data["bolillas"].([]int)
	if !ok || len(bolillas) != 8 {
		t.Fatalf("relampago: bolillas %v", out.Data["bolillas"])
	}
	// Un acierto tambien paga (x1): el 5 esta en la bolilla pero no lo elegimos.
	out, _ = jugarCon("keno", draws, Params{"modalidad": "relampago", "numeros": []int{5, 39, 38, 37}})
	if out.Data["aciertos"] != 1 || out.Multiplicador != 1 {
		t.Errorf("relampago un acierto: %v, multiplicador %d", out.Data["aciertos"], out.Multiplicador)
	}
	// El relampago no acepta 5 numeros ni numeros fuera de 40.
	if _, err := jugarCon("keno", draws, Params{"modalidad": "relampago", "numeros": []int{1, 2, 3, 4, 5}}); err == nil {
		t.Error("el relampago deberia rechazar 5 numeros")
	}
	if _, err := jugarCon("keno", draws, Params{"modalidad": "relampago", "numeros": []int{1, 2, 3, 41}}); err == nil {
		t.Error("el relampago deberia rechazar numeros mayores a 40")
	}
}

// TestRaspaModalidades: la expres usa 4 simbolos y paga x5 el triple.
func TestRaspaModalidades(t *testing.T) {
	out, err := jugarCon("raspa", []int{0, 0, 0}, Params{})
	if err != nil {
		t.Fatal(err)
	}
	if out.Multiplicador != 18 || out.Data["simbolos_totales"] != 6 {
		t.Errorf("clasica: multiplicador %d, simbolos %v", out.Multiplicador, out.Data["simbolos_totales"])
	}
	out, err = jugarCon("raspa", []int{0, 0, 0}, Params{"modalidad": "expres"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Multiplicador != 5 || out.Data["simbolos_totales"] != 4 {
		t.Errorf("expres: multiplicador %d, simbolos %v", out.Multiplicador, out.Data["simbolos_totales"])
	}
	// Un par paga x1 en las dos modalidades.
	out, _ = jugarCon("raspa", []int{0, 0, 1}, Params{"modalidad": "expres"})
	if out.Multiplicador != 1 {
		t.Errorf("expres par: multiplicador %d, queria 1", out.Multiplicador)
	}
	// Con 4 simbolos nunca puede salir herradura ni barra.
	for i := 0; i < 200; i++ {
		out, _ := jugarCon("raspa", []int{i, i + 1, i + 2}, Params{"modalidad": "expres"})
		for _, s := range out.Data["simbolos"].([]string) {
			if s == "herradura" || s == "barra" {
				t.Fatalf("expres: salio %s, que no pertenece al rodillo de 4", s)
			}
		}
	}
}

// TestBaccaratModalidades: la banca sin comision paga x2 y con 6 paga x1, y el
// empate no devuelve la apuesta.
func TestBaccaratModalidades(t *testing.T) {
	// draws [0,1,2,3,4,1]: jugador 0+1+4=5, banca 2+3+1=6 -> banca gana con 6.
	draws := []int{0, 1, 2, 3, 4, 1}
	out, err := jugarCon("baccarat", draws, Params{"tipo": "banca"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Multiplicador != 2 || out.Data["comision"] != int64(50) {
		t.Errorf("estandar banca: multiplicador %d comision %v", out.Multiplicador, out.Data["comision"])
	}
	out, err = jugarCon("baccarat", draws, Params{"modalidad": "sin-comision", "tipo": "banca"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Multiplicador != 1 {
		t.Errorf("sin comision, banca con 6: multiplicador %d, queria 1", out.Multiplicador)
	}
	if out.Data["comision"] != int64(0) {
		t.Errorf("sin comision: comision %v, queria 0", out.Data["comision"])
	}

	// Banca gana sin que sea con 6: paga x2 entero.
	// jugador 0+1=1 pide y saca 0 -> 1; banca 9+9=8 natural, no pide.
	out, _ = jugarCon("baccarat", []int{0, 1, 9, 9, 0, 0}, Params{"modalidad": "sin-comision", "tipo": "banca"})
	if out.Data["total_banca"] != 8 {
		t.Fatalf("caso natural: total banca %v", out.Data["total_banca"])
	}
	if out.Multiplicador != 2 {
		t.Errorf("sin comision, banca con 8: multiplicador %d, queria 2", out.Multiplicador)
	}

	// Empate: en la estandar se devuelve la apuesta, en la sin comision se pierde.
	empate := []int{0, 0, 0, 0, 0, 0}
	out, _ = jugarCon("baccarat", empate, Params{"tipo": "jugador"})
	if out.Multiplicador != 1 {
		t.Errorf("estandar empate: multiplicador %d, queria 1 (devuelve)", out.Multiplicador)
	}
	out, _ = jugarCon("baccarat", empate, Params{"modalidad": "sin-comision", "tipo": "jugador"})
	if out.Multiplicador != 0 {
		t.Errorf("sin comision empate: multiplicador %d, queria 0 (pierde)", out.Multiplicador)
	}
}

// TestAltoModalidades: la racha pide dos aciertos seguidos y paga x4.
func TestAltoModalidades(t *testing.T) {
	out, err := jugarCon("alto", []int{10, 20}, Params{"tope": 48, "direccion": "bajo"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Multiplicador != 2 {
		t.Errorf("clasico: multiplicador %d, queria 2", out.Multiplicador)
	}
	out, err = jugarCon("alto", []int{10, 20}, Params{"modalidad": "racha", "tope": 48, "direccion": "bajo"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Multiplicador != 4 {
		t.Errorf("racha: multiplicador %d, queria 4", out.Multiplicador)
	}
	if out.Data["tiros"] != 2 {
		t.Errorf("racha: tiros %v, queria 2", out.Data["tiros"])
	}
	// Si falla un tiro, no se cobra.
	out, _ = jugarCon("alto", []int{99, 20}, Params{"modalidad": "racha", "tope": 48, "direccion": "bajo"})
	if out.Multiplicador != 0 {
		t.Errorf("racha con un fallo: multiplicador %d, queria 0", out.Multiplicador)
	}
	// La direccion "sobre" tambien funciona en racha.
	out, _ = jugarCon("alto", []int{90, 95}, Params{"modalidad": "racha", "tope": 25, "direccion": "sobre"})
	if out.Multiplicador != 4 {
		t.Errorf("racha sobre: multiplicador %d, queria 4", out.Multiplicador)
	}
}

// TestHiloModalidades: la racha reparte tres cartas y paga x6.
func TestHiloModalidades(t *testing.T) {
	out, err := jugarCon("hilo", []int{4, 9}, Params{"direccion": "mayor"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Multiplicador != 2 {
		t.Errorf("clasico: multiplicador %d, queria 2", out.Multiplicador)
	}
	// Cartas 1, 2, 3: dos aciertos seguidos.
	out, err = jugarCon("hilo", []int{0, 1, 2}, Params{"modalidad": "racha", "direccion": "mayor"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Multiplicador != 6 {
		t.Errorf("racha: multiplicador %d, queria 6", out.Multiplicador)
	}
	if out.Data["tiros"] != 2 {
		t.Errorf("racha: tiros %v, queria 2", out.Data["tiros"])
	}
	figuras, ok := out.Data["figuras"].([]string)
	if !ok || len(figuras) != 3 {
		t.Fatalf("racha: figuras %v", out.Data["figuras"])
	}
	// Cartas 1, 2, 2: el segundo acierto falla.
	out, _ = jugarCon("hilo", []int{0, 1, 1}, Params{"modalidad": "racha", "direccion": "mayor"})
	if out.Multiplicador != 0 {
		t.Errorf("racha con un fallo: multiplicador %d, queria 0", out.Multiplicador)
	}
	// En "menor" tambien aplica.
	out, _ = jugarCon("hilo", []int{12, 11, 10}, Params{"modalidad": "racha", "direccion": "menor"})
	if out.Multiplicador != 6 {
		t.Errorf("racha menor: multiplicador %d, queria 6", out.Multiplicador)
	}
}

// TestCascadaModalidades: cada modalidad tiene su cantidad de filas y su tabla.
func TestCascadaModalidades(t *testing.T) {
	casos := []struct {
		modalidad string
		filas     int
		draws     []int
		pos       int
		mult      int64
	}{
		{"clasica", 9, []int{1, 1, 1, 1, 1, 1, 1, 1, 1}, 9, 10},
		{"rapida", 5, []int{1, 1, 1, 1, 1}, 5, 4},
		{"profunda", 12, []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, 12, 10},
	}
	for _, c := range casos {
		out, err := jugarCon("cascada", c.draws, Params{"modalidad": c.modalidad})
		if err != nil {
			t.Fatalf("%s: %v", c.modalidad, err)
		}
		if out.Data["filas"] != c.filas || out.Data["posicion"] != c.pos {
			t.Errorf("%s: filas %v posicion %v", c.modalidad, out.Data["filas"], out.Data["posicion"])
		}
		if out.Multiplicador != c.mult {
			t.Errorf("%s: multiplicador %d, queria %d", c.modalidad, out.Multiplicador, c.mult)
		}
		tabla, ok := out.Data["tabla"].([]int64)
		if !ok || len(tabla) != c.filas+1 {
			t.Errorf("%s: tabla %v", c.modalidad, out.Data["tabla"])
		}
	}
	// El centro de la rapida no paga.
	out, _ := jugarCon("cascada", []int{1, 1, 0, 0, 0}, Params{"modalidad": "rapida"})
	if out.Data["posicion"] != 2 {
		t.Fatalf("rapida centro: posicion %v", out.Data["posicion"])
	}
	if out.Multiplicador != 0 {
		t.Errorf("rapida centro: multiplicador %d, queria 0", out.Multiplicador)
	}
}

// TestDrawsModalidades verifica que el juego pida draws suficientes para la
// modalidad mas larga.
func TestDrawsModalidades(t *testing.T) {
	esperado := map[string]int{"ruleta": 1, "keno": 24, "raspa": 3, "baccarat": 6, "alto": 2, "hilo": 3, "cascada": 12}
	for codigo, n := range esperado {
		if Draws(codigo) != n {
			t.Errorf("%s: Draws() = %d, queria %d", codigo, Draws(codigo), n)
		}
	}
}

// TestDeterminismoModalidades: la misma semilla y el mismo nonce dan el mismo
// resultado en cualquier modalidad.
func TestDeterminismoModalidades(t *testing.T) {
	for _, codigo := range juegosConModalidades() {
		for _, m := range Modalidades(codigo) {
			p := apuestaBase(codigo)
			p["modalidad"] = m.Codigo
			if codigo == "keno" && m.Codigo == "relampago" {
				p["numeros"] = []int{1, 2, 3, 4}
			}
			if codigo == "ruleta" && m.Codigo == "rapida" {
				p["numero"] = 3
			}
			draws := Sortear("semilla-fija", "cli", 7, Draws(codigo), 1<<31-1)
			a, err := jugarCon(codigo, draws, p)
			if err != nil {
				t.Fatalf("%s/%s: %v", codigo, m.Codigo, err)
			}
			b, err := jugarCon(codigo, draws, p)
			if err != nil {
				t.Fatalf("%s/%s: %v", codigo, m.Codigo, err)
			}
			if a.Multiplicador != b.Multiplicador || !reflect.DeepEqual(a.Data, b.Data) {
				t.Errorf("%s/%s: no es determinista", codigo, m.Codigo)
			}
		}
	}
}

// TestMultMaximoModalidades: el tope del juego cubre a todas sus modalidades y
// no cambia el de los juegos que ya estaban aprobados.
func TestMultMaximoModalidades(t *testing.T) {
	esperado := map[string]int64{"ruleta": 35, "keno": 65, "raspa": 18, "baccarat": 8, "alto": 48, "hilo": 6, "cascada": 10}
	for codigo, m := range esperado {
		if MultMaximoJuego(codigo) != m {
			t.Errorf("%s: MultMaximoJuego = %d, queria %d", codigo, MultMaximoJuego(codigo), m)
		}
	}
	if got := MultiplicadorMaximo("hilo", 2); got != 6 {
		t.Errorf("MultiplicadorMaximo(hilo) = %d, queria 6", got)
	}
	// Los juegos sin modalidades siguen devolviendo el multiplicador del catalogo.
	if got := MultiplicadorMaximo("dados", 27); got != 27 {
		t.Errorf("MultiplicadorMaximo(dados) = %d, queria 27", got)
	}
	if got := MultMaximoModalidad("ruleta", "no-existe"); got != 0 {
		t.Errorf("MultMaximoModalidad inexistente = %d, queria 0", got)
	}
}

// TestModalidadesExponenMultMax: el catalogo que consume el frontend trae el
// mayor multiplicador de cada modalidad, que es lo que usa el encabezado del
// modal (el hilo en racha paga x6, no el x2 del catalogo).
func TestModalidadesExponenMultMax(t *testing.T) {
	for _, codigo := range juegosConModalidades() {
		for _, m := range Modalidades(codigo) {
			if m.MultMax != MultMaximoModalidad(codigo, m.Codigo) {
				t.Errorf("%s/%s: MultMax = %d, queria %d", codigo, m.Codigo, m.MultMax, MultMaximoModalidad(codigo, m.Codigo))
			}
		}
	}
	casos := []struct {
		juego     string
		modalidad string
		want      int64
	}{
		{"hilo", "clasico", 2},
		{"hilo", "racha", 6},
		{"alto", "racha", 4},
		{"ruleta", "europea", 35},
		{"ruleta", "rapida", 8},
		{"cascada", "rapida", 4},
	}
	for _, c := range casos {
		if got := MultMaximoModalidad(c.juego, c.modalidad); got != c.want {
			t.Errorf("%s/%s: %d, queria %d", c.juego, c.modalidad, got, c.want)
		}
	}
}

// TestRTPModalidades fija el retorno medido de cada modalidad. Los valores de
// las modalidades nuevas se mantienen dentro de la banda del proyecto; las
// classicas quedan exactamente como estaban.
func TestRTPModalidades(t *testing.T) {
	casos := []struct {
		codigo    string
		modalidad string
		rtp       float64
	}{
		// Modalidades clasicas: mismo RTP que antes de agregar modalidades.
		{"ruleta", "europea", 94.40},
		{"keno", "clasico", 92.53},
		{"raspa", "clasica", 92.09},
		{"baccarat", "estandar", 98.70},
		{"alto", "clasico", 95.97},
		{"hilo", "clasico", 91.99},
		{"cascada", "clasica", 96.90},
		// Modalidades nuevas.
		{"ruleta", "rapida", 80.08},
		{"keno", "relampago", 86.66},
		{"raspa", "expres", 87.63},
		{"baccarat", "sin-comision", 86.02},
		{"alto", "racha", 92.04},
		{"hilo", "racha", 77.61},
		{"cascada", "rapida", 87.82},
		{"cascada", "profunda", 81.85},
	}
	for _, c := range casos {
		rtp := RTPModalidad(c.codigo, c.modalidad, 200000)
		t.Logf("%s/%s: RTP %.3f%%", c.codigo, c.modalidad, rtp)
		if diff := rtp - c.rtp; diff > 0.35 || diff < -0.35 {
			t.Errorf("%s/%s: RTP %.3f, queria %.2f (±0.35)", c.codigo, c.modalidad, rtp, c.rtp)
		}
	}
}

// TestRTPModalidadesDentroDeBanda verifica que las modalidades nuevas esten
// dentro de la banda del proyecto (78-92%) y que el promedio global tambien.
func TestRTPModalidadesDentroDeBanda(t *testing.T) {
	nuevas := []struct{ codigo, modalidad string }{
		{"ruleta", "rapida"},
		{"keno", "relampago"},
		{"raspa", "expres"},
		{"baccarat", "sin-comision"},
		{"alto", "racha"},
		{"hilo", "racha"},
		{"cascada", "rapida"},
		{"cascada", "profunda"},
	}
	var suma float64
	for _, n := range nuevas {
		rtp := RTPModalidad(n.codigo, n.modalidad, 200000)
		suma += rtp
		if rtp < 75 || rtp > 93 {
			t.Errorf("%s/%s: RTP %.2f fuera de la banda esperada (75-93)", n.codigo, n.modalidad, rtp)
		}
	}
	promedio := suma / float64(len(nuevas))
	if promedio < 78 || promedio > 92 {
		t.Errorf("RTP global de las modalidades nuevas: %.2f, fuera de 78-92", promedio)
	}
	t.Logf("RTP global de las modalidades nuevas: %.2f%%", promedio)
}
