<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api/client'
import type { CasinoGame, CasinoRound, SemillaResponse } from '@/types'
import { colones } from '@/utils/formato'
import { alternarSfx, sfxHabilitado } from '@/utils/sfx'
import { cartas, type CodigoJuego } from '@/assets/casino'
import { TEMAS, type TemaCasino } from '@/assets/casino/temas'
import { agruparJuegos } from '@/utils/casinoGrupos'
import { useAuthStore } from '@/stores/auth'
import Modal from '@/components/Modal.vue'
import EmptyState from '@/components/EmptyState.vue'
import CasinoGameModal from '@/components/casino/CasinoGameModal.vue'

const juegos = ref<CasinoGame[]>([])
const rondas = ref<CasinoRound[]>([])
const cargando = ref(true)
const error = ref('')
const sonido = ref(sfxHabilitado())
const abierto = ref<CasinoGame | null>(null)
const clientSeed = ref('')
const semillaHash = ref('')
const verificando = ref<any | null>(null)
const auth = useAuthStore()
const tema = computed(() => auth.temaCasino)
const grupos = computed(() => agruparJuegos(juegos.value))
const rondasRecientes = computed(() => rondas.value.slice(0, 6))
// Filtro por tipo de juego: evita tener que scrollear toda la lista.
const filtro = ref('todos')
const gruposVisibles = computed(() =>
  filtro.value === 'todos' ? grupos.value : grupos.value.filter((g) => g.id === filtro.value),
)
function elegirFiltro(id: string) {
  filtro.value = id
  document.getElementById('lista-juegos')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

async function elegirTema(codigo: TemaCasino) {
  await auth.fijarTema(codigo)
}

async function cargar() {
  cargando.value = true
  error.value = ''
  try {
    const [j, s, r] = await Promise.all([
      api.get<CasinoGame[]>('/casino/games'),
      api.get<SemillaResponse>('/casino/seed'),
      api.get<CasinoRound[]>('/casino/rounds'),
    ])
    juegos.value = j.data
    clientSeed.value = s.data.client_seed
    semillaHash.value = s.data.hash
    rondas.value = r.data
  } catch (e: any) {
    error.value = e.message
  } finally {
    cargando.value = false
  }
}

onMounted(cargar)

async function verificar(ronda: CasinoRound) {
  try {
    const { data } = await api.get(`/casino/rounds/${ronda.id}/verify`)
    verificando.value = data
  } catch (e: any) {
    error.value = e.message
  }
}
</script>

<template>
  <div class="casino-page" :data-tema="tema">
    <div class="page-head">
      <div class="eyebrow">Mini casino</div>
      <h1>La suerte, en miniatura</h1>
      <p>
        {{ juegos.length }} juegos rapidos con resultado generado en el servidor y pago inmediato al mismo saldo.
        Cada jugada queda comprometida por un hash y puedes auditarla.
      </p>
      <p class="hash-line">
        Semilla del servidor: <span class="hash">{{ semillaHash.slice(0, 24) }}</span>
        · <button class="link" @click="cargar">Rotar semilla</button>
        · <button class="link" @click="sonido = alternarSfx()">{{ sonido ? '🔊 Sonido' : '🔇 Mudo' }}</button>
      </p>
    </div>

    <div class="ambiente">
      <span class="ambiente-etq">Ambiente</span>
      <button
        v-for="t in TEMAS"
        :key="t.codigo"
        class="ambiente-chip"
        :class="{ on: tema === t.codigo }"
        :title="t.descripcion"
        @click="elegirTema(t.codigo)"
      >
        <i class="punto" :style="{ background: t.muestra }"></i>
        {{ t.nombre }}
      </button>
      <span class="ambiente-nota">Se aplica a todos los juegos y queda guardado en tu perfil.</span>
    </div>

    <p class="juego-responsable">
      Juega con responsabilidad · Solo mayores de 18 años · Los topes diarios se aplican por juego y usuario.
    </p>

    <p v-if="error" class="error-text" style="margin-bottom: 14px">{{ error }}</p>
    <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando casino…</div>

    <template v-else>
      <div class="section-title">Tus ultimas jugadas</div>
      <div v-if="rondasRecientes.length" class="table-wrap historial-arriba">
        <table class="table">
          <thead>
            <tr><th>Juego</th><th>Monto</th><th>Multipl</th><th>Premio</th><th>Fecha</th><th></th></tr>
          </thead>
          <tbody>
            <tr v-for="r in rondasRecientes" :key="r.id">
              <td class="fuerte">{{ r.game }}</td>
              <td class="mono">{{ colones(r.monto) }}</td>
              <td class="mono">x{{ r.multiplicador }}</td>
              <td class="mono" :style="{ color: r.premio > 0 ? 'var(--gold)' : 'var(--muted)' }">
                {{ r.premio ? colones(r.premio) : '—' }}
              </td>
              <td>{{ new Date(r.created_at).toLocaleString('es-CR', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' }) }}</td>
              <td><button class="btn btn-ghost btn-sm" @click="verificar(r)">Verificar</button></td>
            </tr>
          </tbody>
        </table>
        <p v-if="rondas.length > rondasRecientes.length" class="mas-jugadas">
          y {{ rondas.length - rondasRecientes.length }} jugadas mas en tu historial.
        </p>
      </div>
      <EmptyState
        v-else
        icono="🎰"
        titulo="Aun no jugaste en el casino"
        texto="Elegi un juego, hacé una apuesta y tu historial aparecera aqui."
      />

      <div id="lista-juegos" class="chips-filtro" role="navigation" aria-label="Tipos de juego">
        <button
          class="chip"
          :class="{ activo: filtro === 'todos' }"
          @click="elegirFiltro('todos')"
        >Todos</button>
        <button
          v-for="g in grupos"
          :key="g.id"
          class="chip"
          :class="{ activo: filtro === g.id }"
          @click="elegirFiltro(g.id)"
        >{{ g.titulo }}</button>
      </div>

      <section v-for="g in gruposVisibles" :key="g.id" class="grupo-casino">
        <div class="grupo-head">
          <h2>{{ g.titulo }}</h2>
          <p v-if="g.descripcion">{{ g.descripcion }}</p>
        </div>
        <div class="grid grid-3">
          <div
            v-for="j in g.juegos"
            :key="j.codigo"
            class="card juego"
            :class="{ apagada: !j.activo }"
          >
            <div class="arte">
              <img :src="cartas[j.codigo as CodigoJuego]" :alt="j.nombre" draggable="false" />
              <span v-if="!j.activo" class="proximamente">Proximamente</span>
            </div>
            <h3>{{ j.nombre }}</h3>
            <p class="desc">{{ j.descripcion }}</p>
            <div class="meta">
              <div>Min<b>{{ colones(j.min_apuesta) }}</b></div>
              <div>RTP<b>{{ j.rtp }}%</b></div>
              <div>Max<b>x{{ j.max_multiplicador }}</b></div>
            </div>
            <div v-if="j.max_premio > 0" class="techo">
              Premio maximo por jugada: <b>{{ colones(j.max_premio) }}</b>
            </div>
            <button class="btn btn-primary btn-block" :disabled="!j.activo" @click="abierto = j">
              Jugar
            </button>
          </div>
        </div>
      </section>
    </template>

    <CasinoGameModal
      v-if="abierto"
      :juego="abierto"
      :client-seed="clientSeed"
      @cerrar="abierto = null; cargar()"
    />

    <Modal v-if="verificando" titulo="Verificacion provably fair" @cerrar="verificando = null">
      <div class="verif">
        <p class="linea"><span>Ronda</span><b>#{{ verificando.ronda_id }}</b></p>
        <p class="linea"><span>Juego</span><b>{{ verificando.juego }}</b></p>
        <p class="linea"><span>Nonce</span><b>{{ verificando.nonce }}</b></p>
        <p class="linea"><span>Client seed</span><b class="mono">{{ verificando.client_seed }}</b></p>
        <p class="linea"><span>Server seed</span><b class="mono">{{ verificando.server_seed }}</b></p>
        <p class="linea"><span>Hash</span><b class="mono">{{ verificando.server_seed_hash }}</b></p>
        <p class="linea">
          <span>Coincide</span>
          <b :style="{ color: verificando.verificado ? 'var(--prim-2)' : '#ff9b9b' }">
            {{ verificando.verificado ? 'SI' : 'NO' }}
          </b>
        </p>
        <p class="nota">
          Reproducibles con HMAC-SHA256(server_seed, "client_seed:nonce:i") en cada posicion.
        </p>
      </div>
      <template #acciones>
        <button class="btn btn-primary" @click="verificando = null">Cerrar</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>.chips-filtro {
  position: sticky;
  top: 0;
  z-index: 5;
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding: 10px 2px 12px;
  background: var(--bg);
  scrollbar-width: none;
}
.chips-filtro::-webkit-scrollbar { display: none; }
.chip {
  flex: 0 0 auto;
  border: 1px solid var(--line);
  background: var(--surface);
  color: var(--muted);
  border-radius: 999px;
  padding: 7px 14px;
  font-size: 13px;
  cursor: pointer;
}
.chip.activo {
  background: var(--prim);
  border-color: var(--prim);
  color: #fff;
  font-weight: 600;
}

.ambiente {
  display: flex; align-items: center; gap: 8px; flex-wrap: wrap;
  background: var(--bg-2); border: 1px solid var(--border-soft);
  border-radius: 12px; padding: 10px 12px; margin-bottom: 14px;
}
.ambiente-etq { font-size: 11px; letter-spacing: .16em; text-transform: uppercase; color: var(--faint); }
.ambiente-chip {
  display: inline-flex; align-items: center; gap: 7px;
  border: 1px solid var(--border-soft); background: var(--surface); color: var(--muted);
  border-radius: 999px; padding: 7px 13px; font-size: 12.5px; cursor: pointer;
}
.ambiente-chip:hover { color: var(--text); border-color: var(--border); }
.ambiente-chip.on { border-color: var(--gold); color: var(--gold); font-weight: 600; }
.ambiente-chip .punto { width: 12px; height: 12px; border-radius: 50%; display: inline-block; }
.ambiente-nota { margin-left: auto; font-size: 11.5px; color: var(--faint); }
.hash-line { font-size: 12px; color: var(--faint); margin-top: 8px; }
.hash { font-family: ui-monospace, monospace; color: var(--muted); }
.link { background: none; border: 0; color: var(--prim-2); cursor: pointer; font-size: 12px; padding: 0; }
.grupo-casino { margin-top: 26px; }
.grupo-head h2 { margin: 0; font-size: 18px; letter-spacing: .02em; }
.grupo-head p { margin: 4px 0 12px; color: var(--muted); font-size: 13px; }
.historial-arriba { margin-bottom: 26px; }
.mas-jugadas { margin: 8px 0 0; font-size: 12px; color: var(--faint); }
.juego-responsable {
  margin: 0 0 18px; font-size: 12px; color: var(--faint);
  border: 1px solid var(--border-soft); border-radius: 10px; padding: 8px 12px;
}
.juego { display: flex; flex-direction: column; }
.juego .arte { position: relative; height: 150px; border-radius: 12px; overflow: hidden; border: 1px solid var(--border-soft); margin-bottom: 12px; }
.juego .arte img { width: 100%; height: 100%; object-fit: cover; display: block; }
.juego h3 { margin: 0 0 4px; }
.juego .desc { color: var(--muted); font-size: 13px; min-height: 34px; margin: 0 0 10px; }
.juego .meta { display: flex; gap: 12px; padding: 10px 0; margin: 0 0 12px; border-top: 1px solid var(--border-soft); border-bottom: 1px solid var(--border-soft); }
.juego .meta div { font-size: 11px; color: var(--faint); text-transform: uppercase; letter-spacing: .06em; }
.juego .meta b { display: block; color: var(--text); font-size: 13px; letter-spacing: 0; text-transform: none; margin-top: 2px; }
.juego .techo { font-size: 11px; color: var(--faint); margin: -4px 0 10px; }
.juego .techo b { color: var(--muted); }
.juego.apagada .arte img { filter: grayscale(.6) brightness(.6); }
.proximamente {
  position: absolute; inset: 0; display: grid; place-items: center;
  background: rgba(3, 6, 10, .55); color: var(--gold); font-weight: 700; letter-spacing: .1em; text-transform: uppercase; font-size: 12px;
}
.verif { display: flex; flex-direction: column; gap: 8px; }
.verif .linea { display: flex; justify-content: space-between; gap: 12px; font-size: 13px; margin: 0; }
.verif .linea span { color: var(--faint); }
.verif .mono { font-family: ui-monospace, monospace; font-size: 12px; word-break: break-all; }
.verif .nota { font-size: 11px; color: var(--faint); margin-top: 10px; }
</style>
