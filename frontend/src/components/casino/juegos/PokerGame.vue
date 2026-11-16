<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { colones } from '@/utils/formato'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
  rondaKey: number
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

const palos = ['♠', '♥', '♦', '♣']
const etiquetas: Record<string, string> = {
  RoyalFlush: 'Escalera real',
  StraightFlush: 'Escalera de color',
  FiveOfAKind: 'Quintilla',
  FourOfAKind: 'Poker',
  FullHouse: 'Full',
  Flush: 'Color',
  Straight: 'Escalera',
  ThreeOfAKind: 'Tercia',
  TwoPair: 'Doble par',
  Pair: 'Par',
}

const cartas = ref<string[]>([])
const retenidas = ref<boolean[]>([false, false, false, false, false])
const fase = ref<'inicio' | 'final'>('inicio')
let timer: number | undefined

function etiqueta(clave: string): { rango: string; palo: string; roja: boolean; comodin: boolean } {
  if (clave === 'comodin') return { rango: 'Joker', palo: '🃏', roja: false, comodin: true }
  const [r, p] = clave.split('-').map(Number)
  const rangos: Record<number, string> = { 11: 'J', 12: 'Q', 13: 'K', 14: 'A' }
  return {
    rango: rangos[r] ?? String(r),
    palo: palos[p] ?? '',
    roja: p === 1 || p === 2,
    comodin: false,
  }
}

watch(
  () => props.rondaKey,
  (k) => {
    if (k === 0) return
    const r = props.resultado
    if (!r?.iniciales) return
    fase.value = 'inicio'
    cartas.value = [...r.iniciales]
    retenidas.value = [false, false, false, false, false]
    window.clearTimeout(timer)
    timer = window.setTimeout(() => {
      fase.value = 'final'
      cartas.value = [...(r.cartas ?? r.iniciales)]
      retenidas.value = Array.isArray(r.retencion) ? [...r.retencion] : retenidas.value
    }, 1100)
  },
)

onUnmounted(() => window.clearTimeout(timer))

function jugar() {
  emit('jugar', {})
}
</script>

<template>
  <div class="poker">
    <div class="mesa">
      <div
        v-for="(c, i) in cartas"
        :key="i"
        class="carta"
        :class="{ roja: etiqueta(c).roja, retenida: fase === 'final' && retenidas[i], comodin: etiqueta(c).comodin }"
      >
        <template v-if="fase === 'final'">
          <span class="rango">{{ etiqueta(c).rango }}</span>
          <span class="palo">{{ etiqueta(c).palo }}</span>
        </template>
        <span v-else class="dorso">♦</span>
        <span v-if="fase === 'final' && retenidas[i]" class="marca">SOSTENIDA</span>
      </div>
    </div>

    <div v-if="resultado" class="estado" :class="{ gana: premio > 0 }">
      <template v-if="premio > 0">
        <span>{{ etiquetas[resultado.mano] ?? resultado.mano }}</span>
        <strong>x{{ resultado.multipl }}</strong>
        <span class="premio">{{ colones(premio) }}</span>
      </template>
      <span v-else class="neutro">Sin premio</span>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando" @click="jugar">
      {{ cargando ? 'Repartiendo…' : 'Jugar' }}
    </button>
  </div>
</template>

<style scoped>
.poker { display: flex; flex-direction: column; gap: 12px; }
.mesa {
  display: flex; gap: 8px; padding: 14px; border-radius: 16px;
  background: radial-gradient(circle at 50% 0%, #14382a, #07120d 70%);
  border: 1px solid var(--border);
}
.carta {
  position: relative; flex: 1; height: 120px; border-radius: 10px;
  background: linear-gradient(160deg, #f8f6ee, #d9d4c3);
  color: #10161c; display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 2px; font-weight: 700; box-shadow: 0 6px 14px rgba(0, 0, 0, .35);
  transition: transform .2s, box-shadow .2s;
}
.carta.roja { color: #b3202c; }
.carta.comodin { background: linear-gradient(160deg, #2a1e3f, #120a1c); color: #e8d9a8; }
.carta.retenida { transform: translateY(-8px); box-shadow: 0 0 0 2px var(--gold), 0 10px 18px rgba(0, 0, 0, .4); }
.carta .rango { font-size: 30px; line-height: 1; }
.carta .palo { font-size: 20px; line-height: 1; }
.carta .dorso { font-size: 34px; color: #2a4d3b; }
.carta .marca {
  position: absolute; bottom: -8px; left: 50%; transform: translateX(-50%);
  background: var(--gold); color: #1a1408; font-size: 8.5px; letter-spacing: .08em;
  padding: 2px 5px; border-radius: 5px; white-space: nowrap;
}
.estado {
  display: flex; align-items: center; justify-content: center; gap: 10px;
  background: rgba(255, 255, 255, .02); border: 1px solid var(--border-soft);
  border-radius: 10px; padding: 10px; color: var(--muted);
}
.estado.gana { background: var(--prim-dim); border-color: rgba(16, 185, 129, .35); color: var(--prim-2); }
.estado strong { color: var(--gold); }
.estado .premio { font-weight: 700; }
.estado .neutro { color: var(--muted); }
</style>
