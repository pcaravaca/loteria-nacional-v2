<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { colones } from '@/utils/formato'
import { astSimbolo } from '@/assets/ast'
import { reproducir } from '@/utils/sfx'

const props = defineProps<{
  codigo: string
  resultado: any
  premio: number
  cargando: boolean
  rondaKey: number
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

const visibles = ref<string[][]>([])
const girando = ref(false)
const lineasGanadoras = ref<any[]>([])
const celdasGanadoras = ref<Set<string>>(new Set())
let timer: number | undefined

function dimensiones(): [number, number] {
  const r = props.resultado?.rodillos
  if (Array.isArray(r) && r.length) return [r.length, r[0].length]
  return [5, 3]
}

function emojiAlAzar() {
  return String(Math.floor(Math.random() * 10))
}

// Que se vean simbolos desde que se abre el juego, sin tener que girar.
function pintarInicial() {
  const [rodillos, filas] = dimensiones()
  visibles.value = Array.from({ length: rodillos }, () =>
    Array.from({ length: filas }, () => emojiAlAzar()),
  )
}
onMounted(pintarInicial)

// Vigilar (rondaKey, resultado) garantiza armar la animacion con el resultado
// de ESTA tirada: si el premio cambia antes que los rodillos, el proximo
// cambio del arreglo vuelve a disparar y recien ahi se fijan.
const ultimaRonda = ref(0)
watch(
  () => [props.rondaKey, props.resultado],
  ([k, r]: [number, any]) => {
    if (!k || k === ultimaRonda.value) return
    if (!r?.rodillos) return
    ultimaRonda.value = k
    fijarRodillos(r.rodillos)
  },
  { flush: 'post', immediate: true },
)

function fijarRodillos(rodillos: string[][]) {
  girando.value = true
  reproducir('rodillo')
  lineasGanadoras.value = []
  celdasGanadoras.value = new Set()
  const cols = Math.max(1, rodillos.length)
  const filas = Math.max(1, Array.isArray(rodillos[0]) ? rodillos[0].length : 3)
  visibles.value = Array.from({ length: cols }, () =>
    Array.from({ length: filas }, () => emojiAlAzar()),
  )
  const repeticiones = [700, 1050, 1400, 1750, 2100]
  const ultimo = Math.max(2, cols - 1)
  // Solo giran las columnas que todavia no se resolvieron: si el interval
  // regenerara toda la matriz pisaria las que ya quedaron fijas.
  const pendientes = new Set<number>(Array.from({ length: cols }, (_, i) => i))
  window.clearInterval(timer)
  timer = window.setInterval(() => {
    const nuevo = visibles.value.map((fila) => [...fila])
    for (const c of pendientes) {
      nuevo[c] = Array.from({ length: filas }, () => emojiAlAzar())
    }
    visibles.value = nuevo
  }, 90)
  repeticiones.slice(0, Math.max(3, cols)).forEach((ms, i) => {
    window.setTimeout(() => {
      const columna = rodillos[i]
      if (Array.isArray(columna)) {
        const nueva = visibles.value.map((fila) => [...fila])
        nueva[i] = columna
        visibles.value = nueva
        pendientes.delete(i)
      }
      if (i === ultimo) {
        window.clearInterval(timer)
        girando.value = false
        lineasGanadoras.value = props.resultado?.lineas ?? []
        const set = new Set<string>()
        for (const l of lineasGanadoras.value) {
          for (const par of l.celdas ?? []) set.add(`${par[0]}:${par[1]}`)
        }
        celdasGanadoras.value = set
      }
    }, ms)
  })
}

onUnmounted(() => window.clearInterval(timer))

function jugar() {
  emit('jugar', {})
}
</script>

<template>
  <div class="ast-slots">
    <div class="maquina" :class="{ girando }">
      <div class="luces-maq">★ ★ ★ ★ ★</div>
      <div v-for="(columna, c) in visibles" :key="c" class="rodillo">
        <div
          v-for="(s, f) in columna"
          :key="f"
          class="celda"
          :class="{ intensa: celdasGanadoras.has(`${c}:${f}`), 'resalta-ganador': celdasGanadoras.has(`${c}:${f}`) }"
        >
          <span>{{ astSimbolo(codigo, s) }}</span>
        </div>
      </div>
    </div>

    <div v-if="girando" class="estado"><span class="neutro">Girando…</span></div>
    <div v-else-if="resultado" class="estado" :class="{ gana: premio > 0 }">
      <template v-if="premio > 0">
        <span>Ganaste</span><strong>{{ colones(premio) }}</strong>
      </template>
      <span v-else class="neutro">Sin premio</span>
    </div>

    <div v-if="lineasGanadoras.length" class="ganadoras">
      <div v-for="(l, i) in lineasGanadoras" :key="i" class="linea-ganadora">
        <span>Linea {{ l.linea + 1 }}</span>
        <b>x{{ l.multipl }}</b>
      </div>
    </div>

    <div v-if="resultado?.tabla" class="paytable">
      <div v-for="(f, i) in resultado.tabla.slice(0, 8)" :key="i" class="fila">
        <span class="combo">
          <em v-for="n in f.repeticion" :key="n">{{ astSimbolo(codigo, f.simbolo) }}</em>
        </span>
        <span class="x">x{{ f.multipl }}</span>
      </div>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando || girando" @click="jugar">
      {{ cargando || girando ? 'Girando…' : 'Girar' }}
    </button>
  </div>
</template>

<style scoped>
.ast-slots { display: flex; flex-direction: column; gap: 12px; }
.maquina {
  display: flex; gap: 8px; padding: 10px; border-radius: 16px;
  background: linear-gradient(180deg, #122a20, #05080c 70%);
  border: 2px solid var(--gold);
  box-shadow: inset 0 14px 26px rgba(0,0,0,.5), 0 12px 26px rgba(0,0,0,.3);
}
.luces-maq {
  position: absolute; top: -9px; left: 50%; transform: translateX(-50%);
  background: var(--surface); border: 1px solid var(--gold); border-radius: 999px;
  padding: 0 10px; color: var(--gold); font-size: 10px; letter-spacing: .18em; white-space: nowrap;
}
.maquina { position: relative; }
.rodillo { flex: 1; display: flex; flex-direction: column; gap: 8px; }
.celda {
  height: 74px; border-radius: 12px; display: grid; place-items: center;
  background: rgba(255, 255, 255, .03); border: 1px solid var(--border-soft);
  font-size: 34px; transition: box-shadow .2s, border-color .2s;
}
.celda.intensa {
  border-color: var(--gold);
  box-shadow: 0 0 14px rgba(201, 162, 75, .35);
}
.maquina.girando .celda { animation: vaiven .18s linear infinite; }
@keyframes vaiven {
  0% { transform: translateY(-8px); filter: blur(1px); }
  50% { transform: translateY(8px); }
  100% { transform: translateY(-8px); filter: blur(1px); }
}
.estado {
  display: flex; align-items: center; justify-content: center; gap: 10px;
  background: rgba(255, 255, 255, .02); border: 1px solid var(--border-soft);
  border-radius: 10px; padding: 10px; color: var(--muted);
}
.estado.gana { background: var(--prim-dim); border-color: rgba(16, 185, 129, .35); color: var(--prim-2); }
.estado strong { font-size: 20px; color: var(--gold); }
.estado .neutro { color: var(--muted); }
.ganadoras { display: flex; flex-wrap: wrap; gap: 6px; }
.linea-ganadora {
  display: flex; align-items: center; gap: 6px; font-size: 11.5px; color: var(--muted);
  background: rgba(201, 162, 75, .1); border: 1px solid rgba(201, 162, 75, .35);
  border-radius: 8px; padding: 4px 8px;
}
.linea-ganadora b { color: var(--gold); }
.paytable { display: grid; grid-template-columns: repeat(4, 1fr); gap: 5px; font-size: 11px; color: var(--muted); }
.fila {
  display: flex; align-items: center; justify-content: space-between; gap: 6px;
  background: rgba(255, 255, 255, .02); border: 1px solid var(--border-soft);
  border-radius: 8px; padding: 4px 7px;
}
.combo { display: inline-flex; gap: 2px; font-style: normal; }
.combo em { font-style: normal; }
.x { color: var(--gold); font-weight: 700; }
</style>
