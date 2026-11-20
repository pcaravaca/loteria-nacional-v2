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

const JUEGO = 'cascada'
const modalidades = modalidadesDe(JUEGO)
const modalidad = modalidadCompartida(JUEGO)
const cfg = computed(() => modalidadPorCodigo(JUEGO, modalidad.value))

// La tabla del servidor es la fuente de verdad; esta copia dibuja el tablero.
const TABLA_DEFECTO = [10, 5, 3, 1, 0, 0, 1, 3, 5, 10]
const tabla = computed<number[]>(() => cfg.value.tablaCascada ?? TABLA_DEFECTO)
const filas = computed(() => cfg.value.filas ?? TABLA_DEFECTO.length - 1)

const animando = ref(false)
const paso = ref(filas.value)
const xPct = ref(50)
const posicion = ref<number | null>(null)

watch(
  () => props.resultado,
  (r) => {
    if (!r) return
    posicion.value = r.posicion
    animar()
  },
)

// Si cambia la modalidad, el tablero se redibuja con la nueva cantidad de filas.
watch([filas, tabla], () => {
  paso.value = filas.value
  posicion.value = null
  animando.value = false
})

function animar() {
  animando.value = true
  reproducir('caida')
  paso.value = 0
  xPct.value = 50
  // 5%..95% segun la casilla dentro de la tabla.
  const destino = (((posicion.value ?? (tabla.value.length - 1) / 2) + 0.5) / tabla.value.length) * 100
  const total = filas.value
  let k = 0
  const timer = setInterval(() => {
    k++
    const progreso = k / total
    const rebote = (k % 2 === 0 ? 1 : -1) * 9 * (1 - progreso)
    xPct.value = 50 + (destino - 50) * progreso + rebote * 0.5
    paso.value = k
    if (k >= total) {
      clearInterval(timer)
      xPct.value = destino
      animando.value = false
    }
  }, 150)
}

function elegirModalidad(codigo: string) {
  modalidad.value = codigo
  reproducir('tecla')
}

function soltar() {
  emit('jugar', { modalidad: modalidad.value })
}
</script>

<template>
  <div class="cascada">
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

    <div class="tablero">
      <div class="filas">
        <div v-for="f in filas" :key="f" class="fila">
          <span v-for="p in f + 1" :key="p" class="pin" />
        </div>
      </div>
      <div
        v-if="resultado"
        class="bola"
        :style="{ top: (paso / filas) * 88 + 4 + '%', left: xPct + '%' }"
      />
      <div class="casillas" :class="{ muchas: tabla.length > 10 }" :style="{ gridTemplateColumns: `repeat(${tabla.length}, minmax(0, 1fr))` }">
        <div
          v-for="(m, i) in tabla"
          :key="i"
          class="casilla"
          :class="{
            ganadora: resultado && !animando && posicion === i && m > 0,
            'resalta-ganador': resultado && !animando && posicion === i && m > 0,
            cero: m === 0,
          }"
        >
          {{ m }}x
        </div>
      </div>
      <div v-if="resultado && !animando" class="resuelto" :class="{ gana: premio > 0 }">
        <span class="n">x{{ resultado.tabla[resultado.posicion] }}</span>
        <span v-if="premio > 0" class="p">{{ colones(premio) }}</span>
        <span v-else class="p cero">Sin premio</span>
      </div>
    </div>

    <div class="paytable">
      <span v-for="p in cfg.pagos" :key="p.etiqueta">{{ p.etiqueta }} {{ p.mult }}</span>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando" @click="soltar">
      {{ cargando ? 'Soltando…' : 'Soltar bola' }}
    </button>
  </div>
</template>

<style scoped>
.cascada { display: flex; flex-direction: column; gap: 14px; }
.modalidades { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
.chip { padding: 6px 12px; border-radius: 999px; border: 1px solid var(--border-soft); background: var(--surface); color: var(--muted); cursor: pointer; font-size: 12px; }
.chip.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
.regla { margin: 0; font-size: 12px; color: var(--faint); text-align: center; line-height: 1.35; }
.tablero {
  position: relative; height: min(340px, 58vh); overflow: hidden;
  border: 1px solid var(--border-soft); border-radius: 14px;
  background: linear-gradient(180deg, rgba(16, 185, 129, 0.06), transparent 60%);
}
.filas { position: absolute; inset: 0 0 46px 0; display: flex; flex-direction: column; justify-content: space-evenly; padding: 8px 16px; }
.fila { display: flex; justify-content: space-between; }
.pin { width: 5px; height: 5px; border-radius: 50%; background: rgba(142, 162, 182, 0.45); }
.bola {
  position: absolute; width: 16px; height: 16px; border-radius: 50%;
  transform: translate(-50%, -50%);
  background: radial-gradient(circle at 30% 30%, #fff7dd, #f5b942 55%, #a8740a);
  box-shadow: 0 0 16px rgba(245, 185, 66, 0.75);
  transition: left 0.14s linear, top 0.14s linear;
}
.casillas {
  position: absolute; left: 0; right: 0; bottom: 0;
  display: grid; gap: 4px; padding: 6px;
}
/* Con 13 casillas (cascada profunda) la celda es angosta: se achica la letra. */
.casillas.muchas { gap: 3px; padding: 5px; }
.casillas.muchas .casilla { font-size: 10px; padding: 6px 0; }
.casilla {
  padding: 7px 0; text-align: center; font-size: 12px; font-weight: 700;
  border-radius: 8px; background: var(--surface); border: 1px solid var(--border-soft);
  color: var(--muted);
}
.casilla.ganadora {
  background: linear-gradient(135deg, var(--prim), #0aa37a);
  color: #04140e; border-color: var(--prim);
}
.casilla.cero { color: var(--faint); }
.resuelto {
  position: absolute; top: 42%; left: 50%; transform: translateX(-50%);
  display: flex; flex-direction: column; align-items: center; gap: 2px;
  padding: 10px 20px; border-radius: 12px;
  background: rgba(8, 11, 16, 0.88); border: 1px solid var(--border);
}
.resuelto .n { font-family: var(--display); font-size: 26px; font-weight: 700; }
.resuelto .p { color: var(--gold); font-weight: 700; }
.resuelto .p.cero { color: var(--muted); font-weight: 500; }
.resuelto.gana { border-color: rgba(245, 185, 66, 0.5); }
.paytable { display: flex; justify-content: center; gap: 12px; flex-wrap: wrap; font-size: 11px; color: var(--faint); }
</style>
