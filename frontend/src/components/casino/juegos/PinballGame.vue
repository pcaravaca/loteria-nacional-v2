<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
  codigo?: string
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

// La fuerza elige cuántos rebotes da la bola (y con eso la tabla de pagos):
// a más fuerza, mayor dispersión y mayor premio máximo.
const FUERZAS = [
  { id: 0, nombre: 'Suave' },
  { id: 1, nombre: 'Media' },
  { id: 2, nombre: 'Fuerte' },
]
const fuerza = ref(1)

// Cada mesa tiene su propia madera / metal y su acento.
const PALETAS: Record<string, { fondo: string; marco: string; vetas: string; acento: string; bumped: string[] }> = {
  'pinball-neon': {
    fondo: '#150a2a', marco: '#6d3fd1', vetas: '#3a1f7a', acento: '#22d3ee',
    bumped: ['#22d3ee', '#e879f9', '#34d399'],
  },
  'pinball-clasico': {
    fondo: '#1c1106', marco: '#a8742c', vetas: '#5e3a14', acento: '#e8ce7a',
    bumped: ['#d9a441', '#c0562f', '#3f8f5f'],
  },
  'plinko-premium': {
    fondo: '#04202a', marco: '#0f766e', vetas: '#0a4a52', acento: '#5eead4',
    bumped: ['#2dd4bf', '#f5d06a', '#7dd3fc'],
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

function aclarar(hex: string, t: number): string {
  const n = parseInt(hex.slice(1), 16)
  const mez = (c: number) => Math.round(c + (255 - c) * t)
  return `rgb(${mez((n >> 16) & 255)},${mez((n >> 8) & 255)},${mez(n & 255)})`
}
function sombrear(hex: string, t: number): string {
  const n = parseInt(hex.slice(1), 16)
  const mez = (c: number) => Math.round(c * (1 - t))
  return `rgb(${mez((n >> 16) & 255)},${mez((n >> 8) & 255)},${mez(n & 255)})`
}

function dibujar() {
  const cv = canvas.value
  if (!cv) return
  const ctx = cv.getContext('2d')!
  const cssW = cv.clientWidth || 320
  const dpr = window.devicePixelRatio || 1
  const W = Math.round(cssW * dpr)
  const H = Math.round(W * 1.3)
  if (cv.width !== W) {
    cv.width = W
    cv.height = H
  }
  const u = W / 320
  const P = paleta.value

  ctx.clearRect(0, 0, W, H)

  // ---- marco de madera / metal del gabinete
  ctx.beginPath()
  ctx.roundRect(2 * u, 2 * u, (320 - 4) * u, (416 - 4) * u, 16 * u)
  const gm = ctx.createLinearGradient(0, 0, W, H)
  gm.addColorStop(0, aclarar(P.marco, 0.25))
  gm.addColorStop(0.45, P.marco)
  gm.addColorStop(1, sombrear(P.marco, 0.55))
  ctx.fillStyle = gm
  ctx.fill()

  // vetas del marco
  ctx.save()
  ctx.beginPath()
  ctx.roundRect(2 * u, 2 * u, (320 - 4) * u, (416 - 4) * u, 16 * u)
  ctx.clip()
  ctx.globalAlpha = 0.22
  ctx.strokeStyle = P.vetas
  for (let i = 0; i < 26; i++) {
    ctx.lineWidth = (1 + (i % 3)) * u
    ctx.beginPath()
    ctx.moveTo(-20 * u, i * 18 * u)
    ctx.bezierCurveTo(90 * u, i * 18 * u + 8 * u, 220 * u, i * 18 * u - 8 * u, 350 * u, i * 18 * u + 4 * u)
    ctx.stroke()
  }
  ctx.restore()
  ctx.globalAlpha = 1

  // ---- campo de juego (acrílico oscuro sobre la mesa)
  ctx.beginPath()
  ctx.roundRect(14 * u, 14 * u, (320 - 28) * u, (416 - 28) * u, 12 * u)
  const gc = ctx.createRadialGradient(W / 2, H * 0.35, 10 * u, W / 2, H / 2, W * 0.75)
  gc.addColorStop(0, aclarar(P.fondo, 0.16))
  gc.addColorStop(0.6, P.fondo)
  gc.addColorStop(1, sombrear(P.fondo, 0.5))
  ctx.fillStyle = gc
  ctx.fill()
  ctx.strokeStyle = aclarar(P.marco, 0.1)
  ctx.lineWidth = 1.5 * u
  ctx.stroke()

  const filas = Math.max(camino.value.length, tabla.value.length ? tabla.value.length - 1 : 0)

  // ---- riel del campo (brillo metálico)
  ctx.beginPath()
  ctx.moveTo(30 * u, 60 * u)
  ctx.quadraticCurveTo(30 * u, 236 * u, 62 * u, 258 * u)
  ctx.lineTo(258 * u, 258 * u)
  ctx.quadraticCurveTo(290 * u, 236 * u, 290 * u, 60 * u)
  ctx.strokeStyle = aclarar(P.marco, 0.35)
  ctx.lineWidth = 5 * u
  ctx.stroke()
  ctx.strokeStyle = 'rgba(255,255,255,.28)'
  ctx.lineWidth = 1.5 * u
  ctx.stroke()

  // ---- bumpers con volumen y halo
  const bumpers = [
    { x: 92, y: 74, r: 22, mult: 2 },
    { x: 228, y: 74, r: 22, mult: 3 },
    { x: 160, y: 128, r: 22, mult: 5 },
  ]
  bumpers.forEach((b, i) => {
    const col = P.bumped[i % P.bumped.length]
    const halo = ctx.createRadialGradient(b.x * u, b.y * u, b.r * 0.2 * u, b.x * u, b.y * u, b.r * 1.9 * u)
    halo.addColorStop(0, col + '66')
    halo.addColorStop(1, col + '00')
    ctx.beginPath()
    ctx.arc(b.x * u, b.y * u, b.r * 1.9 * u, 0, Math.PI * 2)
    ctx.fillStyle = halo
    ctx.fill()

    const cuerpo = ctx.createRadialGradient(
      (b.x - b.r * 0.35) * u, (b.y - b.r * 0.4) * u, b.r * 0.1 * u,
      b.x * u, b.y * u, b.r * u,
    )
    cuerpo.addColorStop(0, aclarar(col, 0.75))
    cuerpo.addColorStop(0.45, col)
    cuerpo.addColorStop(1, sombrear(col, 0.6))
    ctx.beginPath()
    ctx.arc(b.x * u, b.y * u, b.r * u, 0, Math.PI * 2)
    ctx.fillStyle = cuerpo
    ctx.fill()

    ctx.beginPath()
    ctx.arc(b.x * u, b.y * u, b.r * u, 0, Math.PI * 2)
    ctx.strokeStyle = aclarar(P.marco, 0.4)
    ctx.lineWidth = 2.5 * u
    ctx.stroke()

    ctx.beginPath()
    ctx.arc(b.x * u, b.y * u, b.r * 0.55 * u, 0, Math.PI * 2)
    ctx.fillStyle = 'rgba(255,255,255,.16)'
    ctx.fill()
    ctx.strokeStyle = 'rgba(255,255,255,.35)'
    ctx.lineWidth = 1 * u
    ctx.stroke()

    ctx.fillStyle = '#0b0f14'
    ctx.font = `800 ${Math.round(13 * u)}px ui-sans-serif, system-ui`
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText(`x${b.mult}`, b.x * u, b.y * u + 1 * u)
  })

  // ---- guías de caída hacia las troneras
  ctx.strokeStyle = 'rgba(255,255,255,.14)'
  ctx.lineWidth = 2 * u
  for (let i = 1; i < tabla.value.length; i++) {
    const x = (i * 320) / tabla.value.length
    ctx.beginPath()
    ctx.moveTo(x * u, 266 * u)
    ctx.lineTo(x * u, 330 * u)
    ctx.stroke()
  }

  // ---- palancas con gradiente metálico (bajo las troneras)
  const palanca = (x1: number, y1: number, x2: number, y2: number, x3: number, y3: number, x4: number, y4: number) => {
    const gp = ctx.createLinearGradient(x1 * u, y1 * u, x2 * u, y2 * u)
    gp.addColorStop(0, '#f6e6b0')
    gp.addColorStop(0.5, P.marco)
    gp.addColorStop(1, sombrear(P.marco, 0.5))
    ctx.beginPath()
    ctx.moveTo(x1 * u, y1 * u)
    ctx.lineTo(x2 * u, y2 * u)
    ctx.lineTo(x3 * u, y3 * u)
    ctx.lineTo(x4 * u, y4 * u)
    ctx.closePath()
    ctx.fillStyle = gp
    ctx.fill()
  }
  palanca(34, 398, 108, 370, 108, 381, 38, 410)
  palanca(286, 398, 212, 370, 212, 381, 282, 410)

  // ---- troneras con volumen y premio
  const n = Math.max(1, tabla.value.length)
  for (let i = 0; i < n; i++) {
    const x = ((i + 0.5) * 320) / n
    const ancho = 320 / n
    const esGanadora =
      props.resultado && !animando.value && tronera.value === i && tabla.value[i] > 0
    const x0 = (x - ancho / 2 + 2) * u
    const w = (ancho - 4) * u
    const y0 = 330 * u
    const h = 28 * u
    const gt = ctx.createLinearGradient(0, y0, 0, y0 + h)
    if (esGanadora) {
      gt.addColorStop(0, '#f8e2a8')
      gt.addColorStop(0.5, P.bumped[0])
      gt.addColorStop(1, sombrear(P.bumped[0], 0.55))
    } else {
      gt.addColorStop(0, 'rgba(255,255,255,.10)')
      gt.addColorStop(1, 'rgba(0,0,0,.45)')
    }
    ctx.fillStyle = gt
    ctx.fillRect(x0, y0, w, h)
    ctx.strokeStyle = esGanadora ? '#fff3cf' : aclarar(P.marco, 0.15)
    ctx.lineWidth = esGanadora ? 2.5 * u : 1.5 * u
    ctx.strokeRect(x0, y0, w, h)
    ctx.fillStyle = esGanadora ? '#14100a' : 'rgba(232,225,205,.82)'
    ctx.font = `800 ${Math.round(12 * u)}px ui-sans-serif, system-ui`
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText(`${tabla.value[i]}x`, x * u, y0 + h / 2)
  }

  // ---- fila actual del camino
  if (fila.value >= 0 && fila.value < filas) {
    const y = (46 + (fila.value + 1) * (250 - 46) / Math.max(1, filas)) * u
    ctx.strokeStyle = 'rgba(255,255,255,.18)'
    ctx.lineWidth = 1.5 * u
    ctx.beginPath()
    ctx.moveTo(24 * u, y)
    ctx.lineTo(296 * u, y)
    ctx.stroke()
  }

  // ---- bola de marfil con reflejo
  if (props.resultado) {
    const bx = bolaX.value * u
    const by = bolaY.value * u
    const rb = 9 * u
    ctx.beginPath()
    ctx.ellipse(bx + rb * 0.25, by + rb * 0.4, rb, rb * 0.85, 0, 0, Math.PI * 2)
    ctx.fillStyle = 'rgba(0,0,0,.45)'
    ctx.fill()
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
    ctx.fillStyle = 'rgba(255,255,255,.85)'
    ctx.fill()
  }

  // ---- luz superior (bajo el aro)
  const gl = ctx.createRadialGradient(W * 0.5, H * 0.16, 4 * u, W * 0.5, H * 0.16, W * 0.55)
  gl.addColorStop(0, 'rgba(255,255,255,.16)')
  gl.addColorStop(0.5, 'rgba(255,255,255,.04)')
  gl.addColorStop(1, 'rgba(0,0,0,.3)')
  ctx.beginPath()
  ctx.roundRect(14 * u, 14 * u, (320 - 28) * u, (416 - 28) * u, 12 * u)
  ctx.fillStyle = gl
  ctx.fill()
}

function animar() {
  const total = camino.value.length
  animando.value = true
  tronera.value = null
  fila.value = -1
  bolaX.value = 160
  bolaY.value = 14
  dibujar()
  reproducir('caida')
  let k = 0
  const timer = setInterval(() => {
    if (k >= total) {
      clearInterval(timer)
      const final = props.resultado?.pos ?? 0
      const n = Math.max(1, tabla.value.length)
      // la bola queda en el centro de la tronera ganadora, sobre los pagos
      bolaX.value = ((final + 0.5) * 320) / n
      bolaY.value = 294
      tronera.value = final
      animando.value = false
      dibujar()
      if (props.premio > 0) reproducir('ganar')
      else reproducir('perder')
      return
    }
    fila.value = k
    bolaX.value = 160 + (camino.value[k] === 1 ? 1 : -1) * 22
    bolaY.value = 14 + ((k + 1) / Math.max(1, total)) * 240
    reproducir('tecla')
    dibujar()
    k++
  }, 180)
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

function lanzar() {
  emit('jugar', { fuerza: fuerza.value })
}
</script>

<template>
  <div class="pinball">
    <div class="tablero">
      <canvas ref="canvas" class="mesa" />
      <div v-if="resultado && !animando" class="resuelto" :class="{ gana: premio > 0 }">
        <span class="n">x{{ resultado.multiplicador ?? 0 }}</span>
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
      Suave da 4 rebotes y premio chico; media 7; fuerte 10 o más, con el premio más alto.
      Cada rebote sale del sorteo: par = izquierda, impar = derecha.
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
.mesa { display: block; width: 100%; height: auto; }
.resuelto {
  position: absolute; top: 26%; left: 50%; transform: translateX(-50%);
  display: flex; flex-direction: column; align-items: center; gap: 2px;
  padding: 10px 20px; border-radius: 12px; text-align: center;
  background: rgba(6, 8, 12, .88); border: 1px solid var(--border);
  box-shadow: 0 8px 24px rgba(0, 0, 0, .5);
}
.resuelto .n { font-family: var(--display); font-size: 26px; font-weight: 700; }
.resuelto .p { color: var(--gold); font-weight: 700; }
.resuelto .p.cero { color: var(--muted); font-weight: 500; }
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
