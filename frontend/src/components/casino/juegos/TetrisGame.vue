<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'
import { rtpDe, rtpTexto } from '@/utils/rtpCasino'

const props = defineProps<{ monto: number; clientSeed: string }>()
const auth = useAuthStore()

// RTP declarado para este juego (seed del casino): jugando bien y al azar.
const rtp = rtpDe('tetris')!

const FILAS = 12
const COLUMNAS = 6

interface TetrisEstado {
  ronda_id: number
  monto: number
  tablero: number[]
  piezas: number[]
  pieza: number
  piezas_totales: number
  lineas: number
  estado: string
  multiplicador: number
  premio: number
  premio_pagado: number
  jugadas: [number, number][]
  saldo: number
  server_seed_hash: string
  nonce: number
}

const FORMAS: number[][][] = [
  [[0, 1], [1, 1], [2, 1], [3, 1]], // I
  [[1, 0], [2, 0], [1, 1], [2, 1]], // O
  [[1, 0], [0, 1], [1, 1], [2, 1]], // T
  [[1, 0], [2, 0], [0, 1], [1, 1]], // S
  [[0, 0], [1, 0], [1, 1], [2, 1]], // Z
  [[2, 0], [0, 1], [1, 1], [2, 1]], // L
  [[0, 0], [0, 1], [1, 1], [2, 1]], // J
]

const COLORES = [
  'linear-gradient(150deg,#a5f3fc,#0891b2)', // I cian
  'linear-gradient(150deg,#fef08a,#ca8a04)', // O amarillo
  'linear-gradient(150deg,#d8b4fe,#7b3fd1)', // T violeta
  'linear-gradient(150deg,#86efac,#16a34a)', // S verde
  'linear-gradient(150deg,#fca5a5,#dc2626)', // Z rojo
  'linear-gradient(150deg,#fdba74,#ea580c)', // L naranja
  'linear-gradient(150deg,#93c5fd,#2563eb)', // J azul
]

const ronda = ref<TetrisEstado | null>(null)
const cargando = ref(false)
const error = ref('')
const rotacion = ref(0)

const saldo = computed(() => auth.user?.saldo ?? 0)
const abierta = computed(() => ronda.value?.estado === 'abierta')

function rotar(celdas: number[][]): number[][] {
  const out = celdas.map(([x, y]) => [-y, x])
  const minX = Math.min(...out.map((c) => c[0]))
  const minY = Math.min(...out.map((c) => c[1]))
  return out.map(([x, y]) => [x - minX, y - minY])
}

const piezaActual = computed(() => {
  if (!ronda.value) return { celdas: [] as number[][], tipo: -1 }
  const tipo = ronda.value.piezas[ronda.value.pieza] ?? -1
  let celdas = (FORMAS[tipo] ?? []).map((c) => [...c])
  for (let i = 0; i < rotacion.value; i++) celdas = rotar(celdas)
  return { celdas, tipo }
})

function claseCelda(valor: number): string {
  if (!valor) return ''
  return `t${valor - 1}`
}

async function empezar() {
  error.value = ''
  cargando.value = true
  rotacion.value = 0
  try {
    const { data } = await api.post<TetrisEstado>('/casino/tetris/start', {
      monto: props.monto,
      client_seed: props.clientSeed,
    })
    ronda.value = data
    reproducir('tecla')
    await auth.cargar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
    reproducir('error')
  } finally {
    cargando.value = false
  }
}

async function colocar(columna: number) {
  if (!ronda.value || cargando.value || !abierta.value) return
  error.value = ''
  cargando.value = true
  try {
    const { data } = await api.post<TetrisEstado>(`/casino/tetris/${ronda.value.ronda_id}/move`, {
      columna,
      rotacion: rotacion.value,
    })
    ronda.value = data
    rotacion.value = 0
    reproducir(data.estado === 'pagada' ? (data.premio_pagado > 0 ? 'ganar' : 'perder') : 'caida')
    await auth.cargar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
    reproducir('error')
  } finally {
    cargando.value = false
  }
}

function girar() {
  rotacion.value = (rotacion.value + 1) % 4
  reproducir('click')
}
</script>

<template>
  <div class="tetris">
    <div class="tetris-head">
      <div class="contador">
        <span class="etq">Piezas</span>
        <strong>{{ ronda ? ronda.pieza + 1 : 1 }}/{{ ronda?.piezas_totales ?? 8 }}</strong>
      </div>
      <div class="contador">
        <span class="etq">Líneas</span>
        <strong>{{ ronda?.lineas ?? 0 }}</strong>
      </div>
      <div class="contador">
        <span class="etq">Premio</span>
        <strong class="oro">{{ colones(ronda?.premio ?? 0) }}</strong>
      </div>
      <div class="info">
        <span>Saldo <b>{{ colones(saldo) }}</b></span>
        <span>Apuesta <b>{{ colones(monto) }}</b></span>
      </div>
    </div>

    <div class="area">
      <div class="pozo">
        <div v-if="!ronda" class="vacio">
          <span>Tocá <b>Empezar</b> para recibir las piezas.</span>
          <small>Completá líneas y cobrá según el total.</small>
        </div>
        <template v-else>
          <div v-for="f in FILAS" :key="f" class="fila">
            <div
              v-for="c in COLUMNAS"
              :key="c"
              class="celda"
              :class="claseCelda(ronda.tablero[(f - 1) * COLUMNAS + (c - 1)])"
            />
          </div>
        </template>
      </div>

      <div class="lateral">
        <div class="preview">
          <span class="etq">Próxima</span>
          <div class="preview-grid">
            <div
              v-for="n in 16"
              :key="n"
              class="pv-celda"
              :class="{ lleno: piezaActual.celdas.some((cd) => cd[0] === (n - 1) % 4 && cd[1] === Math.floor((n - 1) / 4)) }"
              :style="piezaActual.celdas.some((cd) => cd[0] === (n - 1) % 4 && cd[1] === Math.floor((n - 1) / 4)) ? { background: COLORES[piezaActual.tipo] } : {}"
            />
          </div>
        </div>
        <button class="btn btn-ghost btn-block" :disabled="cargando || !abierta" @click="girar">
          Girar
        </button>
      </div>
    </div>

    <p class="rtp">
      RTP declarado: <b>jugando bien {{ rtpTexto(rtp.rtp) }}</b> ·
      <b>jugando al azar {{ rtpTexto(rtp.rtp_casual) }}</b>
    </p>

    <p v-if="error" class="error-text">{{ error }}</p>

    <div v-if="!ronda" class="controles">
      <button class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
        {{ cargando ? 'Abriendo…' : 'Empezar por ' + colones(monto) }}
      </button>
      <p class="paytable">1 línea x0.28 · 2 x0.55 · 3 x0.85 · 4 x1.15 · 5 x1.45 · 6 x1.75 · 7 x2.05 · 8+ x2.35</p>
    </div>

    <div v-else-if="abierta" class="controles">
      <button
        v-for="c in COLUMNAS"
        :key="c"
        class="btn btn-columna"
        :disabled="cargando"
        @click="colocar(c - 1)"
      >
        {{ c }}
      </button>
      <span class="ayuda">Elegí la columna y girá antes de soltar.</span>
    </div>

    <div v-else class="controles">
      <button class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
        {{ cargando ? 'Abriendo…' : 'Jugar de nuevo por ' + colones(monto) }}
      </button>
    </div>

    <div v-if="ronda && ronda.estado !== 'abierta'" class="resultado" :class="{ gana: ronda.premio_pagado > 0 }">
      <span v-if="ronda.premio_pagado > 0">¡Completaste {{ ronda.lineas }} líneas! Ganaste {{ colones(ronda.premio_pagado) }} (x{{ ronda.multiplicador }})</span>
      <span v-else>Te quedaste sin líneas: se pierde lo apostado</span>
    </div>
  </div>
</template>

<style scoped>
.tetris { display: flex; flex-direction: column; gap: 10px; }
.tetris-head {
  display: flex; align-items: center; gap: 14px; flex-wrap: wrap;
  padding: 10px 12px; border-radius: 12px;
  background: rgba(3, 16, 12, .75); border: 1px solid rgba(232, 206, 122, .35);
}
.contador { display: flex; flex-direction: column; gap: 1px; min-width: 64px; }
.contador .etq { font-size: 10px; letter-spacing: .12em; text-transform: uppercase; color: var(--faint); }
.contador strong { font-family: var(--display); font-size: 18px; color: var(--text); }
.contador strong.oro { color: var(--gold); }
.info { margin-left: auto; display: flex; gap: 14px; font-size: 12px; color: var(--faint); flex-wrap: wrap; }
.info b { color: var(--text); margin-left: 4px; }

.area { display: flex; gap: 10px; align-items: flex-start; justify-content: center; }
.pozo {
  display: grid; grid-template-rows: repeat(12, 1fr); gap: 2px; padding: 6px;
  border-radius: 12px; border: 1px solid #243448;
  background: #080b10; box-shadow: inset 0 14px 30px rgba(0, 0, 0, .45);
  /* Alto acotado: antes el ancho era el unico tope y las 12 filas con `1fr`
     inflaban el tablero hasta 1776 px en un telefono (Peter: "demasiado alto,
     hace demasiado scroll vertical"). Con aspect-ratio el alto deriva del ancho. */
  width: min(170px, 46vw);
  aspect-ratio: 6 / 12;
}
.fila { display: grid; grid-template-columns: repeat(6, 1fr); gap: 2px; min-height: 0; }
.celda { border-radius: 4px; background: rgba(255, 255, 255, .04); border: 1px solid rgba(255, 255, 255, .05); }
.celda.t0 { background: linear-gradient(150deg,#a5f3fc,#0891b2); }
.celda.t1 { background: linear-gradient(150deg,#fef08a,#ca8a04); }
.celda.t2 { background: linear-gradient(150deg,#d8b4fe,#7b3fd1); }
.celda.t3 { background: linear-gradient(150deg,#86efac,#16a34a); }
.celda.t4 { background: linear-gradient(150deg,#fca5a5,#dc2626); }
.celda.t5 { background: linear-gradient(150deg,#fdba74,#ea580c); }
.celda.t6 { background: linear-gradient(150deg,#93c5fd,#2563eb); }
.vacio { grid-column: 1 / -1; grid-row: 1 / -1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 4px; padding: 12px; color: var(--muted); text-align: center; }
.vacio small { color: var(--faint); font-size: 12px; }

.lateral { display: flex; flex-direction: column; gap: 8px; width: 96px; }
.preview { display: flex; flex-direction: column; gap: 4px; }
.preview .etq { font-size: 10px; letter-spacing: .12em; text-transform: uppercase; color: var(--faint); }
.preview-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 2px; }
.pv-celda { aspect-ratio: 1; border-radius: 3px; background: rgba(255, 255, 255, .05); }

.controles { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.controles .btn { flex: 1 1 30px; }
.btn-columna { background: rgba(16, 185, 129, .12); border: 1px solid rgba(16, 185, 129, .4); color: var(--text); font-family: var(--display); }
.btn-columna:hover { background: rgba(16, 185, 129, .22); }
.paytable { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11px; color: var(--faint); line-height: 1.5; }
.rtp { display: flex; justify-content: center; gap: 6px; flex-wrap: wrap; margin: 0; font-size: 12px; color: var(--muted); }
.rtp b { color: var(--text); }
.ayud { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11.5px; color: var(--faint); }

.resultado { text-align: center; padding: 10px; border-radius: 10px; background: rgba(0,0,0,.3); border: 1px solid var(--border-soft); font-size: 14px; color: var(--muted); }
.resultado.gana { background: var(--prim-dim); border-color: rgba(16,185,129,.35); color: var(--prim-2); font-weight: 700; }
.error-text { color: #f87171; font-size: 12px; margin: 0; }
</style>
