<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'

const props = defineProps<{ monto: number; clientSeed: string }>()

const auth = useAuthStore()

const CANTADAS = 42
const LETRAS = ['B', 'I', 'N', 'G', 'O']

interface BingoEstado {
  ronda_id: number
  monto: number
  monto_por_carton: number
  cartones: number
  grids: number[][][]
  orden: number[]
  cantadas: number[]
  bolilla: number
  cantidad_cantadas: number
  estado: string
  lineas: number
  lleno: boolean
  tiers: number[]
  premio: number
  premio_pagado: number
  multiplicador: number
  saldo: number
  server_seed_hash: string
  nonce: number
}

const cartonesQty = ref(1)
const ronda = ref<BingoEstado | null>(null)
const cargando = ref(false)
const error = ref('')
const auto = ref(false)
let timer: number | null = null

const saldo = computed(() => auth.user?.saldo ?? 0)
const totalApuesta = computed(() => props.monto * cartonesQty.value)
const cantadas = computed(() => ronda.value?.cantadas ?? [])
const bolilla = computed(() => ronda.value?.bolilla ?? 0)
const estado = computed(() => ronda.value?.estado ?? '')
const tiers = computed(() => ronda.value?.tiers ?? [])
const premio = computed(() => ronda.value?.premio ?? 0)
const premioPagado = computed(() => ronda.value?.premio_pagado ?? 0)
const abierta = computed(() => estado.value === 'abierta')
const puedeCantar = computed(() => abierta.value && cantadas.value.length < CANTADAS && !cargando.value)

// Casillas marcadas por cartón, según las bolillas cantadas. El centro es libre.
const marcadas = computed<boolean[][][]>(() => {
  const grids = ronda.value?.grids ?? []
  const vistos = new Set(cantadas.value)
  return grids.map((grid) =>
    grid.map((fila, r) =>
      fila.map((num, c) => (r === 2 && c === 2 ? true : num !== 0 && vistos.has(num))),
    ),
  )
})

function detenerAuto() {
  if (timer !== null) {
    window.clearInterval(timer)
    timer = null
  }
  auto.value = false
}

function alternarAuto() {
  if (auto.value) {
    detenerAuto()
    return
  }
  auto.value = true
  reproducir('click')
  timer = window.setInterval(() => {
    if (!puedeCantar.value) {
      detenerAuto()
      return
    }
    void cantar()
  }, 380)
}

async function empezar() {
  error.value = ''
  cargando.value = true
  try {
    const { data } = await api.post<BingoEstado>('/casino/bingo/start', {
      monto: props.monto,
      cartones: cartonesQty.value,
      client_seed: props.clientSeed,
    })
    ronda.value = data
    reproducir('tambor')
    await auth.cargar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
  } finally {
    cargando.value = false
  }
}

async function cantar() {
  if (!ronda.value || cargando.value) return
  error.value = ''
  cargando.value = true
  const antes = marcadas.value.flat(2).filter(Boolean).length
  try {
    const { data } = await api.post<BingoEstado>(`/casino/bingo/${ronda.value.ronda_id}/hit`)
    ronda.value = data
    reproducir('bolilla')
    const despues = marcadas.value.flat(2).filter(Boolean).length
    if (despues > antes) reproducir('acierto')
    if (data.estado !== 'abierta') {
      detenerAuto()
      if (data.premio_pagado > 0) reproducir(data.lleno ? 'bingo' : 'ganar')
      else reproducir('perder')
    }
    await auth.cargar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
    detenerAuto()
  } finally {
    cargando.value = false
  }
}

async function cerrar() {
  if (!ronda.value || cargando.value) return
  error.value = ''
  cargando.value = true
  try {
    const { data } = await api.post<BingoEstado>(`/casino/bingo/${ronda.value.ronda_id}/close`)
    ronda.value = data
    detenerAuto()
    if (data.premio_pagado > 0) reproducir(data.lleno ? 'bingo' : 'ganar')
    else reproducir('perder')
    await auth.cargar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
  } finally {
    cargando.value = false
  }
}

function nueva() {
  detenerAuto()
  ronda.value = null
  error.value = ''
}

onUnmounted(detenerAuto)
</script>

<template>
  <div class="bingo">
    <div class="bingo-head">
      <div class="bolillero">
        <span class="etq">Bolilla</span>
        <strong class="bolilla">{{ bolilla || '—' }}</strong>
        <span class="cantadas">{{ cantadas.length }}/{{ CANTADAS }}</span>
      </div>
      <div class="info">
        <span>Saldo <b>{{ colones(saldo) }}</b></span>
        <span>Apuesta <b>{{ colones(totalApuesta) }}</b></span>
        <span>Premio <b>{{ colones(premio) }}</b></span>
      </div>
    </div>

    <div class="bolas">
      <span
        v-for="b in cantadas"
        :key="b"
        class="bola"
        :class="{ ultima: b === bolilla }"
      >{{ b }}</span>
      <span v-if="!cantadas.length" class="vacio">Todavía no se cantó ninguna bolilla</span>
    </div>

    <div v-if="ronda" class="cartones" :class="'n' + ronda.grids.length">
      <div
        v-for="(grid, n) in ronda.grids"
        :key="n"
        class="carton"
        :class="{ lleno: tiers[n] === 25 }"
      >
        <div class="carton-head">
          <span>Cartón {{ n + 1 }}</span>
          <b v-if="tiers[n]">x{{ tiers[n] }}</b>
        </div>
        <div class="cab">
          <span v-for="l in LETRAS" :key="l">{{ l }}</span>
        </div>
        <div v-for="(fila, r) in grid" :key="r" class="fila">
          <div
            v-for="(num, c) in fila"
            :key="c"
            class="celda"
            :class="{
              libre: num === 0,
              marcada: marcadas[n]?.[r]?.[c],
              'resalta-ganador': marcadas[n]?.[r]?.[c],
            }"
          >
            <span v-if="num === 0">★</span>
            <span v-else>{{ num }}</span>
          </div>
        </div>
      </div>
    </div>

    <div v-if="estado && estado !== 'abierta'" class="resultado" :class="{ gana: premioPagado > 0 }">
      <span v-if="premioPagado > 0">¡Bingo! Ganaste {{ colones(premioPagado) }}</span>
      <span v-else>Sin premio en esta jugada</span>
    </div>

    <p v-if="error" class="error-text">{{ error }}</p>

    <div v-if="!ronda" class="controles">
      <div class="selector">
        <button
          v-for="n in [1, 2, 3, 4]"
          :key="n"
          type="button"
          :class="{ activo: cartonesQty === n }"
          @click="cartonesQty = n; reproducir('click')"
        >{{ n }}</button>
        <span class="selector-etq">cartones</span>
      </div>
      <button class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
        {{ cargando ? 'Abriendo…' : 'Empezar por ' + colones(totalApuesta) }}
      </button>
      <p class="paytable">1 línea x1 · 2 o más líneas x3 · cartón lleno x25</p>
    </div>

    <div v-else-if="abierta" class="controles">
      <button class="btn btn-primary" :disabled="!puedeCantar" @click="cantar">Cantar</button>
      <button class="btn btn-ghost" :disabled="cargando || cantadas.length === 0" @click="cerrar">
        Cobrar {{ colones(premio) }}
      </button>
      <button class="btn btn-ghost" :disabled="cargando" @click="alternarAuto">
        {{ auto ? 'Detener' : 'Automático' }}
      </button>
    </div>

    <div v-else class="controles">
      <button class="btn btn-primary btn-block" :disabled="cargando" @click="nueva">
        Jugar de nuevo
      </button>
    </div>
  </div>
</template>

<style scoped>
.bingo { display: flex; flex-direction: column; gap: 10px; }
.bingo-head {
  display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap;
  padding: 10px 12px; border-radius: 12px;
  background: rgba(3, 16, 12, .75); border: 1px solid rgba(232, 206, 122, .35);
}
.bolillero { display: flex; align-items: center; gap: 10px; }
.bolillero .etq { font-size: 11px; letter-spacing: .16em; text-transform: uppercase; color: var(--faint); }
.bolilla {
  min-width: 44px; height: 44px; padding: 0 8px; border-radius: 999px; display: grid; place-items: center;
  background: linear-gradient(180deg, #f5dd9a, #c9a227); color: #3a2a05;
  font-size: 20px; font-weight: 800; box-shadow: 0 4px 12px rgba(0, 0, 0, .4);
}
.cantadas { font-size: 12px; color: var(--muted); }
.info { display: flex; gap: 14px; font-size: 12px; color: var(--faint); flex-wrap: wrap; }
.info b { color: var(--text); margin-left: 4px; }

.bolas { display: flex; flex-wrap: wrap; gap: 5px; min-height: 28px; align-items: center; }
.bola {
  min-width: 28px; height: 28px; padding: 0 6px; border-radius: 999px; display: grid; place-items: center;
  background: var(--surface); border: 1px solid var(--border-soft); font-size: 12px; color: var(--muted);
}
.bola.ultima { background: var(--prim-dim); border-color: var(--prim); color: var(--prim-2); font-weight: 700; }
.vacio { font-size: 12px; color: var(--faint); font-style: italic; }

.cartones { display: grid; gap: 10px; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); }
.cartones.n1 { grid-template-columns: minmax(210px, 260px); justify-content: center; }
.carton {
  padding: 10px; border-radius: 14px; background: rgba(255, 255, 255, .03);
  border: 1px solid var(--border-soft);
}
.carton.lleno { border-color: var(--gold); box-shadow: 0 0 0 1px var(--gold), 0 0 24px rgba(245, 185, 66, .35); }
.carton-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; font-size: 12px; color: var(--faint); }
.carton-head b { color: var(--gold); font-size: 15px; }
.cab, .fila { display: grid; grid-template-columns: repeat(5, 1fr); gap: 4px; }
.cab { margin-bottom: 4px; }
.cab span { text-align: center; font-size: 11px; font-weight: 800; letter-spacing: .1em; color: var(--gold); }
.celda {
  height: 34px; border-radius: 8px; display: grid; place-items: center;
  background: var(--surface); border: 1px solid var(--border-soft);
  font-size: 13px; color: var(--text); transition: transform .12s, background .12s;
}
.celda.libre { background: var(--prim-dim); color: var(--prim-2); font-size: 15px; }
.celda.marcada { background: rgba(16, 185, 129, .18); border-color: rgba(16, 185, 129, .5); color: var(--prim-2); font-weight: 700; }

.resultado { text-align: center; padding: 10px; border-radius: 10px; background: rgba(0, 0, 0, .3); border: 1px solid var(--border-soft); font-size: 14px; color: var(--muted); }
.resultado.gana { background: var(--prim-dim); border-color: rgba(16, 185, 129, .35); color: var(--prim-2); font-weight: 700; }

.controles { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.controles .btn { flex: 1 1 110px; }
.selector { display: flex; align-items: center; gap: 6px; flex: 1 1 100%; }
.selector button {
  width: 40px; height: 34px; border-radius: 8px; border: 1px solid var(--border-soft);
  background: var(--surface); color: var(--muted); font-weight: 700; cursor: pointer;
}
.selector button.activo { background: var(--prim-dim); border-color: var(--prim); color: var(--prim-2); }
.selector-etq { font-size: 12px; color: var(--faint); margin-left: 4px; }
.paytable { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11px; color: var(--faint); }
.error-text { color: #f87171; font-size: 12px; margin: 0; }
</style>
