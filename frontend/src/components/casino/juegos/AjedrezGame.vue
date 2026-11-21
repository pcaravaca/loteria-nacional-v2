<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'
import { rtpDe, rtpTexto } from '@/utils/rtpCasino'
import { PIEZAS_AJEDREZ } from '@/assets/casino/piezasAjedrez'

const props = defineProps<{ monto: number; clientSeed: string }>()
const auth = useAuthStore()

// RTP declarado para este juego (seed del casino): jugando bien y al azar.
// La modalidad "partida completa" tiene su propia simulacion (el motor contra
// el motor y el azar), medida en backend/internal/casino/ajedrez.go.
const rtpMate = rtpDe('ajedrez')!
const RTP_PARTIDA = { rtp: 85.0, rtp_casual: 5.71 }

type Modalidad = 'mate' | 'partida'

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

interface PartidaEstado {
  partida_id: number
  monto: number
  color: string
  fen: string
  turno: string
  jugadas: string[]
  ultimo: string
  legales: string[]
  estado: string
  resultado: string
  motivo: string
  multiplicador: number
  premio: number
  premio_pagado: number
  saldo: number
  server_seed_hash: string
  nonce: number
}

interface DemoPuzzle {
  fen: string
  correcta: string
  decoys: string[]
}

// Posiciones de muestra para que el tablero se vea apenas se abre la ventana.
// Son posiciones curadas del backend; elegir una aqui no cobra.
const DEMOS: DemoPuzzle[] = [
  { fen: '8/2K5/k7/8/7Q/8/8/8 w - - 0 1', correcta: 'h4a4', decoys: ['h4h5', 'h4b4', 'c7c8'] },
  { fen: '7k/8/7K/8/8/8/8/5Q2 w - - 0 1', correcta: 'f1f8', decoys: ['f1f3', 'h6g6', 'f1h3'] },
  { fen: '8/8/8/8/2B5/8/7B/2K1k3 w - - 0 1', correcta: 'h2g3', decoys: ['h2b8', 'h2f4', 'c1b1'] },
]

const FEN_INICIAL = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'

const modalidad = ref<Modalidad>('mate')
const ronda = ref<AjedrezEstado | null>(null)
const partida = ref<PartidaEstado | null>(null)
const cargando = ref(false)
const error = ref('')
const elegida = ref<number | null>(null)

// Demostracion sin cargo de la modalidad mate.
const demo = ref<{ fen: string; opciones: string[]; correcta: string; elegida: number | null } | null>(null)

// Partida completa: seleccion de casilla, promocion y color.
const seleccionada = ref<string | null>(null)
const promocion = ref<{ desde: string; hasta: string } | null>(null)
const colorElegido = ref<'blancas' | 'negras'>('blancas')

const saldo = computed(() => auth.user?.saldo ?? 0)
const abierta = computed(() => ronda.value?.estado === 'abierta')
const partidaAbierta = computed(() => partida.value?.estado === 'abierta')

const GLIFOS: Record<string, string> = {
  K: '♔', Q: '♕', R: '♖', B: '♗', N: '♘', P: '♙',
  k: '♚', q: '♛', r: '♜', b: '♝', n: '♞', p: '♟',
}

const NOMBRE_PIEZA: Record<string, string> = {
  K: 'Rey', Q: 'Dama', R: 'Torre', B: 'Alfil', N: 'Caballo', P: 'Peon',
  k: 'Rey', q: 'Dama', r: 'Torre', b: 'Alfil', n: 'Caballo', p: 'Peon',
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

function piezaEn(fen: string, sq: string): string {
  const file = sq.charCodeAt(0) - 97
  const rank = sq.charCodeAt(1) - 49
  if (file < 0 || file > 7 || rank < 0 || rank > 7) return ''
  return fenFilas(fen)[7 - rank][file] ?? ''
}

// Mayuscula = pieza blanca, minuscula = negra (notacion FEN).
function esNegra(ch: string): boolean {
  return ch === ch.toLowerCase()
}

function etiqueta(fen: string, uci: string): string {
  if (uci.length < 4) return uci
  const pieza = piezaEn(fen, uci.slice(0, 2))
  return `${NOMBRE_PIEZA[pieza] ?? 'Pieza'} ${uci.slice(0, 2)}-${uci.slice(2, 4)}`
}

function armarDemo() {
  const p = DEMOS[Math.floor(Math.random() * DEMOS.length)]
  const opciones = [p.correcta, ...p.decoys]
  for (let i = opciones.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[opciones[i], opciones[j]] = [opciones[j], opciones[i]]
  }
  demo.value = { fen: p.fen, opciones, correcta: p.correcta, elegida: null }
}

function elegirModalidad(m: Modalidad) {
  modalidad.value = m
  error.value = ''
  seleccionada.value = null
  promocion.value = null
  if (m === 'mate' && !demo.value) armarDemo()
  reproducir('click')
}

const fenActiva = computed(() => {
  if (modalidad.value === 'partida') return partida.value?.fen ?? FEN_INICIAL
  return ronda.value?.fen ?? demo.value?.fen ?? FEN_INICIAL
})

const orientacion = computed<'blancas' | 'negras'>(() =>
  modalidad.value === 'partida' && partida.value?.color === 'negras' ? 'negras' : 'blancas',
)

const filas = computed(() => {
  const base = fenFilas(fenActiva.value)
  if (orientacion.value === 'blancas') return base
  return [...base].reverse().map((f) => [...f].reverse())
})

// Nombre algebraico de la casilla dibujada en (r, c) segun la orientacion.
function nombreDe(r: number, c: number): string {
  const fila = orientacion.value === 'blancas' ? r : 7 - r
  const col = orientacion.value === 'blancas' ? c : 7 - c
  return String.fromCharCode(97 + col) + (8 - fila)
}

const destinos = computed<Set<string>>(() => {
  const set = new Set<string>()
  if (!partida.value || !partidaAbierta.value || !seleccionada.value) return set
  for (const uci of partida.value.legales) {
    if (uci.slice(0, 2) === seleccionada.value) set.add(uci.slice(2, 4))
  }
  return set
})

const ultimoPar = computed<Set<string>>(() => {
  const set = new Set<string>()
  const ultimo = partida.value?.ultimo ?? ''
  if (ultimo.length >= 4) {
    set.add(ultimo.slice(0, 2))
    set.add(ultimo.slice(2, 4))
  }
  return set
})

// El rey que recibe mate queda resaltado con la clase compartida.
const matePar = computed<Set<string>>(() => {
  const set = new Set<string>()
  const p = partida.value
  if (!p || p.estado !== 'pagada' || (p.resultado !== 'blancas' && p.resultado !== 'negras')) return set
  const rey = p.resultado === 'blancas' ? 'k' : 'K'
  const nombre = nombreRey(p.fen, rey)
  if (nombre) set.add(nombre)
  return set
})

function nombreRey(fen: string, rey: string): string | null {
  const filas = fenFilas(fen)
  for (let r = 0; r < 8; r++) {
    for (let c = 0; c < 8; c++) {
      if (filas[r][c] === rey) return String.fromCharCode(97 + c) + (8 - r)
    }
  }
  return null
}

// En la modalidad mate se resalta la jugada que se eligio (y si pagó).
const jugadaPar = computed<Set<string>>(() => {
  const set = new Set<string>()
  const j = ronda.value?.jugadas?.[0] ?? ''
  if (j.length >= 4) {
    set.add(j.slice(0, 2))
    set.add(j.slice(2, 4))
  }
  return set
})

const turnoJugador = computed(() => {
  const p = partida.value
  if (!p || !partidaAbierta.value) return false
  return (p.turno === 'w') === (p.color === 'blancas')
})

function tocarCasilla(r: number, c: number) {
  const p = partida.value
  if (!p || !partidaAbierta.value || cargando.value || !turnoJugador.value) return
  const nombre = nombreDe(r, c)
  const pieza = piezaEn(p.fen, nombre)
  const esPropia = p.color === 'blancas' ? /[A-Z]/.test(pieza) : /[a-z]/.test(pieza)
  if (seleccionada.value && destinos.value.has(nombre)) {
    const desde = seleccionada.value
    const esPromocion = p.legales.some((u) => u.slice(0, 2) === desde && u.slice(2, 4) === nombre && u.length === 5)
    if (esPromocion) {
      promocion.value = { desde, hasta: nombre }
      reproducir('click')
      return
    }
    enviar(desde + nombre)
    return
  }
  seleccionada.value = esPropia ? nombre : null
  promocion.value = null
  if (esPropia) reproducir('click')
}

async function enviar(uci: string) {
  const p = partida.value
  if (!p || cargando.value) return
  error.value = ''
  cargando.value = true
  seleccionada.value = null
  promocion.value = null
  try {
    const { data } = await api.post<PartidaEstado>(`/casino/ajedrez/partida/${p.partida_id}/move`, { uci })
    partida.value = data
    reproducir('ficha')
    if (data.estado === 'pagada') reproducir(data.premio_pagado > 0 ? 'ganar' : 'perder')
    await auth.cargar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
    reproducir('error')
  } finally {
    cargando.value = false
  }
}

async function empezar() {
  error.value = ''
  cargando.value = true
  elegida.value = null
  seleccionada.value = null
  promocion.value = null
  try {
    if (modalidad.value === 'partida') {
      const { data } = await api.post<PartidaEstado>('/casino/ajedrez/partida/start', {
        monto: props.monto,
        client_seed: props.clientSeed,
        color: colorElegido.value,
      })
      partida.value = data
      reproducir('ficha')
    } else {
      const { data } = await api.post<AjedrezEstado>('/casino/ajedrez/start', {
        monto: props.monto,
        client_seed: props.clientSeed,
      })
      ronda.value = data
      reproducir('tecla')
    }
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

function demostracion(opcion: number) {
  if (!demo.value || demo.value.elegida !== null) return
  demo.value.elegida = opcion
  reproducir(demo.value.opciones[opcion] === demo.value.correcta ? 'ganar' : 'perder')
}

onMounted(() => {
  // El tablero se ve apenas se abre la ventana: en mate, una posicion de
  // muestra; en partida completa, la posicion inicial.
  armarDemo()
})
</script>

<template>
  <div class="ajedrez">
    <div class="ajedrez-head">
      <div class="contador" v-if="modalidad === 'mate'">
        <span class="etq">Tema</span>
        <strong>{{ ronda?.tema ?? 'Muestra' }}</strong>
      </div>
      <div class="contador" v-else>
        <span class="etq">Tus piezas</span>
        <strong>{{ partida ? (partida.color === 'blancas' ? 'Blancas' : 'Negras') : '—' }}</strong>
      </div>
      <div class="contador">
        <span class="etq">Premio</span>
        <strong class="oro">{{ colones((modalidad === 'partida' ? partida?.premio : ronda?.premio) ?? 0) }}</strong>
      </div>
      <div class="info">
        <span>Saldo <b>{{ colones(saldo) }}</b></span>
        <span>Apuesta <b>{{ colones(monto) }}</b></span>
      </div>
    </div>

    <div class="modos">
      <button class="modo" :class="{ activa: modalidad === 'mate' }" @click="elegirModalidad('mate')">
        Mate en una
      </button>
      <button class="modo" :class="{ activa: modalidad === 'partida' }" @click="elegirModalidad('partida')">
        Partida completa
      </button>
    </div>

    <div class="tablero">
      <div v-for="(fila, r) in filas" :key="r" class="fila">
        <div
          v-for="(celda, c) in fila"
          :key="c"
          class="celda"
          :class="{
            clara: (r + c) % 2 === 0,
            oscura: (r + c) % 2 === 1,
            seleccionada: nombreDe(r, c) === seleccionada,
            destino: destinos.has(nombreDe(r, c)),
            ultimo: ultimoPar.has(nombreDe(r, c)),
            'resalta-ganador': matePar.has(nombreDe(r, c)) || jugadaPar.has(nombreDe(r, c)),
            tactil: modalidad === 'partida' && partidaAbierta && turnoJugador && celda !== '',
          }"
          @click="tocarCasilla(r, c)"
        >
          <span v-if="c === 0" class="co co-rank">{{ 8 - r }}</span>
          <span v-if="r === 7" class="co co-file">{{ String.fromCharCode(97 + c) }}</span>
          <span
            v-if="celda"
            class="pieza"
            :class="{ negra: esNegra(celda) }"
            v-html="PIEZAS_AJEDREZ[celda]"
          />
          <span
            v-if="destinos.has(nombreDe(r, c))"
            class="dot"
            :class="{ captura: celda !== '' }"
          />
        </div>
      </div>
    </div>

    <div v-if="promocion" class="promocion">
      <span>Promoción:</span>
      <button
        v-for="p in ['q', 'r', 'b', 'n']"
        :key="p"
        class="btn btn-opcion"
        @click="enviar(promocion!.desde + promocion!.hasta + p)"
      >{{ GLIFOS[partida?.color === 'blancas' ? p.toUpperCase() : p] ?? p }}</button>
    </div>

    <p class="rtp">
      <template v-if="modalidad === 'partida'">
        RTP declarado: <b>jugando bien {{ rtpTexto(RTP_PARTIDA.rtp) }}</b> ·
        <b>jugando al azar {{ rtpTexto(RTP_PARTIDA.rtp_casual) }}</b>
        <small>simulado con el motor del adversario; las tablas devuelven lo apostado</small>
      </template>
      <template v-else>
        RTP declarado: <b>jugando bien {{ rtpTexto(rtpMate.rtp) }}</b> ·
        <b>jugando al azar {{ rtpTexto(rtpMate.rtp_casual) }}</b>
        <small>la simulación de referencia es la elección al azar entre las 4 opciones</small>
      </template>
    </p>

    <p v-if="error" class="error-text">{{ error }}</p>

    <!-- Modalidad mate -->
    <template v-if="modalidad === 'mate'">
      <div v-if="demo && !ronda" class="controles">
        <button
          v-for="(uci, i) in demo.opciones"
          :key="uci"
          class="btn btn-opcion"
          :class="{ 'resalta-ganador': demo.elegida === i && uci === demo.correcta }"
          :disabled="demo.elegida !== null"
          @click="demostracion(i)"
        >{{ etiqueta(demo.fen, uci) }}</button>
        <span class="ayuda">
          Posición de muestra, sin cargo.
          <b v-if="demo.elegida !== null">
            {{ demo.opciones[demo.elegida] === demo.correcta ? '¡Es mate!' : 'Esa no era mate.' }}
          </b>
          <template v-else>Pulsá <b>Empezar</b> para jugar una posición real.</template>
        </span>
      </div>

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
        >{{ op.etiqueta }}</button>
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
    </template>

    <!-- Partida completa -->
    <template v-else>
      <div v-if="!partida" class="controles">
        <div class="colores">
          <button class="modo" :class="{ activa: colorElegido === 'blancas' }" @click="colorElegido = 'blancas'">Blancas</button>
          <button class="modo" :class="{ activa: colorElegido === 'negras' }" @click="colorElegido = 'negras'">Negras</button>
        </div>
        <button class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
          {{ cargando ? 'Abriendo…' : 'Empezar por ' + colones(monto) }}
        </button>
        <p class="paytable">
          Ganás x1.7 dando jaque mate · las tablas devuelven lo apostado · la casa responde con su motor.
        </p>
        <p class="paytable">
          Tablas: ahogado, regla de 50 movimientos, triple repetición o material insuficiente.
        </p>
      </div>

      <template v-else>
        <div class="controles">
          <span class="aviso" v-if="partidaAbierta && turnoJugador">Tu turno: tocá una pieza y después el destino.</span>
          <span class="aviso" v-else-if="partidaAbierta">Jugó la casa…</span>
          <span class="aviso" v-if="partida.jugadas.length">Jugadas: {{ partida.jugadas.length }}</span>
        </div>
        <div v-if="!partidaAbierta" class="controles">
          <button class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
            {{ cargando ? 'Abriendo…' : 'Jugar de nuevo por ' + colones(monto) }}
          </button>
        </div>
      </template>

      <div v-if="partida && partida.estado !== 'abierta'" class="resultado" :class="{ gana: partida.premio_pagado > 0 }">
        <span v-if="partida.resultado === partida.color">
          ¡Jaque mate! Ganaste {{ colones(partida.premio_pagado) }} (x{{ partida.multiplicador }})
        </span>
        <span v-else-if="partida.resultado === 'tablas'">
          Tablas ({{ partida.motivo }}): se devuelve lo apostado
        </span>
        <span v-else>La casa dio jaque mate ({{ partida.motivo }}): se pierde lo apostado</span>
      </div>
    </template>
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

.modos { display: flex; gap: 6px; flex-wrap: wrap; }
.modo {
  flex: 1 1 120px; padding: 7px 10px; border-radius: 10px; cursor: pointer;
  background: rgba(16, 185, 129, .08); border: 1px solid rgba(16, 185, 129, .3);
  color: var(--muted); font-family: var(--display); font-size: 13px;
}
.modo.activa { background: rgba(16, 185, 129, .22); border-color: var(--gold); color: var(--text); }
.colores { display: flex; gap: 6px; flex: 1 1 100%; }

.tablero {
  display: flex; flex-direction: column; gap: 0; width: min(360px, 100%);
  margin: 0 auto; border-radius: 12px; overflow: hidden;
  border: 2px solid rgba(232, 206, 122, .45);
  box-shadow: 0 10px 30px rgba(0, 0, 0, .45);
}
.fila { display: grid; grid-template-columns: repeat(8, 1fr); }
.celda { aspect-ratio: 1; display: grid; place-items: center; position: relative; }
/* El ambiente tiñe el tablero; el fallback es el verde/crema de siempre. */
.celda.clara { background: var(--tm-board-light, #e8dcc0); }
.celda.oscura { background: var(--tm-board-dark, #0f9d6b); }
.celda.tactil { cursor: pointer; }
.celda.seleccionada { box-shadow: inset 0 0 0 3px var(--gold); }
.celda.ultimo { box-shadow: inset 0 0 0 3px rgba(232, 206, 122, .8); }
.celda.resalta-ganador { box-shadow: inset 0 0 0 3px var(--gold), inset 0 0 18px rgba(245, 185, 66, .8); }
/* Coordenadas a-h / 1-8: solo en la columna y la fila del borde, como un
   tablero de torneo (el tablero no se carga con 64 etiquetas). */
.co {
  position: absolute; z-index: 2; pointer-events: none;
  font-size: clamp(9px, 2.6vw, 13px); font-weight: 700;
  font-family: var(--display); line-height: 1;
}
.co-rank { top: 2px; left: 3px; }
.co-file { bottom: 2px; right: 3px; }
.celda.clara .co { color: rgba(0, 0, 0, .5); }
.celda.oscura .co { color: rgba(255, 255, 255, .62); }
.celda .pieza {
  display: grid; place-items: center; width: 88%; height: 88%;
  position: relative; z-index: 1;
}
.celda .pieza svg { display: block; width: 100%; height: 100%; filter: drop-shadow(0 2px 2px rgba(0, 0, 0, .35)); }
/* Las piezas negras llevan un halo claro: se leen aunque la casilla sea oscura. */
.celda .pieza.negra svg { filter: drop-shadow(0 0 1.5px rgba(255, 255, 255, .9)) drop-shadow(0 2px 2px rgba(0, 0, 0, .5)); }
.celda .dot {
  position: absolute; width: 28%; height: 28%; border-radius: 50%;
  background: rgba(3, 16, 12, .45); z-index: 3;
}
.celda .dot.captura {
  background: transparent; border: 3px solid rgba(3, 16, 12, .5);
  width: 82%; height: 82%; border-radius: 999px;
}

.promocion { display: flex; gap: 6px; align-items: center; justify-content: center; flex-wrap: wrap; font-size: 13px; color: var(--muted); }

.controles { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.controles .btn { flex: 1 1 140px; }
.btn-opcion { background: rgba(16, 185, 129, .12); border: 1px solid rgba(16, 185, 129, .4); color: var(--text); font-family: var(--display); }
.btn-opcion:hover { background: rgba(16, 185, 129, .22); }
.btn-opcion.resalta-ganador { border-color: var(--gold); box-shadow: 0 0 14px rgba(245, 185, 66, .6); }
.paytable { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11px; color: var(--faint); line-height: 1.5; }
.ayuda { flex: 1 1 100%; margin: 0; text-align: center; font-size: 11.5px; color: var(--faint); }
.ayuda b { color: var(--text); }
.aviso { flex: 1 1 100%; text-align: center; font-size: 12px; color: var(--faint); font-style: italic; }
.rtp { display: flex; justify-content: center; align-items: baseline; gap: 6px; flex-wrap: wrap; margin: 0; font-size: 12px; color: var(--muted); }
.rtp b { color: var(--text); }
.rtp small { color: var(--faint); font-size: 11px; }

.resultado { text-align: center; padding: 10px; border-radius: 10px; background: rgba(0,0,0,.3); border: 1px solid var(--border-soft); font-size: 14px; color: var(--muted); }
.resultado.gana { background: var(--prim-dim); border-color: rgba(16,185,129,.35); color: var(--prim-2); font-weight: 700; }
.error-text { color: #f87171; font-size: 12px; margin: 0; }
</style>
