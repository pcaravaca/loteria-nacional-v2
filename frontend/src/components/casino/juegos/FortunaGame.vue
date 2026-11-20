<script setup lang="ts">
import { computed, ref, watch } from 'vue'
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

// Colores por valor (de mas comun a mas raro).
const colorValor = (v: number) =>
  v === 0 ? '#1a2534' : v === 1 ? '#0f6b45' : v === 2 ? '#0aa37a' : v === 3 ? '#b98a1c' : '#f5b942'

const conico = computed(() => {
  const partes: string[] = []
  const salto = 360 / SEGMENTOS
  TABLA.forEach((v, i) => {
    partes.push(`${colorValor(v)} ${i * salto}deg ${(i + 1) * salto}deg`)
  })
  return `conic-gradient(${partes.join(', ')})`
})

const posicion = ref<number | null>(null)
const rotando = ref(false)
const rotacion = ref(0)

watch(
  () => props.resultado,
  (r) => {
    if (!r) return
    posicion.value = r.posicion
    rotando.value = true
    reproducir('giro')
    // 3 vueltas completas + caer en el segmento ganador (puntoeiro arriba).
    const salto = 360 / SEGMENTOS
    const centro = (r.posicion + 0.5) * salto
    rotacion.value = 360 * 3 + (360 - centro)
    setTimeout(() => (rotando.value = false), 2200)
  },
)

function girar() {
  emit('jugar', {})
}
</script>

<template>
  <div class="fortuna">
    <div class="tablero">
      <div class="punta">▼</div>
      <div
        class="rueda"
        :class="{ girando: rotando }"
        :style="{ background: conico, transform: `rotate(${rotacion}deg)` }"
      />
      <div class="centro" :class="{ 'resalta-ganador': premio > 0 }">
        <span v-if="resultado && !rotando" class="n" :class="{ gana: premio > 0 }">
          x{{ TABLA[posicion ?? 0] }}
        </span>
        <span v-else class="n">?</span>
      </div>
      <div v-if="resultado && !rotando" class="resuelto" :class="{ gana: premio > 0 }">
        <span v-if="premio > 0" class="p">{{ colones(premio) }}</span>
        <span v-else class="p cero">Sin premio</span>
      </div>
    </div>

    <div class="leyenda">
      <span v-for="(v, i) in [5, 3, 2, 1, 0]" :key="i" class="item">
        <i :style="{ background: colorValor(v) }" />x{{ v }}
      </span>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando" @click="girar">
      {{ cargando ? 'Girando…' : 'Girar la ruleta' }}
    </button>
  </div>
</template>

<style scoped>
.fortuna { display: flex; flex-direction: column; gap: 14px; }
.tablero {
  position: relative; display: flex; align-items: center; justify-content: center;
  min-height: 220px; padding: 18px;
  border: 1px solid var(--border-soft); border-radius: 14px;
  background: linear-gradient(180deg, rgba(245, 185, 66, 0.05), transparent 60%);
}
.punta {
  position: absolute; top: 6px; left: 50%; transform: translateX(-50%);
  color: var(--gold); font-size: 22px; z-index: 2;
}
.rueda {
  width: min(190px, 56vw); aspect-ratio: 1 / 1; height: auto; border-radius: 50%;
  border: 4px solid var(--border);
  transition: transform 2.2s cubic-bezier(.15,.85,.25,1);
  position: relative;
}
.centro {
  position: absolute; width: 72px; height: 72px; border-radius: 50%;
  display: grid; place-items: center;
  background: var(--bg-2); border: 1px solid var(--border);
}
.centro .n { font-family: var(--display); font-size: 22px; font-weight: 700; }
.centro .n.gana { color: var(--prim-2); }
.resuelto {
  position: absolute; right: 14px; bottom: 14px;
  padding: 8px 14px; border-radius: 10px;
  background: rgba(8, 11, 16, 0.9); border: 1px solid var(--border);
}
.resuelto .p { color: var(--gold); font-weight: 700; }
.resuelto .p.cero { color: var(--muted); font-weight: 500; }
.resuelto.gana { border-color: rgba(245, 185, 66, 0.5); }
.leyenda { display: flex; justify-content: center; gap: 14px; }
.leyenda .item { display: inline-flex; align-items: center; gap: 5px; font-size: 12px; color: var(--muted); }
.leyenda i { width: 10px; height: 10px; border-radius: 3px; display: inline-block; }
</style>