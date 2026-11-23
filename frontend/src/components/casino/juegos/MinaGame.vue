<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
  codigo?: string
  multiplicador?: number
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

// Cada nivel es una grilla mas larga: misma mecanica, mas filas y mas premio.
const NIVELES = [
  { id: 0, nombre: 'Suave', filas: 4, techo: 8 },
  { id: 1, nombre: 'Media', filas: 5, techo: 10 },
  { id: 2, nombre: 'Fuerte', filas: 6, techo: 18 },
]
const nivel = ref(1)

const filas = computed<number>(() => props.resultado?.filas ?? NIVELES[nivel.value].filas)
const columnas = computed<number>(() => props.resultado?.columnas ?? 5)
const camino = computed<number[]>(() => props.resultado?.camino ?? [])
const minas = computed<number[][]>(() => props.resultado?.minas ?? [])
const pasos = computed<number>(() => props.resultado?.pasos ?? 0)
const tabla = computed<number[]>(() => props.resultado?.tabla ?? [])

// Cantidad de filas ya destapadas en la animacion (0 = ninguna).
const destapadas = ref(0)
const animando = ref(false)
const boom = ref(false)

function celda(f: number, c: number): { tipo: 'oro' | 'mina' | 'roca' | 'vacia'; camino: boolean } {
  if (f >= destapadas.value) return { tipo: 'roca', camino: false }
  const esCamino = camino.value[f] === c
  if (f < pasos.value) {
    return { tipo: esCamino ? 'oro' : 'roca', camino: esCamino }
  }
  // la fila donde piso la mina: se revela el camino y las minas de la fila
  const esMina = minas.value[f]?.includes(c) ?? false
  return { tipo: esMina ? 'mina' : esCamino ? 'roca' : 'vacia', camino: esCamino }
}

const multActual = computed(() => {
  if (!props.resultado) return 0
  const k = Math.min(destapadas.value, pasos.value)
  return tabla.value[k] ?? 0
})

function animar() {
  const total = filas.value
  destapadas.value = 0
  animando.value = true
  boom.value = false
  reproducir('caida')
  let f = 0
  const timer = setInterval(() => {
    destapadas.value = f + 1
    if (f >= pasos.value) {
      // piso la mina en esta fila
      clearInterval(timer)
      boom.value = true
      animando.value = false
      if (props.premio > 0) reproducir('ganar')
      else reproducir('perder')
      return
    }
    reproducir('tecla')
    f++
    if (f >= total) {
      clearInterval(timer)
      animando.value = false
      reproducir('ganar')
    }
  }, 280)
}

watch(
  () => props.resultado,
  async (r) => {
    if (!r) return
    await nextTick()
    animar()
  },
)

function jugar() {
  emit('jugar', { dificultad: nivel.value })
}
</script>

<template>
  <div class="mina">
    <div class="niveles">
      <button
        v-for="n in NIVELES"
        :key="n.id"
        class="chip"
        :class="{ activa: nivel === n.id }"
        :disabled="animando"
        @click="nivel = n.id; reproducir('tecla')"
      >{{ n.nombre }} <small>x{{ n.techo }}</small></button>
    </div>

    <div class="mina-tablero" :class="{ boom }">
      <div v-for="f in filas" :key="f" class="fila">
        <div
          v-for="c in columnas"
          :key="c"
          class="celda"
          :class="[celda(f - 1, c - 1).tipo, { camino: celda(f - 1, c - 1).camino && f - 1 < pasos }]"
        >
          <template v-if="celda(f - 1, c - 1).tipo === 'oro'">◆</template>
          <template v-else-if="celda(f - 1, c - 1).tipo === 'mina'">✸</template>
        </div>
      </div>

      <div v-if="resultado && !animando" class="resuelto" :class="{ gana: premio > 0 }">
        <span class="n">x{{ multiplicador ?? 0 }}</span>
        <span v-if="premio > 0" class="p">{{ colones(premio) }}</span>
        <span v-else class="p cero">Pisaste la mina</span>
      </div>
    </div>

    <div class="progreso">
      <span class="etq">Filas superadas</span>
      <b>{{ resultado ? Math.min(destapadas, pasos) : 0 }}</b>
      <span v-if="resultado" class="mult">x{{ multActual }}</span>
    </div>

    <p class="regla">
      Tres minas por fila. Cada fila que superás sube el multiplicador; si pisás una, perdés todo.
    </p>

    <button class="btn btn-primary btn-block" :disabled="cargando || animando" @click="jugar">
      {{ cargando || animando ? 'Abriendo la mina…' : 'Entrar a la mina' }}
    </button>
  </div>
</template>

<style scoped>
.mina { display: flex; flex-direction: column; gap: 12px; }
.niveles { display: flex; gap: 6px; justify-content: center; flex-wrap: wrap; }
.chip {
  padding: 6px 12px; border-radius: 999px; border: 1px solid var(--border-soft);
  background: var(--surface); color: var(--muted); cursor: pointer; font-size: 12.5px;
}
.chip small { opacity: .7; margin-left: 4px; }
.chip.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
.chip:disabled { opacity: .6; cursor: default; }

.mina-tablero {
  position: relative; padding: 12px 12px 52px; border-radius: 16px;
  border: 1px solid var(--border-soft);
  background: linear-gradient(160deg, #0d1a13, #08110d 70%);
  box-shadow: inset 0 14px 30px rgba(0, 0, 0, .45);
  display: flex; flex-direction: column; gap: 8px;
}
.mina-tablero.boom { animation: sacudir .4s ease; }
@keyframes sacudir {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-5px); }
  50% { transform: translateX(5px); }
  75% { transform: translateX(-3px); }
}

.fila { display: grid; grid-template-columns: repeat(5, 1fr); gap: 8px; }
.celda {
  height: 44px; border-radius: 8px; display: grid; place-items: center;
  font-size: 19px; color: var(--gold);
  background: linear-gradient(160deg, #2a2f36, #171b21);
  border: 1px solid #343b44;
  box-shadow: inset 0 3px 8px rgba(0, 0, 0, .5);
}
.celda.oro {
  background: linear-gradient(160deg, #f2d684, #c9a227);
  border-color: #e8ce7a; color: #3a2a05;
  box-shadow: 0 0 18px rgba(245, 185, 66, .4);
}
.celda.camino { outline: 2px solid rgba(16, 185, 129, .5); }
.celda.mina {
  background: radial-gradient(circle at 50% 40%, #5b1f1f, #1a0808 75%);
  border-color: #ef4444; color: #ff9b9b;
  box-shadow: 0 0 20px rgba(239, 68, 68, .45);
}
.celda.vacia { background: transparent; border-style: dashed; }

/* El resultado es una franja al pie del tablero, no una capa sobre las celdas:
   antes las tapaba y parecia texto solapado. */
.resuelto {
  position: absolute; left: 8px; right: 8px; bottom: 8px; z-index: 2;
  display: flex; align-items: center; justify-content: center; gap: 10px;
  padding: 7px 12px; border-radius: 10px;
  background: rgba(4, 8, 12, .92); border: 1px solid var(--border-soft);
  text-align: center;
}
.resuelto .n { font-family: var(--display); font-size: 24px; font-weight: 700; line-height: 1; }
.resuelto .p { color: var(--gold); font-weight: 700; font-size: 14px; }
.resuelto .p.cero { color: var(--muted); font-weight: 500; font-size: 13px; }
.resuelto.gana { border-color: rgba(245, 185, 66, .6); }

.progreso { display: flex; gap: 10px; align-items: center; justify-content: center; font-size: 12px; }
.etq { color: var(--faint); letter-spacing: .1em; text-transform: uppercase; font-size: 11px; }
.progreso b { color: var(--text); }
.progreso .mult { color: var(--gold); font-weight: 700; }
.regla { margin: 0; text-align: center; font-size: 11.5px; color: var(--faint); line-height: 1.35; }
</style>
