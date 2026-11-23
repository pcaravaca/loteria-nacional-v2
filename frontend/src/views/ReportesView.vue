<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api/client'
import type { Resumen } from '@/types'
import { colones } from '@/utils/formato'
import TablaApuestas from '@/components/reportes/TablaApuestas.vue'
import TablaJugadas from '@/components/reportes/TablaJugadas.vue'
import TablaMovimientos from '@/components/reportes/TablaMovimientos.vue'
import TablaResultados from '@/components/reportes/TablaResultados.vue'
import TablaRifas from '@/components/reportes/TablaRifas.vue'

// Reportes concentra todos los listados del cliente: el resumen, las apuestas
// de sorteo, las rifas, las jugadas del casino y los movimientos de saldo. Cada
// seccion pagina por su lado.
const TABS = [
  { id: 'resumen', titulo: 'Resumen' },
  { id: 'apuestas', titulo: 'Mis apuestas' },
  { id: 'rifas', titulo: 'Mis rifas' },
  { id: 'jugadas', titulo: 'Jugadas del casino' },
  { id: 'movimientos', titulo: 'Movimientos de saldo' },
  { id: 'resultados', titulo: 'Resultados por sorteo' },
] as const

type TabId = (typeof TABS)[number]['id']

const route = useRoute()
const router = useRouter()

function esTab(v: unknown): v is TabId {
  return typeof v === 'string' && TABS.some((t) => t.id === v)
}

const tab = ref<TabId>(esTab(route.query.tab) ? (route.query.tab as TabId) : 'resumen')

watch(tab, (v) => {
  router.replace({ query: { ...route.query, tab: v } }).catch(() => {})
})

const datos = ref<Resumen | null>(null)
const cargando = ref(true)
const error = ref('')

async function cargarResumen() {
  cargando.value = true
  error.value = ''
  try {
    const { data } = await api.get<Resumen>('/reportes/resumen')
    datos.value = data
  } catch (e: any) {
    error.value = e.message
  } finally {
    cargando.value = false
  }
}

onMounted(() => {
  if (tab.value === 'resumen') cargarResumen()
})
</script>

<template>
  <div>
    <div class="page-head">
      <div class="eyebrow">Estadísticas</div>
      <h1>Reportes</h1>
      <p>Tus apuestas, rifas, jugadas, movimientos y resultados, todos paginados.</p>
    </div>

    <div class="chips-filtro" role="navigation" aria-label="Secciones de reportes">
      <button
        v-for="t in TABS"
        :key="t.id"
        class="chip"
        :class="{ activo: tab === t.id }"
        @click="tab = t.id"
      >{{ t.titulo }}</button>
    </div>

    <div v-if="tab === 'resumen'">
      <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando reportes…</div>
      <p v-else-if="error" class="error-text">{{ error }}</p>

      <template v-else-if="datos">
        <div class="grid grid-4">
          <div class="card stat">
            <div class="etq">Total apostado</div>
            <div class="valor">{{ colones(datos.total_apostado) }}</div>
          </div>
          <div class="card stat">
            <div class="etq">Premios pagados</div>
            <div class="valor texto-gold">{{ colones(datos.total_premios) }}</div>
          </div>
          <div class="card stat">
            <div class="etq">Apuestas</div>
            <div class="valor">{{ datos.total_apuestas }}</div>
          </div>
          <div class="card stat">
            <div class="etq">Juegos activos</div>
            <div class="valor">{{ datos.loterias_activas }}</div>
          </div>
        </div>

        <div class="section-title">Por juego</div>
        <div class="table-wrap">
          <table class="table">
            <thead>
              <tr>
                <th>Juego</th>
                <th>Tipo</th>
                <th>Apostado</th>
                <th>Premios</th>
                <th>Apuestas</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="f in datos.por_loteria" :key="f.loteria_id">
                <td class="fuerte">{{ f.nombre }}</td>
                <td>
                  <span class="badge" :class="f.tipo === 'quiniela' ? 'badge-violet' : 'badge-ok'">
                    {{ f.tipo === 'quiniela' ? 'Quiniela' : 'Lotería' }}
                  </span>
                </td>
                <td class="mono">{{ colones(f.apostado) }}</td>
                <td class="mono texto-gold">{{ colones(f.premios) }}</td>
                <td class="mono">{{ f.apuestas }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <p v-if="datos.usuarios" class="hint" style="margin-top: 14px">
          Usuarios registrados: {{ datos.usuarios }}
        </p>
      </template>
    </div>

    <TablaApuestas v-else-if="tab === 'apuestas'" />
    <TablaRifas v-else-if="tab === 'rifas'" />
    <TablaJugadas v-else-if="tab === 'jugadas'" />
    <TablaMovimientos v-else-if="tab === 'movimientos'" />
    <TablaResultados v-else-if="tab === 'resultados'" />
  </div>
</template>

<style scoped>
.chips-filtro {
  display: flex; gap: 8px; flex-wrap: wrap; margin: 4px 0 20px;
}
.chip {
  padding: 8px 14px; border-radius: 999px; border: 1px solid var(--border-soft);
  background: var(--surface); color: var(--muted); cursor: pointer; font-size: 13px;
}
.chip:hover { color: var(--text); border-color: var(--border); }
.chip.activo { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
</style>
