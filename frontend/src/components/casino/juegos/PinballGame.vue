<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
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

const canvas = ref<HTMLCanvasElement | null>(null)
const animando = ref(false)
const fila = ref(-1)
const bolaX = ref(160)
const bolaY = ref(14)
const tronera = ref<number | null>(null)

const tabla = computed<number[]>(() => props.resultado?.tabla ?? [])
const camino = computed<number[]>(() => props.resultado?.camino ?? [])

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

  ctx.clearRect(0, 0, W, H)
  // caja de la mesa
  ctx.fillStyle = 'rgba(6, 12, 20, .9)'
  ctx.beginPath()
  ctx.roundRect(2 * u, 2 * u, (320 - 4) * u, (416 - 4) * u, 14 * u)
  ctx.fill()
  ctx.strokeStyle = 'rgba(232, 206, 122, .35)'
  ctx.lineWidth = 2 * u
  ctx.stroke()

  const filas = Math.max(camino.value.length, tabla.value.length ? tabla.value.length - 1 : 0)

  // bumpers (fijos, decorativos): tres círculos con su multiplicador
  const bumpers = [
    { x: 92, y: 74, r: 20, mult: 2 },
    { x: 228, y: 74, r: 20, mult: 3 },
    { x: 160, y: 126, r: 20, mult: 5 },
  ]
  for (const b of bumpers) {
    ctx.beginPath()
    ctx.arc(b.x * u, b.y * u, b.r * u, 0, Math.PI * 2)
    ctx.fillStyle = 'rgba(16, 185, 129, .14)'
    ctx.fill()
    ctx.lineWidth = 3 * u
    ctx.strokeStyle = 'rgba(16, 185, 129, .7)'
    ctx.stroke()
    ctx.fillStyle = 'rgba(216, 245, 232, .9)'
    ctx.font = `700 ${Math.round(13 * u)}px ui-sans-serif, system-ui`
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText(`x${b.mult}`, b.x * u, b.y * u)
  }

  // guías de caída
  ctx.strokeStyle = 'rgba(142, 162, 182, .2)'
  ctx.lineWidth = 2 * u
  for (let i = 1; i < tabla.value.length; i++) {
    const x = (i * 320) / tabla.value.length
    ctx.beginPath()
    ctx.moveTo(x * u, 252 * u)
    ctx.lineTo(x * u, 332 * u)
    ctx.stroke()
  }

  // palancas (parte de la animación de la mesa), debajo de las troneras
  ctx.fillStyle = 'rgba(232, 206, 122, .85)'
  ctx.beginPath()
  ctx.moveTo(34 * u, 400 * u)
  ctx.lineTo(106 * u, 372 * u)
  ctx.lineTo(106 * u, 382 * u)
  ctx.lineTo(38 * u, 412 * u)
  ctx.closePath()
  ctx.fill()
  ctx.beginPath()
  ctx.moveTo(286 * u, 400 * u)
  ctx.lineTo(214 * u, 372 * u)
  ctx.lineTo(214 * u, 382 * u)
  ctx.lineTo(282 * u, 412 * u)
  ctx.closePath()
  ctx.fill()

  // troneras
  const n = Math.max(1, tabla.value.length)
  for (let i = 0; i < n; i++) {
    const x = ((i + 0.5) * 320) / n
    const ancho = 320 / n
    const esGanadora =
      props.resultado && !animando.value && tronera.value === i && tabla.value[i] > 0
    ctx.fillStyle = esGanadora ? 'rgba(16, 185, 129, .85)' : 'rgba(255,255,255,.05)'
    ctx.fillRect((x - ancho / 2 + 2) * u, 332 * u, (ancho - 4) * u, 26 * u)
    ctx.strokeStyle = esGanadora ? '#d9b45c' : 'rgba(142,162,182,.35)'
    ctx.lineWidth = esGanadora ? 3 * u : 1.5 * u
    ctx.strokeRect((x - ancho / 2 + 2) * u, 332 * u, (ancho - 4) * u, 26 * u)
    ctx.fillStyle = esGanadora ? '#04140e' : 'rgba(180, 194, 210, .8)'
    ctx.font = `800 ${Math.round(12 * u)}px ui-sans-serif, system-ui`
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText(`${tabla.value[i]}x`, x * u, 345 * u)
  }

  // fila actual del camino
  if (fila.value >= 0 && fila.value < filas) {
    const y = (46 + (fila.value + 1) * (252 - 46) / Math.max(1, filas)) * u
    ctx.strokeStyle = 'rgba(232, 206, 122, .25)'
    ctx.lineWidth = 1.5 * u
    ctx.beginPath()
    ctx.moveTo(20 * u, y)
    ctx.lineTo(300 * u, y)
    ctx.stroke()
  }

  // bola
  if (props.resultado) {
    const bx = bolaX.value * u
    const by = bolaY.value * u
    ctx.beginPath()
    ctx.arc(bx, by, 10 * u, 0, Math.PI * 2)
    ctx.fillStyle = 'rgba(0,0,0,.4)'
    ctx.fill()
    const g = ctx.createRadialGradient(bx - 3 * u, by - 3 * u, 1, bx, by, 10 * u)
    g.addColorStop(0, '#ffffff')
    g.addColorStop(0.5, '#f5dd9a')
    g.addColorStop(1, '#a8740a')
    ctx.beginPath()
    ctx.arc(bx, by, 9 * u, 0, Math.PI * 2)
    ctx.fillStyle = g
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
      bolaY.value = 296
      tronera.value = final
      animando.value = false
      dibujar()
      if (props.premio > 0) reproducir('ganar')
      else reproducir('perder')
      return
    }
    fila.value = k
    bolaX.value = 160 + (camino.value[k] === 1 ? 1 : -1) * 22
    bolaY.value = 14 + ((k + 1) / Math.max(1, total)) * 250
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
  border-radius: 14px; overflow: hidden;
  border: 1px solid var(--border-soft);
  background: linear-gradient(180deg, rgba(16, 185, 129, .05), transparent 70%);
}
.mesa { display: block; width: 100%; height: auto; }
.resuelto {
  position: absolute; top: 28%; left: 50%; transform: translateX(-50%);
  display: flex; flex-direction: column; align-items: center; gap: 2px;
  padding: 10px 20px; border-radius: 12px; text-align: center;
  background: rgba(8, 11, 16, .9); border: 1px solid var(--border);
}
.resuelto .n { font-family: var(--display); font-size: 26px; font-weight: 700; }
.resuelto .p { color: var(--gold); font-weight: 700; }
.resuelto .p.cero { color: var(--muted); font-weight: 500; }
.resuelto.gana { border-color: rgba(245, 185, 66, .5); }
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
