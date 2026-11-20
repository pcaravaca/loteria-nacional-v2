<script setup lang="ts">
import { ref, watch } from 'vue'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

// Speigo de multiplicadores: el servidor valida con la misma tabla.
const TOPES: { tope: number; mult: number }[] = [
  { tope: 2, mult: 48 },
  { tope: 5, mult: 19 },
  { tope: 10, mult: 9 },
  { tope: 25, mult: 3 },
  { tope: 48, mult: 2 },
  { tope: 75, mult: 1 },
  { tope: 90, mult: 1 },
  { tope: 95, mult: 1 },
]

const direccion = ref<'bajo' | 'sobre'>('bajo')
const tope = ref(48)
const tirando = ref(false)

watch(
  () => props.resultado,
  (r) => {
    if (!r) return
    tirando.value = true
    reproducir('giro')
    setTimeout(() => (tirando.value = false), 800)
  },
)

function jugar() {
  emit('jugar', { tope: tope.value, direccion: direccion.value })
}
</script>

<template>
  <div class="alto">
    <div class="tablero">
      <div class="dado" :class="{ girando: tirando }">{{ resultado?.tirada ?? '?' }}</div>
      <div class="detalle">
        <span class="dir">{{ resultado ? (resultado.direccion === 'bajo' ? 'Bajo' : 'Sobre') : 'Bajo' }} {{ tope }}%</span>
        <span class="mult">x{{ TOPES.find((t) => t.tope === tope)?.mult }}</span>
      </div>
      <div v-if="resultado && !tirando" class="resuelto" :class="{ gana: premio > 0 }">
        <span v-if="premio > 0" class="p">{{ colones(premio) }}</span>
        <span v-else class="p cero">Sin acierto</span>
      </div>
    </div>

    <div class="direcciones">
      <button class="btn-dir" :class="{ activa: direccion === 'bajo', 'resalta-ganador': premio > 0 && resultado?.direccion === 'bajo' }" @click="direccion = 'bajo'; reproducir('tecla')">
        Bajo
      </button>
      <button class="btn-dir" :class="{ activa: direccion === 'sobre', 'resalta-ganador': premio > 0 && resultado?.direccion === 'sobre' }" @click="direccion = 'sobre'; reproducir('tecla')">
        Sobre
      </button>
    </div>

    <div class="topes">
      <button
        v-for="t in TOPES"
        :key="t.tope"
        class="opcion"
        :class="{ activa: tope === t.tope }"
        @click="tope = t.tope; reproducir('tecla')"
      >
        {{ t.tope }}%
        <small>x{{ t.mult }}</small>
      </button>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando" @click="jugar">
      {{ cargando ? 'Tirando…' : 'Tirar dado' }}
    </button>
  </div>
</template>

<style scoped>
.alto { display: flex; flex-direction: column; gap: 14px; }
.tablero {
  position: relative; display: flex; align-items: center; justify-content: center;
  gap: 18px; min-height: 120px; padding: 18px;
  border: 1px solid var(--border-soft); border-radius: 14px;
  background: linear-gradient(180deg, rgba(16, 185, 129, 0.06), transparent 60%);
}
.dado {
  width: 74px; height: 74px; border-radius: 16px; display: grid; place-items: center;
  font-family: var(--display); font-size: 34px; font-weight: 700;
  background: linear-gradient(145deg, #f4f6f8, #c9d2dc); color: #15181d;
  box-shadow: 0 10px 22px rgba(0, 0, 0, 0.4);
}
.dado.girando { animation: sacudir .8s ease; }
@keyframes sacudir {
  0% { transform: rotate(0) translateY(0); }
  25% { transform: rotate(-16deg) translateY(-8px); }
  50% { transform: rotate(12deg) translateY(4px); }
  75% { transform: rotate(-6deg) translateY(-3px); }
  100% { transform: rotate(0) translateY(0); }
}
.detalle { display: flex; flex-direction: column; gap: 2px; }
.detalle .dir { font-size: 15px; font-weight: 600; }
.detalle .mult { font-size: 26px; font-weight: 800; color: var(--gold); }
.resuelto { position: absolute; right: 14px; top: 50%; transform: translateY(-50%); }
.resuelto .p { color: var(--gold); font-weight: 700; font-size: 17px; }
.resuelto .p.cero { color: var(--muted); font-weight: 500; }
.resuelto.gana .p { color: var(--prim-2); }
.direcciones { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.btn-dir {
  padding: 9px; border-radius: 10px; cursor: pointer; font-size: 14px; font-weight: 600;
  border: 1px solid var(--border-soft); background: var(--surface); color: var(--text);
}
.btn-dir.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
.topes { display: grid; grid-template-columns: repeat(4, 1fr); gap: 6px; }
.opcion {
  padding: 7px 4px; border-radius: 9px; border: 1px solid var(--border-soft);
  background: var(--surface); color: var(--muted); cursor: pointer; font-size: 13px;
  display: flex; flex-direction: column; align-items: center; gap: 1px;
}
.opcion small { font-size: 10px; color: var(--faint); }
.opcion.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
.opcion.activa small { color: var(--prim-2); }
</style>