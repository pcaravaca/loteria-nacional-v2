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

// Multiplicadores declarados: el servidor es la fuente de verdad.
const mult: Record<number, number> = {
  2: 27, 3: 14, 4: 9, 5: 7, 6: 6, 7: 5,
  8: 6, 9: 7, 10: 9, 11: 14, 12: 27,
}

const suma = ref<number>(7)
const girando = ref(false)

watch(
  () => props.resultado,
  (r) => {
    if (!r) return
    girando.value = true
    reproducir('giro')
    setTimeout(() => (girando.value = false), 700)
  },
)

const dados = computed(() => {
  if (!props.resultado) return [5, 2]
  return [props.resultado.dado1, props.resultado.dado2]
})

function tirar() {
  emit('jugar', { suma: suma.value })
}

const puntos: Record<number, number[]> = {
  1: [4], 2: [0, 8], 3: [0, 4, 8], 4: [0, 2, 6, 8],
  5: [0, 2, 4, 6, 8], 6: [0, 2, 3, 5, 6, 8],
}
</script>

<template>
  <div class="dados">
    <div class="tablero">
      <div v-for="(d, i) in dados" :key="i" class="dado" :class="{ girando }">
        <span v-for="p in puntos[d] || []" :key="p" class="pip" :style="{
          gridColumn: (p % 3) + 1,
          gridRow: Math.floor(p / 3) + 1,
        }" />
      </div>
      <div v-if="resultado" class="suma" :class="{ gana: premio > 0 }">
        <span class="n">{{ resultado.suma }}</span>
        <span class="l">elegiste {{ resultado.elegida }}</span>
        <span v-if="premio > 0" class="p">{{ colones(premio) }}</span>
        <span v-else class="p cero">Sin acierto</span>
      </div>
    </div>

    <div class="opciones">
      <button
        v-for="n in 11"
        :key="n"
        class="num"
        :class="{ activa: suma === n + 1, 'resalta-ganador': premio > 0 && resultado?.suma === n + 1 }"
        @click="suma = n + 1; reproducir('tecla')">
        {{ n + 1 }}
        <small>x{{ mult[n + 1] }}</small>
      </button>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando" @click="tirar">
      {{ cargando ? 'Tirando…' : 'Tirar dados' }}
    </button>
  </div>
</template>

<style scoped>
.dados { display: flex; flex-direction: column; gap: 14px; }
.tablero { display: flex; align-items: center; justify-content: center; gap: 14px; min-height: 96px; }
.dado {
  width: 64px; height: 64px; border-radius: 14px; background: linear-gradient(145deg, #f4f6f8, #c9d2dc);
  display: grid; grid-template-columns: repeat(3, 1fr); grid-template-rows: repeat(3, 1fr);
  padding: 9px; box-shadow: 0 8px 18px rgba(0, 0, 0, .4);
}
.dado.girando { animation: sacudir .7s ease; }
.pip { width: 9px; height: 9px; border-radius: 50%; background: #15181d; place-self: center; }
@keyframes sacudir {
  0% { transform: rotate(0) translateY(0); }
  25% { transform: rotate(-18deg) translateY(-8px); }
  50% { transform: rotate(14deg) translateY(4px); }
  75% { transform: rotate(-8deg) translateY(-3px); }
  100% { transform: rotate(0) translateY(0); }
}
.suma { display: flex; flex-direction: column; gap: 2px; min-width: 96px; }
.suma .n { font-size: 30px; font-weight: 800; }
.suma .l { font-size: 11px; color: var(--faint); text-transform: uppercase; letter-spacing: .07em; }
.suma .p { font-size: 16px; font-weight: 700; color: var(--gold); }
.suma .p.cero { color: var(--muted); font-weight: 500; }
.suma.gana .n { color: var(--prim2); }
.opciones { display: grid; grid-template-columns: repeat(6, 1fr); gap: 6px; }
.num {
  padding: 7px 4px; border-radius: 9px; border: 1px solid var(--border-soft);
  background: var(--surface); color: var(--muted); cursor: pointer; font-size: 14px;
  display: flex; flex-direction: column; align-items: center; gap: 1px;
}
.num small { font-size: 10px; color: var(--faint); }
.num.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
.num.activa small { color: var(--prim-2); }
</style>
