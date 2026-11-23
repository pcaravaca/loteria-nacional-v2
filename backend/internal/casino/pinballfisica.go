package casino

import (
	"math"
	"sort"
)

// Motor de pinball determinista (réplica exacta de pinballFisica.ts del frontend).
//
// El servidor reproduce la partida con la traza de inputs del cliente y debe
// llegar AL MISMO puntaje. Cualquier cambio acá exige el mismo cambio allá y
// viceversa: las constantes, el orden de las operaciones y el redondeo son parte
// del contrato. La prueba cruzada es TestMotorGemelo (golden files generados
// por el motor TypeScript).

const (
	gravedad     = 0.18
	restPared    = 0.42
	bolaR        = 7.0
	suelo        = 392.0
	derExterna   = 310.0
	lanzarV      = 16.0
	bumpKick     = 1.15
	golpeVX      = 4.0
	golpeVY      = 10.0
	giroActivo   = 0.5
	giroReposo   = 0.16
	flipperLargo = 62.0
	flipperPivL  = 92.0
	flipperPivR  = 228.0
	maxFrames    = 36000
	bumpPtsCorto = 100
	bumpPtsMedio = 150
	bumpPtsLargo = 250
	slingsPts    = 50
)

type estadoBola struct {
	x, y, vx, vy, r float64
}

// EventoPinball es un flanco de control: L/R cambian de estado, P es el plunger.
type EventoPinball struct {
	F int     `json:"f"`
	K string  `json:"k"`
	V float64 `json:"v"`
}

// ResultadoPinball es lo que el servidor verifica contra la traza del cliente.
type ResultadoPinball struct {
	Puntaje      int   `json:"puntaje"`
	PorBola      []int `json:"porBola"`
	Frames       int   `json:"frames"`
	Lanzamientos int   `json:"lanzamientos"`
}

type bumper struct {
	x, y, r float64
	pts     int
}

var bumpers = []bumper{
	{x: 92, y: 78, r: 24, pts: bumpPtsCorto},
	{x: 228, y: 78, r: 24, pts: bumpPtsMedio},
	{x: 160, y: 134, r: 24, pts: bumpPtsLargo},
}

// paredes en pares de puntos, igual que SPEC.PAREDES del frontend
var paredes = [][2]float64{
	{10, 380}, {10, 40},
	{10, 40}, {28, 24},
	{28, 24}, {292, 24},
	{292, 24}, {310, 44},
	{310, 44}, {310, 416},
	{10, 380}, {flipperPivL, 392},
	{292, 380}, {flipperPivR, 392},
}

func colCirc(b *estadoBola, cx, cy, cr, kick float64) bool {
	dx := b.x - cx
	dy := b.y - cy
	d := math.Sqrt(dx*dx + dy*dy)
	rr := b.r + cr
	if d >= rr || d == 0 {
		return false
	}
	nx := dx / d
	ny := dy / d
	b.x = cx + nx*rr
	b.y = cy + ny*rr
	vn := b.vx*nx + b.vy*ny
	if vn < 0 {
		b.vx -= 2 * vn * nx
		b.vy -= 2 * vn * ny
		b.vx *= kick
		b.vy *= kick
	}
	return true
}

func colSeg(b *estadoBola, x1, y1, x2, y2, rest float64) bool {
	dx := x2 - x1
	dy := y2 - y1
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return false
	}
	t := ((b.x-x1)*dx + (b.y-y1)*dy) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	px := x1 + t*dx
	py := y1 + t*dy
	ox := b.x - px
	oy := b.y - py
	d := math.Sqrt(ox*ox + oy*oy)
	if d >= b.r || d == 0 {
		return false
	}
	nx := ox / d
	ny := oy / d
	b.x = px + nx*b.r
	b.y = py + ny*b.r
	vn := b.vx*nx + b.vy*ny
	if vn < 0 {
		b.vx -= (1 + rest) * vn * nx
		b.vy -= (1 + rest) * vn * ny
	}
	return true
}

// pinballSim es el simulador frame a_frame: lo usa SimularPinball (traza completa)
// y los bots de calibración de RTP.
type pinballSim struct {
	bola          *estadoBola
	flipL         bool
	flipR         bool
	angL          float64
	angR          float64
	carga         float64
	bolaN         int
	f             int
	puntosBola    int
	puntaje       int
	porBola       []int
	cerrado       bool
	eventosVistos int
}

func nuevoPinballSim() *pinballSim {
	return &pinballSim{angL: 0.42, angR: math.Pi - 0.42}
}

func (s *pinballSim) punta(px, py, ang, largo float64) (float64, float64) {
	return px + math.Cos(ang)*largo, py + math.Sin(ang)*largo
}

// paso avanza un frame aplicando los eventos de ese frame.
func (s *pinballSim) paso(f int, evs []EventoPinball) {
	if s.cerrado {
		return
	}
	s.f = f
	for _, ev := range evs {
		switch ev.K {
		case "L":
			if ev.V != boolFloat(s.flipL) {
				s.flipL = ev.V == 1
			}
		case "R":
			if ev.V != boolFloat(s.flipR) {
				s.flipR = ev.V == 1
			}
		case "P":
			s.carga = math.Max(0, math.Min(1, ev.V))
			if s.bola == nil {
				s.bola = &estadoBola{x: derExterna - 9, y: suelo - 18, vx: 0, vy: -s.carga * lanzarV, r: bolaR}
				s.carga = 0
				s.puntosBola = 0
			}
		}
	}
	s.eventosVistos += len(evs)

	objL := 0.42
	if s.flipL {
		objL = -0.5
	}
	prevL := s.angL
	if s.flipL {
		s.angL += (objL - prevL) * giroActivo
	} else {
		s.angL += (objL - prevL) * giroReposo
	}

	objR := math.Pi - 0.42
	if s.flipR {
		objR = math.Pi + 0.5
	}
	prevR := s.angR
	if s.flipR {
		s.angR += (objR - prevR) * giroActivo
	} else {
		s.angR += (objR - prevR) * giroReposo
	}

	if s.bola == nil {
		return
	}

	s.bola.vy += gravedad
	s.bola.x += s.bola.vx
	s.bola.y += s.bola.vy

	for p := 0; p < len(paredes); p += 2 {
		a := paredes[p]
		b := paredes[p+1]
		colSeg(s.bola, a[0], a[1], b[0], b[1], restPared)
	}

	for _, bp := range bumpers {
		if colCirc(s.bola, bp.x, bp.y, bp.r, bumpKick) {
			s.puntosBola += bp.pts
		}
	}
	slings := [][4]float64{
		{26, 340, 62, 366},
		{294, 340, 258, 366},
	}
	for _, sl := range slings {
		if colSeg(s.bola, sl[0], sl[1], sl[2], sl[3], 0.7) {
			s.puntosBola += slingsPts
		}
	}

	tlx, tly := s.punta(flipperPivL, suelo, s.angL, flipperLargo)
	if colSeg(s.bola, flipperPivL, suelo, tlx, tly, 0.5) && s.flipL {
		s.bola.vx -= golpeVX
		s.bola.vy -= golpeVY
	}
	trx, try := s.punta(flipperPivR, suelo, s.angR, flipperLargo)
	if colSeg(s.bola, flipperPivR, suelo, trx, try, 0.5) && s.flipR {
		s.bola.vx += golpeVX
		s.bola.vy -= golpeVY
	}

	if s.bola.y-s.bola.r > 416 {
		s.puntaje += s.puntosBola
		s.porBola = append(s.porBola, s.puntosBola)
		s.bola = nil
		s.bolaN++
	}
}

// cerrar cuenta la bola que quedaba en juego si la simulación cortó por tiempo.
func (s *pinballSim) cerrar() {
	if s.cerrado {
		return
	}
	s.cerrado = true
	if s.bola != nil {
		s.puntaje += s.puntosBola
		s.porBola = append(s.porBola, s.puntosBola)
		s.bola = nil
		s.bolaN++
	}
}

// SimularPinball reproduce una partida completa a partir de la traza de inputs.
// Debe devolver exactamente lo mismo que simularPinball del frontend.
func SimularPinball(eventos []EventoPinball) ResultadoPinball {
	orden := make([]EventoPinball, len(eventos))
	copy(orden, eventos)
	// estable, como Array.sort del frontend: los empates conservan el orden de llegada
	sort.SliceStable(orden, func(i, j int) bool { return orden[i].F < orden[j].F })

	sim := nuevoPinballSim()
	i := 0
	for f := 0; f < maxFrames && sim.bolaN < 3; f++ {
		var evs []EventoPinball
		for i < len(orden) && orden[i].F == f {
			evs = append(evs, orden[i])
			i++
		}
		sim.paso(f, evs)
	}
	sim.cerrar()

	return ResultadoPinball{
		Puntaje:      sim.puntaje,
		PorBola:      sim.porBola,
		Frames:       sim.f,
		Lanzamientos: len(sim.porBola),
	}
}

func boolFloat(v bool) float64 {
	if v {
		return 1
	}
	return 0
}
