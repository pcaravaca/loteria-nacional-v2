<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { colones } from '@/utils/formato'

const props = defineProps<{ monto: number; clientSeed: string }>()

const auth = useAuthStore()

const manoId = ref<number | null>(null)
const mano = ref<number[]>([])
const crupier = ref<number[]>([])
const estado = ref('')
const premio = ref(0)
const seguro = ref(0)
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

const valor = (cartas: number[]) => {
  let total = 0
  let ases = 0
  for (const c of cartas) {
    const v = c % 13
    if (v === 0) ases++
    total += v === 0 ? 11 : v >= 10 ? 10 : v + 1
  }
  while (total > 21 && ases > 0) {
    total -= 10
    ases--
  }
  return total
}

const valorJugador = computed(() => valor(mano.value))
const valorCrupier = computed(() => valor(crupier.value))
const cerrada = computed(() => estado.value !== '' && estado.value !== 'abierta')
// El seguro solo aplica con un As visible del crupier y una sola vez por mano.
const puedeAsegurar = computed(
  () => !cerrada.value && crupier.value.length > 0 && crupier.value[0] % 13 === 0 && seguro.value === 0,
)
// Rendirse es con la mano inicial de dos cartas, antes de pedir o doblar.
const puedeRendirse = computed(() => !cerrada.value && mano.value.length === 2)

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
    manoId.value = data.mano_id
    mano.value = data.mano
    crupier.value = data.crupier
    estado.value = data.estado
    premio.value = data.premio ?? 0
    seguro.value = data.seguro ?? 0
    await auth.cargar()
  } catch (e: any) {
    error.value = e.message
  } finally {
    cargando.value = false
  }
}

function empezar() {
  mano.value = []
  crupier.value = []
  estado.value = ''
  premio.value = 0
  seguro.value = 0
  llamar('/casino/blackjack', { monto: props.monto, client_seed: props.clientSeed })
}

function pedir() { llamar(`/casino/blackjack/${manoId.value}/hit`) }
function plantarse() { llamar(`/casino/blackjack/${manoId.value}/stand`) }
function doblar() { llamar(`/casino/blackjack/${manoId.value}/double`) }
function rendirse() { llamar(`/casino/blackjack/${manoId.value}/surrender`) }
function asegurar() { llamar(`/casino/blackjack/${manoId.value}/insurance`) }

// No se reparte al abrir la ventana: la apuesta solo se cobra cuando el jugador
// toca "Repartir". Antes abría el modal y ya te descontaba una mano.
const listaParaJugar = computed(() => !mano.value.length && !cargando.value)
</script>

<template>
  <div class="blackjack">
    <div class="mesa">
      <span class="placa">Limites {{ colones(monto) }} - {{ colones(monto * 20) }}</span>
      <div class="mesa-head">
        <h3>Blackjack <span class="suits">♠ ♥</span></h3>
        <p class="reglas">Paga 3 a 2 · La casa se planta en 17 y pide hasta 16</p>
      </div>

      <div class="zona">
        <div class="zona-head"><b>Casa</b><span class="valor" v-if="crupier.length">{{ cerrada ? valorCrupier : valor(crupier.slice(0,1)) }}</span></div>
        <div class="cartas">
          <div v-for="(c, i) in crupier" :key="i" class="carta"
               :class="[roja(c) ? 'roja' : 'negra', { oculta: !cerrada && i === 1 }]">
            <template v-if="cerrada || i !== 1"><span class="fig">{{ figura(c) }}</span><span class="palo">{{ palo(c) }}</span></template>
            <span v-else class="dorso"></span>
          </div>
          <span v-if="!crupier.length" class="vacio">Esperando reparto</span>
        </div>
      </div>

      <div class="separador"><span>♠</span><i></i><span>♥</span></div>

      <div class="zona">
        <div class="zona-head"><b>Vos</b><span class="valor" v-if="mano.length">{{ valorJugador }}</span></div>
        <div class="cartas">
          <div v-for="(c, i) in mano" :key="i" class="carta" :class="roja(c) ? 'roja' : 'negra'">
            <span class="fig">{{ figura(c) }}</span><span class="palo">{{ palo(c) }}</span>
          </div>
          <span v-if="!mano.length" class="vacio">Tu mano aparece aca</span>
        </div>
      </div>

      <div class="barra">
        <span class="barra-item">Saldo <b>{{ colones(auth.user?.saldo ?? 0) }}</b></span>
        <span class="barra-item">Apuesta <b>{{ colones(monto) }}</b></span>
        <span class="barra-item aviso">{{ cerrada ? 'Mano terminada' : 'La casa juega en 17' }}</span>
      </div>
    </div>

    <div v-if="error" class="error-text">{{ error }}</div>

    <div v-if="estado && estado !== 'abierta'" class="estado" :class="etiquetas[estado]?.clase">
      <span>{{ etiquetas[estado]?.texto ?? estado }}</span>
      <strong v-if="premio > 0">{{ colones(premio) }}</strong>
    </div>

    <button v-if="listaParaJugar" class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
      {{ cargando ? 'Repartiendo…' : 'Repartir por ' + colones(monto) }}
    </button>

    <div v-else-if="!cerrada" class="acciones">
      <button class="btn btn-primary" :disabled="cargando" @click="pedir">Pedir</button>
      <button class="btn btn-ghost" :disabled="cargando" @click="plantarse">Plantarse</button>
      <button class="btn btn-gold" :disabled="cargando" @click="doblar">Doblar</button>
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
.mesa-head { text-align: center; margin-bottom: 8px; padding-top: 6px; }
.mesa-head h3 { margin: 0; font-size: 22px; letter-spacing: .3em; text-transform: uppercase; color: #f5d778; text-shadow: 0 2px 0 #6b4a12; }
.mesa-head .suits { letter-spacing: .06em; }
.reglas { margin: 3px 0 0; font-size: 10.5px; letter-spacing: .08em; color: #e8ce7a; opacity: .9; }
.zona { display: flex; flex-direction: column; align-items: center; gap: 6px; }
.zona-head { width: 100%; display: flex; align-items: center; justify-content: space-between; padding: 0 6px; }
.zona-head b { font-size: 11px; text-transform: uppercase; letter-spacing: .24em; color: #e8ce7a; }
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
.estado { text-align: center; padding: 9px; border-radius: 10px; background: rgba(0,0,0,.3); border: 1px solid rgba(232,206,122,.35); font-size: 13px; }
.estado.gana { color: #f5d778; }
.estado strong { color: #f5d778; font-size: 17px; margin-left: 6px; }
.acciones { display: flex; gap: 8px; flex-wrap: wrap; }
.acciones .btn { flex: 1 1 90px; }
</style>