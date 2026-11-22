<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { api } from '@/api/client'
import type { Movimiento } from '@/types'
import { colones, fechaHora } from '@/utils/formato'
import Paginador from './Paginador.vue'

interface Pagina<T> {
  items: T[]
  total: number
  pagina: number
  por_pagina: number
  paginas: number
}

const items = ref<Movimiento[]>([])
const pagina = ref(1)
const paginas = ref(1)
const total = ref(0)
const cargando = ref(true)
const error = ref('')

const etiquetas: Record<string, { texto: string; clase: string }> = {
  recarga: { texto: 'Recarga', clase: 'badge-ok' },
  apuesta: { texto: 'Apuesta', clase: 'badge-gold' },
  casino_apuesta: { texto: 'Apuesta casino', clase: 'badge-gold' },
  casino_premio: { texto: 'Premio casino', clase: 'badge-info' },
  premio: { texto: 'Premio', clase: 'badge-info' },
  bono: { texto: 'Bono', clase: 'badge-violet' },
  retiro: { texto: 'Retiro', clase: 'badge-bad' },
  ajuste: { texto: 'Ajuste', clase: '' },
}

async function cargar() {
  cargando.value = true
  error.value = ''
  try {
    const { data } = await api.get<Pagina<Movimiento>>(
      `/movimientos?pagina=${pagina.value}&por_pagina=20`,
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
          <tr><th>Tipo</th><th>Descripción</th><th>Fecha</th><th>Monto</th></tr>
        </thead>
        <tbody>
          <tr v-for="m in items" :key="m.id">
            <td>
              <span class="badge" :class="(etiquetas[m.tipo] ?? {}).clase">
                {{ (etiquetas[m.tipo] ?? {}).texto ?? m.tipo }}
              </span>
            </td>
            <td>{{ m.descripcion || '—' }}</td>
            <td>{{ fechaHora(m.created_at) }}</td>
            <td class="mono" :style="{ color: m.monto >= 0 ? 'var(--prim-2)' : '#ff9b9b' }">
              {{ m.monto >= 0 ? '+' : '' }}{{ colones(m.monto) }}
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

    <p v-else class="hint">Todavía no tenés movimientos de saldo.</p>
  </div>
</template>

<style scoped>
.hint { color: var(--faint); }
</style>
