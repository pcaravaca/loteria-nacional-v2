<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { colones } from '@/utils/formato'
import { simbolos } from '@/assets/casino'
import { reproducir } from '@/utils/sfx'
import { modalidadCompartida, modalidadPorCodigo, modalidadesDe } from './modalidades'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

const JUEGO = 'raspa'
const modalidades = modalidadesDe(JUEGO)
const modalidad = modalidadCompartida(JUEGO)
const cfg = computed(() => modalidadPorCodigo(JUEGO, modalidad.value))

// Que casilleros pagaron: tres iguales marcan todos; un par marca las dos iguales.
const pagaRaspa = computed(() => {
  const s = props.resultado?.simbolos ?? []
  if (!s.length || props.premio <= 0) return [false, false, false]
  return s.map((v: any, i: number) => s.some((o: any, j: number) => j !== i && o === v))
})

// Raspado a mano: el ticket llega comprada del servidor y el jugador va
// levantando la platea arrastrando el dedo, como en un ticket fisico.
const raspadas = ref<boolean[]>([false, false, false])
const raspando = ref(false)
const comprada = ref(false)
const todas = computed(() => raspadas.value.every(Boolean))

function revelar(i: number) {
  if (!comprada.value || raspadas.value[i]) return
  raspadas.value[i] = true
  reproducir('rodillo')
  if (raspadas.value.every(Boolean)) {
    if (props.premio > 0) reproducir('acierto')
    else reproducir('perder')
  }
}

// El puntero va levantando la platea de cada casilla por la que pasa.
function porPuntero(e: PointerEvent) {
  if (!raspando.value) return
  const el = e.currentTarget as HTMLElement
  const r = el.getBoundingClientRect()
  const x = Math.min(Math.max(e.clientX - r.left, 0), r.width)
  const idx = Math.min(2, Math.floor((x / r.width) * 3))
  revelar(idx)
}

watch(
  () => props.resultado,
  (r) => {
    if (!r) return
    comprada.value = true
    raspadas.value = [false, false, false]
  },
)

function elegirModalidad(codigo: string) {
  modalidad.value = codigo
  comprada.value = false
  raspadas.value = [false, false, false]
  reproducir('tecla')
}

function comprar() {
  comprada.value = false
  raspadas.value = [false, false, false]
  reproducir('ficha')
  emit('jugar', { modalidad: modalidad.value })
}
</script>

<template>
  <div class="raspa">
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

    <div
      class="panel"
      :class="{ comprada, lista: todas }"
      @pointerdown="raspando = true"
      @pointermove="porPuntero"
      @pointerup="raspando = false"
      @pointerleave="raspando = false"
    >
      <div
        v-for="(s, i) in (resultado?.simbolos ?? ['?', '?', '?'])"
        :key="i"
        class="casilla"
        :class="{ 'resalta-ganador': todas && premio > 0 && pagaRaspa[i] }"
      >
        <img v-if="raspadas[i]" :src="simbolos[s]" :alt="s" />
        <span v-else class="plata">
          <span class="signo">?</span>
          <span class="texto">Raspa acá</span>
        </span>
      </div>
    </div>

    <p v-if="comprada && !todas" class="pista">Deslizá el dedo sobre el ticket para raspar.</p>

    <div v-if="todas" class="estado" :class="{ gana: premio > 0 }">
      <template v-if="premio > 0"><span>Ganaste</span><strong>{{ colones(premio) }}</strong></template>
      <span v-else class="neutro">Sin premio · tres iguales pagan {{ cfg.pagos[0].mult }}</span>
    </div>

    <div class="paytable">
      <span v-for="p in cfg.pagos" :key="p.etiqueta">{{ p.etiqueta }} {{ p.mult }}</span>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando" @click="comprar">
      {{ cargando ? 'Comprando…' : comprada ? 'Otro ticket' : 'Comprar ticket' }}
    </button>
  </div>
</template>

<style scoped>
.raspa { display: flex; flex-direction: column; gap: 12px; }
.modalidades { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
.chip { padding: 6px 12px; border-radius: 999px; border: 1px solid var(--border-soft); background: var(--surface); color: var(--muted); cursor: pointer; font-size: 12px; }
.chip.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
.regla { margin: 0; font-size: 12px; color: var(--faint); text-align: center; line-height: 1.35; }
.panel {
  display: flex; gap: 10px; padding: 6px; border-radius: 16px;
  background: linear-gradient(160deg, rgba(255,255,255,.04), rgba(255,255,255,.01));
  border: 1px solid var(--border-soft); touch-action: none; user-select: none;
}
.panel.lista { border-color: rgba(245,185,66,.4); }
.casilla {
  flex: 1; height: 118px; border-radius: 12px; overflow: hidden; position: relative;
  background: #05080c; border: 1px solid var(--border); display: grid; place-items: center;
}
.casilla img { width: 72px; height: 72px; object-fit: contain; animation: aparecer .3s ease; }
@keyframes aparecer { from { transform: scale(.7); opacity: 0; } to { transform: scale(1); opacity: 1; } }
.plata {
  position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 4px; background: linear-gradient(140deg, #d7dde4, #98a3b0 40%, #cfd6de 60%, #8b95a1);
  color: #4a5460; cursor: crosshair;
}
.plata .signo { font-size: 30px; font-weight: 800; }
.plata .texto { font-size: 10px; text-transform: uppercase; letter-spacing: .08em; }
.pista { margin: 0; text-align: center; font-size: 11.5px; color: var(--prim-2); }
.estado { text-align: center; padding: 10px; border-radius: 10px; background: rgba(255,255,255,.02); border: 1px solid var(--border-soft); color: var(--muted); }
.estado.gana { background: var(--prim-dim); border-color: rgba(16,185,129,.35); color: var(--prim-2); }
.estado strong { color: var(--gold); font-size: 18px; margin-left: 6px; }
.neutro { font-size: 12px; }
.paytable { display: flex; justify-content: center; gap: 12px; flex-wrap: wrap; font-size: 11px; color: var(--faint); }
</style>
