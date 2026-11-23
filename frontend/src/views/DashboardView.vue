<script setup lang="ts">
import { onMounted, ref } from 'vue'
// El panel es la pantalla de inicio de la app: saldo, accesos y próximos sorteos.
import { api } from '@/api/client'
import type { Dashboard, Sorteo } from '@/types'
import { colones, estadoApuesta, fechaHora } from '@/utils/formato'
import StatCard from '@/components/StatCard.vue'
import SorteoCard from '@/components/SorteoCard.vue'
import ApuestaModal from '@/components/ApuestaModal.vue'
import EmptyState from '@/components/EmptyState.vue'

const datos = ref<Dashboard | null>(null)
const proximos = ref<Sorteo[]>([])
const cargando = ref(true)
const sorteoParaJugar = ref<Sorteo | null>(null)

async function cargarDatos() {
  try {
    const [d, s] = await Promise.all([
      api.get<Dashboard>('/dashboard'),
      api.get<Sorteo[]>('/sorteos?estado=abierto'),
    ])
    datos.value = d.data
    proximos.value = s.data
  } catch {
    /* el shell ya redirige si la sesión expira */
  }
}

onMounted(async () => {
  await cargarDatos()
  cargando.value = false
})

// Tras comprar un tiquete el panel se refresca solo (saldo y ultimas apuestas).
async function onComprada() {
  sorteoParaJugar.value = null
  await cargarDatos()
}
</script>

<template>
  <div>
    <div class="page-head">
      <div class="eyebrow">Resumen</div>
      <h1>Tu panel</h1>
      <p>Saldo, premios pendientes y los próximos sorteos disponibles.</p>
    </div>

    <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando tu panel…</div>

    <template v-else-if="datos">
      <div class="grid grid-4">
        <StatCard
          etiqueta="Disponible"
          :valor="colones(datos.saldo)"
          nota="Saldo para jugar"
          color="rgba(16,185,129,.16)"
          icono="💰"
        />
        <StatCard
          etiqueta="Pendiente"
          :valor="colones(datos.pendiente)"
          :nota="`${datos.apuestas_activas} apuesta(s) en juego`"
          color="rgba(245,185,66,.16)"
          icono="🎟"
        />
        <StatCard
          etiqueta="Premios"
          :valor="colones(datos.premios)"
          nota="Ganados historicamente"
          color="rgba(245,185,66,.16)"
          icono="🏆"
        />
        <StatCard
          etiqueta="Bono"
          :valor="colones(datos.bono)"
          nota="Promociones"
          color="rgba(56,189,248,.16)"
          icono="🎁"
        />
      </div>

      <div class="section-title">Accesos</div>
      <div class="accesos">
        <router-link class="acceso" to="/loterias"><b>🎱</b><span>Loterías</span></router-link>
        <router-link class="acceso" to="/quinielas"><b>⚽</b><span>Quinielas</span></router-link>
        <router-link class="acceso" to="/rifas"><b>🎁</b><span>Rifas</span></router-link>
        <router-link class="acceso" to="/casino"><b>🎰</b><span>Casino</span></router-link>
        <router-link class="acceso" to="/reportes?tab=apuestas"><b>📊</b><span>Reportes</span></router-link>
      </div>

      <div class="section-title">Próximos sorteos</div>
      <div v-if="proximos.length" class="grid grid-3">
        <SorteoCard v-for="s in proximos" :key="s.id" :sorteo="s" @jugar="sorteoParaJugar = s" />
      </div>
      <div v-else class="card empty">
        <div class="ico">⏳</div>
        <h3>Sin sorteos abiertos</h3>
        <p>En cuanto se publiquen nuevos sorteos aparecerán aquí.</p>
      </div>

      <div class="section-title">Tus últimas apuestas</div>
      <div v-if="datos.ultimas_apuestas.length" class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>Sorteo</th>
              <th>Números</th>
              <th>Monto</th>
              <th>Estado</th>
              <th>Fecha del evento</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in datos.ultimas_apuestas" :key="a.id">
              <td class="fuerte">{{ a.sorteo?.titulo ?? 'Sorteo' }}</td>
              <td>
                <span class="numeros">
                  <span v-for="n in a.numeros" :key="n" class="num">{{ String(n).padStart(2, '0') }}</span>
                </span>
              </td>
              <td class="mono">{{ colones(a.monto) }}</td>
              <td>
                <span class="badge" :class="(estadoApuesta[a.estado] ?? {}).clase">
                  {{ (estadoApuesta[a.estado] ?? {}).texto ?? a.estado }}
                </span>
              </td>
              <td>
                <div>{{ fechaHora(a.sorteo?.fecha ?? a.created_at) }}</div>
                <div class="hint">la apostaste {{ fechaHora(a.created_at) }}</div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else>
        <EmptyState
          icono="🎟"
          titulo="Aún no tienes apuestas"
          texto="Elige un sorteo abierto y marca tus números."
        >
          <router-link to="/loterias" class="btn btn-primary btn-sm" style="margin-top: 14px">Ver loterías</router-link>
        </EmptyState>
      </div>
    </template>

<style scoped>
.accesos { display: grid; grid-template-columns: repeat(5, 1fr); gap: 10px; }
.acceso {
  display: flex; flex-direction: column; align-items: center; gap: 6px;
  padding: 12px 6px; border-radius: 12px; text-align: center;
  background: linear-gradient(180deg, var(--surface), var(--bg-2));
  border: 1px solid var(--border-soft); color: var(--muted);
}
.acceso b { font-size: 20px; }
.acceso span { font-size: 11.5px; }
.acceso:hover { color: var(--text); border-color: var(--border); }
.hint { font-size: 10.5px; color: var(--faint); margin-top: 2px; }
@media (max-width: 560px) { .accesos { grid-template-columns: repeat(3, 1fr); } }
</style>

    <ApuestaModal
      v-if="sorteoParaJugar"
      :sorteo="sorteoParaJugar"
      @cerrar="sorteoParaJugar = null"
      @ok="onComprada"
    />
  </div>
</template>
