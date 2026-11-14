<script setup lang="ts">
import { onMounted, ref } from 'vue'
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

onMounted(async () => {
  try {
    const [d, s] = await Promise.all([
      api.get<Dashboard>('/dashboard'),
      api.get<Sorteo[]>('/sorteos?estado=abierto'),
    ])
    datos.value = d.data
    proximos.value = s.data
  } catch {
    /* el shell ya redirige si la sesión expira */
  } finally {
    cargando.value = false
  }
})
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
              <th>Fecha</th>
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
              <td>{{ fechaHora(a.created_at) }}</td>
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

    <ApuestaModal
      v-if="sorteoParaJugar"
      :sorteo="sorteoParaJugar"
      @cerrar="sorteoParaJugar = null"
      @ok="sorteoParaJugar = null"
    />
  </div>
</template>
