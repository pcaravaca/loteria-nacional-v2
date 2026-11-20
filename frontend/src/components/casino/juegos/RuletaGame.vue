<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'
import { colorDeCasilla, modalidadCompartida, modalidadPorCodigo, modalidadesDe } from './modalidades'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

const JUEGO = 'ruleta'
const modalidades = modalidadesDe(JUEGO)
const modalidad = modalidadCompartida(JUEGO)
const rueda = computed(() => modalidadPorCodigo(JUEGO, modalidad.value))
// Casillas por las que se apuesta: la europea son los numeros del 0 al 36 y la
// rapida una rueda corta con 0 y 00.
const casillas = computed(() => rueda.value.casillas ?? [])
// Orden de la rueda europea (el 0 seguido del 32), que es solo visual.
const ORDEN_EUROPEA = ['0', '32', '15', '19', '4', '21', '2', '25', '17', '34', '6', '27', '13', '36', '11', '30', '8', '23', '10', '5', '24', '16', '33', '1', '20', '14', '31', '9', '22', '18', '29', '7', '28', '12', '35', '3', '26']
// Lo que se dibuja: la rueda corta se muestra en el orden de sus casillas.
const orden = computed(() => (rueda.value.codigo === 'rapida' ? casillas.value : ORDEN_EUROPEA))
const paso = computed(() => (Math.PI * 2) / Math.max(1, orden.value.length))

const canvas = ref<HTMLCanvasElement | null>(null)
const tipo = ref<'pleno' | 'color'>('pleno')
const numero = ref(17)
const color = ref<'rojo' | 'negro'>('rojo')
const rotacion = ref(0)
const girando = ref(false)
let animacion = 0

function colorDe(casilla: string): string {
  const c = colorDeCasilla(rueda.value, casilla)
  return c === 'rojo' ? '#b3242f' : c === 'verde' ? '#10b981' : '#14181f'
}

// Al cambiar de modalidad la rueda se puede achicar: el pleno elegido tiene que
// seguir siendo un indice valido.
watch(casillas, (cs) => {
  if (!cs.length) return
  if (numero.value > cs.length - 1) numero.value = cs.length - 1
  if (numero.value < 0) numero.value = 0
})

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
  for (let i = 0; i < orden.value.length; i++) {
    const casilla = orden.value[i]
    const a0 = i * paso.value - Math.PI / 2
    ctx.beginPath()
    ctx.moveTo(0, 0)
    ctx.arc(0, 0, radio - 8, a0, a0 + paso.value)
    ctx.closePath()
    ctx.fillStyle = colorDe(casilla)
    ctx.fill()
    ctx.strokeStyle = 'rgba(0,0,0,.5)'
    ctx.lineWidth = 1
    ctx.stroke()

    ctx.save()
    ctx.rotate(a0 + paso.value / 2)
    ctx.fillStyle = casilla === '0' || casilla === '00' ? '#04140e' : '#f3f5f7'
    ctx.font = 'bold 11px ui-sans-serif, system-ui'
    ctx.textAlign = 'center'
    ctx.fillText(casilla, radio - 22, 4)
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

function anguloFinal(casilla: string): number {
  const indice = orden.value.indexOf(casilla)
  // Queremos la casilla en la parte superior (-PI/2).
  const base = -Math.PI / 2 - indice * paso.value
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

  const pasoAnim = (ahora: number) => {
    const t = Math.min(1, (ahora - inicio) / duracion)
    const eased = 1 - Math.pow(1 - t, 3)
    rotacion.value = desde + (hasta - desde) * eased
    dibujar()
    if (t < 1) {
      animacion = requestAnimationFrame(pasoAnim)
    } else {
      window.clearTimeout(seguro)
      girando.value = false
    }
  }
  animacion = requestAnimationFrame(pasoAnim)
}

watch(
  () => props.resultado,
  async (r) => {
    if (!r) return
    await nextTick()
    reproducir('giro')
    animar(anguloFinal(String(r.casilla ?? r.numero)))
  },
)

// La rueda se redibuja al cambiar de modalidad y en cada apertura.
watch([orden, rotacion], dibujar)
onMounted(() => dibujar())

function elegirModalidad(codigo: string) {
  modalidad.value = codigo
  reproducir('tecla')
}

function jugar() {
  if (tipo.value === 'pleno') emit('jugar', { tipo: 'pleno', numero: numero.value, modalidad: modalidad.value })
  else emit('jugar', { tipo: 'color', color: color.value, modalidad: modalidad.value })
}
</script>

<template>
  <div class="ruleta">
    <div class="lienzo">
      <canvas ref="canvas" width="260" height="260" />
      <div v-if="resultado" class="resultado" :class="{ gana: premio > 0 }">
        <span
          class="n"
          :class="{ 'resalta-ganador': premio > 0 }"
          :style="{ background: resultado.color === 'verde' ? '#10b981' : resultado.color === 'rojo' ? '#b3242f' : '#14181f' }"
        >
          {{ resultado.casilla ?? resultado.numero }}
        </span>
        <span class="c">{{ resultado.color }}</span>
        <strong v-if="premio > 0">{{ colones(premio) }}</strong>
        <span v-else class="neutro">Sin acierto</span>
      </div>
    </div>

    <div class="modalidades">
      <button
        v-for="m in modalidades"
        :key="m.codigo"
        class="chip"
        :class="{ activa: modalidad === m.codigo }"
        @click="elegirModalidad(m.codigo)"
      >
        {{ m.nombre }}
      </button>
    </div>
    <p class="regla">{{ rueda.descripcion }}</p>

    <div class="modo">
      <button class="chip" :class="{ activa: tipo === 'pleno' }" @click="tipo = 'pleno'; reproducir('tecla')">Pleno {{ rueda.pagos[0].mult }}</button>
      <button class="chip" :class="{ activa: tipo === 'color' }" @click="tipo = 'color'; reproducir('tecla')">Color {{ rueda.pagos[1].mult }}</button>
    </div>

    <div v-if="tipo === 'pleno'" class="selector">
      <button class="step" @click="numero = (numero + casillas.length - 1) % casillas.length; reproducir('tecla')">−</button>
      <div class="valor">{{ casillas[numero] }}</div>
      <button class="step" @click="numero = (numero + 1) % casillas.length; reproducir('tecla')">+</button>
    </div>
    <div v-else class="selector">
      <button class="color-btn rojo" :class="{ activa: color === 'rojo' }" @click="color = 'rojo'; reproducir('tecla')">Rojo</button>
      <button class="color-btn negro" :class="{ activa: color === 'negro' }" @click="color = 'negro'; reproducir('tecla')">Negro</button>
    </div>

    <div class="paytable">
      <span v-for="p in rueda.pagos" :key="p.etiqueta">{{ p.etiqueta }} {{ p.mult }}</span>
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
.modalidades { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
.regla { margin: 0; font-size: 12px; color: var(--faint); text-align: center; line-height: 1.35; }
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
.paytable { display: flex; justify-content: center; gap: 12px; flex-wrap: wrap; font-size: 11px; color: var(--faint); }
</style>
