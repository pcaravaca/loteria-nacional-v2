<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'
import { modalidadCompartida, modalidadPorCodigo, modalidadesDe } from './modalidades'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

const JUEGO = 'keno'
const modalidades = modalidadesDe(JUEGO)
const modalidad = modalidadCompartida(JUEGO)
const cfg = computed(() => modalidadPorCodigo(JUEGO, modalidad.value))

const elegidos = ref<number[]>([])
const numeros = computed(() =>
  Array.from({ length: cfg.value.bolillasTotales ?? 60 }, (_, i) => i + 1),
)
const mult = computed(() => cfg.value.tablaKeno ?? {})
// La grilla se acomoda a la cantidad de bolillas: 8 columnas para 40, 10 para 60.
const columnas = computed(() => ((cfg.value.bolillasTotales ?? 60) <= 40 ? 8 : 10))

function toggle(n: number) {
  const i = elegidos.value.indexOf(n)
  if (i >= 0) elegidos.value.splice(i, 1)
  else if (elegidos.value.length < (cfg.value.elegidos ?? 5)) elegidos.value.push(n)
  reproducir('tecla')
}

const bolillas = computed(() => props.resultado?.bolillas ?? [])
const aciertos = computed(() => props.resultado?.aciertos ?? 0)

watch(
  () => props.resultado,
  (r) => {
    if (r?.tus_numeros) elegidos.value = [...r.tus_numeros]
    reproducir('bolilla')
  },
)

// Al cambiar de modalidad cambia la cantidad de numeros que se pueden elegir.
watch(modalidad, () => {
  elegidos.value = []
})

function elegirModalidad(codigo: string) {
  modalidad.value = codigo
  reproducir('tecla')
}

function jugar() {
  if (elegidos.value.length !== cfg.value.elegidos) return
  emit('jugar', {
    numeros: [...elegidos.value].sort((a, b) => a - b),
    modalidad: modalidad.value,
  })
}
</script>

<template>
  <div class="keno">
    <div class="modalidades">
      <button
        v-for="m in modalidades"
        :key="m.codigo"
        class="chip"
        :class="{ activa: modalidad === m.codigo }"
        @click="elegirModalidad(m.codigo)"
      >
        {{ m.nombre }}
      </button>
    </div>
    <p class="regla">{{ cfg.descripcion }}</p>

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

    <div class="grilla" :style="{ gridTemplateColumns: `repeat(${columnas}, minmax(0, 1fr))` }">
      <button
        v-for="n in numeros"
        :key="n"
        class="num"
        :class="{ activa: elegidos.includes(n), salida: bolillas.includes(n), 'resalta-ganador': premio > 0 && bolillas.includes(n) && elegidos.includes(n) }"
        @click="toggle(n)">{{ n }}</button>
    </div>

    <div class="ayuda">Elegí {{ cfg.elegidos }} números ({{ elegidos.length }}/{{ cfg.elegidos }})</div>
    <div class="paytable">
      <span v-for="p in cfg.pagos" :key="p.etiqueta">{{ p.etiqueta }} {{ p.mult }}</span>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando || elegidos.length !== cfg.elegidos" @click="jugar">
      {{ cargando ? 'Sorteando…' : 'Jugar Keno' }}
    </button>
  </div>
</template>

<style scoped>
.keno { display: flex; flex-direction: column; gap: 12px; }
.modalidades { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
.chip { padding: 6px 12px; border-radius: 999px; border: 1px solid var(--border-soft); background: var(--surface); color: var(--muted); cursor: pointer; font-size: 12px; }
.chip.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
.regla { margin: 0; font-size: 12px; color: var(--faint); text-align: center; line-height: 1.35; }
.bolillas { display: flex; flex-wrap: wrap; gap: 6px; justify-content: center; }
.bolilla {
  min-width: 30px; height: 30px; padding: 0 6px; border-radius: 999px; display: grid; place-items: center;
  background: var(--surface); border: 1px solid var(--border); font-size: 12px; color: var(--muted);
  animation: aparecer .3s ease both;
}
@keyframes aparecer { from { transform: scale(.4); opacity: 0; } to { transform: scale(1); opacity: 1; } }
.bolilla.mia { background: var(--prim-dim); border-color: var(--prim); color: var(--prim-2); font-weight: 700; }
.estado { display: flex; align-items: center; justify-content: center; gap: 8px; padding: 10px; border-radius: 10px; background: rgba(255,255,255,.02); border: 1px solid var(--border-soft); color: var(--muted); }
.estado.gana { background: var(--prim-dim); border-color: rgba(16,185,129,.35); color: var(--prim-2); }
.estado strong { color: var(--gold); font-size: 18px; }
.neutro { font-size: 12px; }
.grilla { display: grid; gap: 5px; }
.num {
  min-width: 0; padding: 0; height: 30px; border-radius: 8px; border: 1px solid var(--border-soft); background: var(--surface);
  color: var(--muted); font-size: 12px; cursor: pointer;
}
.num.activa { background: var(--prim-dim); border-color: var(--prim); color: var(--prim-2); font-weight: 700; }
.num.salida:not(.activa) { background: rgba(245,185,66,.12); border-color: rgba(245,185,66,.4); color: var(--gold); }
.ayuda { font-size: 12px; color: var(--faint); text-align: center; }
.paytable { display: flex; justify-content: center; gap: 12px; flex-wrap: wrap; font-size: 11px; color: var(--faint); }
</style>
