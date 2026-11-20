<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'

const props = defineProps<{ monto: number; clientSeed: string }>()

const auth = useAuthStore()

const FILAS = 8
const COLUMNAS = 8
const DIVISOR = 300 // 300 puntos = lo apostado (ajustado por simulación)

interface GemasPaso {
  antes: number[]
  despues: number[]
  eliminadas: number[]
  especiales: number[]
  puntos: number
  cascada: number
}

interface GemasEstado {
  ronda_id: number
  monto: number
  tablero: number[]
  jugadas: [number, number][]
  movimientos: number
  movimientos_max: number
  puntos: number
  multiplicador: number
  premio: number
  premio_pagado: number
  estado: string
  hay_movimientos: boolean
  saldo: number
  server_seed_hash: string
  nonce: number
  pasos?: GemasPaso[]
}

// Tipos de gema: el orden es el del backend (internal/casino/gemas.go).
const TIPOS = [
  { nombre: 'Esmeralda', puntos: 3, clase: 't0' },
  { nombre: 'Zafiro', puntos: 4, clase: 't1' },
  { nombre: 'Amatista', puntos: 5, clase: 't2' },
  { nombre: 'Rubí', puntos: 7, clase: 't3' },
  { nombre: 'Topacio', puntos: 9, clase: 't4' },
  { nombre: 'Naranja', puntos: 12, clase: 't5' },
  { nombre: 'Diamante', puntos: 20, clase: 't6' },
  { nombre: 'Gema rayo', puntos: 40, clase: 't7' },
]

const ronda = ref<GemasEstado | null>(null)
const tablero = ref<number[]>([])
const seleccionada = ref<number | null>(null)
const cargando = ref(false)
const animando = ref(false)
const error = ref('')
const resaltando = ref<Set<number>>(new Set())
const cayendo = ref<Set<number>>(new Set())

const saldo = computed(() => auth.user?.saldo ?? 0)
const estado = computed(() => ronda.value?.estado ?? '')
const abierta = computed(() => estado.value === 'abierta')
const puntos = computed(() => ronda.value?.puntos ?? 0)
const premio = computed(() => ronda.value?.premio ?? 0)
const movimientos = computed(() => ronda.value?.movimientos ?? 0)
const movimientosMax = computed(() => ronda.value?.movimientos_max ?? 25)
const resto = computed(() => Math.max(0, movimientosMax.value - movimientos.value))
const multiplicador = computed(() => (puntos.value / DIVISOR).toFixed(2))

// Path de la gema: se dibuja con CSS; el rayo lleva un brillo animado.
function claseGema(tipo: number): string {
  return `gema t${tipo}`
}

function esAdyacente(a: number, b: number): boolean {
  const ra = Math.floor(a / COLUMNAS)
  const ca = a % COLUMNAS
  const rb = Math.floor(b / COLUMNAS)
  const cb = b % COLUMNAS
  return Math.abs(ra - rb) + Math.abs(ca - cb) === 1
}

function esperar(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function empezar() {
  error.value = ''
  cargando.value = true
  seleccionada.value = null
  try {
    const { data } = await api.post<GemasEstado>('/casino/gemas/start', {
      monto: props.monto,
      client_seed: props.clientSeed,
    })
    ronda.value = data
    tablero.value = data.tablero
    reproducir('tambor')
    await auth.cargar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
    reproducir('error')
  } finally {
    cargando.value = false
  }
}

async function tocarCelda(i: number) {
  if (!ronda.value || !abierta.value || cargando.value || animando.value) return
  if (seleccionada.value === null) {
    seleccionada.value = i
    reproducir('click')
    return
  }
  if (seleccionada.value === i) {
    seleccionada.value = null
    reproducir('click')
    return
  }
  if (!esAdyacente(seleccionada.value, i)) {
    seleccionada.value = i
    reproducir('click')
    return
  }
  await mover(seleccionada.value, i)
}

async function mover(a: number, b: number) {
  if (!ronda.value || cargando.value || animando.value) return
  error.value = ''
  cargando.value = true
  const puntosAntes = puntos.value
  try {
    const { data } = await api.post<GemasEstado>(`/casino/gemas/${ronda.value.ronda_id}/move`, { a, b })
    seleccionada.value = null
    await animar(data.pasos ?? [])
    ronda.value = data
    tablero.value = data.tablero
    if (data.estado !== 'abierta') {
      reproducir(data.premio_pagado > 0 ? 'ganar' : 'perder')
    } else if (data.puntos > puntosAntes) {
      reproducir('acierto')
    }
    await auth.cargar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
    reproducir('error')
  } finally {
    cargando.value = false
  }
}

async function animar(pasos: GemasPaso[]) {
  if (!pasos.length) return
  animando.value = true
  for (const paso of pasos) {
    tablero.value = paso.antes
    resaltando.value = new Set(paso.eliminadas)
    reproducir(paso.cascada > 1 ? 'caida' : 'acierto')
    await esperar(300)
    const dif = new Set<number>()
    for (let i = 0; i < paso.despues.length; i++) {
      if (paso.antes[i] !== paso.despues[i]) dif.add(i)
    }
    tablero.value = paso.despues
    resaltando.value = new Set()
    cayendo.value = dif
    await esperar(220)
    cayendo.value = new Set()
  }
  animando.value = false
}

async function reventar() {
  if (!ronda.value || cargando.value || animando.value) return
  error.value = ''
  cargando.value = true
  try {
    const { data } = await api.post<GemasEstado>(`/casino/gemas/${ronda.value.ronda_id}/reventar`)
    ronda.value = data
    reproducir(data.premio_pagado > 0 ? 'ganar' : 'perder')
    await auth.cargar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
    reproducir('error')
  } finally {
    cargando.value = false
  }
}
</script>

<template>
  <div class="gemas">
    <div class="gemas-head">
      <div class="contador">
        <span class="etq">Movimientos</span>
        <strong>{{ resto }}/{{ movimientosMax }}</strong>
      </div>
      <div class="contador">
        <span class="etq">Puntos</span>
        <strong>{{ puntos }}</strong>
      </div>
      <div class="contador">
        <span class="etq">Premio</span>
        <strong class="oro">{{ colones(premio) }}</strong>
      </div>
      <div class="info">
        <span>Saldo <b>{{ colones(saldo) }}</b></span>
        <span>Apuesta <b>{{ colones(monto) }}</b></span>
      </div>
    </div>

    <div class="tablero" :class="{ animando: animando || cargando }">
      <div
        v-for="(tipo, i) in tablero"
        :key="i"
        class="celda"
        :class="{
          seleccionada: seleccionada === i,
          'resalta-ganador': resaltando.has(i),
          cayendo: cayendo.has(i),
        }"
        @click="tocarCelda(i)"
      >
        <div :class="claseGema(tipo)">
          <span class="brillo" />
          <span v-if="tipo === 7" class="rayo" />
        </div>
      </div>
      <div v-if="!tablero.length" class="vacio">
        <span>Tocá <b>Empezar</b> para abrir el tablero.</span>
        <small>Una jugada es un tablero de 8×8. Tenés {{ movimientosMax }} intercambios.</small>
      </div>
    </div>

    <div v-if="ronda && abierta && !ronda.hay_movimientos" class="aviso">
      El tablero se quedó sin jugadas: se liquidó solo.
    </div>

    <p v-if="error" class="error-text">{{ error }}</p>

    <div v-if="!ronda" class="controles">
      <button class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
        {{ cargando ? 'Abriendo…' : 'Empezar por ' + colones(monto) }}
      </button>
      <p class="paytable">
        3 esmeraldas x0.01 · 3 zafiros x0.01 · 3 amatistas x0.02 · 3 rubíes x0.02 ·
        3 topacios x0.03 · 3 naranjas x0.04 · 3 diamantes x0.07 ·
        grupo de 4+ deja una gema rayo
      </p>
    </div>

    <div v-else-if="abierta" class="controles">
      <button class="btn btn-primary" :disabled="cargando || animando" @click="reventar">
        Reventar y cobrar {{ colones(premio) }}
      </button>
      <span class="ayuda">Tocá una gema y después una vecina para intercambiarlas.</span>
    </div>

    <div v-else class="controles">
      <button class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
        {{ cargando ? 'Abriendo…' : 'Jugar de nuevo por ' + colones(monto) }}
      </button>
    </div>

    <div v-if="ronda && estado !== 'abierta'" class="resultado" :class="{ gana: ronda.premio_pagado > 0 }">
      <span v-if="ronda.premio_pagado > 0">
        ¡Reventaste! Ganaste {{ colones(ronda.premio_pagado) }} (x{{ multiplicador }})
      </span>
      <span v-else>Sin premio en esta jugada</span>
    </div>

    <details class="guia-detalle">
      <summary>Tabla de pagos</summary>
      <div class="tabla">
        <div v-for="t in TIPOS" :key="t.clase" class="fila">
          <span class="muestra"><i :class="`punto ${t.clase}`" />{{ t.nombre }}</span>
          <b>{{ t.puntos }} pts</b>
        </div>
        <p class="nota">
          300 puntos = x1 · grupo de 4 paga x2, de 5 x3 y así · cada cascada multiplica
          x1, x1, x2, x2, x3… · la gema rayo limpia su fila y su columna.
        </p>
      </div>
    </details>
  </div>
</template>

<style scoped>
.gemas { display: flex; flex-direction: column; gap: 10px; }
.gemas-head {
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

.tablero {
  position: relative;
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 4px;
  padding: 8px;
  border-radius: 16px;
  border: 1px solid #243448;
  background:
    linear-gradient(180deg, rgba(16, 185, 129, .05), transparent 55%),
    #080b10;
  box-shadow: inset 0 14px 30px rgba(0, 0, 0, .45);
}
.tablero.animando { pointer-events: none; }
.celda {
  aspect-ratio: 1;
  border-radius: 10px;
  background: rgba(255, 255, 255, .03);
  border: 1px solid rgba(255, 255, 255, .05);
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: transform .12s ease, background .12s ease, border-color .12s ease;
}
.celda:hover { background: rgba(255, 255, 255, .07); }
.celda.seleccionada {
  border-color: var(--gold);
  box-shadow: 0 0 0 2px var(--gold), 0 0 18px rgba(245, 185, 66, .5);
  transform: scale(1.05);
}
.celda.cayendo .gema { animation: caer .22s ease-out; }
@keyframes caer {
  from { transform: translateY(-28%); opacity: .35; }
  to { transform: translateY(0); opacity: 1; }
}

.gema {
  position: relative;
  width: 76%;
  height: 76%;
  display: grid;
  place-items: center;
  box-shadow: inset -2px -3px 6px rgba(0, 0, 0, .35), inset 2px 3px 6px rgba(255, 255, 255, .35), 0 2px 6px rgba(0, 0, 0, .4);
}
.brillo {
  position: absolute;
  top: 12%;
  left: 16%;
  width: 34%;
  height: 22%;
  border-radius: 50%;
  background: rgba(255, 255, 255, .75);
  filter: blur(.5px);
  opacity: .85;
}
.t0 { /* esmeralda: óvalo */ border-radius: 48% 52% 46% 54% / 58% 44% 56% 42%; background: linear-gradient(150deg, #8ffcd0, #0f9d6b 60%, #065f46); }
.t1 { /* zafiro: gota */ border-radius: 50% 50% 50% 6px; transform: rotate(45deg); background: linear-gradient(150deg, #bfe4ff, #2277cc 60%, #0d3f7d); }
.t1 .brillo { transform: rotate(-45deg); }
.t2 { /* amatista: cuadrado redondeado */ border-radius: 24%; background: linear-gradient(150deg, #e6d0ff, #7b3fd1 60%, #3d1478); }
.t3 { /* rubí: cuadrado con swirl */ border-radius: 24%; background: linear-gradient(150deg, #ffc2c2, #d62839 60%, #7a0c18); }
.t4 { /* topacio: diamante */ border-radius: 14%; transform: rotate(45deg); background: linear-gradient(150deg, #fff6bf, #e0a200 60%, #8a5b00); }
.t4 .brillo { transform: rotate(-45deg); }
.t5 { /* naranja: círculo */ border-radius: 50%; background: linear-gradient(150deg, #ffd9a8, #e2680a 60%, #8a3a04); }
.t6 { /* diamante: hexágono */ clip-path: polygon(25% 5%, 75% 5%, 100% 50%, 75% 95%, 25% 95%, 0 50%); background: linear-gradient(150deg, #ffffff, #b9cddb 60%, #5f7a92); }
.t6 .brillo { clip-path: polygon(25% 5%, 75% 5%, 100% 50%, 75% 95%, 25% 95%, 0 60%); }
.t7 { /* gema rayo: esfera arcoíris */ border-radius: 50%; background: conic-gradient(#7cf7c4, #38bdf8, #a78bfa, #f5b942, #ff7a9c, #7cf7c4); animation: girar 2.4s linear infinite; }
.t7 .brillo { width: 40%; height: 40%; background: #fff7d6; opacity: .95; }
.t7 .rayo {
  position: absolute;
  inset: -26%;
  background:
    conic-gradient(from 0deg, transparent 0 12deg, rgba(255,255,255,.95) 12deg 18deg, transparent 18deg 90deg, rgba(255,255,255,.95) 90deg 96deg, transparent 96deg 180deg, rgba(255,255,255,.95) 180deg 186deg, transparent 186deg 270deg, rgba(255,255,255,.95) 270deg 276deg, transparent 276deg 360deg);
  -webkit-mask: radial-gradient(circle, transparent 34%, #000 36%);
  mask: radial-gradient(circle, transparent 34%, #000 36%);
  animation: girar 3.2s linear infinite reverse;
}
@keyframes girar { to { transform: rotate(360deg); } }

.vacio {
  grid-column: 1 / -1;
  display: flex; flex-direction: column; align-items: center; gap: 4px;
  padding: 40px 16px; text-align: center; color: var(--muted); font-size: 14px;
}
.vacio small { color: var(--faint); font-size: 12px; }

.aviso {
  padding: 8px 12px; border-radius: 10px; font-size: 12px;
  background: rgba(245, 185, 66, .12); border: 1px solid rgba(245, 185, 66, .35); color: var(--gold);
}

.controles { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.controles .btn { flex: 1 1 140px; }
.paytable { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11px; color: var(--faint); line-height: 1.5; }
.ayuda { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11.5px; color: var(--faint); }

.resultado { text-align: center; padding: 10px; border-radius: 10px; background: rgba(0,0,0,.3); border: 1px solid var(--border-soft); font-size: 14px; color: var(--muted); }
.resultado.gana { background: var(--prim-dim); border-color: rgba(16,185,129,.35); color: var(--prim-2); font-weight: 700; }

.guia-detalle { border-radius: 12px; border: 1px solid var(--border-soft); background: rgba(255,255,255,.04); }
.guia-detalle > summary { cursor: pointer; padding: 9px 12px; font-size: 11px; letter-spacing: .14em; text-transform: uppercase; color: var(--gold); list-style: none; }
.guia-detalle > summary::-webkit-details-marker { display: none; }
.tabla { padding: 10px 12px; display: flex; flex-direction: column; gap: 5px; }
.fila { display: flex; align-items: center; justify-content: space-between; font-size: 12.5px; color: var(--muted); }
.fila b { color: var(--text); font-family: var(--display); }
.muestra { display: inline-flex; align-items: center; gap: 8px; }
.punto { width: 14px; height: 14px; display: inline-block; box-shadow: inset -1px -2px 3px rgba(0,0,0,.35), inset 1px 2px 3px rgba(255,255,255,.35); }
.punto.t0 { border-radius: 48% 52% 46% 54% / 58% 44% 56% 42%; background: linear-gradient(150deg, #8ffcd0, #0f9d6b); }
.punto.t1 { border-radius: 50% 50% 50% 4px; background: linear-gradient(150deg, #bfe4ff, #2277cc); }
.punto.t2 { border-radius: 24%; background: linear-gradient(150deg, #e6d0ff, #7b3fd1); }
.punto.t3 { border-radius: 24%; background: linear-gradient(150deg, #ffc2c2, #d62839); }
.punto.t4 { border-radius: 14%; transform: rotate(45deg); background: linear-gradient(150deg, #fff6bf, #e0a200); }
.punto.t5 { border-radius: 50%; background: linear-gradient(150deg, #ffd9a8, #e2680a); }
.punto.t6 { clip-path: polygon(25% 5%, 75% 5%, 100% 50%, 75% 95%, 25% 95%, 0 60%); background: linear-gradient(150deg, #ffffff, #b9cddb); }
.punto.t7 { border-radius: 50%; background: conic-gradient(#7cf7c4, #38bdf8, #a78bfa, #f5b942, #ff7a9c, #7cf7c4); }
.nota { margin: 6px 0 0; font-size: 11px; color: var(--faint); line-height: 1.5; }
.error-text { color: #f87171; font-size: 12px; margin: 0; }
</style>
