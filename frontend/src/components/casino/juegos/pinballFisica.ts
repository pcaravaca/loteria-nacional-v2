// Motor de pinball determinista — versión TypeScript.
//
// Existe DOS veces, literalmente: acá y en backend/internal/casino/pinballfisica.go.
// El servidor replica la partida con la misma semilla y la traza de inputs del
// cliente y tiene que llegar AL MISMO puntaje. Por eso este archivo no se puede
// tocar sin tocar el gemelo: el orden de las operaciones, las constantes y el
// redondeo son parte del contrato (ver TestMotorGemelo en el backend).
//
// Reglas del contrato:
//   - Paso fijo de simulación: 1 frame = 1 paso. No hay dt real: el cliente
//     acumula tiempo de reloj y avanza la simulación en pasos enteros (con
//     interpolación al dibujar), así el resultado no depende de los fps.
//   - Todo en float64 y enteros; el puntaje es entero, que es lo que se compara.
//   - Los inputs son eventos: cambios de estado de los flippers y el plunger.
//     Sin posiciones absolutas: solo flancos, para que la traza sea chica.
//   - La semilla no interviene en la física: la partida es habilidad pura y lo
//     que se verifica es que la traza presentada reproduce el puntaje declarado.

export type Evento = { f: number; k: 'L' | 'R' | 'P'; v: number }

// ---- geometría (viewBox 320x416: la misma que usa el SVG del playfield) -----
export const SPEC = {
  W: 320,
  H: 416,
  GRAV: 0.18,
  REST_PARED: 0.42,
  BOLA_R: 7,
  TECHO: 24,
  SUELO: 392,
  DER_EXTERNA: 310,
  BUMPERS: [
    { x: 92, y: 78, r: 24, pts: 100 },
    { x: 228, y: 78, r: 24, pts: 150 },
    { x: 160, y: 134, r: 24, pts: 250 },
  ],
  SLINGS: [
    { x1: 26, y1: 340, x2: 62, y2: 366, x3: 24, y3: 372, pts: 50 },
    { x1: 294, y1: 340, x2: 258, y2: 366, x3: 296, y3: 372, pts: 50 },
  ],
  FLIP_L: { px: 92, py: 392, len: 62, reposo: 0.42, activo: -0.5 },
  FLIP_R: { px: 228, py: 392, len: 62, reposo: Math.PI - 0.42, activo: Math.PI + 0.5 },
  LANZAR_V: 16,
  // golpe del flipper: mientras el brazo esté activo, todo contacto lanza la bola
  GOLPE_VX: 4,
  GOLPE_VY: 10,
  BUMP_KICK: 1.15,
  GIRO_ACTIVO: 0.5,
  GIRO_REPOSO: 0.16,
  // paredes en pares; el carril de salida no tiene tabique: el techo en diagonal
  // ([292,24] -> [310,44]) desvía la bola hacia el campo, como el arco real.
  PAREDES: [
    [10, 380], [10, 40],
    [10, 40], [28, 24],
    [28, 24], [292, 24],
    [292, 24], [310, 44],
    [310, 44], [310, 416],
    [10, 380], [92, 392],
    [292, 380], [228, 392],
  ] as number[][],
  BOLAS: 3,
  MAX_FRAMES: 36000,
} as const

export type EstadoBola = { x: number; y: number; vx: number; vy: number; r: number }

export type ResultadoSim = {
  puntaje: number
  porBola: number[]
  frames: number
  lanzamientos: number
}

export type SimPinball = {
  paso: (f: number, evs: Evento[]) => void
  /** avanza la simulación hasta el frame indicado (por si el cliente va a menos fps) */
  bola: () => EstadoBola | null
  puntaje: () => number
  puntosEnJuego: () => number
  porBola: () => number[]
  bolas: () => number
  frames: () => number
  flippers: () => { angL: number; angR: number; activoL: boolean; activoR: boolean }
  cerrar: () => void
}

const hypoth = (dx: number, dy: number) => Math.sqrt(dx * dx + dy * dy)

function colCirc(b: EstadoBola, cx: number, cy: number, cr: number, kick = 1): boolean {
  const dx = b.x - cx
  const dy = b.y - cy
  const d = hypoth(dx, dy)
  const rr = b.r + cr
  if (d >= rr || d === 0) return false
  const nx = dx / d
  const ny = dy / d
  b.x = cx + nx * rr
  b.y = cy + ny * rr
  const vn = b.vx * nx + b.vy * ny
  if (vn < 0) {
    b.vx -= 2 * vn * nx
    b.vy -= 2 * vn * ny
    b.vx *= kick
    b.vy *= kick
  }
  return true
}

function colSeg(
  b: EstadoBola, x1: number, y1: number, x2: number, y2: number, rest: number,
): boolean {
  const dx = x2 - x1
  const dy = y2 - y1
  const l2 = dx * dx + dy * dy
  if (l2 === 0) return false
  let t = ((b.x - x1) * dx + (b.y - y1) * dy) / l2
  if (t < 0) t = 0
  else if (t > 1) t = 1
  const px = x1 + t * dx
  const py = y1 + t * dy
  const ox = b.x - px
  const oy = b.y - py
  const d = hypoth(ox, oy)
  if (d >= b.r || d === 0) return false
  const nx = ox / d
  const ny = oy / d
  b.x = px + nx * b.r
  b.y = py + ny * b.r
  const vn = b.vx * nx + b.vy * ny
  if (vn < 0) {
    b.vx -= (1 + rest) * vn * nx
    b.vy -= (1 + rest) * vn * ny
  }
  return true
}

// crearSim expone el simulador frame a frame: lo usa el cliente para ir decidiendo
// los flancos según lo que ve, y la calibración para jugar con políticas.
export function crearSim(): SimPinball {
  const S = SPEC
  let bola: EstadoBola | null = null
  let flipL = false
  let flipR = false
  let angL = S.FLIP_L.reposo
  let angR = S.FLIP_R.reposo
  let carga = 0
  let bolaN = 0
  let f = 0
  let puntosBola = 0
  let puntajeTotal = 0
  let cerrado = false
  const porBola: number[] = []

  const punta = (px: number, py: number, ang: number, len: number) => ({
    x: px + Math.cos(ang) * len,
    y: py + Math.sin(ang) * len,
  })

  return {
    paso(frame: number, evs: Evento[]) {
      if (cerrado) return
      f = frame
      for (const ev of evs) {
        if (ev.f !== frame) continue
        if (ev.k === 'L') {
          if (ev.v !== (flipL ? 1 : 0)) flipL = ev.v === 1
        } else if (ev.k === 'R') {
          if (ev.v !== (flipR ? 1 : 0)) flipR = ev.v === 1
        } else if (ev.k === 'P') {
          carga = Math.max(0, Math.min(1, ev.v))
          if (!bola) {
            bola = {
              x: S.DER_EXTERNA - 9,
              y: S.SUELO - 18,
              vx: 0,
              vy: -carga * S.LANZAR_V,
              r: S.BOLA_R,
            }
            carga = 0
            puntosBola = 0
          }
        }
      }

      const objL = flipL ? S.FLIP_L.activo : S.FLIP_L.reposo
      const prevL = angL
      angL += (objL - prevL) * (flipL ? S.GIRO_ACTIVO : S.GIRO_REPOSO)
      const objR = flipR ? S.FLIP_R.activo : S.FLIP_R.reposo
      const prevR = angR
      angR += (objR - prevR) * (flipR ? S.GIRO_ACTIVO : S.GIRO_REPOSO)

      if (!bola) return

      bola.vy += S.GRAV
      bola.x += bola.vx
      bola.y += bola.vy

      for (let p = 0; p < S.PAREDES.length; p += 2) {
        const a = S.PAREDES[p]
        const b = S.PAREDES[p + 1]
        colSeg(bola, a[0], a[1], b[0], b[1], S.REST_PARED)
      }
      for (const bp of S.BUMPERS) {
        if (colCirc(bola, bp.x, bp.y, bp.r, S.BUMP_KICK)) puntosBola += bp.pts
      }
      for (const sl of S.SLINGS) {
        if (colSeg(bola, sl.x1, sl.y1, sl.x2, sl.y2, 0.7)) puntosBola += sl.pts
      }

      const tl = punta(S.FLIP_L.px, S.FLIP_L.py, angL, S.FLIP_L.len)
      if (colSeg(bola, S.FLIP_L.px, S.FLIP_L.py, tl.x, tl.y, 0.5) && flipL) {
        bola.vx -= S.GOLPE_VX
        bola.vy -= S.GOLPE_VY
      }
      const tr = punta(S.FLIP_R.px, S.FLIP_R.py, angR, S.FLIP_R.len)
      if (colSeg(bola, S.FLIP_R.px, S.FLIP_R.py, tr.x, tr.y, 0.5) && flipR) {
        bola.vx += S.GOLPE_VX
        bola.vy -= S.GOLPE_VY
      }

      if (bola.y - bola.r > 416) {
        puntajeTotal += puntosBola
        porBola.push(puntosBola)
        bola = null
        bolaN++
      }
    },
    bola: () => bola,
    puntaje: () => puntajeTotal,
    puntosEnJuego: () => puntosBola,
    porBola: () => porBola.slice(),
    bolas: () => bolaN,
    frames: () => f,
    flippers: () => ({ angL, angR, activoL: flipL, activoR: flipR }),
    cerrar() {
      if (cerrado) return
      cerrado = true
      if (bola) {
        puntajeTotal += puntosBola
        porBola.push(puntosBola)
        bola = null
        bolaN++
      }
    },
  }
}

// Traza que manda el cliente: solo flancos de los flippers y lanzamientos.
export type TrazaPinball = {
  eventos: Evento[]
  puntaje: number
  porBola: number[]
}

export function simularPinball(eventos: Evento[]): ResultadoSim {
  const S = SPEC
  const sim = crearSim()
  const porFrame = new Map<number, Evento[]>()
  // agrupar por frame conservando el orden de llegada = sort estable por f
  for (const ev of eventos.slice().sort((a, b) => a.f - b.f)) {
    const lista = porFrame.get(ev.f)
    if (lista) lista.push(ev)
    else porFrame.set(ev.f, [ev])
  }
  for (let f = 0; f < S.MAX_FRAMES && sim.bolas() < S.BOLAS; f++) {
    sim.paso(f, porFrame.get(f) ?? [])
  }
  sim.cerrar()
  return {
    puntaje: sim.puntaje(),
    porBola: sim.porBola(),
    frames: sim.frames(),
    lanzamientos: sim.porBola().length,
  }
}
