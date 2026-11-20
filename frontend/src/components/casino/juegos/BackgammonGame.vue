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
const rtp = rtpDe('backgammon')!

interface BackgammonEstado {
  ronda_id: number
  monto: number
  jugador: number[]
  casa: number[]
  turno: string
  dado1: number
  dado2: number
  dobles: boolean
  lance: number
  estado: string
  multiplicador: number
  premio: number
  premio_pagado: number
  jugadas: number[]
  saldo: number
  server_seed_hash: string
  nonce: number
}

const META = 12
const ronda = ref<BackgammonEstado | null>(null)
const cargando = ref(false)
const error = ref('')
const seleccionada = ref<number | null>(null)

const saldo = computed(() => auth.user?.saldo ?? 0)
const abierta = computed(() => ronda.value?.estado === 'abierta')
const avance = computed(() => {
  if (!ronda.value) return 0
  return ronda.value.dobles ? (ronda.value.dado1 + ronda.value.dado2) * 2 : ronda.value.dado1 + ronda.value.dado2
})

function fichasEn(posiciones: number[], punto: number): number {
  return posiciones.filter((p) => p === punto).length
}

async function empezar() {
  error.value = ''
  cargando.value = true
  seleccionada.value = null
  try {
    const { data } = await api.post<BackgammonEstado>('/casino/backgammon/start', {
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
    const { data } = await api.post<BackgammonEstado>(`/casino/backgammon/${ronda.value.ronda_id}/move`, { ficha })
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
</script>

<template>
  <div class="backgammon">
    <div class="bg-head">
      <div class="contador">
        <span class="etq">Dados</span>
        <strong>{{ ronda ? ronda.dado1 + '-' + ronda.dado2 : '—' }}</strong>
      </div>
      <div class="contador">
        <span class="etq">Avance</span>
        <strong class="oro">+{{ avance }}</strong>
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

    <div class="pista">
      <div v-if="!ronda" class="vacio">
        <span>Tocá <b>Empezar</b> para tirar los dados.</span>
        <small>Sacá las tres fichas antes que la casa. Los dobles avanzan el doble.</small>
      </div>
      <template v-else>
        <div class="lado">
          <span class="etq">Vos</span>
          <div class="puntos">
            <div v-for="p in META" :key="p" class="punto">
              <span
                v-for="n in fichasEn(ronda.jugador, p - 1)"
                :key="n"
                class="ficha tuya"
                :class="{ 'resalta-ganador': seleccionada !== null && ronda.jugador[seleccionada] === p - 1 }"
              />
              <small>{{ p - 1 }}</small>
            </div>
          </div>
        </div>
        <div class="lado">
          <span class="etq">Casa</span>
          <div class="puntos">
            <div v-for="p in META" :key="p" class="punto">
              <span
                v-for="n in fichasEn(ronda.casa, p - 1)"
                :key="n"
                class="ficha casa"
              />
              <small>{{ p - 1 }}</small>
            </div>
          </div>
        </div>
      </template>
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
      <p class="paytable">Ganaste x1.85 · los dobles duplican el avance · la casa mueve la ficha más atrasada.</p>
    </div>

    <div v-else-if="abierta" class="controles">
      <button
        v-for="i in 3"
        :key="i"
        class="btn btn-opcion"
        :class="{ 'resalta-ganador': seleccionada === i - 1 }"
        :disabled="cargando || ronda.turno !== 'jugador'"
        @click="jugar(i - 1)"
      >
        Ficha {{ i }} ({{ ronda.jugador[i - 1] }})
      </button>
      <span class="ayuda">Elegí qué ficha avanza {{ avance }} puntos.</span>
    </div>

    <div v-else class="controles">
      <button class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
        {{ cargando ? 'Abriendo…' : 'Jugar de nuevo por ' + colones(monto) }}
      </button>
    </div>

    <div v-if="ronda && ronda.estado !== 'abierta'" class="resultado" :class="{ gana: ronda.premio_pagado > 0 }">
      <span v-if="ronda.premio_pagado > 0">¡Llegaste primero! Ganaste {{ colones(ronda.premio_pagado) }} (x{{ ronda.multiplicador }})</span>
      <span v-else>La casa sacó sus fichas antes</span>
    </div>
  </div>
</template>

<style scoped>
.backgammon { display: flex; flex-direction: column; gap: 10px; }
.bg-head {
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

.pista {
  display: flex; flex-direction: column; gap: 8px; padding: 12px;
  border-radius: 16px; border: 1px solid #243448;
  background: linear-gradient(180deg, rgba(16, 185, 129, .05), transparent 55%), #080b10;
}
.lado { display: flex; flex-direction: column; gap: 4px; }
.lado .etq { font-size: 10px; letter-spacing: .12em; text-transform: uppercase; color: var(--faint); }
.puntos { display: grid; grid-template-columns: repeat(12, 1fr); gap: 3px; }
.punto {
  display: flex; flex-direction: column; align-items: center; gap: 1px;
  min-height: 34px; padding: 2px; border-radius: 6px;
  background: rgba(255, 255, 255, .03); border: 1px solid rgba(255, 255, 255, .05);
}
.punto small { font-size: 8px; color: var(--faint); }
.ficha { width: 12px; height: 12px; border-radius: 50%; display: inline-block; }
.ficha.tuya { background: linear-gradient(150deg, #8ffcd0, #0f9d6b); box-shadow: 0 0 6px rgba(16, 185, 129, .7); }
.ficha.casa { background: linear-gradient(150deg, #ffe9a8, #c99a1e); }
.ficha.resalta-ganador { outline: 2px solid var(--gold); }
.vacio { display: flex; flex-direction: column; align-items: center; gap: 4px; padding: 24px 16px; color: var(--muted); }
.vacio small { color: var(--faint); font-size: 12px; }

.controles { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.controles .btn { flex: 1 1 120px; }
.btn-opcion { background: rgba(16, 185, 129, .12); border: 1px solid rgba(16, 185, 129, .4); color: var(--text); font-family: var(--display); }
.btn-opcion:hover { background: rgba(16, 185, 129, .22); }
.paytable { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11px; color: var(--faint); line-height: 1.5; }
.ayuda { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11.5px; color: var(--faint); }
.rtp { display: flex; justify-content: center; gap: 6px; flex-wrap: wrap; margin: 0; font-size: 12px; color: var(--muted); }
.rtp b { color: var(--text); }

.resultado { text-align: center; padding: 10px; border-radius: 10px; background: rgba(0,0,0,.3); border: 1px solid var(--border-soft); font-size: 14px; color: var(--muted); }
.resultado.gana { background: var(--prim-dim); border-color: rgba(16,185,129,.35); color: var(--prim-2); font-weight: 700; }
.error-text { color: #f87171; font-size: 12px; margin: 0; }
</style>
