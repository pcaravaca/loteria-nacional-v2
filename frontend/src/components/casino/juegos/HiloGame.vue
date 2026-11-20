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

const direccion = ref<'mayor' | 'menor'>('mayor')
const vuelta = ref(false)

// Rojas aproximadas para pintar la carta (corazones/diamantes).
const rojas = new Set([1, 3, 5, 7, 9, 12, 13])

watch(
  () => props.resultado,
  (r) => {
    if (!r) return
    vuelta.value = false
    reproducir('carta')
    setTimeout(() => (vuelta.value = true), 400)
  },
)

function apostar(d: 'mayor' | 'menor') {
  direccion.value = d
  emit('jugar', { direccion: d })
}
</script>

<template>
  <div class="hilo">
    <div class="cartas">
      <div class="carta" :class="{ roja: rojas.has(resultado?.base ?? 0) }">
        <span>{{ resultado?.figura_base ?? 'A' }}</span>
        <small>Base</small>
      </div>
      <div class="vs">→</div>
      <div class="carta" :class="[vuelta ? 'revelada' : 'dorso', rojas.has(resultado?.siguiente ?? 0) ? 'roja' : '']">
        <span v-if="vuelta && resultado">{{ resultado.figura_siguiente }}</span>
        <span v-else>?</span>
        <small>{{ vuelta && resultado ? 'Salió' : 'Tapada' }}</small>
      </div>
    </div>

    <div v-if="resultado" class="resuelto" :class="{ gana: premio > 0 }">
      <span class="t">
        Elegiste {{ resultado.direccion }} · {{ resultado.figura_base }} → {{ resultado.figura_siguiente }}
      </span>
      <span v-if="premio > 0" class="p">{{ colones(premio) }}</span>
      <span v-else class="p cero">Sin acierto</span>
    </div>

    <div class="opciones">
      <button
        class="btn btn-dir"
        :class="{ activa: direccion === 'mayor', 'resalta-ganador': premio > 0 && resultado?.direccion === 'mayor' }"
        :disabled="cargando"
        @click="apostar('mayor')"
      >
        Mayor <small>x2</small>
      </button>
      <button
        class="btn btn-dir"
        :class="{ activa: direccion === 'menor', 'resalta-ganador': premio > 0 && resultado?.direccion === 'menor' }"
        :disabled="cargando"
        @click="apostar('menor')"
      >
        Menor <small>x2</small>
      </button>
    </div>
    <p class="nota">Si la carta sale igual, se pierde la apuesta.</p>
  </div>
</template>

<style scoped>
.hilo { display: flex; flex-direction: column; gap: 14px; }
.cartas { display: flex; align-items: center; justify-content: center; gap: 16px; min-height: 132px; }
.carta {
  width: 88px; height: 120px; border-radius: 12px;
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 2px;
  font-family: var(--display); font-size: 34px; font-weight: 700;
  background: linear-gradient(160deg, #1c2733, #10161d);
  border: 1px solid var(--border);
}
.carta small { font-size: 10px; font-weight: 500; color: var(--faint); text-transform: uppercase; letter-spacing: .08em; }
.carta.roja { color: #ff7b72; }
.carta.revelada { animation: aparecer .35s ease; }
@keyframes aparecer { from { transform: rotateY(90deg) scale(.9); opacity: .4; } to { transform: none; opacity: 1; } }
.carta.dorso {
  background: repeating-linear-gradient(45deg, #0f2a20, #0f2a20 6px, #12362a 6px, #12362a 12px);
  color: var(--prim-2);
}
.vs { color: var(--faint); font-size: 20px; }
.resuelto {
  display: flex; flex-direction: column; align-items: center; gap: 2px;
  padding: 10px 18px; border-radius: 12px; text-align: center;
  background: var(--surface); border: 1px solid var(--border-soft);
}
.resuelto .t { font-size: 13px; color: var(--muted); }
.resuelto .p { color: var(--gold); font-weight: 700; font-size: 16px; }
.resuelto .p.cero { color: var(--muted); font-weight: 500; }
.resuelto.gana { border-color: rgba(245, 185, 66, 0.5); }
.opciones { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.btn-dir {
  display: flex; flex-direction: column; align-items: center; gap: 1px;
  padding: 10px; border-radius: 10px; cursor: pointer;
  border: 1px solid var(--border-soft); background: var(--surface);
  color: var(--text); font-size: 15px; font-weight: 600;
}
.btn-dir small { font-size: 11px; color: var(--faint); }
.btn-dir.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
.btn-dir.activa small { color: var(--prim-2); }
.nota { margin: 0; font-size: 12px; color: var(--faint); text-align: center; }
</style>