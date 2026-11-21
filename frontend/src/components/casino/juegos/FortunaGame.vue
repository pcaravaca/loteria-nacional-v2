<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

// La tabla del servidor es la fuente de verdad; esta copia dibuja la ruleta.
const TABLA = [
  0, 0, 0, 0, 0, 0, 0, 0, 0,
  1, 1, 1, 1, 1, 1, 1,
  2, 2,
  3,
  5,
]
const SEGMENTOS = TABLA.length
const SALTO = 360 / SEGMENTOS
const BULBOS = 18

// Colores por valor (de mas comun a mas raro): del azul apagado al oro pleno.
const colorValor = (v: number) =>
  v === 0 ? '#22364f' : v === 1 ? '#0d5c3c' : v === 2 ? '#0a8f6c' : v === 3 ? '#8f6d12' : '#f5b942'

const canvas = ref<HTMLCanvasElement | null>(null)
const posicion = ref<number | null>(null)
const rotando = ref(false)
const rotacion = ref(0)

function dibujar() {
  const cv = canvas.value
  if (!cv) return
  const css = cv.clientWidth || 200
  const dpr = window.devicePixelRatio || 1
  const want = Math.round(css * dpr)
  if (cv.width !== want || cv.height !== want) {
    cv.width = want
    cv.height = want
  }
  const size = cv.width
  const centro = size / 2
  const radio = centro - 4
  const ctx = cv.getContext('2d')
  if (!ctx) return
  ctx.clearRect(0, 0, size, size)

  // Segmentos con su multiplicador visible.
  for (let i = 0; i < SEGMENTOS; i++) {
    const a0 = ((i * SALTO - 90) * Math.PI) / 180
    const a1 = (((i + 1) * SALTO - 90) * Math.PI) / 180
    ctx.beginPath()
    ctx.moveTo(centro, centro)
    ctx.arc(centro, centro, radio, a0, a1)
    ctx.closePath()
    ctx.fillStyle = colorValor(TABLA[i])
    ctx.fill()
    ctx.strokeStyle = 'rgba(0,0,0,.55)'
    ctx.lineWidth = Math.max(1, size / 200)
    ctx.stroke()

    ctx.save()
    ctx.translate(centro, centro)
    ctx.rotate((a0 + a1) / 2)
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    // Contorno oscuro detras del valor para que se lea sobre cualquier color.
    ctx.font = `bold ${Math.max(10, Math.round(size / 10))}px ui-sans-serif, system-ui`
    ctx.lineWidth = Math.max(2, size / 80)
    ctx.strokeStyle = 'rgba(0,0,0,.75)'
    ctx.strokeText(`x${TABLA[i]}`, -radio * 0.62, 0)
    ctx.fillStyle = TABLA[i] >= 2 ? '#1a1408' : '#f3f5f7'
    ctx.fillText(`x${TABLA[i]}`, -radio * 0.62, 0)
    ctx.restore()
  }

  // Filete dorado que separa el disco del cubo central.
  ctx.beginPath()
  ctx.arc(centro, centro, radio * 0.8, 0, Math.PI * 2)
  ctx.strokeStyle = 'rgba(245,185,66,.5)'
  ctx.lineWidth = Math.max(1, size / 130)
  ctx.stroke()
}

onMounted(() => {
  dibujar()
  window.addEventListener('resize', dibujar)
})
onUnmounted(() => window.removeEventListener('resize', dibujar))

watch(
  () => props.resultado,
  (r) => {
    if (!r) return
    posicion.value = r.posicion
    rotando.value = true
    reproducir('giro')
    // Cuatro vueltas y el puntero arriba; la cuenta cruza multiplos de 360
    // para que la transicion siempre avance hacia adelante.
    const centro = (r.posicion + 0.5) * SALTO
    const base = 360 - centro
    const actual = rotacion.value % 360
    let destino = base + 360 * 4
    while (destino < actual) destino += 360
    rotacion.value = destino
    window.setTimeout(() => {
      rotando.value = false
      if (props.premio > 0) reproducir('ganar')
      else reproducir('perder')
    }, 4200)
  },
)

function girar() {
  emit('jugar', {})
}

// Posicion de cada lamparita en el aro, en porcentaje del contenedor (sirve a
// cualquier tamano de pantalla sin depender del font-size).
function bulboPos(i: number): { x: number; y: number } {
  const ang = ((i - 1) * (360 / BULBOS) - 90) * (Math.PI / 180)
  return { x: 50 + 46.5 * Math.cos(ang), y: 50 + 46.5 * Math.sin(ang) }
}
</script>

<template>
  <div class="fortuna">
    <div class="tablero">
      <!-- Aro de lamparitas: fijas, en cascada; quedan prendidas si hay premio. -->
      <div class="aro" :class="{ premio: resultado && !rotando && premio > 0 }">
        <span
          v-for="i in BULBOS"
          :key="i"
          class="bulbo"
          :style="{
            left: bulboPos(i).x + '%',
            top: bulboPos(i).y + '%',
            animationDelay: '0ms',
          }"
        />
      </div>

      <div class="punta" :class="{ 'premio-pulso': resultado && !rotando && premio > 0 }"><span class="flecha">▾</span></div>

      <canvas ref="canvas" class="disco" :style="{ transform: `rotate(${rotacion}deg)` }" />

      <div class="centro" :class="{ 'resalta-ganador': resultado && !rotando && premio > 0 }">
        <span v-if="resultado && !rotando" class="n" :class="{ gana: premio > 0 }">
          {{ posicion != null ? 'x' + TABLA[posicion] : '?' }}
        </span>
        <span v-else class="n">?</span>
      </div>
    </div>

    <div class="marcador" :class="{ gana: resultado && !rotando && premio > 0 }">
      <span class="etq">Premio</span>
      <strong v-if="resultado && !rotando">{{ premio > 0 ? colones(premio) : 'Sin premio' }}</strong>
      <strong v-else>—</strong>
    </div>

    <div class="leyenda">
      <span v-for="(v, i) in [5, 3, 2, 1, 0]" :key="i" class="item">
        <i :style="{ background: colorValor(v) }" />x{{ v }}
      </span>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando || rotando" @click="girar">
      {{ cargando || rotando ? 'Girando…' : 'Girar la ruleta' }}
    </button>
  </div>
</template>

<style scoped>
.fortuna { display: flex; flex-direction: column; gap: 14px; align-items: center; }

.tablero {
  position: relative;
  display: grid;
  place-items: center;
  width: min(250px, 70vw);
  aspect-ratio: 1 / 1;
  margin: 0 auto;
}

/* Aro del programa con lamparitas. */
.aro {
  position: absolute;
  inset: 3.5%;
  border-radius: 50%;
  border: .45em solid #2a2113;
  outline: 2px solid rgba(245, 185, 66, .35);
  outline-offset: -0.45em;
  pointer-events: none;
  filter: drop-shadow(0 8px 18px rgba(0, 0, 0, .45));
}
.bulbo {
  position: absolute;
  width: .55em; height: .55em;
  margin: -.275em 0 0 -.275em;
  border-radius: 50%;
  background: #5c4a1c;
  box-shadow: inset 0 0 0 2px rgba(0, 0, 0, .45);
  animation: bulbo 1.1s ease-in-out infinite;
}
.aro.premio .bulbo {
  background: #ffe9a8;
  box-shadow: 0 0 10px 2px rgba(245, 185, 66, .85);
  animation: none;
}
@keyframes bulbo {
  0%, 100% { background: #5c4a1c; }
  50% { background: #ffd873; box-shadow: 0 0 12px 2px rgba(245, 185, 66, .8); }
}

.disco {
  position: absolute;
  inset: 9%;
  width: 82%;
  height: 82%;
  border-radius: 50%;
  transition: transform 4.2s cubic-bezier(.12, .68, .16, 1);
  will-change: transform;
}

/* Puntero grande y metalico: lo primero que se mira. */
.punta {
  position: absolute;
  top: -16px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 3;
  display: grid;
  place-items: center;
  width: 46px;
  height: 46px;
  border-radius: 50%;
  background: radial-gradient(circle at 30% 30%, #fff7dd, #c9a227 60%, #7a5c17);
  border: 2px solid #f5d67b;
  box-shadow: 0 4px 12px rgba(0, 0, 0, .55);
}
.punta .flecha {
  font-size: 20px;
  line-height: 1;
  color: #2b1d05;
  transform: translateY(2px);
  text-shadow: 0 1px 0 rgba(255, 255, 255, .5);
}
.premio-pulso { animation: pulso 1s ease-in-out infinite alternate; }
@keyframes pulso {
  from { transform: translateX(-50%) scale(1); }
  to { transform: translateX(-50%) scale(1.1); }
}
/* El cubo central vive en el flujo del grid del tablero: queda centrado sin
   transform, asi la animacion .resalta-ganador no lo desplaza. */
.centro {
  position: relative;
  width: 28%;
  aspect-ratio: 1 / 1;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: radial-gradient(circle at 35% 30%, #20332a, #08110d 70%);
  border: 2px solid rgba(245, 185, 66, .5);
  box-shadow: inset 0 3px 10px rgba(0, 0, 0, .7), 0 4px 14px rgba(0, 0, 0, .4);
}
.centro .n {
  font-family: var(--display), ui-sans-serif, system-ui;
  font-size: clamp(18px, 6.4vw, 28px);
  font-weight: 800;
  color: var(--muted);
}
.centro .n.gana {
  color: var(--gold);
  text-shadow: 0 0 18px rgba(245, 185, 66, .75);
}

/* Marcador tipo panel de programa, sin tapar el area de juego. */
.marcador {
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: 10px;
  min-width: 190px;
  padding: 8px 18px;
  border-radius: 12px;
  border: 1px solid var(--border-soft);
  background: linear-gradient(180deg, rgba(245, 185, 66, .07), rgba(0, 0, 0, .35));
}
.marcador .etq {
  font-size: 11px;
  letter-spacing: .18em;
  text-transform: uppercase;
  color: var(--faint);
}
.marcador strong {
  font-family: var(--display), ui-sans-serif, system-ui;
  font-size: 20px;
  color: var(--text);
}
.marcador.gana {
  border-color: rgba(245, 185, 66, .6);
  box-shadow: 0 0 22px rgba(245, 185, 66, .25);
}
.marcador.gana strong { color: var(--gold); }

.leyenda { display: flex; justify-content: center; gap: 14px; flex-wrap: wrap; }
.leyenda .item { display: inline-flex; align-items: center; gap: 5px; font-size: 12px; color: var(--muted); }
.leyenda i { width: 10px; height: 10px; border-radius: 3px; display: inline-block; }
</style>
