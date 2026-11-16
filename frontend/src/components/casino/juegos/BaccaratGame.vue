<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { colones } from '@/utils/formato'

const props = defineProps<{
  resultado: any
  premio: number
  monto: number
  cargando: boolean
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

type Tipo = 'jugador' | 'banca' | 'empate'

const tipo = ref<Tipo>('jugador')
const PALOS = ['♠', '♥', '♦', '♣']
const opciones: { t: Tipo; etiqueta: string; nota: string }[] = [
  { t: 'jugador', etiqueta: 'Jugador x2', nota: 'Paga 1 a 1' },
  { t: 'banca', etiqueta: 'Banca x2', nota: '5% de comisión' },
  { t: 'empate', etiqueta: 'Empate x8', nota: 'Paga 8 a 1' },
]

// Las figuras y los diez valen 0 en baccarat; el as vale 1.
function figura(v: number): string {
  if (v === 0) return '10'
  if (v === 1) return 'A'
  return String(v)
}

function palo(indice: number): string {
  return PALOS[indice % PALOS.length]
}

function colorDe(indice: number): string {
  return indice % 2 === 1 ? '#b3242f' : '#14181f'
}

const limpiar = ref(0)
watch(
  () => props.resultado,
  async () => {
    if (!props.resultado) return
    await nextTick()
    limpiar.value++
  },
)

function jugar() {
  emit('jugar', { tipo: tipo.value })
}
</script>

<template>
  <div class="baccarat">
    <div class="manos">
      <div class="mano">
        <span class="etiqueta">Jugador</span>
        <div class="cartas" :key="'j' + limpiar">
          <span
            v-for="(c, i) in (resultado?.cartas_jugador ?? [])"
            :key="i"
            class="carta"
            :style="{ color: colorDe(i) }"
          >
            <b>{{ figura(c) }}</b>
            <i>{{ palo(c + i) }}</i>
          </span>
          <span v-if="!resultado" class="carta vacia">·</span>
        </div>
        <strong class="total">{{ resultado?.total_jugador ?? '—' }}</strong>
      </div>

      <div class="vs">VS</div>

      <div class="mano">
        <span class="etiqueta">Banca</span>
        <div class="cartas" :key="'b' + limpiar">
          <span
            v-for="(c, i) in (resultado?.cartas_banca ?? [])"
            :key="i"
            class="carta"
            :style="{ color: colorDe(i + 1) }"
          >
            <b>{{ figura(c) }}</b>
            <i>{{ palo(c + i + 2) }}</i>
          </span>
          <span v-if="!resultado" class="carta vacia">·</span>
        </div>
        <strong class="total">{{ resultado?.total_banca ?? '—' }}</strong>
      </div>
    </div>

    <div v-if="resultado" class="resumen" :class="{ gana: premio > monto }">
      <span>{{ resultado.resultado }}</span>
      <strong v-if="premio > monto">{{ colones(premio) }}</strong>
      <span v-else-if="premio === monto">Apuesta devuelta</span>
      <span v-else>Sin acierto</span>
    </div>

    <div class="modo">
      <button
        v-for="o in opciones"
        :key="o.t"
        class="chip"
        :class="{ activa: tipo === o.t }"
        @click="tipo = o.t"
      >
        {{ o.etiqueta }}
        <small>{{ o.nota }}</small>
      </button>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando" @click="jugar">
      {{ cargando ? 'Repartiendo…' : 'Repartir cartas' }}
    </button>
  </div>
</template>

<style scoped>
.baccarat { display: flex; flex-direction: column; gap: 14px; }
.manos { display: grid; grid-template-columns: 1fr auto 1fr; gap: 10px; align-items: stretch; }
.mano { background: #05080c; border: 1px solid var(--border); border-radius: 12px; padding: 12px; display: flex; flex-direction: column; gap: 8px; align-items: center; }
.etiqueta { font-size: 10px; letter-spacing: .12em; text-transform: uppercase; color: var(--muted); }
.cartas { display: flex; gap: 6px; min-height: 62px; align-items: center; }
.carta {
  width: 42px; height: 58px; border-radius: 7px; background: #f4f1e8; display: flex; flex-direction: column;
  align-items: center; justify-content: center; gap: 2px; font-size: 17px; font-weight: 800;
  box-shadow: 0 6px 14px rgba(0, 0, 0, .45);
  animation: aparición .3s ease both;
}
.carta i { font-size: 11px; font-style: normal; }
.carta.vacia { background: transparent; border: 1px dashed var(--border); box-shadow: none; color: var(--faint); }
.total { font-size: 20px; color: var(--gold); }
.vs { align-self: center; font-size: 11px; letter-spacing: .1em; color: var(--faint); }
.resumen { display: flex; align-items: center; justify-content: center; gap: 10px; font-size: 13px; color: var(--muted); }
.resumen strong { color: var(--gold); font-size: 16px; }
.resumen.gana { color: var(--prim-2); }
.modo { display: flex; gap: 8px; }
.chip {
  flex: 1; display: flex; flex-direction: column; gap: 2px; padding: 8px 10px; border-radius: 10px;
  border: 1px solid var(--border-soft); background: var(--surface); color: var(--text);
  cursor: pointer; font-size: 12.5px; font-weight: 700;
}
.chip small { font-weight: 400; color: var(--faint); font-size: 10.5px; }
.chip.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
@keyframes aparición { from { opacity: 0; transform: translateY(-8px) rotate(-6deg); } to { opacity: 1; transform: none; } }
</style>
