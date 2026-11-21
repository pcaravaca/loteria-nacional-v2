<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'
import { rtpDe, rtpTexto } from '@/utils/rtpCasino'

const props = defineProps<{ monto: number; clientSeed: string }>()
const auth = useAuthStore()

// RTP declarado para este juego (seed del casino): jugando bien y al azar.
const rtp = rtpDe('domino')!

interface Ficha {
  0: number
  1: number
}

interface DominoEstado {
  ronda_id: number
  monto: number
  mano: Ficha[]
  casa: Ficha[] | null
  casa_fichas: number
  tablero: Ficha[]
  extremos: [number, number]
  turno: string
  estado: string
  multiplicador: number
  premio: number
  premio_pagado: number
  jugadas: [number, number][]
  saldo: number
  server_seed_hash: string
  nonce: number
}

// ---------------------------------------------------------------- muestra ---
// La ventana se abre con una mano y una mesa de ejemplo ya dibujadas, para que
// el juego se entienda de una. La muestra es local y no cuesta ni paga nada:
// recien al pulsar Empezar se pide /start y este estado se reemplaza.

function hashCadena(texto: string): number {
  let h = 2166136261 >>> 0
  for (let i = 0; i < texto.length; i++) {
    h ^= texto.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

function rngDeterminista(a: number) {
  return function () {
    a |= 0
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

// Pipas de cada mitad en una grilla 3x3 (indices 0..8).
const PIPOS: number[][] = [
  [],
  [4],
  [0, 8],
  [0, 4, 8],
  [0, 2, 6, 8],
  [0, 2, 4, 6, 8],
  [0, 2, 3, 5, 6, 8],
]

function piposDe(n: number): boolean[] {
  const out = new Array<boolean>(9).fill(false)
  for (const k of PIPOS[n] ?? []) out[k] = true
  return out
}

const ronda = ref<DominoEstado | null>(null)
const cargando = ref(false)
const error = ref('')
const seleccionada = ref<number | null>(null)
// Marca qué fichas de la mesa son de la casa (solo aplica a la muestra).
const duenoPreview = ref<boolean[]>([])

const saldo = computed(() => auth.user?.saldo ?? 0)
const esPreview = computed(() => ronda.value?.estado === 'preview')
const abierta = computed(() => ronda.value?.estado === 'abierta')

const duenoCasa = computed<boolean[]>(() => {
  if (esPreview.value) return duenoPreview.value
  return (ronda.value?.tablero ?? []).map(() => false)
})

function esExtremo(i: number): boolean {
  const t = ronda.value?.tablero ?? []
  return t.length > 0 && (i === 0 || i === t.length - 1)
}

// Genera la mano y la mesa de ejemplo de forma determinista a partir de la
// clientSeed: mismo valor -> misma muestra, y nunca implica apuesta.
function generarPreview() {
  const rng = rngDeterminista(hashCadena('domino-preview:' + props.clientSeed))
  const mazo: Ficha[] = []
  for (let a = 0; a <= 6; a++) {
    for (let b = a; b <= 6; b++) mazo.push({ 0: a, 1: b })
  }
  for (let i = mazo.length - 1; i > 0; i--) {
    const j = Math.floor(rng() * (i + 1))
    const tmp = mazo[i]
    mazo[i] = mazo[j]
    mazo[j] = tmp
  }
  const mano = mazo.slice(0, 7)
  const casa = mazo.slice(7, 14)
  const pool = mazo.slice(14)

  // Cadena de ejemplo: arranca con una ficha y se extiende por un extremo.
  const primera = pool.splice(Math.floor(rng() * pool.length), 1)[0]
  const cadena: Ficha[] = [primera]
  const dueno: boolean[] = [false]
  let izq = primera[0]
  let der = primera[1]
  const target = 6 + Math.floor(rng() * 3)
  while (cadena.length < target && pool.length) {
    const candidatas = pool
      .map((f, i) => ({ f, i }))
      .filter(({ f }) => f[0] === izq || f[1] === izq || f[0] === der || f[1] === der)
    if (!candidatas.length) break
    const { f, i } = candidatas[Math.floor(rng() * candidatas.length)]
    pool.splice(i, 1)
    if (f[0] === der) {
      cadena.push(f)
      der = f[1]
    } else if (f[1] === der) {
      cadena.push({ 0: f[1], 1: f[0] })
      der = f[0]
    } else if (f[0] === izq) {
      cadena.unshift({ 0: f[1], 1: f[0] })
      izq = f[1]
    } else {
      cadena.unshift(f)
      izq = f[0]
    }
    dueno.push(dueno.length % 2 === 1)
  }

  duenoPreview.value = dueno
  ronda.value = {
    ronda_id: 0,
    monto: props.monto,
    mano,
    casa,
    casa_fichas: casa.length,
    tablero: cadena,
    extremos: [izq, der],
    turno: 'jugador',
    estado: 'preview',
    multiplicador: 0,
    premio: 0,
    premio_pagado: 0,
    jugadas: [],
    saldo: auth.user?.saldo ?? 0,
    server_seed_hash: '',
    nonce: 0,
  }
}

generarPreview()
watch(
  () => props.clientSeed,
  () => {
    if (esPreview.value) generarPreview()
  },
)

// ------------------------------------------------------------------ juego ---

function esValida(f: Ficha): boolean {
  if (!ronda.value) return false
  if (ronda.value.tablero.length === 0) return true
  const [izq, der] = ronda.value.extremos
  return f[0] === izq || f[1] === izq || f[0] === der || f[1] === der
}

const jugables = computed(() => {
  const set = new Set<number>()
  if (!ronda.value || esPreview.value) return set
  ronda.value.mano.forEach((f, i) => {
    if (esValida(f)) set.add(i)
  })
  return set
})

const sinJugada = computed(() => abierta.value && jugables.value.size === 0)

async function empezar() {
  error.value = ''
  cargando.value = true
  seleccionada.value = null
  try {
    const { data } = await api.post<DominoEstado>('/casino/domino/start', {
      monto: props.monto,
      client_seed: props.clientSeed,
    })
    ronda.value = data
    reproducir('ficha')
    await auth.cargar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
    reproducir('error')
  } finally {
    cargando.value = false
  }
}

async function jugar(ficha: number) {
  if (!ronda.value || cargando.value || !abierta.value) return
  error.value = ''
  cargando.value = true
  try {
    const { data } = await api.post<DominoEstado>(`/casino/domino/${ronda.value.ronda_id}/move`, { ficha })
    ronda.value = data
    seleccionada.value = null
    reproducir(data.estado === 'pagada' ? (data.premio_pagado > 0 ? 'ganar' : 'perder') : 'click')
    await auth.cargar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
    reproducir('error')
  } finally {
    cargando.value = false
  }
}

function tocarFicha(i: number) {
  if (esPreview.value || !abierta.value || cargando.value) return
  if (!jugables.value.has(i)) {
    error.value = 'Esa ficha no encaja en los extremos.'
    reproducir('error')
    return
  }
  seleccionada.value = i
  reproducir('click')
}
</script>

<template>
  <div class="domino">
    <div class="domino-head">
      <div class="contador">
        <span class="etq">Fichas tuyas</span>
        <strong>{{ ronda?.mano.length ?? 0 }}</strong>
      </div>
      <div class="contador">
        <span class="etq">Fichas casa</span>
        <strong>{{ ronda?.casa_fichas ?? 0 }}</strong>
      </div>
      <div class="contador">
        <span class="etq">Premio</span>
        <strong class="oro">{{ esPreview ? '—' : colones(ronda?.premio ?? 0) }}</strong>
      </div>
      <div class="info">
        <span>Saldo <b>{{ colones(saldo) }}</b></span>
        <span>{{ esPreview ? 'Se cobra al empezar' : 'Apuesta' }} <b>{{ colones(monto) }}</b></span>
      </div>
    </div>

    <div class="mesa">
      <div class="mesa-top">
        <span class="etq">Mesa</span>
        <span class="encaje">
          Encaja
          <b class="extremo-chip">{{ ronda?.extremos[0] ?? '—' }}</b>
          <i>o</i>
          <b class="extremo-chip">{{ ronda?.extremos[1] ?? '—' }}</b>
        </span>
        <span v-if="esPreview" class="badge-muestra">Muestra · sin cargo</span>
        <span v-else-if="ronda?.turno === 'casa'" class="aviso-casa">Jugó la casa…</span>
      </div>

      <div class="tablero" :class="{ vacio: !ronda?.tablero.length }">
        <span v-if="!ronda?.tablero.length" class="vacio-texto">
          Tocá una ficha de tu mano para abrir la mesa.
        </span>
        <div
          v-for="(f, i) in ronda?.tablero ?? []"
          :key="i"
          class="ficha horizontal"
          :class="{
            extremo: esExtremo(i),
            'ultima-casa': duenoCasa[i],
            'resalta-ganador': ronda?.estado === 'pagada' && i === ronda.tablero.length - 1,
          }"
        >
          <div class="mitad">
            <span v-for="(on, k) in piposDe(f[0])" :key="k" class="pipo" :class="{ on }" />
          </div>
          <div class="media-luna" />
          <div class="mitad">
            <span v-for="(on, k) in piposDe(f[1])" :key="k" class="pipo" :class="{ on }" />
          </div>
          <span v-if="duenoCasa[i]" class="titin" title="Ficha de la casa" />
        </div>
      </div>

      <p v-if="esPreview" class="mesa-nota">
        <span class="titin inline" />
        <span>Las fichas con el punto dorado son de la casa. Muestra sin cargo.</span>
      </p>
      <p v-else class="mesa-nota">La casa responde con la ficha válida de mayor puntaje.</p>
    </div>

    <div class="mano-seccion">
      <div class="mano-top">
        <span class="etq">Tu mano</span>
        <span v-if="esPreview" class="mano-hint">Así se ve una mano de 7 fichas.</span>
        <span v-else-if="abierta" class="mano-hint">Tocá una ficha y después <b>Jugar ficha</b>.</span>
      </div>
      <div class="mano">
        <button
          v-for="(f, i) in ronda?.mano ?? []"
          :key="i"
          class="ficha vertical boton"
          :class="{
            seleccionada: seleccionada === i,
            jugable: !esPreview && jugables.has(i),
            'resalta-ganador': seleccionada === i,
          }"
          :disabled="cargando || esPreview"
          @click="tocarFicha(i)"
        >
          <div class="mitad">
            <span v-for="(on, k) in piposDe(f[0])" :key="k" class="pipo" :class="{ on }" />
          </div>
          <div class="media-luna" />
          <div class="mitad">
            <span v-for="(on, k) in piposDe(f[1])" :key="k" class="pipo" :class="{ on }" />
          </div>
        </button>
      </div>
    </div>

    <p class="rtp">
      RTP declarado: <b>jugando bien {{ rtpTexto(rtp.rtp) }}</b> ·
      <b>jugando al azar {{ rtpTexto(rtp.rtp_casual) }}</b>
    </p>

    <p v-if="error" class="error-text">{{ error }}</p>

    <div v-if="esPreview" class="controles">
      <button class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
        {{ cargando ? 'Abriendo…' : 'Empezar por ' + colones(monto) }}
      </button>
      <p class="paytable">Ganaste x1.52 · empate devuelve lo apostado · la casa juega la ficha de mayor puntaje.</p>
    </div>

    <div v-else-if="abierta" class="controles">
      <button class="btn btn-primary" :disabled="cargando || seleccionada === null" @click="jugar(seleccionada ?? -1)">
        Jugar ficha
      </button>
      <button class="btn btn-ghost" :disabled="cargando || !sinJugada" @click="jugar(-1)">
        Pasar
      </button>
      <span class="ayuda">Tocá una ficha y después <b>Jugar ficha</b>. Si no encaja ninguna, pasás.</span>
    </div>

    <div v-else class="controles">
      <button class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
        {{ cargando ? 'Abriendo…' : 'Jugar de nuevo por ' + colones(monto) }}
      </button>
    </div>

    <div
      v-if="ronda && !esPreview && ronda.estado !== 'abierta'"
      class="resultado"
      :class="{ gana: ronda.premio_pagado > 0 }"
    >
      <span v-if="ronda.premio_pagado > 0">¡Ganaste {{ colones(ronda.premio_pagado) }} (x{{ ronda.multiplicador }})</span>
      <span v-else>Sin premio en esta mano</span>
    </div>
  </div>
</template>

<style scoped>
.domino { display: flex; flex-direction: column; gap: 10px; }
.domino-head {
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

/* ------------------------------------------------------------- mesa --- */
.mesa {
  display: flex; flex-direction: column; gap: 7px; padding: 10px;
  border-radius: 16px; border: 1px solid #243448;
  background: linear-gradient(180deg, rgba(16, 185, 129, .06), transparent 60%), #080b10;
}
.mesa-top { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.mesa-top .etq { font-size: 10px; letter-spacing: .14em; text-transform: uppercase; color: var(--faint); }
.encaje { display: inline-flex; align-items: center; gap: 5px; font-size: 12px; color: var(--muted); }
.encaje i { color: var(--faint); font-style: normal; }
.extremo-chip {
  display: inline-grid; place-items: center; min-width: 20px; height: 20px; padding: 0 4px;
  border-radius: 6px; background: rgba(245, 185, 66, .14); border: 1px solid rgba(245, 185, 66, .45);
  color: var(--gold); font-size: 12px; font-weight: 700;
}
.badge-muestra {
  margin-left: auto; font-size: 10px; letter-spacing: .1em; text-transform: uppercase;
  color: var(--prim-2); background: var(--prim-dim); border: 1px solid rgba(16, 185, 129, .35);
  padding: 3px 8px; border-radius: 999px;
}
.aviso-casa { margin-left: auto; color: var(--faint); font-style: italic; font-size: 12px; }

.tablero {
  display: flex; flex-wrap: wrap; gap: 4px; justify-content: center; align-items: center;
  min-height: 46px; padding: 8px; border-radius: 12px;
  background: rgba(0, 0, 0, .28); border: 1px dashed rgba(120, 140, 160, .18);
}
.tablero.vacio { min-height: 54px; }
.vacio-texto { color: var(--faint); font-size: 12.5px; }
.mesa-nota { margin: 0; font-size: 11px; color: var(--faint); display: flex; align-items: center; gap: 6px; }
.mesa-nota .titin.inline { position: static; display: inline-block; margin: 0; }

/* ------------------------------------------------------------ fichas --- */
.ficha {
  position: relative;
  display: flex;
  background: linear-gradient(145deg, #fbf3df 0%, #efe3c2 55%, #d8c69b 100%);
  border-radius: 10px;
  box-shadow:
    inset 0 0 0 1px rgba(96, 76, 44, .38),
    inset -2px -3px 6px rgba(0, 0, 0, .18),
    0 3px 8px rgba(0, 0, 0, .45);
}
.ficha.vertical { flex-direction: column; width: 42px; height: 62px; padding: 4px 3px; }
.ficha.horizontal { flex-direction: row; width: 62px; height: 38px; padding: 3px 4px; }

.mitad { flex: 1; display: grid; grid-template-columns: repeat(3, 1fr); grid-template-rows: repeat(3, 1fr); place-items: center; }
.pipo { width: 6px; height: 6px; border-radius: 50%; background: transparent; }
.pipo.on { background: radial-gradient(circle at 35% 30%, #4a4438, #141210 72%); box-shadow: inset 0 0 1px rgba(0, 0, 0, .6); }
.ficha.horizontal .pipo { width: 6px; height: 6px; }

.media-luna { position: relative; background: rgba(80, 64, 38, .55); }
.vertical .media-luna { height: 1px; width: 76%; margin: 2px auto; }
.horizontal .media-luna { width: 1px; height: 76%; margin: auto 2px; }
.media-luna::after {
  content: ''; position: absolute; left: 50%; top: 50%;
  transform: translate(-50%, -50%) rotate(45deg);
  width: 6px; height: 6px; border-radius: 50%;
  border: 2px solid rgba(120, 92, 40, .8); border-left-color: transparent;
}
.horizontal .media-luna::after { transform: translate(-50%, -50%) rotate(-45deg); }

.titin {
  position: absolute; top: 2px; right: 2px; width: 6px; height: 6px; margin: 0;
  border-radius: 50%; background: transparent;
  border: 1.5px solid #f5b942;
  box-shadow: 0 0 3px rgba(245, 185, 66, .8);
}

.ficha.boton { cursor: pointer; border: 2px solid transparent; transition: transform .12s ease, border-color .12s ease; }
.ficha.boton:hover { transform: translateY(-2px); }
.ficha.boton:disabled { cursor: default; }
.ficha.boton:disabled:hover { transform: none; }
.ficha.boton.jugable { border-color: rgba(16, 185, 129, .55); }
.ficha.boton.seleccionada { border-color: var(--gold); box-shadow: 0 0 0 2px var(--gold), 0 0 18px rgba(245, 185, 66, .5); }
.ficha.extremo { box-shadow: inset 0 0 0 1px rgba(96, 76, 44, .38), 0 0 0 2px rgba(245, 185, 66, .7), 0 3px 8px rgba(0, 0, 0, .45); }
.ficha.resalta-ganador { border-color: var(--gold); box-shadow: 0 0 14px rgba(245, 185, 66, .6); }

/* ------------------------------------------------------------- mano --- */
.mano-seccion { display: flex; flex-direction: column; gap: 6px; }
.mano-top { display: flex; align-items: baseline; gap: 10px; flex-wrap: wrap; }
.mano-top .etq { font-size: 10px; letter-spacing: .14em; text-transform: uppercase; color: var(--faint); }
.mano-hint { font-size: 11.5px; color: var(--faint); }
.mano-hint b { color: var(--muted); }
.mano { display: flex; flex-wrap: wrap; gap: 4px; justify-content: center; }

.controles { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.controles .btn { flex: 1 1 140px; }
.paytable { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11px; color: var(--faint); line-height: 1.5; }
.ayuda { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11.5px; color: var(--faint); }
.rtp { display: flex; justify-content: center; gap: 6px; flex-wrap: wrap; margin: 0; font-size: 12px; color: var(--muted); }
.rtp b { color: var(--text); }

.resultado { text-align: center; padding: 10px; border-radius: 10px; background: rgba(0, 0, 0, .3); border: 1px solid var(--border-soft); font-size: 14px; color: var(--muted); }
.resultado.gana { background: var(--prim-dim); border-color: rgba(16, 185, 129, .35); color: var(--prim-2); font-weight: 700; }
.error-text { color: #f87171; font-size: 12px; margin: 0; }
</style>
