<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import { colones } from '@/utils/formato'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

const ROJO = new Set([1, 3, 5, 7, 9, 12, 14, 16, 18, 19, 21, 23, 25, 27, 30, 32, 34, 36])
// Orden de la ruleta europea (el 0 seguido del 32).
const ORDEN = [0, 32, 15, 19, 4, 21, 2, 25, 17, 34, 6, 27, 13, 36, 11, 30, 8, 23, 10, 5, 24, 16, 33, 1, 20, 14, 31, 9, 22, 18, 29, 7, 28, 12, 35, 3, 26]
const PASO = (Math.PI * 2) / 37

const canvas = ref<HTMLCanvasElement | null>(null)
const tipo = ref<'pleno' | 'color'>('pleno')
const numero = ref<number>(17)
const color = ref<'rojo' | 'negro'>('rojo')
const rotacion = ref(0)
const girando = ref(false)
let animacion = 0

function colorDe(n: number): string {
  if (n === 0) return '#10b981'
  return ROJO.has(n) ? '#b3242f' : '#14181f'
}

function dibujar() {
  const cv = canvas.value
  if (!cv) return
  const ctx = cv.getContext('2d')!
  // El canvas se adapta al ancho real (moviles) y usa la densidad de pixeles
  // del dispositivo para que la rueda no se vea borrosa.
  const css = cv.clientWidth || 260
  const dpr = window.devicePixelRatio || 1
  const want = Math.round(css * dpr)
  if (cv.width !== want || cv.height !== want) {
    cv.width = want
    cv.height = want
  }
  const size = cv.width
  const centro = size / 2
  const radio = centro - 6

  ctx.clearRect(0, 0, size, size)
  ctx.save()
  ctx.translate(centro, centro)

  // aro exterior
  ctx.beginPath()
  ctx.arc(0, 0, radio, 0, Math.PI * 2)
  ctx.fillStyle = '#0d2a22'
  ctx.fill()
  ctx.lineWidth = 4
  ctx.strokeStyle = '#c9a227'
  ctx.stroke()

  ctx.save()
  ctx.rotate(rotacion.value)
  for (let i = 0; i < ORDEN.length; i++) {
    const n = ORDEN[i]
    const a0 = i * PASO - Math.PI / 2
    ctx.beginPath()
    ctx.moveTo(0, 0)
    ctx.arc(0, 0, radio - 8, a0, a0 + PASO)
    ctx.closePath()
    ctx.fillStyle = colorDe(n)
    ctx.fill()
    ctx.strokeStyle = 'rgba(0,0,0,.5)'
    ctx.lineWidth = 1
    ctx.stroke()

    ctx.save()
    ctx.rotate(a0 + PASO / 2)
    ctx.fillStyle = n === 0 ? '#04140e' : '#f3f5f7'
    ctx.font = 'bold 11px ui-sans-serif, system-ui'
    ctx.textAlign = 'center'
    ctx.fillText(String(n), radio - 22, 4)
    ctx.restore()
  }
  ctx.restore()

  // bolita fija en la parte superior
  ctx.beginPath()
  ctx.arc(0, -radio + 2, 6, 0, Math.PI * 2)
  ctx.fillStyle = '#f8fafc'
  ctx.shadowColor = 'rgba(0,0,0,.6)'
  ctx.shadowBlur = 6
  ctx.fill()
  ctx.restore()
}

function anguloFinal(n: number): number {
  const indice = ORDEN.indexOf(n)
  // Queremos el numero en la parte superior (-PI/2).
  const base = -Math.PI / 2 - indice * PASO
  const vueltas = Math.PI * 2 * (4 + Math.floor(Math.random() * 2))
  // Normalizamos para que gire siempre hacia adelante.
  let destino = base + vueltas
  while (destino < rotacion.value) destino += Math.PI * 2
  return destino
}

function animar(hasta: number) {
  const desde = rotacion.value
  const inicio = performance.now()
  const duracion = 3400
  girando.value = true

  // Seguridad: si requestAnimationFrame se pausa (pestania en segundo plano),
  // la ruleta se queda girando. Se garantiza el estado final.
  const seguro = window.setTimeout(() => {
    window.cancelAnimationFrame(animacion)
    rotacion.value = hasta
    dibujar()
    girando.value = false
  }, duracion + 300)

  const paso = (ahora: number) => {
    const t = Math.min(1, (ahora - inicio) / duracion)
    const eased = 1 - Math.pow(1 - t, 3)
    rotacion.value = desde + (hasta - desde) * eased
    dibujar()
    if (t < 1) {
      animacion = requestAnimationFrame(paso)
    } else {
      window.clearTimeout(seguro)
      girando.value = false
    }
  }
  animacion = requestAnimationFrame(paso)
}

watch(
  () => props.resultado,
  async (r) => {
    if (!r) return
    await nextTick()
    animar(anguloFinal(r.numero))
  },
)

onMounted(() => dibujar())

function jugar() {
  if (tipo.value === 'pleno') emit('jugar', { tipo: 'pleno', numero: numero.value })
  else emit('jugar', { tipo: 'color', color: color.value })
}
</script>

<template>
  <div class="ruleta">
    <div class="lienzo">
      <canvas ref="canvas" width="260" height="260" />
      <div v-if="resultado" class="resultado" :class="{ gana: premio > 0 }">
        <span class="n" :style="{ background: resultado.color === 'verde' ? '#10b981' : resultado.color === 'rojo' ? '#b3242f' : '#14181f' }">
          {{ resultado.numero }}
        </span>
        <span class="c">{{ resultado.color }}</span>
        <strong v-if="premio > 0">{{ colones(premio) }}</strong>
        <span v-else class="neutro">Sin acierto</span>
      </div>
    </div>

    <div class="modo">
      <button class="chip" :class="{ activa: tipo === 'pleno' }" @click="tipo = 'pleno'">Pleno x35</button>
      <button class="chip" :class="{ activa: tipo === 'color' }" @click="tipo = 'color'">Color x2</button>
    </div>

    <div v-if="tipo === 'pleno'" class="selector">
      <button class="step" @click="numero = (numero + 36) % 37">−</button>
      <div class="valor">{{ numero }}</div>
      <button class="step" @click="numero = (numero + 1) % 37">+</button>
    </div>
    <div v-else class="selector">
      <button class="color-btn rojo" :class="{ activa: color === 'rojo' }" @click="color = 'rojo'">Rojo</button>
      <button class="color-btn negro" :class="{ activa: color === 'negro' }" @click="color = 'negro'">Negro</button>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando || girando" @click="jugar">
      {{ cargando || girando ? 'Girando…' : 'Girar la ruleta' }}
    </button>
  </div>
</template>

<style scoped>
.ruleta { display: flex; flex-direction: column; gap: 12px; align-items: center; }
.lienzo { position: relative; width: min(260px, 74vw); aspect-ratio: 1 / 1; margin: 0 auto; }
.lienzo canvas { display: block; width: 100%; height: 100%; }
.resultado {
  position: absolute; inset: auto 0 -8px 0; display: flex; align-items: center; justify-content: center;
  gap: 8px; font-size: 13px; color: var(--muted);
}
.resultado .n {
  min-width: 30px; height: 30px; padding: 0 6px; border-radius: 8px; display: grid; place-items: center;
  color: #fff; font-weight: 800;
}
.resultado strong { color: var(--gold); font-size: 16px; }
.resultado.gana { color: var(--prim-2); }
.modo { display: flex; gap: 8px; }
.chip { padding: 6px 12px; border-radius: 999px; border: 1px solid var(--border-soft); background: var(--surface); color: var(--muted); cursor: pointer; font-size: 12px; }
.chip.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
.selector { display: flex; gap: 10px; align-items: center; }
.step { width: 38px; height: 38px; border-radius: 10px; border: 1px solid var(--border); background: var(--surface); color: var(--text); font-size: 18px; cursor: pointer; }
.valor { min-width: 54px; height: 38px; display: grid; place-items: center; border-radius: 10px; background: #05080c; border: 1px solid var(--border); font-weight: 800; font-size: 18px; }
.color-btn { flex: 1; padding: 10px; border-radius: 10px; border: 1px solid var(--border); color: #fff; font-weight: 700; cursor: pointer; opacity: .6; }
.color-btn.rojo { background: #b3242f; }
.color-btn.negro { background: #14181f; }
.color-btn.activa { opacity: 1; border-color: var(--gold); }
</style>
