<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { colones } from '@/utils/formato'
import { reproducir } from '@/utils/sfx'

const props = defineProps<{ codigo: string; monto: number; clientSeed: string }>()

const auth = useAuthStore()
const palos = ['♠', '♥', '♦', '♣']
const etiquetas: Record<string, string> = {
  RoyalFlush: 'Escalera real',
  StraightFlush: 'Escalera de color',
  FiveOfAKind: 'Quintilla',
  FourOfAKind: 'Poker',
  FullHouse: 'Full',
  Flush: 'Color',
  Straight: 'Escalera',
  ThreeOfAKind: 'Tercia',
  TwoPair: 'Doble par',
  Pair: 'Par',
}

type Fase = 'apuesta' | 'cartas' | 'final'
const fase = ref<Fase>('apuesta')
const cartas = ref<string[]>([])
const retenidas = ref<boolean[]>([false, false, false, false, false])
const manoId = ref<number | null>(null)
const premio = ref(0)
const multiplicador = ref(0)
const manoFinal = ref('')
const cargando = ref(false)
const error = ref('')

function etiqueta(clave: string): { rango: string; palo: string; roja: boolean; comodin: boolean } {
  if (clave === 'comodin') return { rango: 'Joker', palo: '🃏', roja: false, comodin: true }
  const [r, p] = clave.split('-').map(Number)
  const rangos: Record<number, string> = { 11: 'J', 12: 'Q', 13: 'K', 14: 'A' }
  return {
    rango: rangos[r] ?? String(r),
    palo: palos[p] ?? '',
    roja: p === 1 || p === 2,
    comodin: false,
  }
}

async function repartir() {
  error.value = ''
  cargando.value = true
  try {
    const { data } = await api.post('/casino/poker', {
      juego: props.codigo,
      monto: props.monto,
      client_seed: props.clientSeed,
    })
    manoId.value = data.mano_id
    cartas.value = data.cartas
    retenidas.value = [false, false, false, false, false]
    premio.value = 0
    multiplicador.value = 0
    fase.value = 'cartas'
    reproducir('carta')
    await auth.cargar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
  } finally {
    cargando.value = false
  }
}

async function cobrar() {
  if (!manoId.value) return
  error.value = ''
  cargando.value = true
  try {
    const indices = retenidas.value.flatMap((r, i) => (r ? [i] : []))
    const { data } = await api.post(`/casino/poker/${manoId.value}/cobrar`, { retener: indices })
    cartas.value = data.resultado?.cartas ?? cartas.value
    retenidas.value = data.resultado?.retencion ?? retenidas.value
    premio.value = data.premio ?? 0
    multiplicador.value = data.resultado?.multipl ?? 0
    manoFinal.value = data.resultado?.mano ?? ''
    fase.value = 'final'
    reproducir('carta')
    window.setTimeout(() => reproducir(premio.value > 0 ? 'ganar' : 'perder'), 280)
    await auth.cargar()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? e.message
  } finally {
    cargando.value = false
  }
}

function nueva() {
  fase.value = 'apuesta'
  cartas.value = []
  retenidas.value = [false, false, false, false, false]
  manoId.value = null
  premio.value = 0
  multiplicador.value = 0
  manoFinal.value = ''
  error.value = ''
}

onMounted(nueva)
</script>

<template>
  <div class="poker">
    <div v-if="fase === 'apuesta'" class="esperando">
      <span>Elegí las cartas que querés sostener y cobrá según la tabla.</span>
      <button class="btn btn-primary btn-block" :disabled="cargando" @click="repartir">
        {{ cargando ? 'Repartiendo…' : 'Repartir' }}
      </button>
    </div>

    <template v-else>
      <div class="mesa">
        <button
          v-for="(c, i) in cartas"
          :key="i"
          type="button"
          class="carta"
          :class="{
            roja: etiqueta(c).roja,
            retenida: fase === 'cartas' && retenidas[i],
            fija: fase === 'final' && retenidas[i],
            'resalta-ganador': fase === 'final' && premio > 0,
            comodin: etiqueta(c).comodin,
          }"
          :disabled="fase === 'final' || cargando"
          @click="retenidas[i] = !retenidas[i]"
        >
          <span class="rango">{{ etiqueta(c).rango }}</span>
          <span class="palo">{{ etiqueta(c).palo }}</span>
          <span v-if="fase === 'cartas' && retenidas[i]" class="marca">SOSTENIDA</span>
          <span v-else-if="fase === 'final' && retenidas[i]" class="marca">QUEDÓ</span>
        </button>
      </div>

      <div v-if="fase === 'final'" class="estado" :class="{ gana: premio > 0 }">
        <template v-if="premio > 0">
          <span>{{ etiquetas[manoFinal] ?? manoFinal }}</span>
          <strong>x{{ multiplicador }}</strong>
          <span class="premio">{{ colones(premio) }}</span>
        </template>
        <span v-else class="neutro">Sin premio</span>
      </div>

      <div v-if="fase === 'cartas'" class="acciones-poker">
        <span class="ayuda">Tocá una carta para sostenerla; lo demás se reemplaza.</span>
        <button class="btn btn-primary btn-block" :disabled="cargando" @click="cobrar">
          {{ cargando ? 'Cobrando…' : 'Cobrar' }}
        </button>
      </div>

      <button v-if="fase === 'final'" class="btn btn-ghost btn-block" :disabled="cargando" @click="nueva">
        Jugar de nuevo
      </button>
    </template>

    <p v-if="error" class="error-text">{{ error }}</p>
  </div>
</template>

<style scoped>
.poker { display: flex; flex-direction: column; gap: 12px; }
.esperando { display: flex; flex-direction: column; gap: 10px; color: var(--muted); font-size: 12.5px; }
.mesa {
  display: flex; gap: 8px; padding: 14px; border-radius: 16px;
  background: radial-gradient(circle at 50% 0%, #14382a, #07120d 70%);
  border: 1px solid var(--border);
}
.carta {
  position: relative; flex: 1; height: 120px; border-radius: 10px; cursor: pointer;
  background: linear-gradient(160deg, #f8f6ee, #d9d4c3);
  color: #10161c; display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 2px; font-weight: 700; box-shadow: 0 6px 14px rgba(0, 0, 0, .35);
  border: 1px solid transparent; transition: transform .15s, box-shadow .15s;
  animation: aparecer .25s ease both;
}
@keyframes aparecer { from { transform: translateY(-8px); opacity: 0; } to { transform: translateY(0); opacity: 1; } }
.carta.roja { color: #b3202c; }
.carta.comodin { background: linear-gradient(160deg, #2a1e3f, #120a1c); color: #e8d9a8; }
.carta.retenida { transform: translateY(-8px); box-shadow: 0 0 0 2px var(--gold), 0 10px 18px rgba(0, 0, 0, .4); }
.carta.fija { box-shadow: 0 0 0 2px rgba(201, 162, 75, .5); }
.carta:disabled { cursor: default; }
.carta .rango { font-size: 30px; line-height: 1; }
.carta .palo { font-size: 20px; line-height: 1; }
.carta .marca {
  position: absolute; bottom: -8px; left: 50%; transform: translateX(-50%);
  background: var(--gold); color: #1a1408; font-size: 8.5px; letter-spacing: .08em;
  padding: 2px 5px; border-radius: 5px; white-space: nowrap;
}
.acciones-poker { display: flex; flex-direction: column; gap: 8px; }
.acciones-poker .ayuda { font-size: 11.5px; color: var(--muted); text-align: center; }
.estado {
  display: flex; align-items: center; justify-content: center; gap: 10px;
  background: rgba(255, 255, 255, .02); border: 1px solid var(--border-soft);
  border-radius: 10px; padding: 10px; color: var(--muted);
}
.estado.gana { background: var(--prim-dim); border-color: rgba(16, 185, 129, .35); color: var(--prim-2); }
.estado strong { color: var(--gold); }
.estado .premio { font-weight: 700; }
.estado .neutro { color: var(--muted); }
.error-text { color: #f87171; font-size: 12px; margin: 0; }
</style>
