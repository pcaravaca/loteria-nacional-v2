<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api/client'
import type { Sorteo } from '@/types'
import { colones, fechaHora } from '@/utils/formato'
import EmptyState from '@/components/EmptyState.vue'

interface Fila {
  sorteo: Sorteo
  aciertos: number
  premio: number
  estado_apuesta: string
}

const filas = ref<Fila[]>([])
const cargando = ref(true)

onMounted(async () => {
  try {
    const { data } = await api.get<{ resultados: Fila[] }>('/resultados')
    // La API ya ordena por fecha descendente; se reordena acá como rede de
    // seguridad para que la vista nunca muestre los sorteos viejos primero.
    filas.value = [...data.resultados].sort((a, b) =>
      b.sorteo.fecha.localeCompare(a.sorteo.fecha),
    )
  } finally {
    cargando.value = false
  }
})
</script>

<template>
  <div>
    <div class="page-head">
      <div class="eyebrow">Sorteos cerrados</div>
      <h1>Resultados</h1>
      <p>Números ganadores de cada sorteo y cómo te fue en tus apuestas.</p>
    </div>

    <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando resultados…</div>

    <div v-else-if="filas.length" class="grid" style="gap: 14px">
      <div v-for="f in filas" :key="f.sorteo.id" class="card">
        <div style="display: flex; justify-content: space-between; gap: 12px; flex-wrap: wrap; align-items: flex-start">
          <div>
            <div class="eyebrow" style="color: var(--prim-2); font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: .1em">
              {{ f.sorteo.loteria?.nombre }}
            </div>
            <h3 style="font-size: 17px; margin-top: 5px">{{ f.sorteo.titulo }}</h3>
            <p class="hint" style="margin-top: 4px">{{ fechaHora(f.sorteo.fecha) }}</p>
          </div>
          <div style="text-align: right">
            <div class="hint">{{ f.estado_apuesta ? 'Tu resultado' : 'Sin tu apuesta' }}</div>
            <div v-if="f.estado_apuesta" class="mono" :class="f.aciertos > 0 ? 'texto-prim' : 'texto-muted'">
              {{ f.aciertos }} acierto(s)
            </div>
            <div v-else class="mono texto-muted">—</div>
            <div v-if="f.premio" class="mono texto-gold">{{ colones(f.premio) }}</div>
          </div>
        </div>

        <div style="margin-top: 14px; padding-top: 14px; border-top: 1px solid var(--border-soft)">
          <div class="hint" style="margin-bottom: 7px">Números ganadores</div>
          <div class="numeros">
            <span v-for="n in f.sorteo.numeros_ganadores" :key="n" class="num num-ganador">
              {{ String(n).padStart(2, '0') }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <EmptyState
      v-else
      icono="🏆"
      titulo="Aún no hay resultados"
      texto="Cuando se cierren sorteos, los números ganadores aparecerán aquí."
    />
  </div>
</template>
