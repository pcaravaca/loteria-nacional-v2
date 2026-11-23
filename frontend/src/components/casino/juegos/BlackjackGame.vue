<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'

const props = defineProps<{ monto: number; clientSeed: string }>()

const auth = useAuthStore()

const manos = ref<any[]>([])
const activa = ref<number | null>(null)
const grupoPremio = ref(0)
const cargando = ref(false)
const error = ref('')

const figura = (i: number) => {
  const v = i % 13
  if (v === 0) return 'A'
  if (v === 10) return 'J'
  if (v === 11) return 'Q'
  if (v === 12) return 'K'
  return String(v + 1)
}
const palo = (i: number) => ['♠', '♥', '♦', '♣'][Math.floor(i / 13)] || ''
const roja = (i: number) => Math.floor(i / 13) === 1 || Math.floor(i / 13) === 2
const valorCarta = (i: number) => {
  const v = i % 13
  return v === 0 ? 11 : v >= 10 ? 10 : v + 1
}

const valor = (cartas: number[]) => {
  let total = 0
  let ases = 0
  for (const c of cartas) {
    if (c % 13 === 0) ases++
    total += valorCarta(c)
  }
  while (total > 21 && ases > 0) {
    total -= 10
    ases--
  }
  return total
}

// La mano en juego: la activa si esta dividida, si no la unica.
const actual = computed(() => manos.value.find((m) => m.id === activa.value) ?? manos.value[0] ?? null)
const mano = computed<number[]>(() => actual.value?.mano ?? [])
const crupier = computed<number[]>(() => actual.value?.crupier ?? [])
const estado = computed(() => actual.value?.estado ?? '')
const premio = computed(() => actual.value?.premio ?? 0)
const valorJugador = computed(() => valor(mano.value))
const valorCrupier = computed(() => valor(crupier.value))
const cerrada = computed(() => estado.value !== '' && estado.value !== 'abierta')
const grupoCerrado = computed(() => manos.value.length > 0 && manos.value.every((m) => m.estado !== 'abierta'))
const dividida = computed(() => manos.value.length > 1)

// El seguro solo aplica con un As visible del crupier y una sola vez por mano.
const puedeAsegurar = computed(
  () => !cerrada.value && crupier.value.length > 0 && crupier.value[0] % 13 === 0 && !actual.value?.seguro,
)
// Rendirse es con la mano inicial de dos cartas, antes de pedir o doblar.
const puedeRendirse = computed(() => !cerrada.value && !dividida.value && mano.value.length === 2)
// Dividir: solo la mano inicial, con dos cartas del mismo valor y sin haber dividido.
const puedeDividir = computed(
  () =>
    !cerrada.value &&
    !dividida.value &&
    mano.value.length === 2 &&
    valorCarta(mano.value[0]) === valorCarta(mano.value[1]),
)

const etiquetas: Record<string, { texto: string; clase: string }> = {
  blackjack: { texto: 'Blackjack · pagas 3:2', clase: 'gana' },
  ganada: { texto: 'Ganaste', clase: 'gana' },
  perdida: { texto: 'Perdiste', clase: 'pierde' },
  empate: { texto: 'Empate · devuelve tu apuesta', clase: 'empate' },
  rendida: { texto: 'Te rendiste · recuperas la mitad', clase: 'empate' },
  seguro: { texto: 'El crupier tenía blackjack · recuperaste tu apuesta (seguro)', clase: 'empate' },
}

async function llamar(path: string, body: any = {}) {
  error.value = ''
  cargando.value = true
  try {
    const { data } = await api.post(path, body)
    manos.value = data.manos ?? []
    activa.value = data.activa || null
    grupoPremio.value = data.grupo_premio ?? data.premio ?? 0
    // Reparto/pedida: tono de carta; al cerrar, el arpegio del resultado.
    const ultimo = data.manos?.length ? data.manos[data.manos.length - 1].estado : data.estado
    if (ultimo && ultimo !== 'abierta') {
      if (ultimo === 'blackjack' || ultimo === 'ganada') reproducir('ganar')
      else if (ultimo === 'perdida' || ultimo === 'rendida') reproducir('perder')
      else reproducir('acierto')
    } else {
      reproducir('carta')
    }
    await auth.cargar()
  } catch (e: any) {
    error.value = e.message
  } finally {
    cargando.value = false
  }
}

function empezar() {
  manos.value = []
  activa.value = null
  grupoPremio.value = 0
  llamar('/casino/blackjack', { monto: props.monto, client_seed: props.clientSeed })
}

const idActiva = computed(() => activa.value ?? manos.value[0]?.id ?? null)
function pedir() { llamar(`/casino/blackjack/${idActiva.value}/hit`) }
function plantarse() { llamar(`/casino/blackjack/${idActiva.value}/stand`) }
function doblar() { llamar(`/casino/blackjack/${idActiva.value}/double`) }
function rendirse() { llamar(`/casino/blackjack/${idActiva.value}/surrender`) }
function asegurar() { llamar(`/casino/blackjack/${idActiva.value}/insurance`) }
function dividir() { llamar(`/casino/blackjack/${idActiva.value}/split`) }

// No se reparte al abrir la ventana: la apuesta solo se cobra cuando el jugador
// toca "Repartir". Antes abría el modal y ya te descontaba una mano.
const listaParaJugar = computed(() => manos.value.length === 0 && !cargando.value)
const apostadoTotal = computed(() => props.monto * (dividida.value ? 2 : 1))
</script>

<template>
  <div class="blackjack">
    <div class="mesa">
      <span class="placa">Limites {{ colones(monto) }} - {{ colones(monto * 20) }}</span>
      <div class="mesa-head">
        <h3>Blackjack <span class="suits">♠ ♥</span></h3>
        <p class="reglas">Paga 3 a 2 · Dividís dos cartas iguales · La casa se planta en 17</p>
      </div>

      <div class="zona" :class="{ 'resalta-ganador': grupoCerrado && estado === 'perdida' }">
        <div class="zona-head"><b>Casa</b><span class="valor" v-if="crupier.length">{{ grupoCerrado ? valorCrupier : valor(crupier.slice(0,1)) }}</span></div>
        <div class="cartas">
          <div v-for="(c, i) in crupier" :key="i" class="carta"
               :class="[roja(c) ? 'roja' : 'negra', { oculta: !grupoCerrado && i === 1 }]">
            <template v-if="grupoCerrado || i !== 1"><span class="fig">{{ figura(c) }}</span><span class="palo">{{ palo(c) }}</span></template>
            <span v-else class="dorso"></span>
          </div>
          <span v-if="!crupier.length" class="vacio">Esperando reparto</span>
        </div>
      </div>

      <div class="separador"><span>♠</span><i></i><span>♥</span></div>

      <div v-for="(m, idx) in manos" :key="m.id" class="zona"
           :class="[{ 'en-juego': m.id === activa && !grupoCerrado, 'resalta-ganador': grupoCerrado && (m.estado === 'blackjack' || m.estado === 'ganada') }, 'e-' + m.estado]">
        <div class="zona-head">
          <b>{{ dividida ? 'Vos ' + (idx + 1) : 'Vos' }}</b>
          <span class="valor" v-if="m.mano.length">{{ valor(m.mano) }}</span>
          <span v-if="dividida" class="mini-estado">{{ m.estado === 'abierta' ? 'En juego' : etiquetas[m.estado]?.texto ?? m.estado }}</span>
        </div>
        <div class="cartas">
          <div v-for="(c, i) in m.mano" :key="i" class="carta" :class="roja(c) ? 'roja' : 'negra'">
            <span class="fig">{{ figura(c) }}</span><span class="palo">{{ palo(c) }}</span>
          </div>
          <span v-if="!m.mano.length" class="vacio">Tu mano aparece aca</span>
        </div>
      </div>

      <div class="barra">
        <span class="barra-item">Saldo<b>{{ colones(auth.user?.saldo ?? 0) }}</b></span>
        <span class="barra-item">Apuesta<b>{{ colones(apostadoTotal) }}</b></span>
        <span class="barra-item aviso">{{ grupoCerrado ? 'Mano terminada' : 'La casa juega en 17' }}</span>
      </div>
    </div>

    <div v-if="error" class="error-text">{{ error }}</div>

    <div v-if="grupoCerrado" class="estado"
         :class="grupoPremio > apostadoTotal ? 'gana' : grupoPremio < apostadoTotal ? 'pierde' : 'empate'">
      <span>{{ grupoPremio > apostadoTotal ? 'Ganaste' : grupoPremio < apostadoTotal ? 'Perdiste' : 'Empate · devuelve tu apuesta' }}</span>
      <strong v-if="grupoPremio > apostadoTotal">{{ colones(grupoPremio) }}</strong>
    </div>

    <button v-if="listaParaJugar" class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
      {{ cargando ? 'Repartiendo…' : 'Repartir por ' + colones(monto) }}
    </button>

    <div v-else-if="!cerrada" class="acciones">
      <button class="btn btn-primary" :disabled="cargando" @click="pedir">Pedir</button>
      <button class="btn btn-ghost" :disabled="cargando" @click="plantarse">Plantarse</button>
      <button class="btn btn-gold" :disabled="cargando" @click="doblar">Doblar</button>
      <button v-if="puedeDividir" class="btn btn-gold" :disabled="cargando" @click="dividir">Dividir</button>
      <button v-if="puedeAsegurar" class="btn btn-ghost" :disabled="cargando" @click="asegurar">Seguro {{ colones(Math.floor(monto / 2)) }}</button>
      <button v-if="puedeRendirse" class="btn btn-ghost" :disabled="cargando" @click="rendirse">Rendirse</button>
    </div>
    <button v-else class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
      {{ cargando ? 'Repartiendo…' : 'Otra mano' }}
    </button>
  </div>
</template>

<style scoped>
.blackjack { display: flex; flex-direction: column; gap: 10px; }
.placa {
  position: absolute; top: 10px; left: 12px; z-index: 1;
  background: linear-gradient(180deg, #f5dd9a, #c9a227); color: #3a2a05;
  font-size: 9px; font-weight: 800; letter-spacing: .1em; text-transform: uppercase;
  padding: 4px 7px; border-radius: 4px; box-shadow: 0 3px 8px rgba(0,0,0,.4);
}
.mesa { position: relative; padding: 12px 14px 14px; border-radius: 14px; background: linear-gradient(160deg, #0d3b2c, #062018 70%); border: 1px solid rgba(232,206,122,.35); box-shadow: inset 0 14px 30px rgba(0,0,0,.5); }
.mesa-head { text-align: center; margin-bottom: 8px; padding-top: 6px; }
.mesa-head h3 { margin: 0; font-size: 22px; letter-spacing: .3em; text-transform: uppercase; color: #f5d778; text-shadow: 0 2px 0 #6b4a12; }
.mesa-head .suits { letter-spacing: .06em; }
.reglas { margin: 3px 0 0; font-size: 10.5px; letter-spacing: .08em; color: #e8ce7a; opacity: .9; }
.zona { display: flex; flex-direction: column; align-items: center; gap: 6px; padding: 6px 8px; border-radius: 10px; border: 1px solid transparent; }
.zona.en-juego { border-color: rgba(232,206,122,.4); background: rgba(232,206,122,.06); }
.zona.e-blackjack, .zona.e-ganada { border-color: rgba(245,185,66,.5); }
.zona.e-perdida, .zona.e-rendida { border-color: rgba(180,60,60,.45); }
.zona-head { width: 100%; display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 0 6px; }
.zona-head b { font-size: 11px; text-transform: uppercase; letter-spacing: .24em; color: #e8ce7a; }
.mini-estado { font-size: 10px; opacity: .75; }
.cartas { display: flex; gap: 8px; flex-wrap: wrap; min-height: 82px; align-items: center; justify-content: center; }
.vacio { font-size: 12px; opacity: .55; font-style: italic; }
.carta {
  width: 56px; height: 78px; border-radius: 9px; background: #fdfaf0; color: #15181d;
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 2px;
  font-weight: 800; box-shadow: 0 6px 12px rgba(0,0,0,.45); animation: aparecer .25s ease; border: 1px solid #d8c79a;
}
.carta .fig { font-size: 21px; line-height: 1; }
.carta .palo { font-size: 15px; line-height: 1; }
.carta.roja { color: #b3242f; }
.carta.oculta { background: repeating-linear-gradient(45deg,#0d4a35 0 8px,#0a3a29 8px 16px); border-color: #0a3a29; }
.carta.oculta .dorso { width: 24px; height: 15px; border-radius: 3px; background: #e8ce7a; opacity: .85; }
@keyframes aparecer { from { transform: translateY(-6px); opacity: 0; } to { transform: translateY(0); opacity: 1; } }
.separador { display: flex; align-items: center; gap: 8px; margin: 8px 6px; opacity: .55; }
.separador span { color: #e8ce7a; font-size: 11px; }
.separador i { flex: 1; height: 1px; background: linear-gradient(90deg, transparent, #e8ce7a, transparent); }
.valor { min-width: 30px; text-align: center; font-weight: 800; font-size: 16px; background: rgba(0,0,0,.35); border: 1px solid rgba(232,206,122,.5); border-radius: 8px; padding: 3px 7px; color: #f5d778; }
.barra {
  margin-top: 10px; padding: 8px 10px; display: flex; align-items: center; justify-content: space-between; gap: 10px;
  background: rgba(3,16,12,.75); border: 1px solid rgba(232,206,122,.35); border-radius: 10px;
}
.barra-item { font-size: 11px; color: #e8ce7a; opacity: .9; }
.barra-item b { color: #fff; margin-left: 4px; }
.barra-item.aviso { opacity: .55; font-style: italic; }
.error-text { color: #ffb3b3; font-size: 12px; text-align: center; }
.estado { text-align: center; padding: 9px; border-radius: 10px; background: rgba(0,0,0,.3); border: 1px solid rgba(232,206,122,.35); font-size: 13px; }
.estado.gana { color: #f5d778; border-color: rgba(245,185,66,.5); }
.estado.pierde { color: #ffb3b3; border-color: rgba(180,60,60,.45); }
.estado.empate { color: #e8ce7a; opacity: .9; }
.estado strong { color: #f5d778; font-size: 17px; margin-left: 6px; }
.acciones { display: flex; gap: 8px; flex-wrap: wrap; }
.acciones .btn { flex: 1 1 90px; }
</style>

<style scoped>
/* En horizontal (tablets/telefonos apaisados) la mesa se compacta para no exigir scroll. */
@media (max-height: 900px) {
  .mesa { padding: 8px 12px; border-radius: 14px; }
  .mesa-head h3 { font-size: 17px; letter-spacing: .2em; }
  .reglas { font-size: 9.5px; }
  .placa { top: 7px; left: 8px; font-size: 8px; padding: 3px 5px; }
  .zona { gap: 4px; padding: 4px 6px; }
  .cartas { min-height: 52px; gap: 5px; }
  .carta { width: 42px; height: 58px; border-radius: 7px; }
  .carta .fig { font-size: 17px; }
  .carta .palo { font-size: 12px; }
  .separador { margin: 6px 4px; }
  .barra { margin-top: 6px; padding: 5px 8px; }
  .blackjack { gap: 8px; }
}
</style>
