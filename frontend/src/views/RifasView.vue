<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { colones, fechaHora } from '@/utils/formato'
import Modal from '@/components/Modal.vue'

interface NumeroRifa {
  numero: number
  vendido: boolean
  mio: boolean
}

interface RifaCard {
  id: number
  titulo: string
  descripcion: string
  premio: string
  premio_monto: number
  precio_numero: number
  desde_numero: number
  hasta_numero: number
  fecha_sorteo: string
  estado: string
  numeros_ganadores: number[]
  vendidos: number
  total: number
  ganadores?: { numero: number; nombre: string }[]
}

interface MiNumero {
  rifa_id: number
  titulo: string
  premio: string
  estado: string
  numero: number
  monto: number
  fecha_sorteo: string
  ganador: boolean
}

const auth = useAuthStore()
const rifas = ref<RifaCard[]>([])
const mias = ref<MiNumero[]>([])
const cargando = ref(true)
const error = ref('')

const abierta = ref<RifaCard | null>(null)
const numeros = ref<NumeroRifa[]>([])
const elegidos = ref<number[]>([])
const cargandoDetalle = ref(false)
const comprando = ref(false)
const errorCompra = ref('')
const inputManual = ref('')

const progreso = computed(() => (r: RifaCard) =>
  r.total > 0 ? Math.round((r.vendidos / r.total) * 100) : 0,
)

async function cargar() {
  cargando.value = true
  error.value = ''
  try {
    const [{ data }, miasResp] = await Promise.all([
      api.get<RifaCard[]>('/rifas'),
      auth.isAutenticado ? api.get<MiNumero[]>('/rifas/mias') : Promise.resolve({ data: [] as MiNumero[] }),
    ])
    rifas.value = data
    mias.value = miasResp.data ?? []
  } catch (e: any) {
    error.value = e.message
  } finally {
    cargando.value = false
  }
}

onMounted(cargar)

function pad(n: number, r: RifaCard): string {
  const digitos = String(r.hasta_numero).length
  return String(n).padStart(digitos, '0')
}

function toggleNumero(n: number, vendido: boolean, mio: boolean) {
  if (vendido && !mio) return
  const i = elegidos.value.indexOf(n)
  if (i >= 0) elegidos.value.splice(i, 1)
  else {
    if (elegidos.value.length >= 50) return
    elegidos.value.push(n)
  }
}

function agregarManual(r: RifaCard) {
  const n = parseInt(inputManual.value, 10)
  if (!isNaN(n)) toggleNumero(n, numeros.value.find((x) => x.numero === n)?.vendido ?? false, false)
  inputManual.value = ''
}

function aleatorio(r: RifaCard) {
  const libres = numeros.value.filter((n) => !n.vendido || n.mio).map((n) => n.numero)
  if (!libres.length) return
  const n = libres[Math.floor(Math.random() * libres.length)]
  if (!elegidos.value.includes(n)) elegidos.value.push(n)
}

async function abrir(r: RifaCard) {
  if (r.estado !== 'abierta') return
  abierta.value = r
  elegidos.value = []
  errorCompra.value = ''
  cargandoDetalle.value = true
  try {
    const { data } = await api.get(`/rifas/${r.id}`)
    numeros.value = data.numeros ?? []
  } catch (e: any) {
    errorCompra.value = e.message
  } finally {
    cargandoDetalle.value = false
  }
}

async function comprar() {
  if (!abierta.value || elegidos.value.length === 0) return
  comprando.value = true
  errorCompra.value = ''
  try {
    await api.post(`/rifas/${abierta.value.id}/comprar`, { numeros: elegidos.value })
    await auth.cargar()
    abierta.value = null
    await cargar()
  } catch (e: any) {
    errorCompra.value = e.response?.data?.error ?? e.message
  } finally {
    comprando.value = false
  }
}

const totalCompra = computed(() =>
  abierta.value ? abierta.value.precio_numero * elegidos.value.length : 0,
)
</script>

<template>
  <div>
    <div class="page-head">
      <div class="eyebrow">Rifas</div>
      <h1>Rifas y sorteos especiales</h1>
      <p>
        Comprá números de rifas abiertas por el operador. Cada número es tuyo: seguilo hasta el
        sorteo y cobrá si sale premiado.
      </p>
    </div>

    <p v-if="error" class="error-text">{{ error }}</p>
    <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando rifas…</div>

    <template v-else>
      <div class="grid grid-3">
        <div v-for="r in rifas" :key="r.id" class="card card-hover rifa">
          <div style="display:flex; justify-content:space-between; align-items:flex-start; gap:8px">
            <h3 style="font-size:17px">{{ r.titulo }}</h3>
            <span class="badge" :class="r.estado === 'abierta' ? 'badge-ok' : r.estado === 'sorteada' ? 'badge-gold' : 'badge-bad'">
              {{ r.estado }}
            </span>
          </div>
          <p class="texto-muted" style="font-size:13px; margin:6px 0 0">{{ r.descripcion }}</p>
          <div class="premio">
            <span class="etq">Premio</span>
            <b>{{ r.premio }}</b>
            <span v-if="r.premio_monto > 0" class="etq">o {{ colones(r.premio_monto) }}</span>
          </div>
          <div class="meta">
            <span>Precio: <b class="mono texto-gold">{{ colones(r.precio_numero) }}</b> / número</span>
            <span>Rango: <b class="mono">{{ pad(r.desde_numero, r) }}–{{ pad(r.hasta_numero, r) }}</b></span>
            <span>Sorteo: <b>{{ fechaHora(r.fecha_sorteo) }}</b></span>
          </div>
          <div class="progreso">
            <div class="barra"><div class="llenado" :style="{ width: progreso(r) + '%' }" /></div>
            <span class="hint mono">{{ r.vendidos }}/{{ r.total }} ({{ progreso(r) }}%)</span>
          </div>

          <div v-if="r.estado === 'sorteada'" class="ganadores">
            <div class="etq">Ganadores</div>
            <ul>
              <li v-for="g in r.ganadores" :key="g.numero">
                <b class="mono">#{{ pad(g.numero, r) }}</b> · {{ g.nombre }}
              </li>
            </ul>
          </div>

          <button
            v-if="r.estado === 'abierta'"
            class="btn btn-primary btn-block"
            @click="abrir(r)"
          >
            Jugar
          </button>
        </div>
      </div>

      <div v-if="auth.isAutenticado" class="section-title">Mis números</div>
      <div v-if="auth.isAutenticado && mias.length" class="grid grid-3">
        <div v-for="(m, i) in mias" :key="i" class="card mia">
          <div style="display:flex; justify-content:space-between; align-items:center">
            <b>#{{ m.numero }}</b>
            <span class="badge" :class="m.ganador ? 'badge-gold' : 'badge-ok'">
              {{ m.ganador ? 'Ganador' : m.estado }}
            </span>
          </div>
          <p class="texto-muted" style="font-size:13px; margin:8px 0 0">{{ m.titulo }}</p>
          <p class="hint" style="margin:4px 0 0">{{ m.premio }} · pagaste {{ colones(m.monto) }}</p>
        </div>
      </div>
      <div v-else-if="auth.isAutenticado" class="card empty">
        <div class="ico">🎁</div>
        <h3>Aún no tenés números de rifas</h3>
        <p>Elegí una rifa abierta y marcá tus números.</p>
      </div>
    </template>

    <Modal v-if="abierta" :titulo="abierta.titulo" @cerrar="abierta = null">
      <div class="compra">
        <p class="hint">
          Precio por número: <b class="mono texto-gold">{{ colones(abierta.precio_numero) }}</b> ·
          Elegidos: <b class="mono">{{ elegidos.length }}</b>/50
        </p>
        <div v-if="cargandoDetalle" class="cargando"><span class="spinner"></span> Cargando números…</div>
        <template v-else>
          <div v-if="abierta.total <= 200" class="tabla">
            <button
              v-for="n in numeros"
              :key="n.numero"
              class="casilla"
              :class="{ vendida: n.vendido && !n.mio, mia: n.mio, elegida: elegidos.includes(n.numero) }"
              :disabled="n.vendido && !n.mio"
              @click="toggleNumero(n.numero, n.vendido, n.mio)"
            >
              {{ pad(n.numero, abierta) }}
            </button>
          </div>
          <div v-else class="manual">
            <input v-model="inputManual" placeholder="Número" class="input" @keyup.enter="agregarManual(abierta)" />
            <button class="btn btn-ghost" @click="agregarManual(abierta)">Agregar</button>
            <button class="btn btn-ghost" @click="aleatorio(abierta)">Suerte</button>
          </div>
          <div v-if="elegidos.length" class="elegidos">
            <span v-for="n in elegidos" :key="n" class="chip">{{ pad(n, abierta) }}</span>
          </div>
          <p v-if="errorCompra" class="error-text">{{ errorCompra }}</p>
          <button
            class="btn btn-primary btn-block"
            :disabled="elegidos.length === 0 || comprando"
            @click="comprar"
          >
            {{ comprando ? 'Comprando…' : 'Comprar por ' + colones(totalCompra) }}
          </button>
        </template>
      </div>
    </Modal>
  </div>
</template>

<style scoped>
.rifa { display: flex; flex-direction: column; gap: 10px; }
.premio { display: flex; align-items: baseline; gap: 8px; padding-top: 10px; border-top: 1px solid var(--border-soft); }
.premio .etq, .ganadores .etq { font-size: 10px; text-transform: uppercase; letter-spacing: .09em; color: var(--faint); }
.meta { display: flex; flex-direction: column; gap: 2px; font-size: 13px; color: var(--muted); }
.progreso { display: flex; align-items: center; gap: 10px; }
.barra { flex: 1; height: 8px; border-radius: 999px; background: var(--surface-2); overflow: hidden; }
.llenado { height: 100%; background: linear-gradient(90deg, var(--prim), var(--gold)); border-radius: 999px; }
.ganadores { padding: 10px; border-radius: 10px; background: var(--gold-dim); }
.ganadores ul { margin: 6px 0 0; padding-left: 16px; color: var(--text); font-size: 13px; }
.compra { display: flex; flex-direction: column; gap: 12px; }
.tabla { display: grid; grid-template-columns: repeat(10, 1fr); gap: 5px; max-height: 320px; overflow-y: auto; }
.casilla {
  padding: 8px 0; border-radius: 8px; border: 1px solid var(--border-soft);
  background: var(--surface); color: var(--muted); cursor: pointer; font-size: 13px;
}
.casilla.vendida { opacity: .3; cursor: not-allowed; }
.casilla.mia { border-color: var(--info); color: var(--info); }
.casilla.elegida { background: var(--prim-dim); border-color: var(--prim); color: var(--prim-2); font-weight: 700; }
.manual { display: flex; gap: 8px; }
.input { flex: 1; padding: 9px 11px; border-radius: 10px; border: 1px solid var(--border); background: var(--surface); color: var(--text); }
.elegidos { display: flex; flex-wrap: wrap; gap: 6px; }
.chip { padding: 4px 9px; border-radius: 999px; background: var(--prim-dim); color: var(--prim-2); font-size: 12px; font-weight: 600; }
.mia { display: flex; flex-direction: column; gap: 2px; }
</style>