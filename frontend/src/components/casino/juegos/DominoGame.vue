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

const ronda = ref<DominoEstado | null>(null)
const cargando = ref(false)
const error = ref('')
const seleccionada = ref<number | null>(null)

const saldo = computed(() => auth.user?.saldo ?? 0)
const abierta = computed(() => ronda.value?.estado === 'abierta')
const casaVisible = computed(() => ronda.value?.casa ?? [])

function esValida(f: Ficha): boolean {
  if (!ronda.value) return false
  if (ronda.value.tablero.length === 0) return true
  const [izq, der] = ronda.value.extremos
  return f[0] === izq || f[1] === izq || f[0] === der || f[1] === der
}

const jugables = computed(() => {
  if (!ronda.value) return new Set<number>()
  const set = new Set<number>()
  ronda.value.mano.forEach((f, i) => {
    if (esValida(f)) set.add(i)
  })
  return set
})

const sinJugada = computed(() => abierta.value && jugables.value.size === 0)

function puntos(f: Ficha): number {
  return f[0] + f[1]
}

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
  if (!abierta.value || cargando.value) return
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
        <strong class="oro">{{ colones(ronda?.premio ?? 0) }}</strong>
      </div>
      <div class="info">
        <span>Saldo <b>{{ colones(saldo) }}</b></span>
        <span>Apuesta <b>{{ colones(monto) }}</b></span>
      </div>
    </div>

    <div class="tablero">
      <div v-if="!ronda || !ronda.tablero.length" class="vacio">
        <span>Tocá <b>Empezar</b> para recibir tus 7 fichas.</span>
        <small>La casa también recibe 7. Gana quien se queda sin fichas.</small>
      </div>
      <div
        v-for="(f, i) in ronda?.tablero ?? []"
        :key="i"
        class="ficha"
        :class="{ 'resalta-ganador': ronda?.estado === 'pagada' && i === ronda.tablero.length - 1 }"
      >
        <span>{{ f[0] }}</span>
        <i />
        <span>{{ f[1] }}</span>
      </div>
    </div>

    <div v-if="ronda && abierta" class="extremos">
      <span>Extremos: <b>{{ ronda.extremos[0] }}</b> y <b>{{ ronda.extremos[1] }}</b></span>
      <span v-if="ronda.turno === 'casa'" class="aviso-casa">Jugó la casa…</span>
    </div>

    <div v-if="ronda && abierta" class="mano">
      <button
        v-for="(f, i) in ronda.mano"
        :key="i"
        class="ficha boton"
        :class="{
          seleccionada: seleccionada === i,
          jugable: jugables.has(i),
          'resalta-ganador': seleccionada === i,
        }"
        :disabled="cargando"
        @click="tocarFicha(i)"
      >
        <span>{{ f[0] }}</span>
        <i />
        <span>{{ f[1] }}</span>
      </button>
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

    <div v-if="ronda && ronda.estado !== 'abierta'" class="resultado" :class="{ gana: ronda.premio_pagado > 0 }">
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

.tablero {
  display: flex; flex-wrap: wrap; gap: 6px; justify-content: center;
  padding: 12px; border-radius:  16px; border: 1px solid #243448;
  background: linear-gradient(180deg, rgba(16, 185, 129, .05), transparent 55%), #080b10;
  min-height: 96px; align-items: center;
}
.mano { display: flex; flex-wrap: wrap; gap: 6px; justify-content: center; }
.vacio { width: 100%; display: flex; flex-direction: column; align-items: center; gap: 4px; color: var(--muted); font-size: 14px; }
.vacio small { color: var(--faint); font-size: 12px; }

.ficha {
  display: inline-flex; flex-direction: column; align-items: center; justify-content: center;
  min-width: 34px; height: 52px; padding: 3px 5px; border-radius: 8px;
  background: linear-gradient(160deg, #fdf6e3, #d9c9a3);
  color: #1b1b1b; font-family: var(--display); font-size: 15px; font-weight: 700;
  box-shadow: inset -2px -3px 5px rgba(0, 0, 0, .25), 0 2px 5px rgba(0, 0, 0, .4);
}
.ficha i { width: 80%; height: 1px; background: rgba(0, 0, 0, .35); margin: 3px 0; }
.ficha.boton { cursor: pointer; border: 2px solid transparent; transition: transform .12s ease, border-color .12s ease; }
.ficha.boton:hover { transform: translateY(-2px); }
.ficha.boton.jugable { border-color: rgba(16, 185, 129, .55); }
.ficha.boton.seleccionada { border-color: var(--gold); box-shadow: 0 0 0 2px var(--gold), 0 0 18px rgba(245, 185, 66, .5); }
.ficha.resalta-ganador { border-color: var(--gold); box-shadow: 0 0 14px rgba(245, 185, 66, .6); }

.extremos { display: flex; justify-content: space-between; gap: 10px; font-size: 12.5px; color: var(--muted); }
.extremos b { color: var(--gold); }
.aviso-casa { color: var(--faint); font-style: italic; }

.controles { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.controles .btn { flex: 1 1 140px; }
.paytable { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11px; color: var(--faint); line-height: 1.5; }
.ayuda { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11.5px; color: var(--faint); }
.rtp { display: flex; justify-content: center; gap: 6px; flex-wrap: wrap; margin: 0; font-size: 12px; color: var(--muted); }
.rtp b { color: var(--text); }

.resultado { text-align: center; padding: 10px; border-radius: 10px; background: rgba(0,0,0,.3); border: 1px solid var(--border-soft); font-size: 14px; color: var(--muted); }
.resultado.gana { background: var(--prim-dim); border-color: rgba(16,185,129,.35); color: var(--prim-2); font-weight: 700; }
.error-text { color: #f87171; font-size: 12px; margin: 0; }
</style>
