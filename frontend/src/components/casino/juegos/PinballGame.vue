<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'

// El playfield es SVG: asi bumpers, troneras y paletas comparten las mismas
// coordenadas que la bola. Con una imagen de fondo el canvas quedaba desalineado
// y las troneras dibujadas no coincidian con las del arte.
const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
  codigo?: string
  multiplicador?: number
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

const FUERZAS = [
  { id: 0, nombre: 'Suave' },
  { id: 1, nombre: 'Media' },
  { id: 2, nombre: 'Fuerte' },
]
const fuerza = ref(1)

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

const canvas = ref<HTMLCanvasElement | null>(null)
const animando = ref(false)
const fila = ref(-1)
const bolaX = ref(160)
const bolaY = ref(14)
const tronera = ref<number | null>(null)

const tabla = computed<number[]>(() => props.resultado?.tabla ?? [])
const camino = computed<number[]>(() => props.resultado?.camino ?? [])

// Geometria del campo (viewBox 320x416), compartida por el SVG y la bola.
const BUMPERS = [
  { x: 92, y: 78, r: 24, mult: 2 },
  { x: 228, y: 78, r: 24, mult: 3 },
  { x: 160, y: 134, r: 24, mult: 5 },
]
const TRONERA_Y = 330
const TRONERA_H = 30

function aclarar(hex: string, t: number): string {
  const n = parseInt(hex.slice(1), 16)
  const mez = (c: number) => Math.round(c + (255 - c) * t)
  return `rgb(${mez((n >> 16) & 255)},${mez((n >> 8) & 255)},${mez(n & 255)})`
}

function dibujar() {
  const cv = canvas.value
  if (!cv) return
  const ctx = cv.getContext('2d')!
  const cssW = cv.clientWidth || 320
  const dpr = window.devicePixelRatio || 1
  const W = Math.round(cssW * dpr)
  const H = Math.round(W * 416 / 320)
  if (cv.width !== W) {
    cv.width = W
    cv.height = H
  }
  const u = W / 320
  ctx.clearRect(0, 0, W, H)
  const P = paleta.value

  // ---- sombra de la bola (solo cuando ya hay jugada)
  if (props.resultado) {
    const bx = bolaX.value * u
    const by = bolaY.value * u
    const rb = 10 * u
    ctx.beginPath()
    ctx.ellipse(bx + rb * 0.3, by + rb * 0.5, rb, rb * 0.8, 0, 0, Math.PI * 2)
    ctx.fillStyle = 'rgba(0,0,0,.5)'
    ctx.fill()
  }

  // ---- bola de marfil con reflejo
  if (props.resultado) {
    const bx = bolaX.value * u
    const by = bolaY.value * u
    const rb = 10 * u
    const gb = ctx.createRadialGradient(bx - rb * 0.35, by - rb * 0.4, rb * 0.1, bx, by, rb)
    gb.addColorStop(0, '#ffffff')
    gb.addColorStop(0.35, '#f7ecc9')
    gb.addColorStop(0.75, '#c9922a')
    gb.addColorStop(1, '#6d4a08')
    ctx.beginPath()
    ctx.arc(bx, by, rb, 0, Math.PI * 2)
    ctx.fillStyle = gb
    ctx.fill()
    ctx.beginPath()
    ctx.arc(bx - rb * 0.3, by - rb * 0.35, rb * 0.22, 0, Math.PI * 2)
    ctx.fillStyle = 'rgba(255,255,255,.9)'
    ctx.fill()
  }

  // ---- fila actual del camino
  const filas = Math.max(camino.value.length, tabla.value.length ? tabla.value.length - 1 : 0)
  if (fila.value >= 0 && fila.value < filas) {
    const y = (46 + (fila.value + 1) * (250 - 46) / Math.max(1, filas)) * u
    ctx.strokeStyle = 'rgba(255,255,255,.22)'
    ctx.lineWidth = 2 * u
    ctx.beginPath()
    ctx.moveTo(20 * u, y)
    ctx.lineTo(300 * u, y)
    ctx.stroke()
  }

  // ---- halo del bumper por el que pasa la bola
  if (animando.value && camino.value.length && fila.value >= 0) {
    const k = Math.min(fila.value, camino.value.length - 1)
    const b = BUMPERS[k % BUMPERS.length]
    const halo = ctx.createRadialGradient(b.x * u, b.y * u, 2 * u, b.x * u, b.y * u, b.r * 2.6 * u)
    halo.addColorStop(0, aclarar(P.bumped[k % P.bumped.length], 0.5))
    halo.addColorStop(1, 'rgba(255,255,255,0)')
    ctx.beginPath()
    ctx.arc(b.x * u, b.y * u, b.r * 2.6 * u, 0, Math.PI * 2)
    ctx.fillStyle = halo
    ctx.fill()
  }
}

function animar() {
  const total = camino.value.length
  animando.value = true
  tronera.value = null
  fila.value = -1
  bolaX.value = 160
  bolaY.value = 14
  try {
    dibujar()
    reproducir('caida')
  } catch {
    // Si el primer dibujado falla, la jugada igual arranca.
  }
  let k = 0
  // setTimeout recursivo (no setInterval): en una pestaña en segundo plano el
  // intervalo se pausaba y la jugada quedaba "Jugando…" para siempre.
  let pasos = 0
  const tick = () => {
    pasos++
    if (k >= total || pasos > 60) {
      const final = props.resultado?.pos ?? 0
      const n = Math.max(1, tabla.value.length)
      bolaX.value = ((final + 0.5) * 320) / n
      bolaY.value = TRONERA_Y + TRONERA_H / 2
      tronera.value = final
      animando.value = false
      dibujar()
      if (props.premio > 0) reproducir('ganar')
      else reproducir('perder')
      return
    }
    fila.value = k
    bolaX.value = 160 + (camino.value[k] === 1 ? 1 : -1) * 34
    bolaY.value = 14 + ((k + 1) / Math.max(1, total)) * (TRONERA_Y - 20)
    try {
      reproducir('tecla')
      dibujar()
    } catch {
      // Un dibujado que falle no debe trabar la jugada: la bola sigue avanzando.
    }
    k++
    setTimeout(tick, 190)
  }
  setTimeout(tick, 190)
}

watch(
  () => props.resultado,
  async (r) => {
    if (!r) return
    await nextTick()
    animar()
  },
)

watch(() => props.codigo, async () => {
  await nextTick()
  dibujar()
})

onMounted(async () => {
  await nextTick()
  dibujar()
})

function lanzar() {
  emit('jugar', { fuerza: fuerza.value })
}
</script>

<template>
  <div class="pinball">
    <div class="tablero">
      <svg class="campo" viewBox="0 0 320 416" preserveAspectRatio="xMidYMid meet" aria-hidden="true">
        <defs>
          <linearGradient :id="`fondo-${codigo ?? 'x'}`" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" :stop-color="paleta.fondo1" />
            <stop offset="1" :stop-color="paleta.fondo2" />
          </linearGradient>
          <radialGradient :id="`bumper-${codigo ?? 'x'}`" cx="35%" cy="30%" r="75%">
            <stop offset="0" :stop-color="paleta.metal" />
            <stop offset="0.45" :stop-color="paleta.acento" />
            <stop offset="1" :stop-color="paleta.metal2" />
          </radialGradient>
          <linearGradient :id="`paleta-${codigo ?? 'x'}`" x1="0" y1="0" x2="1" y2="0">
            <stop offset="0" :stop-color="paleta.metal" />
            <stop offset="0.5" :stop-color="paleta.marco" />
            <stop offset="1" :stop-color="paleta.metal2" />
          </linearGradient>
          <linearGradient :id="`tronera-${codigo ?? 'x'}`" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stop-color="rgba(255,255,255,.14)" />
            <stop offset="1" stop-color="rgba(0,0,0,.5)" />
          </linearGradient>
        </defs>

        <rect x="0" y="0" width="320" height="416" :fill="`url(#fondo-${codigo ?? 'x'})`" />
        <!-- vetas del playfield -->
        <g :stroke="paleta.veta" stroke-width="1" opacity=".55">
          <path d="M20 40 Q20 250 58 268 L40 268 Q8 240 8 40 Z" fill="none" />
          <path d="M300 40 Q300 250 262 268 L280 268 Q312 240 312 40 Z" fill="none" />
        </g>
        <!-- marco del campo -->
        <path d="M28 40 Q28 246 62 268 L258 268 Q292 246 292 40"
              fill="none" :stroke="paleta.marco" stroke-width="5" opacity=".85" />
        <path d="M28 40 Q28 246 62 268 L258 268 Q292 246 292 40"
              fill="none" stroke="rgba(255,255,255,.3)" stroke-width="1.4" />

        <!-- salidas laterales -->
        <g :fill="paleta.marco" opacity=".5">
          <rect x="10" y="286" width="14" height="70" rx="4" />
          <rect x="296" y="286" width="14" height="70" rx="4" />
        </g>

        <!-- bumpers -->
        <g v-for="(b, i) in BUMPERS" :key="i">
          <circle :cx="b.x" :cy="b.y" :r="b.r * 1.5" :fill="paleta.bumped[i % paleta.bumped.length]" opacity=".22" />
          <circle :cx="b.x" :cy="b.y" :r="b.r" :fill="`url(#bumper-${codigo ?? 'x'})`"
                  :stroke="paleta.metal" stroke-width="2.5" />
          <circle :cx="b.x" :cy="b.y" :r="b.r * .62" fill="rgba(255,255,255,.18)"
                  stroke="rgba(255,255,255,.45)" stroke-width="1" />
          <text :x="b.x" :y="b.y + 1" text-anchor="middle" dominant-baseline="middle"
                font-family="ui-sans-serif, system-ui" font-size="15" font-weight="800"
                :fill="paleta.fondo2">x{{ b.mult }}</text>
        </g>

        <!-- guías de caída -->
        <g v-if="tabla.length > 1" stroke="rgba(255,255,255,.16)" stroke-width="2">
          <line v-for="i in tabla.length - 1" :key="i" :x1="(i * 320) / tabla.length"
                y1="268" :x2="(i * 320) / tabla.length" y2="330" />
        </g>

        <!-- troneras -->
        <g v-for="i in Math.max(1, tabla.length)" :key="i">
          <rect :x="(i - 1) * (320 / Math.max(1, tabla.length)) + 2" :y="TRONERA_Y"
                :width="320 / Math.max(1, tabla.length) - 4" :height="TRONERA_H" rx="5"
                :fill="`url(#tronera-${codigo ?? 'x'})`"
                :stroke="tronera === i - 1 && tabla[i - 1] > 0 ? '#fff3cf' : paleta.marco"
                :stroke-width="tronera === i - 1 && tabla[i - 1] > 0 ? 2.5 : 1.5" />
          <text v-if="tabla.length" :x="(i - 0.5) * (320 / Math.max(1, tabla.length))"
                :y="TRONERA_Y + TRONERA_H / 2" text-anchor="middle" dominant-baseline="middle"
                font-family="ui-sans-serif, system-ui" font-size="13" font-weight="800"
                :fill="tronera === i - 1 && tabla[i - 1] > 0 ? '#fff3cf' : 'rgba(232,225,205,.85)'">
            {{ tabla[i - 1] }}x
          </text>
        </g>

        <!-- paletas -->
        <g>
          <path d="M36 398 L112 366 L112 382 L40 414 Z" :fill="`url(#paleta-${codigo ?? 'x'})`"
                :stroke="paleta.metal" stroke-width="1.6" />
          <path d="M284 398 L208 366 L208 382 L280 414 Z" :fill="`url(#paleta-${codigo ?? 'x'})`"
                :stroke="paleta.metal" stroke-width="1.6" />
        </g>
      </svg>

      <canvas ref="canvas" class="mesa" />

      <div v-if="resultado && !animando" class="resuelto" :class="{ gana: premio > 0 }">
        <span class="n">x{{ multiplicador ?? resultado.multiplicador ?? 0 }}</span>
        <span v-if="premio > 0" class="p">{{ colones(premio) }}</span>
        <span v-else class="p cero">Sin premio</span>
      </div>
    </div>

    <div class="controles">
      <div class="grupo">
        <span class="etq">Fuerza</span>
        <button
          v-for="f in FUERZAS"
          :key="f.id"
          class="chip"
          :class="{ activa: fuerza === f.id }"
          @click="fuerza = f.id; reproducir('tecla')"
        >{{ f.nombre }}</button>
      </div>
    </div>

    <p class="regla">
      La bola rebota en los bumpers (x2, x3, x5) y cae en una tronera: la de más valor paga más.
      Suave da 4 rebotes y premio chico; media 7; fuerte 10 o más.
    </p>

    <button class="btn btn-primary btn-block" :disabled="cargando || animando" @click="lanzar">
      {{ cargando || animando ? 'Jugando…' : 'Lanzar bola' }}
    </button>
  </div>
</template>

<style scoped>
.pinball { display: flex; flex-direction: column; gap: 12px; }
.tablero {
  position: relative; width: min(340px, 78vw); margin: 0 auto;
  border-radius: 16px; overflow: hidden;
  border: 1px solid var(--border-soft);
  box-shadow: 0 18px 40px rgba(0, 0, 0, .45);
}
.campo { display: block; width: 100%; height: auto; aspect-ratio: 320 / 416; }
.mesa { position: absolute; inset: 0; display: block; width: 100%; height: 100%; }
.resuelto {
  position: absolute; left: 8px; right: 8px; bottom: 8px; z-index: 2;
  display: flex; align-items: center; justify-content: center; gap: 10px;
  padding: 7px 12px; border-radius: 10px;
  background: rgba(4, 6, 10, .92); border: 1px solid var(--border-soft);
}
.resuelto .n { font-family: var(--display); font-size: 22px; font-weight: 700; line-height: 1; }
.resuelto .p { color: var(--gold); font-weight: 700; font-size: 14px; }
.resuelto .p.cero { color: var(--muted); font-weight: 500; font-size: 13px; }
.resuelto.gana { border-color: rgba(245, 185, 66, .55); }
.controles { display: flex; flex-direction: column; gap: 8px; }
.grupo { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; justify-content: center; }
.etq {
  font-size: 11px; letter-spacing: .12em; text-transform: uppercase;
  color: var(--faint); min-width: 58px; text-align: right;
}
.chip {
  padding: 6px 12px; border-radius: 999px; border: 1px solid var(--border-soft);
  background: var(--surface); color: var(--muted); cursor: pointer; font-size: 12px;
}
.chip.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
.regla { margin: 0; font-size: 11.5px; color: var(--faint); text-align: center; line-height: 1.35; }
</style>
