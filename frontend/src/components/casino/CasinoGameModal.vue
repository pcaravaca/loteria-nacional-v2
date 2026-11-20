<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import type { CasinoGame, PlayResponse } from '@/types'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'
import Modal from '@/components/Modal.vue'
import { heroes, type CodigoJuego } from '@/assets/casino'
import { astSlots, astPokers } from '@/assets/ast'
import DadosGame from './juegos/DadosGame.vue'
import SlotsGame from './juegos/SlotsGame.vue'
import RuletaGame from './juegos/RuletaGame.vue'
import RaspaGame from './juegos/RaspaGame.vue'
import KenoGame from './juegos/KenoGame.vue'
import BlackjackGame from './juegos/BlackjackGame.vue'
import CascadaGame from './juegos/CascadaGame.vue'
import HiloGame from './juegos/HiloGame.vue'
import AltoGame from './juegos/AltoGame.vue'
import FortunaGame from './juegos/FortunaGame.vue'
import BaccaratGame from './juegos/BaccaratGame.vue'
import AstSlotGame from './juegos/AstSlotGame.vue'
import PokerGame from './juegos/PokerGame.vue'

const props = defineProps<{ juego: CasinoGame; clientSeed: string }>()
const emit = defineEmits<{ cerrar: [] }>()

const auth = useAuthStore()

const monto = ref<number>(props.juego.min_apuesta)
const cargando = ref(false)
const error = ref('')
const respuesta = ref<PlayResponse | null>(null)
const rondaKey = ref(0)

const rapidos = [500, 1000, 5000, 10000]
// Guía por juego (aprobada por Peter: se muestra arriba del tablero).
// Los multiplicadores son los del backend (internal/casino/games.go).
type Guia = { pasos: { t: string; d: string }[]; premios: { etq: string; mult: string }[] }
const guias: Partial<Record<CodigoJuego, Guia>> = {
  dados: {
    pasos: [
      { t: 'Elegí la suma', d: 'Del 2 al 12. Cuanto más rara, más paga.' },
      { t: 'Definí el monto', d: 'Mínimo ¢200 por tirada.' },
      { t: 'Lanzá los dados', d: 'El servidor sortea y acredita al instante.' },
    ],
    premios: [
      { etq: 'Suma 2 o 12', mult: 'x30' }, { etq: 'Suma 3 o 11', mult: 'x15' },
      { etq: 'Suma 4 o 10', mult: 'x10' }, { etq: 'Suma 5 o 9', mult: 'x8' },
      { etq: 'Suma 6 u 8', mult: 'x6' }, { etq: 'Suma 7', mult: 'x5' },
    ],
  },
  slots: {
    pasos: [
      { t: 'Definí el monto', d: 'Mínimo según el juego.' },
      { t: 'Girá los rodillos', d: 'Tres iguales pagan el triple; si solo coinciden las dos primeras, pagan el par.' },
    ],
    premios: [
      { etq: 'Tres sietes', mult: 'x50' }, { etq: 'Tres diamantes', mult: 'x25' },
      { etq: 'Tres campanas', mult: 'x18' }, { etq: 'Tres cerezas', mult: 'x8' },
      { etq: 'Dos sietes', mult: 'x20' }, { etq: 'Dos diamantes', mult: 'x8' },
      { etq: 'Dos herraduras', mult: 'x5' }, { etq: 'Dos campanas', mult: 'x3' },
      { etq: 'Dos barras', mult: 'x2' }, { etq: 'Dos cerezas', mult: 'x1' },
    ],
  },
  ruleta: {
    pasos: [
      { t: 'Elegí pleno o color', d: 'Pleno: un número del 0 al 36. Color: rojo o negro.' },
      { t: 'Definí el monto', d: 'Mínimo según el juego.' },
      { t: 'Girá la ruleta', d: 'El 0 es verde: no paga ni rojo ni negro.' },
    ],
    premios: [{ etq: 'Número pleno', mult: 'x35' }, { etq: 'Color rojo o negro', mult: 'x2' }],
  },
  raspa: {
    pasos: [
      { t: 'Definí el monto', d: 'Mínimo según el juego.' },
      { t: 'Raspá los tres casilleros', d: 'Tres iguales triplican; un par devuelve lo apostado.' },
    ],
    premios: [{ etq: 'Tres iguales', mult: 'x18' }, { etq: 'Un par', mult: 'x1 (devuelve)' }],
  },
  keno: {
    pasos: [
      { t: 'Marcá 5 números', d: 'Del 1 al 60, sin repetir.' },
      { t: 'Definí el monto', d: 'Mínimo según el juego.' },
      { t: 'Esperá las bolillas', d: 'Se sortean 15 de 60; tus aciertos definen el pago.' },
    ],
    premios: [
      { etq: '5 aciertos', mult: 'x65' }, { etq: '4 aciertos', mult: 'x8' },
      { etq: '3 aciertos', mult: 'x3' }, { etq: '2 aciertos', mult: 'x1' },
    ],
  },
  cascada: {
    pasos: [
      { t: 'Definí el monto', d: 'La bola baja por 9 filas de pinchos.' },
      { t: 'Soltá la bola', d: 'En el centro casi no paga; en los bordes llega a x10.' },
    ],
    premios: [
      { etq: 'Cae en el borde', mult: 'x10' }, { etq: 'Segundas casillas', mult: 'x5' },
      { etq: 'Terceras casillas', mult: 'x3' }, { etq: 'Centro', mult: 'x1 o x0' },
    ],
  },
  hilo: {
    pasos: [
      { t: 'Mirá la carta base', d: 'No importa el palo.' },
      { t: 'Elegí mayor o menor', d: 'La siguiente carta define el resultado; si empatan, se pierde.' },
    ],
    premios: [{ etq: 'Acierta mayor o menor', mult: 'x2' }],
  },
  alto: {
    pasos: [
      { t: 'Elegí dirección y tope', d: 'Bajo: sale ≤ tope. Sobre: sale > 100 − tope.' },
      { t: 'Definí el monto', d: 'El dado va del 1 al 100; más riesgo, más paga.' },
    ],
    premios: [
      { etq: 'Tope 2%', mult: 'x48' }, { etq: 'Tope 5%', mult: 'x19' },
      { etq: 'Tope 10%', mult: 'x9' }, { etq: 'Tope 25%', mult: 'x3' },
      { etq: 'Tope 48%', mult: 'x2' }, { etq: 'Tope 75 / 90 / 95%', mult: 'x1' },
    ],
  },
  fortuna: {
    pasos: [
      { t: 'Definí el monto', d: 'Son 21 casillas con premios que crecen hacia el centro.' },
      { t: 'Girá la ruleta de fortuna', d: 'Donde se detenga, cobrás el multiplicador de esa casilla.' },
    ],
    premios: [
      { etq: 'Casilla dorada', mult: 'x5' }, { etq: 'Rodeando el centro', mult: 'x3' },
      { etq: 'Laterales', mult: 'x2' }, { etq: 'Siete casillas', mult: 'x1' },
      { etq: 'Resto', mult: 'x0' },
    ],
  },
  blackjack: {
    pasos: [
      { t: 'Pedí o plantate', d: 'Sumá 21 sin pasarte; la casa se planta en 17 o más.' },
      { t: 'Podés doblar', d: 'Duplicás el monto y recibís una sola carta.' },
    ],
    premios: [{ etq: 'Natural (21 inicial)', mult: 'x1.5' }, { etq: 'Ganar la mano', mult: 'x1' }, { etq: 'Con Doblar', mult: 'hasta x4' }],
  },
  moneda: {
    pasos: [
      { t: 'Elegí cara o sello', d: 'Acertarte duplica la apuesta.' },
    ],
    premios: [{ etq: 'Acierto', mult: 'x2' }],
  },
  suertudo: {
    pasos: [
      { t: 'Marcá un número', d: 'Del 00 al 99.' },
      { t: 'Definí el monto', d: 'Si sale tu número, cobrás x75.' },
    ],
    premios: [{ etq: 'Tu número', mult: 'x75' }],
  },
  parley: {
    pasos: [
      { t: 'Marcá 2 números', d: 'Del 1 al 20, sin repetir.' },
      { t: 'Salidas', d: 'Si salen los dos, x18; si sale uno, recuperás lo apostado.' },
    ],
    premios: [{ etq: 'Los dos', mult: 'x18' }, { etq: 'Uno solo', mult: 'x1' }],
  },
  baccarat: {
    pasos: [
      { t: 'Elegí jugador, banca o empate', d: 'El empate paga x8; a la banca se le cobra 5% de comisión.' },
      { t: 'Reparto automático', d: 'Dos cartas por mano y la regla de la tercera carta define el resultado.' },
    ],
    premios: [{ etq: 'Jugador', mult: 'x2' }, { etq: 'Banca', mult: 'x2 menos 5%' }, { etq: 'Empate', mult: 'x8' }],
  },
}

// Guías genéricas para el catalogo AST (slots y video poker).
const guiaAstSlot: Guia = {
  pasos: [
    { t: 'Definí el monto', d: 'La apuesta se reparte entre todas las lineas.' },
    { t: 'Girá los rodillos', d: 'Las lineas se leen de izquierda a derecha; paga la mayor coincidencia.' },
    { t: 'Cobrar', d: 'El servidor acredita el premio al instante y la ronda queda verificable.' },
  ],
  premios: [],
}
const guiaAstPoker: Guia = {
  pasos: [
    { t: 'Definí el monto', d: 'Corresponde a cinco fichas: la tabla paga como apuesta maxima.' },
    { t: 'Repartí', d: 'La apuesta se descuenta al repartir; las cinco cartas quedan committed con el hash de la semilla.' },
    { t: 'Elegí y cobrá', d: 'Tocá las cartas que querés sostener y cobrá; lo demás se reemplaza. Si no tocás nada, el sistema sostiene la mejor jugada.' },
  ],
  premios: [],
}

const guia = computed<Guia | undefined>(() => {
  const codigo = props.juego.codigo as CodigoJuego
  const propia = guias[codigo]
  if (propia) return propia
  if (astSlots.some((s) => s.codigo === codigo)) return guiaAstSlot
  if (astPokers.some((p) => p.codigo === codigo)) return guiaAstPoker
  return undefined
})
const esSlotAst = computed(() => astSlots.some((s) => s.codigo === props.juego.codigo))
const esPokerAst = computed(() => astPokers.some((p) => p.codigo === props.juego.codigo))
const saldo = computed(() => auth.user?.saldo ?? 0)
// El tope de premio puede recortar la apuesta maxima; el backend envia ese valor
// ya calculado y es el que vale.
const apuestaMax = computed(() => props.juego.apuesta_max ?? props.juego.max_apuesta)

function ajustar(delta: number) {
  monto.value = Math.max(props.juego.min_apuesta, Math.min(apuestaMax.value, monto.value + delta))
}

async function jugar(params: Record<string, unknown>) {
  error.value = ''
  cargando.value = true
  reproducir('ficha')
  try {
    const { data } = await api.post<PlayResponse>('/casino/play', {
      game: props.juego.codigo,
      monto: monto.value,
      client_seed: props.clientSeed,
      params,
    })
    respuesta.value = data
    rondaKey.value++
    reproducir(data.premio > 0 ? 'ganar' : 'perder')
    await auth.cargar()
  } catch (e: any) {
    error.value = e.message
  } finally {
    cargando.value = false
  }
}

function repetir() {
  respuesta.value = null
  error.value = ''
}
</script>

<template>
  <Modal :titulo="juego.nombre" :sub="juego.descripcion" ancho="lg" mostrar-cerrar @cerrar="emit('cerrar')">
    <div class="casino-modal" :data-tema="auth.temaCasino">
      <div class="tema-marquesina">
        <b>{{ juego.nombre }}</b>
        <span>Min {{ colones(juego.min_apuesta) }} · Max x{{ juego.max_multiplicador }}</span>
      </div>

      <div class="apuesta">
        <button class="step" @click="ajustar(-juego.min_apuesta)">−</button>
        <div class="monto">
          <span class="etq">Apuesta</span>
          <strong>{{ colones(monto) }}</strong>
        </div>
        <button class="step" @click="ajustar(juego.min_apuesta)">+</button>
      </div>
      <div class="rapidos">
        <button
          v-for="r in rapidos"
          :key="r"
          class="chip"
          :class="{ activa: monto === r }"
          @click="monto = Math.max(juego.min_apuesta, Math.min(apuestaMax, r))"
        >{{ colones(r) }}</button>
        <span class="saldo">Saldo: <b>{{ colones(saldo) }}</b></span>
      </div>

      <p v-if="error" class="error-text">{{ error }}</p>

      <div class="tema-arena">
      <DadosGame
        v-if="juego.codigo === 'dados'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        @jugar="jugar"
      />
      <SlotsGame
        v-else-if="juego.codigo === 'slots'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        :ronda-key="rondaKey"
        @jugar="jugar"
      />
      <RuletaGame
        v-else-if="juego.codigo === 'ruleta'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        @jugar="jugar"
      />
      <RaspaGame
        v-else-if="juego.codigo === 'raspa'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        @jugar="jugar"
      />
      <KenoGame
        v-else-if="juego.codigo === 'keno'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        @jugar="jugar"
      />
      <CascadaGame
        v-else-if="juego.codigo === 'cascada'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        @jugar="jugar"
      />
      <HiloGame
        v-else-if="juego.codigo === 'hilo'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        @jugar="jugar"
      />
      <AltoGame
        v-else-if="juego.codigo === 'alto'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        @jugar="jugar"
      />
      <FortunaGame
        v-else-if="juego.codigo === 'fortuna'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        @jugar="jugar"
      />
      <BlackjackGame
        v-else-if="juego.codigo === 'blackjack'"
        :monto="monto"
        :client-seed="clientSeed"
      />
      <BaccaratGame
        v-else-if="juego.codigo === 'baccarat'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :monto="monto"
        :cargando="cargando"
        @jugar="jugar"
      />
      <AstSlotGame
        v-else-if="esSlotAst"
        :codigo="juego.codigo"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        :ronda-key="rondaKey"
        @jugar="jugar"
      />
      <PokerGame
        v-else-if="esPokerAst"
        :codigo="juego.codigo"
        :monto="monto"
        :client-seed="clientSeed"
      />
      </div>

      <details v-if="guia" class="guia-detalle">
        <summary>Cómo jugar y pagos</summary>
      <div class="guia">
        <h4>Cómo jugar</h4>
        <div class="pasos">
          <div v-for="(p, i) in guia.pasos" :key="i" class="paso">
            <b>{{ i + 1 }} · {{ p.t }}</b>
            <span>{{ p.d }}</span>
          </div>
        </div>
        <div class="premios">
          <div v-for="(pr, i) in guia.premios" :key="i" class="premio">
            <span>{{ pr.etq }}</span>
            <b>{{ pr.mult }}</b>
          </div>
        </div>
      </div>

      </details>

      <div v-if="respuesta" class="acciones-resultado">
        <button class="btn btn-ghost" @click="repetir">Jugar de nuevo</button>
        <button class="btn btn-primary" @click="emit('cerrar')">Cerrar</button>
      </div>
      <div v-else class="acciones-resultado">
        <button class="btn btn-ghost" @click="emit('cerrar')">Cerrar</button>
      </div>

      <p class="nota">
        Resultado generado en el servidor · provably fair ·
        <span class="hash">{{ respuesta?.server_seed_hash?.slice(0, 16) ?? '' }}</span>
      </p>
    </div>
  </Modal>
</template>

<style scoped>

.guia-detalle { border-radius: 12px; border: 1px solid var(--tm-reel-border); background: color-mix(in srgb, var(--tm-shell) 12%, transparent); }
.guia-detalle > summary {
  cursor: pointer; padding: 9px 12px; font-size: 11px; letter-spacing: .14em;
  text-transform: uppercase; color: var(--tm-accent); list-style: none;
}
.guia-detalle > summary::-webkit-details-marker { display: none; }
.guia-detalle[open] > summary { border-bottom: 1px solid var(--tm-reel-border); }
.guia-detalle .guia { border: 0; background: transparent; padding: 10px 12px; }
.rapidos { gap: 6px; }
.rapidos .chip { padding: 5px 9px; font-size: 11.5px; }
.apuesta { gap: 8px; }
.step { width: 34px; height: 34px; font-size: 16px; }
.monto strong { font-size: 17px; }

.casino-modal { display: flex; flex-direction: column; gap: 10px; }
.tema-marquesina {
  border-radius: 12px; padding: 10px 14px; display: flex; align-items: center; gap: 10px;
  text-transform: uppercase; letter-spacing: .18em; font-size: 13px; font-weight: 800;
  border: 1px solid transparent;
}
.tema-marquesina span {
  margin-left: auto; text-transform: none; letter-spacing: .02em; font-weight: 400;
  font-size: 11px; opacity: .8;
}
.tema-arena {
  border-radius: 16px; border: 1px solid; padding: 10px;
  box-shadow: inset 0 14px 30px rgba(0,0,0,.45);
}

.hero { height: 96px; border-radius: 12px; overflow: hidden; border: 1px solid var(--border-soft);
  background: linear-gradient(120deg, #0f2b22, #123f31 55%, #2a2410); }
.hero img { width: 100%; height: 100%; object-fit: cover; }
.guia { background: rgba(56, 189, 248, .06); border: 1px solid rgba(56, 189, 248, .22); border-radius: 14px; padding: 14px 16px; }
.guia h4 { margin: 0 0 10px; font-size: 11px; letter-spacing: .12em; text-transform: uppercase; color: var(--info); }
.pasos { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 8px; }
.paso { background: var(--surface-2); border: 1px solid var(--border-soft); border-radius: 10px; padding: 9px 10px; display: flex; flex-direction: column; gap: 3px; }
.paso b { font-size: 12.5px; color: var(--text); }
.paso span { font-size: 11.5px; color: var(--muted); line-height: 1.35; }
.premios { margin-top: 10px; display: flex; flex-wrap: wrap; gap: 6px; }
.premio { display: flex; align-items: center; gap: 6px; background: var(--bg-2); border: 1px solid var(--border-soft); border-radius: 8px; padding: 4px 9px; font-size: 11.5px; color: var(--muted); }
.premio b { color: var(--gold); font-family: 'Space Grotesk', var(--font); }
.apuesta { display: flex; align-items: center; gap: 10px; }
.step { width: 40px; height: 40px; border-radius: 10px; border: 1px solid var(--border); background: var(--surface); color: var(--text); font-size: 18px; cursor: pointer; }
.monto { flex: 1; text-align: center; background: #05080c; border: 1px solid var(--border); border-radius: 10px; padding: 8px; }
.monto .etq { display: block; font-size: 10px; text-transform: uppercase; letter-spacing: .09em; color: var(--faint); }
.monto strong { font-size: 20px; color: var(--text); }
.rapidos { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.chip { padding: 6px 10px; border-radius: 999px; border: 1px solid var(--border-soft); background: var(--surface); color: var(--muted); font-size: 12px; cursor: pointer; }
.chip.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
.saldo { margin-left: auto; font-size: 12px; color: var(--faint); }
.saldo b { color: var(--prim-2); }
.acciones-resultado { display: flex; gap: 10px; justify-content: flex-end; }
.nota { font-size: 11px; color: var(--faint); text-align: center; }
.hash { font-family: ui-monospace, monospace; }
</style>
