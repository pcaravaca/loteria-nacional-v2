<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { colones } from '@/utils/formato'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

const elegidos = ref<number[]>([])
const numeros = Array.from({ length: 60 }, (_, i) => i + 1)

const mult: Record<number, number> = { 2: 1, 3: 3, 4: 8, 5: 65 }

function toggle(n: number) {
  const i = elegidos.value.indexOf(n)
  if (i >= 0) elegidos.value.splice(i, 1)
  else if (elegidos.value.length < 5) elegidos.value.push(n)
}

const bolillas = computed(() => props.resultado?.bolillas ?? [])
const aciertos = computed(() => props.resultado?.aciertos ?? 0)

watch(
  () => props.resultado,
  (r) => {
    if (r?.tus_numeros) elegidos.value = [...r.tus_numeros]
  },
)

function jugar() {
  if (elegidos.value.length !== 5) return
  emit('jugar', { numeros: [...elegidos.value].sort((a, b) => a - b) })
}
</script>

<template>
  <div class="keno">
    <div v-if="resultado" class="bolillas">
      <span
        v-for="b in bolillas"
        :key="b"
        class="bolilla"
        :class="{ mia: elegidos.includes(b) }"
      >{{ b }}</span>
    </div>

    <div v-if="resultado" class="estado" :class="{ gana: premio > 0 }">
      <span>{{ aciertos }} acierto{{ aciertos === 1 ? '' : 's' }}</span>
      <strong v-if="premio > 0">{{ colones(premio) }}</strong>
      <span v-else class="neutro">x{{ mult[aciertos] ?? 0 }}</span>
    </div>

    <div class="grilla">
      <button
        v-for="n in numeros"
        :key="n"
        class="num"
        :class="{ activa: elegidos.includes(n), salida: bolillas.includes(n) }"
        @click="toggle(n)"
      >{{ n }}</button>
    </div>

    <div class="ayuda">Elegí 5 números ({{ elegidos.length }}/5)</div>
    <div class="paytable">
      <span>2 aciertos x1</span><span>3 x3</span><span>4 x8</span><span>5 x65</span>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando || elegidos.length !== 5" @click="jugar">
      {{ cargando ? 'Sorteando…' : 'Jugar Keno' }}
    </button>
  </div>
</template>

<style scoped>
.keno { display: flex; flex-direction: column; gap: 12px; }
.bolillas { display: flex; flex-wrap: wrap; gap: 6px; justify-content: center; }
.bolilla {
  min-width: 30px; height: 30px; padding: 0 6px; border-radius: 999px; display: grid; place-items: center;
  background: var(--surface); border: 1px solid var(--border); font-size: 12px; color: var(--muted);
}
.bolilla.mia { background: var(--prim-dim); border-color: var(--prim); color: var(--prim-2); font-weight: 700; }
.estado { display: flex; align-items: center; justify-content: center; gap: 8px; padding: 10px; border-radius: 10px; background: rgba(255,255,255,.02); border: 1px solid var(--border-soft); color: var(--muted); }
.estado.gana { background: var(--prim-dim); border-color: rgba(16,185,129,.35); color: var(--prim-2); }
.estado strong { color: var(--gold); font-size: 18px; }
.neutro { font-size: 12px; }
.grilla { display: grid; grid-template-columns: repeat(10, 1fr); gap: 5px; }
.num {
  height: 30px; border-radius: 8px; border: 1px solid var(--border-soft); background: var(--surface);
  color: var(--muted); font-size: 12px; cursor: pointer;
}
.num.activa { background: var(--prim-dim); border-color: var(--prim); color: var(--prim-2); font-weight: 700; }
.num.salida:not(.activa) { background: rgba(245,185,66,.12); border-color: rgba(245,185,66,.4); color: var(--gold); }
.ayuda { font-size: 12px; color: var(--faint); text-align: center; }
.paytable { display: flex; justify-content: center; gap: 12px; font-size: 11px; color: var(--faint); }
</style>
