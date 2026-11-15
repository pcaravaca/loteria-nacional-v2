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

// Al montar se reparte sola la primera mano.
empezar()
</script>

<template>
  <div class="blackjack">
    <div class="mesa">
      <div class="fila">
        <span class="quien">Casa</span>
        <div class="cartas">
          <div
            v-for="(c, i) in crupier"
            :key="i"
            class="carta"
            :class="[roja(c) ? 'roja' : 'negra', { oculta: !cerrada && i === 1 }]"
          >
            <template v-if="cerrada || i !== 1">
              <span class="fig">{{ figura(c) }}</span>
              <span class="palo">{{ palo(c) }}</span>
            </template>
            <span v-else class="dorso">LN</span>
          </div>
        </div>
        <span class="valor" v-if="crupier.length">{{ cerrada ? valorCrupier : valor(crupier.slice(0,1)) }}</span>
      </div>

      <div class="fila">
        <span class="quien">Vos</span>
        <div class="cartas">
          <div v-for="(c, i) in mano" :key="i" class="carta" :class="roja(c) ? 'roja' : 'negra'">
            <span class="fig">{{ figura(c) }}</span>
            <span class="palo">{{ palo(c) }}</span>
          </div>
        </div>
        <span class="valor" v-if="mano.length">{{ valorJugador }}</span>
      </div>
    </div>

    <div v-if="error" class="error-text">{{ error }}</div>

    <div v-if="estado && estado !== 'abierta'" class="estado" :class="etiquetas[estado]?.clase">
      <span>{{ etiquetas[estado]?.texto ?? estado }}</span>
      <strong v-if="premio > 0">{{ colones(premio) }}</strong>
    </div>

    <div v-if="!cerrada" class="acciones">
      <button class="btn btn-primary" :disabled="cargando" @click="pedir">Pedir</button>
      <button class="btn btn-ghost" :disabled="cargando" @click="plantarse">Plantarse</button>
      <button class="btn btn-gold" :disabled="cargando" @click="doblar">Doblar</button>
      <button v-if="puedeAsegurar" class="btn btn-ghost" :disabled="cargando" title="Mitad de la apuesta; paga 2:1 si el crupier tiene blackjack" @click="asegurar">
        Seguro {{ colones(Math.floor(monto / 2)) }}
      </button>
      <button v-if="puedeRendirse" class="btn btn-ghost" :disabled="cargando" @click="rendirse">Rendirse</button>
    </div>
    <button v-else class="btn btn-primary btn-block" :disabled="cargando" @click="empezar">
      {{ cargando ? 'Repartiendo…' : 'Otra mano' }}
    </button>
  </div>
</template>

<style scoped>
.blackjack { display: flex; flex-direction: column; gap: 14px; }
.mesa {
  background: linear-gradient(180deg, rgba(16,185,129,.10), rgba(16,185,129,.03));
  border: 1px solid var(--border-soft); border-radius: 12px; padding: 14px;
  display: flex; flex-direction: column; gap: 12px;
}
.fila { display: flex; align-items: center; gap: 10px; }
.quien { width: 44px; font-size: 11px; text-transform: uppercase; letter-spacing: .08em; color: var(--faint); }
.cartas { display: flex; gap: 6px; flex: 1; flex-wrap: wrap; }
.carta {
  width: 44px; height: 60px; border-radius: 8px; background: #f4f6f8; color: #15181d;
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 1px;
  font-weight: 800; box-shadow: 0 4px 10px rgba(0,0,0,.35); animation: aparecer .25s ease;
}
.carta .fig { font-size: 17px; line-height: 1; }
.carta .palo { font-size: 13px; line-height: 1; }
.carta.roja { color: #b3242f; }
.carta.negra { color: #15181d; }
.carta.oculta { background: linear-gradient(140deg, #0b8f68, #064e3b); color: #d1fae5; }
.carta.oculta .dorso { font-size: 11px; letter-spacing: .12em; font-weight: 700; }
@keyframes aparecer { from { transform: translateY(-6px); opacity: 0; } to { transform: translateY(0); opacity: 1; } }
.valor {
  min-width: 28px; text-align: center; font-weight: 800; font-size: 18px;
  background: #05080c; border: 1px solid var(--border); border-radius: 8px; padding: 5px 6px;
}
.estado { text-align: center; padding: 10px; border-radius: 10px; background: rgba(255,255,255,.02); border: 1px solid var(--border-soft); }
.estado.gana { background: var(--prim-dim); border-color: rgba(16,185,129,.35); color: var(--prim-2); }
.estado.pierde { color: var(--muted); }
.estado.empate { color: var(--gold); }
.estado strong { color: var(--gold); font-size: 18px; margin-left: 6px; }
.acciones { display: flex; gap: 8px; flex-wrap: wrap; }
.acciones .btn { flex: 1 1 96px; }
</style>
