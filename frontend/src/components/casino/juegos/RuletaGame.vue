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

function aclarar(hex: string, t: number): string {
  const n = parseInt(hex.slice(1), 16)
  const r = (n >> 16) & 255, g = (n >> 8) & 255, b = n & 255
  const mez = (c: number) => Math.round(c + (255 - c) * t)
  return `rgb(${mez(r)},${mez(g)},${mez(b)})`
}
function sombrear(hex: string, t: number): string {
  const n = parseInt(hex.slice(1), 16)
  const r = (n >> 16) & 255, g = (n >> 8) & 255, b = n & 255
  const mez = (c: number) => Math.round(c * (1 - t))
  return `rgb(${mez(r)},${mez(g)},${mez(b)})`
}

// Rueda realista: cuenco de madera y aro metalico fijos, rotor (casillas,
// separadores, numeros y cono) que gira, y bola de marfil en el carril.
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
  const C = size / 2
  const R = C - 4

  ctx.clearRect(0, 0, size, size)

  // ---- cuenco de madera (fijo)
  let g = ctx.createRadialGradient(C, C, R * 0.72, C, C, R)
  g.addColorStop(0, '#5a2f1c')
  g.addColorStop(1, '#22100a')
  ctx.beginPath(); ctx.arc(C, C, R, 0, Math.PI * 2); ctx.fillStyle = g; ctx.fill()

  ctx.save()
  ctx.beginPath(); ctx.arc(C, C, R - 1, 0, Math.PI * 2); ctx.clip()
  ctx.globalAlpha = 0.14
  for (let i = 0; i < 40; i++) {
    const rr = R * 0.74 + i * (R * 0.26 / 40)
    ctx.beginPath()
    ctx.arc(C, C, rr, i * 0.7, i * 0.7 + 2.4)
    ctx.strokeStyle = i % 2 ? '#7a452a' : '#3a1c10'
    ctx.lineWidth = 1 + (i % 3)
    ctx.stroke()
  }
  ctx.restore()
  ctx.globalAlpha = 1

  // ---- aro metalico (fijo)
  const rOut = R * 0.99
  const rIn = R * 0.855
  g = ctx.createLinearGradient(C - R, C - R, C + R, C + R)
  g.addColorStop(0, '#8a6a1f')
  g.addColorStop(0.35, '#f4e2a0')
  g.addColorStop(0.6, '#b8922f')
  g.addColorStop(1, '#5e4712')
  ctx.beginPath()
  ctx.arc(C, C, rOut, 0, Math.PI * 2)
  ctx.arc(C, C, rIn, 0, Math.PI * 2, true)
  ctx.fillStyle = g
  ctx.fill()
  ctx.beginPath()
  ctx.arc(C, C, (rOut + rIn) / 2, 0, Math.PI * 2)
  ctx.strokeStyle = 'rgba(255,236,170,.55)'
  ctx.lineWidth = Math.max(1.5, size * 0.006)
  ctx.stroke()

  // ---- rotor (gira)
  ctx.save()
  ctx.translate(C, C)
  ctx.rotate(rotacion.value)

  const rBol = R * 0.60
  const n = orden.value.length

  for (let i = 0; i < n; i++) {
    const casilla = orden.value[i]
    const a0 = i * paso.value - Math.PI / 2 - paso.value / 2
    const a1 = a0 + paso.value
    const base = colorDe(casilla)
    ctx.beginPath()
    ctx.arc(0, 0, rIn, a0, a1)
    ctx.arc(0, 0, rBol, a1, a0, true)
    ctx.closePath()
    const rg = ctx.createRadialGradient(0, 0, rBol, 0, 0, rIn)
    rg.addColorStop(0, sombrear(base, 0.45))
    rg.addColorStop(0.55, base)
    rg.addColorStop(1, aclarar(base, 0.18))
    ctx.fillStyle = rg
    ctx.fill()
  }

  // separadores dorados (en los bordes de las casillas, no en el centro)
  ctx.strokeStyle = '#d9b45c'
  ctx.lineWidth = Math.max(1.2, size * 0.005)
  for (let i = 0; i < n; i++) {
    const a = -Math.PI / 2 + i * paso.value - paso.value / 2
    ctx.beginPath()
    ctx.moveTo(Math.cos(a) * rBol, Math.sin(a) * rBol)
    ctx.lineTo(Math.cos(a) * rIn, Math.sin(a) * rIn)
    ctx.stroke()
  }

  // numeros de las casillas
  ctx.fillStyle = '#f4f1e8'
  ctx.font = `700 ${Math.max(8, Math.round(size * 0.042))}px ui-sans-serif, system-ui`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  for (let i = 0; i < n; i++) {
    const a = -Math.PI / 2 + i * paso.value
    const rr = (rIn + rBol) / 2
    ctx.save()
    ctx.rotate(a)
    ctx.translate(rr, 0)
    ctx.rotate(Math.PI / 2)
    ctx.fillText(orden.value[i], 0, 0)
    ctx.restore()
  }

  // cono central
  g = ctx.createRadialGradient(-R * 0.14, -R * 0.18, 4, 0, 0, rBol)
  g.addColorStop(0, '#8a4a2c')
  g.addColorStop(0.55, '#4a2415')
  g.addColorStop(1, '#1d0d07')
  ctx.beginPath(); ctx.arc(0, 0, rBol, 0, Math.PI * 2); ctx.fillStyle = g; ctx.fill()
  ctx.beginPath()
  ctx.arc(0, 0, rBol * 0.99, 0, Math.PI * 2)
  ctx.strokeStyle = 'rgba(255,236,170,.4)'
  ctx.lineWidth = Math.max(1, size * 0.004)
  ctx.stroke()

  // husillo
  g = ctx.createLinearGradient(-size * 0.02, -size * 0.11, size * 0.02, size * 0.05)
  g.addColorStop(0, '#f4e2a0')
  g.addColorStop(0.5, '#b8922f')
  g.addColorStop(1, '#5e4712')
  ctx.beginPath()
  ctx.moveTo(0, -R * 0.235)
  ctx.quadraticCurveTo(R * 0.14, -R * 0.05, R * 0.08, R * 0.11)
  ctx.lineTo(-R * 0.08, R * 0.11)
  ctx.quadraticCurveTo(-R * 0.14, -R * 0.05, 0, -R * 0.235)
  ctx.fillStyle = g
  ctx.fill()

  ctx.restore()

  // ---- bola de marfil en el carril exterior (fija, marca la casilla ganadora)
  const rb = Math.max(4, size * 0.024)
  const bx = C
  const by = C - R * 0.92
  ctx.beginPath()
  ctx.ellipse(bx + rb * 0.35, by + rb * 0.45, rb, rb * 0.9, 0, 0, Math.PI * 2)
  ctx.fillStyle = 'rgba(0,0,0,.35)'
  ctx.fill()
  g = ctx.createRadialGradient(bx - rb * 0.3, by - rb * 0.35, 1, bx, by, rb)
  g.addColorStop(0, '#ffffff')
  g.addColorStop(0.45, '#f0e9d6')
  g.addColorStop(1, '#8d8574')
  ctx.beginPath(); ctx.arc(bx, by, rb, 0, Math.PI * 2); ctx.fillStyle = g; ctx.fill()

  // ---- luz superior
  g = ctx.createRadialGradient(C - R * 0.35, C - R * 0.45, 10, C, C, R * 1.15)
  g.addColorStop(0, 'rgba(255,255,255,.18)')
  g.addColorStop(0.45, 'rgba(255,255,255,.05)')
  g.addColorStop(1, 'rgba(0,0,0,.28)')
  ctx.beginPath(); ctx.arc(C, C, R, 0, Math.PI * 2); ctx.fillStyle = g; ctx.fill()
}

function anguloFinal(casilla: string): number {
  const indice = orden.value.indexOf(casilla)
  // La casilla tiene que quedar bajo la bola (arriba, -PI/2). El centro de la
  // casilla i en el rotor es `-PI/2 + i*paso`, asi que el giro necesario es
  // exactamente `-i*paso`. (Antes sobraba un -PI/2 y la casilla quedaba ~90° corrida.)
  const base = -indice * paso.value
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
