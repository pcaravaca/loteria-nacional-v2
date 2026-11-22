<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { colones } from '@/utils/formato'
import { reproducir, silenciado } from '@/utils/sfx'
import {
  cantarBolilla,
  callar,
  fijarPresetVoz,
  presetPorId,
  presetVoz,
  PRESETS_VOZ,
} from '@/utils/vozBingo'
import {
  bingoVariante,
  celdasGanadoras,
  etiquetaBolilla,
  type BingoVarianteCfg,
} from './bingoVariantes'

const props = defineProps<{ monto: number; clientSeed: string; variante?: string }>()

const auth = useAuthStore()

interface BingoEstado {
  ronda_id: number
  variante: string
  nombre_variante: string
  monto: number
  monto_por_carton: number
  cartones: number
  grids: number[][][]
  orden: number[]
  cantadas: number[]
  bolilla: number
  cantidad_cantadas: number
  tope_cantadas: number
  bolillas: number
  filas: number
  columnas: number
  centro_libre: boolean
  estado: string
  lineas: number
  lleno: boolean
  patron: string
  tiers: number[]
  premio: number
  premio_pagado: number
  multiplicador: number
  saldo: number
  server_seed_hash: string
  nonce: number
}

const cfg = computed<BingoVarianteCfg>(() => bingoVariante(props.variante ?? 'bingo'))

const cartonesQty = ref(1)
const ronda = ref<BingoEstado | null>(null)
const cargando = ref(false)
const error = ref('')
// El canto es automático por defecto: se puede pausar o cantar a mano.
const auto = ref(true)
// Voz que canta las bolillas (estilo Waze: el cliente elige el preset).
const vozId = ref(presetVoz())
const reloj = ref(0)
let timeout: number | null = null
let ticker: number | null = null
let proximoEn = 0

const saldo = computed(() => auth.user?.saldo ?? 0)
const totalApuesta = computed(() => props.monto * cartonesQty.value)
const cantadas = computed(() => ronda.value?.cantadas ?? [])
const bolilla = computed(() => ronda.value?.bolilla ?? 0)
const estado = computed(() => ronda.value?.estado ?? '')
const tiers = computed(() => ronda.value?.tiers ?? [])
const premio = computed(() => ronda.value?.premio ?? 0)
const premioPagado = computed(() => ronda.value?.premio_pagado ?? 0)
const abierta = computed(() => estado.value === 'abierta')
const tope = computed(() => ronda.value?.tope_cantadas ?? cfg.value.tope)
const cantadasSet = computed(() => new Set(cantadas.value))
const puedeCantar = computed(
  () => abierta.value && cantadas.value.length < tope.value && !cargando.value,
)

// Tablero completo de bolillas: se dibuja apenas se abre la ventana, sin esperar
// al primer canto.
const bolitasTablero = computed(() => {
  const out: number[] = []
  for (let n = 1; n <= cfg.value.bolillas; n++) out.push(n)
  return out
})

// El pozo de bolillas es una tira ancha (15 columnas en el bingo de 75), no una
// columna de 15 filas: antes reusaba las columnas del cartón y quedaba altísimo.
const tableroColumnas = computed(() =>
  Math.max(1, Math.round(cfg.value.bolillas / Math.max(1, cfg.value.columnas))),
)

// Casillas marcadas por cartón, según las bolillas cantadas. El centro es libre.
const marcadas = computed<boolean[][][]>(() => {
  const grids = ronda.value?.grids ?? []
  const vistos = cantadasSet.value
  const cm = Math.floor(cfg.value.filas / 2)
  const cc = Math.floor(cfg.value.columnas / 2)
  return grids.map((grid) =>
    grid.map((fila, r) =>
      fila.map(
        (num, c) =>
          (cfg.value.centroLibre && r === cm && c === cc) ||
          (num !== 0 && vistos.has(num)),
      ),
    ),
  )
})

// Casillas que formaron el patrón ganador de cada cartón.
const ganadoras = computed<Set<string>[]>(() => {
  const grids = ronda.value?.grids ?? []
  return grids.map((grid, n) => celdasGanadoras(cfg.value, grid, marcadas.value[n] ?? []))
})

// Bolitas que se mezclan en la tómbola. Las posiciones son fijas (no saltan al
// re-render) y el tambor gira con CSS.
function pseudo(n: number) {
  const x = Math.sin(n * 12.9898) * 43758.5453
  return x - Math.floor(x)
}
const bolitas = computed(() =>
  Array.from({ length: 18 }, (_, i) => ({
    left: `${14 + pseudo(i + 1) * 68}%`,
    top: `${14 + pseudo(i + 50) * 68}%`,
    background: `hsl(${Math.round(pseudo(i + 99) * 360)} 70% 55%)`,
  })),
)

function etiqueta(n: number) {
  return etiquetaBolilla(cfg.value, n)
}

function demora() {
  return 1500 + Math.random() * 1000
}

function elegirVoz(id: string) {
  vozId.value = id
  fijarPresetVoz(id)
  reproducir('click')
}

function probarVoz() {
  const texto = ronda.value?.bolilla ? etiqueta(ronda.value.bolilla) : 'B 12'
  cantarBolilla(`¡${texto}!`, vozId.value, true)
}

function detenerTimers() {
  if (timeout !== null) {
    window.clearTimeout(timeout)
    timeout = null
  }
  if (ticker !== null) {
    window.clearInterval(ticker)
    ticker = null
  }
}

function programar() {
  detenerTimers()
  if (!auto.value || !abierta.value) return
  const d = demora()
  proximoEn = Date.now() + d
  reloj.value = d / 1000
  timeout = window.setTimeout(() => {
    void cantoAutomatico()
  }, d)
  ticker = window.setInterval(() => {
    reloj.value = Math.max(0, (proximoEn - Date.now()) / 1000)
  }, 100)
}

function pausarAuto() {
  auto.value = false
  detenerTimers()
  reloj.value = 0
}

function reanudarAuto() {
  if (!abierta.value) return
  auto.value = true
  reproducir('click')
  programar()
}

async function cantoAutomatico() {
  if (!auto.value || cargando.value || !puedeCantar.value) return
  await cantar()
  if (auto.value && abierta.value) programar()
}

async function empezar() {
  error.value = ''
  cargando.value = true
  try {
    const { data } = await api.post<BingoEstado>('/casino/bingo/start', {
      monto: props.monto,
      cartones: cartonesQty.value,
      client_seed: props.clientSeed,
      variante: props.variante ?? 'bingo',
    })
    ronda.value = data
    if (auth.user) auth.user.saldo = data.saldo
    reproducir('tambor')
    auto.value = true
    programar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
  } finally {
    cargando.value = false
  }
}

async function cantar(manual = false) {
  if (!ronda.value || cargando.value || !puedeCantar.value) return
  // Si el jugador canta a mano, se respeta su ritmo: se pausa el automático.
  if (manual) pausarAuto()
  error.value = ''
  cargando.value = true
  const antes = marcadas.value.flat(2).filter(Boolean).length
  const cantadasAntes = cantadas.value.length
  try {
    const { data } = await api.post<BingoEstado>(
      `/casino/bingo/${ronda.value.ronda_id}/hit`,
      {},
    )
    ronda.value = data
    if (auth.user) auth.user.saldo = data.saldo
    reproducir('bolilla')
    // La bolilla cantada en voz alta con el preset elegido (si salio una nueva).
    if (data.bolilla > 0 && data.cantadas.length > cantadasAntes) {
      cantarBolilla(`¡${etiqueta(data.bolilla)}!`, vozId.value)
    }
    const despues = marcadas.value.flat(2).filter(Boolean).length
    if (despues > antes) reproducir('acierto')
    if (data.estado !== 'abierta') {
      detenerTimers()
      auto.value = false
      if (data.premio_pagado > 0) reproducir(data.lleno ? 'bingo' : 'ganar')
      else reproducir('perder')
    }
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
    detenerTimers()
    auto.value = false
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
    detenerTimers()
    auto.value = false
    if (auth.user) auth.user.saldo = data.saldo
    if (data.premio_pagado > 0) reproducir(data.lleno ? 'bingo' : 'ganar')
    else reproducir('perder')
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
  } finally {
    cargando.value = false
  }
}

function nueva() {
  detenerTimers()
  ronda.value = null
  error.value = ''
  auto.value = true
}

onUnmounted(() => {
  detenerTimers()
  callar()
})
</script>

<template>
  <div class="bingo">
    <div class="bingo-head">
      <span class="nombre">{{ cfg.nombre }}</span>
      <span>Saldo <b>{{ colones(saldo) }}</b></span>
      <span>Apuesta <b>{{ colones(totalApuesta) }}</b></span>
      <span>Premio <b>{{ colones(premio) }}</b></span>
      <span v-if="auto && abierta" class="reloj">Próxima en <b>{{ reloj.toFixed(1) }} s</b></span>
      <span v-else-if="abierta" class="reloj pausa">Automático en pausa</span>
    </div>

    <div class="bingo-cuerpo">
      <div class="zap">
        <div class="tombola" aria-hidden="true">
          <div class="tambor">
            <span v-for="(b, i) in bolitas" :key="i" class="bolita" :style="b"></span>
          </div>
          <div :key="bolilla" class="bola-grande">{{ etiqueta(bolilla) }}</div>
        </div>
        <div class="registro">
          <span v-for="b in cantadas" :key="b" class="bola" :class="{ ultima: b === bolilla }">
            {{ b }}
          </span>
          <span v-if="!cantadas.length" class="vacio">Todavía no se cantó ninguna bolilla</span>
        </div>
      </div>

      <div class="zona">
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
            <div class="cab" :style="{ gridTemplateColumns: `repeat(${cfg.columnas}, 1fr)` }">
              <span v-for="l in cfg.letras" :key="l">{{ l }}</span>
            </div>
            <div
              v-for="(fila, r) in grid"
              :key="r"
              class="fila"
              :style="{ gridTemplateColumns: `repeat(${cfg.columnas}, 1fr)` }"
            >
              <div
                v-for="(num, c) in fila"
                :key="c"
                class="celda"
                :class="{
                  libre: num === 0,
                  marcada: marcadas[n]?.[r]?.[c],
                  'resalta-ganador': ganadoras[n]?.has(r + ',' + c),
                }"
              >
                <span v-if="num === 0">★</span>
                <span v-else>{{ num }}</span>
              </div>
            </div>
          </div>
        </div>

        <div
          class="tablero"
          :style="{ gridTemplateColumns: `repeat(${tableroColumnas}, minmax(0, 1fr))` }"
        >
          <button
            v-for="n in bolitasTablero"
            :key="n"
            type="button"
            class="casilla"
            :class="{ 'resalta-ganador': cantadasSet.has(n), ultima: n === bolilla }"
            disabled
          >
            {{ n }}
          </button>
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
      <p class="paytable">{{ cfg.descripcion }}</p>
    </div>

    <div v-else-if="abierta" class="controles">
      <button class="btn btn-primary" :disabled="!puedeCantar" @click="cantar(true)">
        Cantar ahora
      </button>
      <button class="btn btn-ghost" :disabled="cargando || cantadas.length === 0" @click="cerrar">
        Cobrar {{ colones(premio) }}
      </button>
      <button class="btn btn-ghost" :disabled="cargando" @click="auto ? pausarAuto() : reanudarAuto()">
        {{ auto ? 'Pausar automático' : 'Reanudar automático' }}
      </button>
    </div>

    <div v-else class="controles">
      <button class="btn btn-primary btn-block" :disabled="cargando" @click="nueva">
        Jugar de nuevo
      </button>
    </div>

    <div class="voz">
      <span class="voz-etq">Voz de la bolilla</span>
      <div class="voz-chips">
        <button
          v-for="p in PRESETS_VOZ"
          :key="p.id"
          type="button"
          class="voz-chip"
          :class="{ activo: vozId === p.id }"
          :title="p.descripcion"
          @click="elegirVoz(p.id)"
        >{{ p.nombre }}</button>
        <button type="button" class="voz-probar" title="Escuchar una bolilla de muestra" @click="probarVoz">
          ▶ Probar
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.bingo { display: flex; flex-direction: column; gap: 10px; }

/* Info arriba, en una sola fila: antes era una columna que ocupaba media pantalla. */
.bingo-head {
  display: flex; align-items: center; gap: 14px; flex-wrap: wrap;
  padding: 8px 12px; border-radius: 12px;
  background: rgba(3, 16, 12, .75); border: 1px solid rgba(232, 206, 122, .35);
  font-size: 12px; color: var(--faint);
}
.bingo-head .nombre { font-size: 13px; font-weight: 700; color: var(--gold); margin-right: 4px; }
.bingo-head b { color: var(--text); margin-left: 4px; }
.bingo-head .reloj b { color: var(--prim-2); }
.bingo-head .reloj.pausa { color: var(--muted); font-style: italic; }

/* Tómbola a la par del cartón: el cartón era la sección más alta y se llevaba
   todo el espacio vertical. */
.bingo-cuerpo { display: flex; gap: 14px; align-items: flex-start; }
.zap { flex: 0 0 148px; display: flex; flex-direction: column; gap: 10px; }
.zona { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 10px; }

/* Tómbola: tambor que gira con bolitas mezclándose y la bola cantada afuera. */
.tombola {
  position: relative; width: 148px; height: 148px; margin: 0 auto;
  border-radius: 50%; overflow: hidden;
  background: radial-gradient(circle at 35% 30%, #1d3b2a, #0a1a12 70%);
  border: 3px solid rgba(232, 206, 122, .55);
  box-shadow: inset 0 0 24px rgba(0, 0, 0, .6), 0 6px 18px rgba(0, 0, 0, .4);
}
.tombola::after {
  content: ''; position: absolute; inset: 0; border-radius: 50%;
  background: radial-gradient(circle at 50% 45%, rgba(255, 255, 255, .12), transparent 55%);
  pointer-events: none;
}
.tambor { position: absolute; inset: 0; animation: girar 7s linear infinite; }
.bolita {
  position: absolute; width: 17px; height: 17px; border-radius: 50%;
  transform: translate(-50%, -50%); box-shadow: inset -2px -2px 4px rgba(0, 0, 0, .35);
}
.bola-grande {
  position: absolute; left: 50%; top: 50%; transform: translate(-50%, -50%);
  min-width: 80px; height: 80px; padding: 0 10px; border-radius: 50%;
  display: grid; place-items: center; text-align: center;
  background: linear-gradient(180deg, #f5dd9a, #c9a227); color: #3a2a05;
  font-size: 22px; font-weight: 800; letter-spacing: .02em;
  box-shadow: 0 6px 16px rgba(0, 0, 0, .5), inset 0 -6px 10px rgba(0, 0, 0, .15);
  animation: pop .35s ease;
}
@keyframes girar { to { transform: rotate(360deg); } }
@keyframes pop {
  0% { transform: translate(-50%, -50%) scale(.2); opacity: 0; }
  70% { transform: translate(-50%, -50%) scale(1.12); opacity: 1; }
  100% { transform: translate(-50%, -50%) scale(1); opacity: 1; }
}

.registro { display: flex; flex-wrap: wrap; gap: 5px; min-height: 28px; align-content: flex-start; }
.bola {
  min-width: 28px; height: 28px; padding: 0 6px; border-radius: 999px; display: grid; place-items: center;
  background: var(--surface); border: 1px solid var(--border-soft); font-size: 12px; color: var(--muted);
}
.bola.ultima { background: var(--prim-dim); border-color: var(--prim); color: var(--prim-2); font-weight: 700; }
.vacio { font-size: 12px; color: var(--faint); font-style: italic; }

/* Cartón de papel: se ve como el cartón físico de salón, no como una tabla. */
.cartones { display: grid; gap: 10px; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); }
.cartones.n1 { grid-template-columns: minmax(210px, 300px); }
.carton {
  padding: 10px; border-radius: 6px;
  background: linear-gradient(180deg, #f8f3e3, #e6dec6);
  border: 1px solid #b9ad8c;
  box-shadow: 0 8px 20px rgba(0, 0, 0, .45), inset 0 0 40px rgba(120, 100, 60, .12);
}
.carton.lleno { border-color: #c9a227; box-shadow: 0 0 0 1px #c9a227, 0 0 24px rgba(245, 185, 66, .4); }
.carton-head {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 6px; font-size: 11px; letter-spacing: .1em; text-transform: uppercase;
  color: #7a6a3f; font-weight: 700;
}
.carton-head b { color: #8a6d1f; font-size: 14px; }
.cab { display: grid; gap: 4px; margin-bottom: 4px; }
.cab span { text-align: center; font-size: 12px; font-weight: 800; letter-spacing: .18em; color: #8a6d1f; }
.fila { display: grid; gap: 4px; }
.celda {
  height: 34px; border-radius: 3px; display: grid; place-items: center;
  background: #fffdf5; border: 1px solid #c8bd9c;
  font-size: 14px; font-weight: 600; color: #2b2b2b;
  transition: transform .12s, background .12s;
}
.celda.libre { background: #d8ecd9; color: #2f6b45; font-size: 16px; }
.celda.marcada {
  background: #1f8f5f; border-color: #126843; color: #fff; font-weight: 800;
  box-shadow: inset 0 -2px 0 rgba(0, 0, 0, .18);
}

.tablero { display: grid; gap: 3px; }
.casilla {
  height: 22px; border-radius: 5px; border: 1px solid var(--border-soft);
  background: var(--surface); color: var(--muted); font-size: 10.5px; font-weight: 600;
  cursor: default; padding: 0;
}
.casilla:disabled { opacity: 1; }
.casilla.ultima { border-color: var(--gold); color: var(--gold); }

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

/* Voz de la bolilla: presets configurables por el cliente (estilo Waze). */
.voz {
  display: flex; align-items: center; gap: 8px; flex-wrap: wrap;
  padding: 8px 10px; border-radius: 12px;
  background: rgba(3, 16, 12, .5); border: 1px solid var(--border-soft);
}
.voz-etq { font-size: 11px; letter-spacing: .12em; text-transform: uppercase; color: var(--faint); }
.voz-chips { display: flex; gap: 6px; flex-wrap: wrap; align-items: center; }
.voz-chip {
  border: 1px solid var(--border-soft); background: var(--surface); color: var(--muted);
  border-radius: 999px; padding: 6px 11px; font-size: 12px; cursor: pointer;
}
.voz-chip:hover { color: var(--text); border-color: var(--border); }
.voz-chip.activo {
  border-color: var(--gold); color: var(--gold); font-weight: 700;
  background: var(--gold-dim);
}
.voz-probar {
  border: 1px solid var(--prim); background: var(--prim-dim); color: var(--prim-2);
  border-radius: 999px; padding: 6px 11px; font-size: 12px; font-weight: 700; cursor: pointer;
}

/* Variante C (elegida por Peter): en pantallas chicas la tómbola se achica a
   104 px y se pone a la par del registro, el cartón ocupa el ancho completo con
   celdas altas y el bolillero queda como tira: entra todo sin scrollear. */
@media (max-width: 620px) {
  .bingo { gap: 8px; }
  .bingo-head { gap: 10px; padding: 6px 10px; font-size: 11.5px; }
  .bingo-cuerpo { flex-direction: column; gap: 8px; }
  .zap { flex: 1 1 auto; flex-direction: row; align-items: center; gap: 10px; width: 100%; }
  .tombola { width: 104px; height: 104px; margin: 0; flex: 0 0 auto; }
  .bola-grande { min-width: 52px; height: 52px; font-size: 15px; }
  .registro { flex: 1 1 auto; gap: 4px; }
  .bola { min-width: 24px; height: 24px; padding: 0 5px; font-size: 11px; }
  .zona { gap: 8px; }
  .cartones { grid-template-columns: 1fr; gap: 8px; }
  .carton { padding: 8px; }
  .celda { height: 44px; font-size: 16px; border-radius: 4px; }
  .cab span { font-size: 13px; }
  .casilla { height: 24px; font-size: 10.5px; }
  .controles .btn { flex: 1 1 100%; }
  .voz { padding: 6px 8px; }
  .voz-chips { width: 100%; }
}

.error-text { color: #f87171; font-size: 12px; margin: 0; }
</style>
