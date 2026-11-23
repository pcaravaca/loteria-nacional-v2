<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api/client'
import { colones, fechaHora } from '@/utils/formato'

interface RifaMia {
  rifa_id: number
  titulo: string
  premio: string
  estado: string
  numero: number
  monto: number
  fecha_sorteo: string
  ganador: boolean
}

const items = ref<RifaMia[]>([])
const cargando = ref(true)
const error = ref('')

const badges: Record<string, string> = {
  abierta: 'badge-ok',
  sorteada: 'badge-violet',
  cerrada: 'badge-muted',
  cancelada: 'badge-bad',
}

async function cargar() {
  cargando.value = true
  error.value = ''
  try {
    const { data } = await api.get<RifaMia[]>('/rifas/mias')
    items.value = data
  } catch (e: any) {
    error.value = e.message
  } finally {
    cargando.value = false
  }
}

onMounted(cargar)
</script>

<template>
  <div>
    <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando tus rifas…</div>
    <p v-else-if="error" class="error-text">{{ error }}</p>

    <div v-else-if="items.length" class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Rifa</th>
            <th>Tu número</th>
            <th>Premio</th>
            <th>Monto</th>
            <th>Sorteo</th>
            <th>Estado</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in items" :key="r.rifa_id + '-' + r.numero">
            <td class="fuerte">{{ r.titulo }}</td>
            <td>
              <span class="num num-ganador">{{ String(r.numero).padStart(2, '0') }}</span>
            </td>
            <td>{{ r.premio }}</td>
            <td class="mono">{{ colones(r.monto) }}</td>
            <td>
              <div>{{ fechaHora(r.fecha_sorteo) }}</div>
            </td>
            <td>
              <span class="badge" :class="badges[r.estado] ?? 'badge-muted'">
                {{ r.ganador ? 'Ganaste' : r.estado }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
      <p class="hint" style="margin-top: 10px">
        Tenés {{ items.length }} número{{ items.length === 1 ? '' : 's' }} en rifas.
      </p>
    </div>

    <p v-else class="hint">Todavía no compraste números en ninguna rifa.</p>
  </div>
</template>

<style scoped>
.hint { color: var(--faint); }
</style>
