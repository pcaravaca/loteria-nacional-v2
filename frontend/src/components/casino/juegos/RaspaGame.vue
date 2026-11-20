<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { colones } from '@/utils/formato'
import { simbolos } from '@/assets/casino'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

// Que casilleros pagaron: tres iguales marcan todos; un par marca las dos iguales.
const pagaRaspa = computed(() => {
  const s = props.resultado?.simbolos ?? []
  if (!s.length || props.premio <= 0) return [false, false, false]
  // Si dos o mas casilleros son iguales, esos son los que pagaron.
  return s.map((v: any, i: number) => s.some((o: any, j: number) => j !== i && o === v))
})

const reveladas = ref<boolean[]>([false, false, false])

watch(
  () => props.resultado,
  (r) => {
    if (!r) return
    reveladas.value = [false, false, false]
    r.simbolos.forEach((_: string, i: number) => {
      window.setTimeout(() => (reveladas.value[i] = true), 500 + i * 450)
    })
  },
)

function rascar() {
  reveladas.value = [false, false, false]
  emit('jugar', {})
}
</script>

<template>
  <div class="raspa">
    <div class="panel">
      <div v-for="(s, i) in (resultado?.simbolos ?? ['?', '?', '?'])" :key="i" class="casilla"
           :class="{ 'resalta-ganador': premio > 0 && pagaRaspa[i] }">
        <img v-if="reveladas[i]" :src="simbolos[s]" :alt="s" />
        <span v-else class="plata">
          <span class="signo">?</span>
          <span class="texto">Raspa aqui</span>
        </span>
      </div>
    </div>

    <div v-if="resultado" class="estado" :class="{ gana: premio > 0 }">
      <template v-if="premio > 0"><span>Ganaste</span><strong>{{ colones(premio) }}</strong></template>
      <span v-else class="neutro">Sin premio · tres iguales pagan x18</span>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando" @click="rascar">
      {{ cargando ? 'Rascando…' : 'Rascar y jugar' }}
    </button>
  </div>
</template>

<style scoped>
.raspa { display: flex; flex-direction: column; gap: 12px; }
.panel { display: flex; gap: 10px; }
.casilla {
  flex: 1; height: 108px; border-radius: 12px; overflow: hidden; position: relative;
  background: #05080c; border: 1px solid var(--border); display: grid; place-items: center;
}
.casilla img { width: 72px; height: 72px; object-fit: contain; animation: aparecer .35s ease; }
@keyframes aparecer { from { transform: scale(.7); opacity: 0; } to { transform: scale(1); opacity: 1; } }
.plata {
  position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 4px; background: linear-gradient(140deg, #d7dde4, #98a3b0 40%, #cfd6de 60%, #8b95a1);
  color: #4a5460;
}
.plata .signo { font-size: 30px; font-weight: 800; }
.plata .texto { font-size: 10px; text-transform: uppercase; letter-spacing: .08em; }
.estado { text-align: center; padding: 10px; border-radius: 10px; background: rgba(255,255,255,.02); border: 1px solid var(--border-soft); color: var(--muted); }
.estado.gana { background: var(--prim-dim); border-color: rgba(16,185,129,.35); color: var(--prim-2); }
.estado strong { color: var(--gold); font-size: 18px; margin-left: 6px; }
.neutro { font-size: 12px; }
</style>
