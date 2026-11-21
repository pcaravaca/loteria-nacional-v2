<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api/client'
import type { Apuesta } from '@/types'
import { colones, estadoApuesta, fechaHora } from '@/utils/formato'
import EmptyState from '@/components/EmptyState.vue'

const apuestas = ref<Apuesta[]>([])
const cargando = ref(true)
const error = ref('')

function esAcierto(a: Apuesta, n: number): boolean {
  if (a.estado === 'pendiente') return false
  return !!a.sorteo?.numeros_ganadores?.includes(n)
}

onMounted(async () => {
  try {
    const { data } = await api.get<Apuesta[]>('/apuestas/mias')
    apuestas.value = data
  } catch (e: any) {
    error.value = e.message
  } finally {
    cargando.value = false
  }
})
</script>

<template>
  <div>
    <div class="page-head">
      <div class="eyebrow">Historial</div>
      <h1>Mis apuestas</h1>
      <p>Todas tus jugadas, con su estado y premios.</p>
    </div>

    <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando…</div>
    <p v-else-if="error" class="error-text">{{ error }}</p>

    <div v-else-if="apuestas.length" class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Sorteo</th>
            <th>Números</th>
            <th>Monto</th>
            <th>Aciertos</th>
            <th>Premio</th>
            <th>Estado</th>
            <th>Fecha sorteo</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in apuestas" :key="a.id">
            <td class="fuerte">{{ a.sorteo?.titulo ?? 'Sorteo' }}</td>
            <td>
              <span class="numeros">
                <span
                  v-for="n in a.numeros"
                  :key="n"
                  class="num"
                  :class="{ 'num-acierto': esAcierto(a, n) }"
                >{{ String(n).padStart(2, '0') }}</span>
              </span>
            </td>
            <td class="mono">{{ colones(a.monto) }}</td>
            <td class="mono">{{ a.aciertos }}</td>
            <td class="mono texto-gold">{{ a.premio ? colones(a.premio) : '—' }}</td>
            <td>
              <span class="badge" :class="(estadoApuesta[a.estado] ?? {}).clase">
                {{ (estadoApuesta[a.estado] ?? {}).texto ?? a.estado }}
              </span>
            </td>
            <td>
              <div>{{ fechaHora(a.sorteo?.fecha ?? a.created_at) }}</div>
              <div class="hint">apuesta {{ fechaHora(a.created_at) }}</div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <EmptyState
      v-else
      icono="🎟"
      titulo="Sin apuestas todavia"
      texto="Cuando juegues tu primera apuesta aparecerá aquí."
    >
      <router-link to="/loterias" class="btn btn-primary btn-sm" style="margin-top: 14px">Ver sorteos</router-link>
    </EmptyState>
  </div>
</template>

<style scoped>
.hint { font-size: 10.5px; color: var(--faint); margin-top: 2px; }
</style>
