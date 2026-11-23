<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'
import { crearSim, SPEC, type Evento, type SimPinball } from './pinballFisica'

// Pinball real: el jugador controla los flippers y el plunger con un motor
// determinista de paso fijo (1 frame = 1/60 s). La traza de inputs se manda al
// servidor, que la replica y paga segun la tabla de tramos de la casa.
//
// La tabla está duplicada acá y en backend/internal/casino/pinball.go: si se
// cambia una, cambia la otra (TestPinballRTP verifica el RTP de la del server).
// Espejo de casino.pinballTramos; el componente prefiere los tramos que manda
// el servidor en la respuesta y usa estos solo si llegan vacíos.
const TRAMOS_RESPALDO: Array<[number, number]> = [
  [0, 0],
  [900, 1],
  [1500, 2],
  [2400, 3],
  [3600, 5],
  [5000, 7],
]

type TramoPinball = { desde?: number; mult?: number; Desde?: number; Mult?: number }

function tramosDeServidor(raw: unknown): Array<[number, number]> | null {
  if (!Array.isArray(raw) || raw.length === 0) return null
  const out: Array<[number, number]> = []
  for (const t of raw as TramoPinball[]) {
    const desde = Number(t.desde ?? t.Desde ?? 0)
    const mult = Number(t.mult ?? t.Mult ?? 0)
    if (!Number.isFinite(desde) || !Number.isFinite(mult)) continue
    out.push([desde, Math.round(mult)])
  }
  return out.length > 0 ? out : null
}

const tramos = computed<Array<[number, number]>>(() =>
  tramosDeServidor(props.resultado?.tramos) ?? TRAMOS_RESPALDO,
)

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
  codigo?: string
  multiplicador?: number
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

const PALETAS: Record<string, {
  fondo1: string; fondo2: string; marco: string; veta: string; acento: string
  bumped: string[]; metal: string; metal2: string
}> = {
  'pinball-neon': {
    fondo1: '#1b0f38', fondo2: '#070313', marco: '#6d3fd1', veta: '#3a1f7a', acento: '#22d3ee',
    bumped: ['#22d3ee', '#e879f9', '#34d399'], metal: '#c7d2fe', metal2: '#4c1d95',
  },
  'pinball-clasico': {
    fondo1: '#2a1a08', fondo2: '#0d0702', marco: '#a8742c', veta: '#5e3a14', acento: '#e8ce7a',
    bumped: ['#d9a441', '#c0562f', '#3f8f5f'], metal: '#f2d9a4', metal2: '#7c4a16',
  },
  'plinko-premium': {
    fondo1: '#062b33', fondo2: '#021418', marco: '#0f766e', veta: '#0a4a52', acento: '#5eead4',
    bumped: ['#2dd4bf', '#f5d06a', '#7dd3fc'], metal: '#c9f5ec', metal2: '#0b5a55',
  },
}
const paleta = computed(() => PALETAS[props.codigo ?? ''] ?? PALETAS['pinball-neon'])
const uid = computed(() => props.codigo ?? 'x')

// ---- estado de la simulacion (refs porque se actualizan 60 veces por segundo)
let sim: SimPinball = crearSim()
const fase = ref<'lista' | 'jugando' | 'cobrando'>('lista')
const bola = ref<{ x: number; y: number; r: number } | null>(null)
const angL = ref<number>(SPEC.FLIP_L.reposo)
const angR = ref<number>(SPEC.FLIP_R.reposo)
const actL = ref(false)
const actR = ref(false)
const puntos = ref(0)
const enJuego = ref(0)
const bolasRestantes = ref<number>(SPEC.BOLAS)
const carga = ref(0)
const cargandoPlunger = ref(false)

let frame = 0
let cola: Array<{ k: 'L' | 'R' | 'P'; v: number }> = []
const traza: Evento[] = []
let flipLState = false
let flipRState = false
let temporizador = 0
let ultimo = 0
let acumulado = 0
let cobroPendiente = 0

const PASO = 1 / 60

const multEstimado = computed(() => {
  let m = 0
  for (const [desde, mult] of tramos.value) if (puntos.value >= desde) m = mult
  return m
})

// Texto de la regla derivado de la tabla vigente: si cambia la tabla no hace
// falta tocar el template.
const textoRegla = computed(() => {
  const pagan = tramos.value.filter(([, m]) => m > 0)
  const primero = pagan[0]
  const ultimo = pagan[pagan.length - 1]
  if (!primero || !ultimo) return ''
  return `Cada partida son 3 bolas; los puntos de bumpers y slingshots definen el tramo: `
    + `desde ${primero[0]} puntos pagás x${primero[1]}, y el tramo más alto paga x${ultimo[1]} `
    + `a partir de ${ultimo[0]} puntos.`
})

const paredes = computed(() =>
  SPEC.PAREDES.reduce<Array<{ x1: number; y1: number; x2: number; y2: number }>>((acc, p, i) => {
    if (i % 2 === 0 && SPEC.PAREDES[i + 1]) {
      acc.push({ x1: p[0], y1: p[1], x2: SPEC.PAREDES[i + 1][0], y2: SPEC.PAREDES[i + 1][1] })
    }
    return acc
  }, []),
)

const paletaL = computed(() => ({
  x1: SPEC.FLIP_L.px,
  y1: SPEC.FLIP_L.py,
  x2: SPEC.FLIP_L.px + Math.cos(angL.value) * SPEC.FLIP_L.len,
  y2: SPEC.FLIP_L.py + Math.sin(angL.value) * SPEC.FLIP_L.len,
}))
const paletaR = computed(() => ({
  x1: SPEC.FLIP_R.px,
  y1: SPEC.FLIP_R.py,
  x2: SPEC.FLIP_R.px + Math.cos(angR.value) * SPEC.FLIP_R.len,
  y2: SPEC.FLIP_R.py + Math.sin(angR.value) * SPEC.FLIP_R.len,
}))

// halo del bumper al que se está acercando la bola
const halos = computed(() => {
  const out: number[] = [0, 0, 0]
  const b = bola.value
  if (!b) return out
  SPEC.BUMPERS.forEach((bp, i) => {
    const d = Math.hypot(b.x - bp.x, b.y - bp.y)
    out[i] = Math.max(0, Math.min(1, 1 - (d - bp.r) / 26))
  })
  return out
})

function sincronizar() {
  const b = sim.bola()
  bola.value = b ? { x: b.x, y: b.y, r: b.r } : null
  const f = sim.flippers()
  angL.value = f.angL
  angR.value = f.angR
  actL.value = f.activoL
  actR.value = f.activoR
  puntos.value = sim.puntaje()
  enJuego.value = sim.puntosEnJuego()
  bolasRestantes.value = Math.max(0, SPEC.BOLAS - sim.bolas())
}

function tick() {
  const now = performance.now()
  if (!ultimo) ultimo = now
  const dt = Math.min(0.25, (now - ultimo) / 1000)
  ultimo = now
  acumulado += dt

  while (acumulado >= PASO) {
    acumulado -= PASO
    const evs: Evento[] = cola.map((e) => ({ f: frame, k: e.k, v: e.v }))
    if (evs.length) traza.push(...evs)
    cola = []
    sim.paso(frame, evs)
    frame++
  }

  if (cargandoPlunger.value) {
    carga.value = Math.min(1, carga.value + dt / 1.4)
  }

  sincronizar()

  if (fase.value === 'jugando' && sim.bolas() >= SPEC.BOLAS) {
    fase.value = 'cobrando'
    sincronizar()
    cobroPendiente = window.setTimeout(cobrar, 900)
    temporizador = 0
    return // no re-armar el bucle: la jugada termino
  }
  temporizador = window.setTimeout(tick, 1000 * PASO)
}

function cobrar() {
  if (traza.length === 0) return
  emit('jugar', { eventos: traza.slice() })
}

function pulsar(k: 'L' | 'R') {
  if (fase.value !== 'jugando') return
  if (k === 'L') {
    if (flipLState) return
    flipLState = true
  } else {
    if (flipRState) return
    flipRState = true
  }
  cola.push({ k, v: 1 })
  reproducir('tecla')
}

function soltar(k: 'L' | 'R') {
  if (k === 'L') {
    if (!flipLState) return
    flipLState = false
  } else {
    if (!flipRState) return
    flipRState = false
  }
  cola.push({ k, v: 0 })
}

function plungerDown() {
  if (fase.value !== 'jugando' || sim.bola()) return
  cargandoPlunger.value = true
  carga.value = 0
}

function plungerUp() {
  if (!cargandoPlunger.value) return
  cargandoPlunger.value = false
  if (fase.value !== 'jugando' || sim.bola()) return
  if (carga.value < 0.08) carga.value = 0.08
  // Si ya quedó un lanzamiento sin aplicar (pulsacion rapida antes de que el
  // bucle avance), no lo acumules: el motor solo crea una bola.
  if (cola.some((c) => c.k === 'P')) return
  cola.push({ k: 'P', v: Math.round(carga.value * 100) / 100 })
  reproducir('deflector')
  carga.value = 0
}

function arrancar() {
  if (fase.value === 'cobrando') return
  fase.value = 'jugando'
  if (!temporizador) {
    ultimo = 0
    acumulado = 0
    temporizador = window.setTimeout(tick, 1000 * PASO)
  }
}

function reiniciar() {
  sim = crearSim()
  frame = 0
  cola = []
  traza.length = 0
  flipLState = false
  flipRState = false
  carga.value = 0
  cargandoPlunger.value = false
  window.clearTimeout(temporizador)
  temporizador = 0
  fase.value = 'lista'
  sincronizar()
}

// teclado: flechas o Z/M para las paletas, espacio para el plunger
function onKey(e: KeyboardEvent, down: boolean) {
  const t = e.target as HTMLElement | null
  if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA')) return
  if (e.code === 'ArrowLeft' || e.key === 'z' || e.key === 'Z') {
    if (down) { if (fase.value === 'lista') arrancar(); pulsar('L') } else soltar('L')
    e.preventDefault()
  } else if (e.code === 'ArrowRight' || e.key === 'm' || e.key === 'M') {
    if (down) { if (fase.value === 'lista') arrancar(); pulsar('R') } else soltar('R')
    e.preventDefault()
  } else if (e.code === 'Space') {
    if (down) { if (fase.value === 'lista') arrancar(); plungerDown() } else plungerUp()
    e.preventDefault()
  }
}

watch(
  () => props.resultado,
  (r) => {
    if (!r) reiniciar()
  },
)

let tecladoBajo: (e: KeyboardEvent) => void = () => {}
let tecladoAlto: (e: KeyboardEvent) => void = () => {}

onMounted(() => {
  sincronizar()
  tecladoBajo = (e) => onKey(e, true)
  tecladoAlto = (e) => onKey(e, false)
  window.addEventListener('keydown', tecladoBajo)
  window.addEventListener('keyup', tecladoAlto)
})

onBeforeUnmount(() => {
  window.clearTimeout(temporizador)
  window.clearTimeout(cobroPendiente)
  window.removeEventListener('keydown', tecladoBajo)
  window.removeEventListener('keyup', tecladoAlto)
})
</script>

<template>
  <div class="pinball">
    <div class="tablero">
      <svg class="campo" viewBox="0 0 320 416" preserveAspectRatio="xMidYMid meet" aria-hidden="true">
        <defs>
          <linearGradient :id="`fondo-${uid}`" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" :stop-color="paleta.fondo1" />
            <stop offset="1" :stop-color="paleta.fondo2" />
          </linearGradient>
          <radialGradient :id="`bumper-${uid}`" cx="35%" cy="30%" r="75%">
            <stop offset="0" :stop-color="paleta.metal" />
            <stop offset="0.45" :stop-color="paleta.acento" />
            <stop offset="1" :stop-color="paleta.metal2" />
          </radialGradient>
          <linearGradient :id="`brazo-${uid}`" x1="0" y1="0" x2="1" y2="0">
            <stop offset="0" :stop-color="paleta.metal" />
            <stop offset="0.5" :stop-color="paleta.marco" />
            <stop offset="1" :stop-color="paleta.metal2" />
          </linearGradient>
          <radialGradient :id="`bola-${uid}`" cx="35%" cy="30%" r="80%">
            <stop offset="0" stop-color="#ffffff" />
            <stop offset="0.35" stop-color="#f7ecc9" />
            <stop offset="0.75" stop-color="#c9922a" />
            <stop offset="1" stop-color="#6d4a08" />
          </radialGradient>
        </defs>

        <rect x="0" y="0" width="320" height="416" :fill="`url(#fondo-${uid})`" />

        <!-- vetas del playfield -->
        <g :stroke="paleta.veta" stroke-width="1" opacity=".55" fill="none">
          <path d="M20 40 Q20 250 58 268 L40 268 Q8 240 8 40 Z" />
          <path d="M300 40 Q300 250 262 268 L280 268 Q312 240 312 40 Z" />
        </g>

        <!-- paredes reales del motor -->
        <g stroke-linecap="round">
          <line v-for="(p, i) in paredes" :key="i"
                :x1="p.x1" :y1="p.y1" :x2="p.x2" :y2="p.y2"
                :stroke="paleta.marco" stroke-width="6" opacity=".9" />
          <line v-for="(p, i) in paredes" :key="`b${i}`"
                :x1="p.x1" :y1="p.y1" :x2="p.x2" :y2="p.y2"
                stroke="rgba(255,255,255,.28)" stroke-width="1.6" />
        </g>

        <!-- slingshots -->
        <g :fill="paleta.acento" opacity=".8">
          <polygon :points="`${SPEC.SLINGS[0].x1},${SPEC.SLINGS[0].y1} ${SPEC.SLINGS[0].x2},${SPEC.SLINGS[0].y2} ${SPEC.SLINGS[0].x3},${SPEC.SLINGS[0].y3}`" />
          <polygon :points="`${SPEC.SLINGS[1].x1},${SPEC.SLINGS[1].y1} ${SPEC.SLINGS[1].x2},${SPEC.SLINGS[1].y2} ${SPEC.SLINGS[1].x3},${SPEC.SLINGS[1].y3}`" />
        </g>

        <!-- bumpers -->
        <g v-for="(b, i) in SPEC.BUMPERS" :key="i">
          <circle :cx="b.x" :cy="b.y" :r="b.r * 1.6" :fill="paleta.bumped[i]"
                  :opacity="0.06 + halos[i] * 0.3" />
          <circle :cx="b.x" :cy="b.y" :r="b.r" :fill="`url(#bumper-${uid})`"
                  :stroke="paleta.metal" stroke-width="2.5" />
          <circle :cx="b.x" :cy="b.y" :r="b.r * .6" fill="rgba(255,255,255,.16)"
                  stroke="rgba(255,255,255,.45)" stroke-width="1" />
          <text :x="b.x" :y="b.y + 1" text-anchor="middle" dominant-baseline="middle"
                font-family="ui-sans-serif, system-ui" font-size="14" font-weight="800"
                :fill="paleta.fondo2">{{ b.pts }}</text>
        </g>

        <!-- paletas -->
        <line :x1="paletaL.x1" :y1="paletaL.y1" :x2="paletaL.x2" :y2="paletaL.y2"
              :stroke="actL ? paleta.metal : `url(#brazo-${uid})`"
              :stroke-width="actL ? 13 : 12" stroke-linecap="round" />
        <line :x1="paletaR.x1" :y1="paletaR.y1" :x2="paletaR.x2" :y2="paletaR.y2"
              :stroke="actR ? paleta.metal : `url(#brazo-${uid})`"
              :stroke-width="actR ? 13 : 12" stroke-linecap="round" />
        <circle :cx="SPEC.FLIP_L.px" :cy="SPEC.FLIP_L.py" r="5" :fill="paleta.metal" />
        <circle :cx="SPEC.FLIP_R.px" :cy="SPEC.FLIP_R.py" r="5" :fill="paleta.metal" />

        <!-- bola -->
        <circle v-if="bola" :cx="bola.x" :cy="bola.y" :r="bola.r"
                :fill="`url(#bola-${uid})`" stroke="rgba(0,0,0,.35)" stroke-width="1" />

        <!-- resorte del plunger -->
        <g v-if="!bola && fase === 'jugando'">
          <rect :x="298" :y="374 + carga * 26" width="10" :height="18 - carga * 26"
                rx="3" :fill="paleta.acento" opacity=".85" />
        </g>
      </svg>

      <div class="hud">
        <span class="bola">{{ bolasRestantes }}</span>
        <span class="etq">bolas</span>
        <span class="pts">{{ puntos + enJuego }}</span>
        <span class="etq">pts</span>
        <span class="mult">x{{ multEstimado }}</span>
      </div>
    </div>

    <div v-if="resultado && !cargando" class="resuelto" :class="{ gana: premio > 0 }">
      <span class="n">x{{ multiplicador ?? resultado.multiplicador ?? 0 }}</span>
      <span v-if="premio > 0" class="p">{{ colones(premio) }}</span>
      <span v-else class="p cero">Sin premio</span>
    </div>

    <div class="controles">
      <button class="paleta-btn izq" :class="{ activa: actL }"
              @pointerdown.prevent="fase === 'lista' && arrancar(); pulsar('L')"
              @pointerup.prevent="soltar('L')" @pointerleave.prevent="soltar('L')">
        ◀ Paleta
      </button>
      <button class="paleta-btn der" :class="{ activa: actR }"
              @pointerdown.prevent="fase === 'lista' && arrancar(); pulsar('R')"
              @pointerup.prevent="soltar('R')" @pointerleave.prevent="soltar('R')">
        Paleta ▶
      </button>
    </div>

    <button v-if="fase !== 'cobrando'" class="btn btn-primary btn-block"
            :disabled="cargando || (fase === 'jugando' && !!bola)"
            @pointerdown.prevent="arrancar(); plungerDown()"
            @pointerup.prevent="plungerUp()" @pointerleave.prevent="plungerUp()">
      {{ cargando ? 'Cobrando…' : fase === 'lista' ? 'Mantené para lanzar' : 'Mantené para lanzar la bola' }}
    </button>
    <p v-else class="cobrando">Cobrando la jugada…</p>

    <p class="regla">
      Mantené apretadas las paletas (flechas o Z/M) y soltá el impulsor con espacio.
      {{ textoRegla }}
    </p>
  </div>
</template>

<style scoped>
.pinball { display: flex; flex-direction: column; gap: 12px; }
.tablero {
  position: relative; width: min(340px, 78vw); margin: 0 auto;
  border-radius: 16px; overflow: hidden;
  border: 1px solid var(--border-soft);
  box-shadow: 0 18px 40px rgba(0, 0, 0, .45);
  touch-action: none;
}
.campo { display: block; width: 100%; height: auto; aspect-ratio: 320 / 416; }
.hud {
  display: flex; align-items: baseline; gap: 6px;
  padding: 5px 8px 7px;
  font-family: ui-monospace, monospace;
}
.hud .bola { font-size: 18px; font-weight: 800; color: var(--gold); }
.hud .pts { font-size: 16px; font-weight: 700; color: #fff; margin-left: auto; }
.hud .mult { font-size: 15px; font-weight: 800; color: var(--prim-2, #7dd3fc); }
.hud .etq { font-size: 10px; letter-spacing: .1em; text-transform: uppercase; color: var(--faint); }
.resuelto {
  display: flex; align-items: center; justify-content: center; gap: 10px;
  min-height: 40px; padding: 7px 12px; border-radius: 10px;
  background: rgba(4, 6, 10, .92); border: 1px solid var(--border-soft);
}
.resuelto .n { font-family: var(--display); font-size: 22px; font-weight: 700; line-height: 1; }
.resuelto .p { color: var(--gold); font-weight: 700; font-size: 14px; }
.resuelto .p.cero { color: var(--muted); font-weight: 500; font-size: 13px; }
.resuelto.gana { border-color: rgba(245, 185, 66, .55); }
.controles { display: flex; gap: 8px; }
.paleta-btn {
  flex: 1; padding: 14px 8px; border-radius: 12px; cursor: pointer;
  border: 1px solid var(--border-soft); background: var(--surface);
  color: var(--muted); font-weight: 700; font-size: 13px;
  touch-action: none; user-select: none;
}
.paleta-btn.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
.cobrando { margin: 0; text-align: center; font-size: 12px; color: var(--faint); }
.regla { margin: 0; font-size: 11.5px; color: var(--faint); text-align: center; line-height: 1.35; }
</style>
