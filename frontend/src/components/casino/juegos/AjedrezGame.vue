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
const rtp = rtpDe('ajedrez')!

interface Opcion {
  uci: string
  etiqueta: string
}

interface AjedrezEstado {
  ronda_id: number
  monto: number
  fen: string
  tema: string
  opciones: Opcion[]
  estado: string
  multiplicador: number
  premio: number
  premio_pagado: number
  jugadas: string[]
  saldo: number
  server_seed_hash: string
  nonce: number
}

const ronda = ref<AjedrezEstado | null>(null)
const cargando = ref(false)
const error = ref('')
const elegida = ref<number | null>(null)

const saldo = computed(() => auth.user?.saldo ?? 0)
const abierta = computed(() => ronda.value?.estado === 'abierta')

const GLIFOS: Record<string, string> = {
  K: '♔', Q: '♕', R: '♖', B: '♗', N: '♘', P: '♙',
  k: '♚', q: '♛', r: '♜', b: '♝', n: '♞', p: '♟',
}

function fenFilas(fen: string): string[][] {
  const parte = fen.split(' ')[0]
  const filas: string[][] = []
  for (const fila of parte.split('/')) {
    const celdas: string[] = []
    for (const ch of fila) {
      if (ch >= '1' && ch <= '8') {
        for (let i = 0; i < Number(ch); i++) celdas.push('')
      } else {
        celdas.push(ch)
      }
    }
    filas.push(celdas)
  }
  return filas
}

const filas = computed(() => (ronda.value ? fenFilas(ronda.value.fen) : []))

async function empezar() {
  error.value = ''
  cargando.value = true
  elegida.value = null
  try {
    const { data } = await api.post<AjedrezEstado>('/casino/ajedrez/start', {
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

async function jugar(opcion: number) {
  if (!ronda.value || cargando.value || !abierta.value) return
  error.value = ''
  cargando.value = true
  try {
    const { data } = await api.post<AjedrezEstado>(`/casino/ajedrez/${ronda.value.ronda_id}/move`, { opcion })
    ronda.value = data
    elegida.value = opcion
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
  <div class="ajedrez">
    <div class="ajedrez-head">
      <div class="contador">
        <span class="etq">Tema</span>
        <strong>{{ ronda?.tema ?? '—' }}</strong>
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
      <div v-if="!ronda" class="vacio">
        <span>Tocá <b>Empezar</b> para ver la posición.</span>
        <small>Encontrá el mate en una entre cuatro jugadas.</small>
      </div>
      <div v-for="(fila, r) in filas" :key="r" class="fila">
        <div
          v-for="(celda, c) in fila"
          :key="c"
          class="celda"
          :class="{ clara: (r + c) % 2 === 0, oscura: (r + c) % 2 === 1 }"
        >
          <span v-if="celda" class="pieza">{{ GLIFOS[celda] ?? celda }}</span>
        </div>
      </div>
    </div>

    <p class="rtp">
      RTP declarado: <b>jugando bien {{ rtpTexto(rtp.rtp) }}</b> ·
      <b>jugando al azar {{ rtpTexto(rtp.rtp_casual) }}</b>
      <small>la simulación de referencia es la elección al azar entre las 4 opciones</small>
    </p>

    <p v-if="error" class="error-text">{{ error }}</p>

    <div v-if="!ronda" class="controles">
      <button class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
        {{ cargando ? 'Abriendo…' : 'Empezar por ' + colones(monto) }}
      </button>
      <p class="paytable">Acierto x3.5 · fallar pierde lo apostado · las posiciones están verificadas.</p>
    </div>

    <div v-else-if="abierta" class="controles">
      <button
        v-for="(op, i) in ronda.opciones"
        :key="op.uci"
        class="btn btn-opcion"
        :disabled="cargando"
        @click="jugar(i)"
      >
        {{ op.etiqueta }}
      </button>
      <span class="ayuda">Elegí la jugada que da jaque mate.</span>
    </div>

    <div v-else class="controles">
      <button class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
        {{ cargando ? 'Abriendo…' : 'Jugar de nuevo por ' + colones(monto) }}
      </button>
    </div>

    <div v-if="ronda && ronda.estado !== 'abierta'" class="resultado" :class="{ gana: ronda.premio_pagado > 0 }">
      <span v-if="ronda.premio_pagado > 0">¡Mate! Ganaste {{ colones(ronda.premio_pagado) }} (x{{ ronda.multiplicador }})</span>
      <span v-else>Esa jugada no era mate: se pierde lo apostado</span>
    </div>
  </div>
</template>

<style scoped>
.ajedrez { display: flex; flex-direction: column; gap: 10px; }
.ajedrez-head {
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
  display: flex; flex-direction: column; gap: 0; width: min(360px, 100%);
  margin: 0 auto; border-radius: 12px; overflow: hidden;
  border: 2px solid rgba(232, 206, 122, .45);
  box-shadow: 0 10px 30px rgba(0, 0, 0, .45);
}
.fila { display: grid; grid-template-columns: repeat(8, 1fr); }
.celda { aspect-ratio: 1; display: grid; place-items: center; }
.celda.clara { background: #e8dcc0; }
.celda.oscura { background: #0f9d6b; }
.pieza { font-size: clamp(20px, 6vw, 34px); line-height: 1; }
.vacio { grid-column: 1 / -1; display: flex; flex-direction: column; align-items: center; gap: 4px; padding: 40px 16px; color: var(--muted); background: #080b10; }
.vacio small { color: var(--faint); font-size: 12px; }

.controles { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.controles .btn { flex: 1 1 140px; }
.btn-opcion { background: rgba(16, 185, 129, .12); border: 1px solid rgba(16, 185, 129, .4); color: var(--text); font-family: var(--display); }
.btn-opcion:hover { background: rgba(16, 185, 129, .22); }
.paytable { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11px; color: var(--faint); line-height: 1.5; }
.ayuda { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11.5px; color: var(--faint); }
.rtp { display: flex; justify-content: center; align-items: baseline; gap: 6px; flex-wrap: wrap; margin: 0; font-size: 12px; color: var(--muted); }
.rtp b { color: var(--text); }
.rtp small { color: var(--faint); font-size: 11px; }

.resultado { text-align: center; padding: 10px; border-radius: 10px; background: rgba(0,0,0,.3); border: 1px solid var(--border-soft); font-size: 14px; color: var(--muted); }
.resultado.gana { background: var(--prim-dim); border-color: rgba(16,185,129,.35); color: var(--prim-2); font-weight: 700; }
.error-text { color: #f87171; font-size: 12px; margin: 0; }
</style>
