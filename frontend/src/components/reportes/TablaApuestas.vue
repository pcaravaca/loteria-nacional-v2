<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { api } from '@/api/client'
import type { Apuesta } from '@/types'
import { colones, estadoApuesta, fechaHora } from '@/utils/formato'
import Paginador from './Paginador.vue'

interface Pagina<T> {
  items: T[]
  total: number
  pagina: number
  por_pagina: number
  paginas: number
}

const items = ref<Apuesta[]>([])
const pagina = ref(1)
const paginas = ref(1)
const total = ref(0)
const cargando = ref(true)
const error = ref('')

function esAcierto(a: Apuesta, n: number): boolean {
  if (a.estado === 'pendiente') return false
  return !!a.sorteo?.numeros_ganadores?.includes(n)
}

async function cargar() {
  cargando.value = true
  error.value = ''
  try {
    const { data } = await api.get<Pagina<Apuesta>>(
      `/apuestas/mias?pagina=${pagina.value}&por_pagina=20`,
    )
    items.value = data.items
    paginas.value = data.paginas
    total.value = data.total
  } catch (e: any) {
    error.value = e.message
  } finally {
    cargando.value = false
  }
}

onMounted(cargar)
watch(pagina, cargar)
</script>

<template>
  <div>
    <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando…</div>
    <p v-else-if="error" class="error-text">{{ error }}</p>

    <div v-else-if="items.length" class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Sorteo</th>
            <th>Números</th>
            <th>Monto</th>
            <th>Aciertos</th>
            <th>Premio</th>
            <th>Estado</th>
            <th>Fecha del evento</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in items" :key="a.id">
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
              <div class="hint">la apostaste {{ fechaHora(a.created_at) }}</div>
            </td>
          </tr>
        </tbody>
      </table>
      <Paginador
        :pagina="pagina"
        :paginas="paginas"
        :total="total"
        @cambiar="pagina = $event"
      />
    </div>

    <p v-else class="hint">Todavía no tenés apuestas de sorteos.</p>
  </div>
</template>

<style scoped>
.hint { font-size: 10.5px; color: var(--faint); margin-top: 2px; }
</style>
