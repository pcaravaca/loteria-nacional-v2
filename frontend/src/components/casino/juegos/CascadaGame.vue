<script setup lang="ts">
import { ref, watch } from 'vue'
import { colones } from '@/utils/formato'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

// La tabla del servidor es la fuente de verdad; esta copia dibuja el tablero.
const TABLA = [10, 5, 3, 1, 0, 0, 1, 3, 5, 10]
const FILAS = 9

const animando = ref(false)
const paso = ref(FILAS)
const xPct = ref(50)
const posicion = ref<number | null>(null)

watch(
  () => props.resultado,
  (r) => {
    if (!r) return
    posicion.value = r.posicion
    animar()
  },
)

function animar() {
  animando.value = true
  paso.value = 0
  xPct.value = 50
  const destino = ((posicion.value ?? 4.5) + 0.5) * 10 // 5%..95%
  let k = 0
  const timer = setInterval(() => {
    k++
    const progreso = k / FILAS
    const rebote = (k % 2 === 0 ? 1 : -1) * 9 * (1 - progreso)
    xPct.value = 50 + (destino - 50) * progreso + rebote * 0.5
    paso.value = k
    if (k >= FILAS) {
      clearInterval(timer)
      xPct.value = destino
      animando.value = false
    }
  }, 150)
}

function soltar() {
  emit('jugar', {})
}
</script>

<template>
  <div class="cascada">
    <div class="tablero">
      <div class="filas">
        <div v-for="f in FILAS" :key="f" class="fila">
          <span v-for="p in f + 1" :key="p" class="pin" />
        </div>
      </div>
      <div
        v-if="resultado"
        class="bola"
        :style="{ top: (paso / FILAS) * 88 + 4 + '%', left: xPct + '%' }"
      />
      <div class="casillas">
        <div
          v-for="(m, i) in TABLA"
          :key="i"
          class="casilla"
          :class="{
            ganadora: resultado && !animando && posicion === i && m > 0,
            cero: m === 0,
          }"
        >
          {{ m }}x
        </div>
      </div>
      <div v-if="resultado && !animando" class="resuelto" :class="{ gana: premio > 0 }">
        <span class="n">x{{ resultado.tabla[resultado.posicion] }}</span>
        <span v-if="premio > 0" class="p">{{ colones(premio) }}</span>
        <span v-else class="p cero">Sin premio</span>
      </div>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando" @click="soltar">
      {{ cargando ? 'Soltando…' : 'Soltar bola' }}
    </button>
  </div>
</template>

<style scoped>
.cascada { display: flex; flex-direction: column; gap: 14px; }
.tablero {
  position: relative; height: min(340px, 58vh); overflow: hidden;
  border: 1px solid var(--border-soft); border-radius: 14px;
  background: linear-gradient(180deg, rgba(16, 185, 129, 0.06), transparent 60%);
}
.filas { position: absolute; inset: 0 0 46px 0; display: flex; flex-direction: column; justify-content: space-evenly; padding: 8px 16px; }
.fila { display: flex; justify-content: space-between; }
.pin { width: 5px; height: 5px; border-radius: 50%; background: rgba(142, 162, 182, 0.45); }
.bola {
  position: absolute; width: 16px; height: 16px; border-radius: 50%;
  transform: translate(-50%, -50%);
  background: radial-gradient(circle at 30% 30%, #fff7dd, #f5b942 55%, #a8740a);
  box-shadow: 0 0 16px rgba(245, 185, 66, 0.75);
  transition: left 0.14s linear, top 0.14s linear;
}
.casillas {
  position: absolute; left: 0; right: 0; bottom: 0;
  display: grid; grid-template-columns: repeat(10, 1fr); gap: 4px; padding: 6px;
}
.casilla {
  padding: 7px 0; text-align: center; font-size: 12px; font-weight: 700;
  border-radius: 8px; background: var(--surface); border: 1px solid var(--border-soft);
  color: var(--muted);
}
.casilla.ganadora {
  background: linear-gradient(135deg, var(--prim), #0aa37a);
  color: #04140e; border-color: var(--prim);
}
.casilla.cero { color: var(--faint); }
.resuelto {
  position: absolute; top: 42%; left: 50%; transform: translateX(-50%);
  display: flex; flex-direction: column; align-items: center; gap: 2px;
  padding: 10px 20px; border-radius: 12px;
  background: rgba(8, 11, 16, 0.88); border: 1px solid var(--border);
}
.resuelto .n { font-family: var(--display); font-size: 26px; font-weight: 700; }
.resuelto .p { color: var(--gold); font-weight: 700; }
.resuelto .p.cero { color: var(--muted); font-weight: 500; }
.resuelto.gana { border-color: rgba(245, 185, 66, 0.5); }
</style>