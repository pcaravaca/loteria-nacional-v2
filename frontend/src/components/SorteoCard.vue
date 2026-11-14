<script setup lang="ts">
import { computed } from 'vue'
import type { Sorteo } from '@/types'
import { colones, estadoSorteo } from '@/utils/formato'

const props = defineProps<{
  sorteo: Sorteo
  selectable?: boolean
}>()

const emit = defineEmits<{ jugar: [Sorteo] }>()

const estado = computed(() => estadoSorteo[props.sorteo.estado] ?? { texto: props.sorteo.estado, clase: '' })
</script>

<template>
  <div class="card card-hover">
    <div style="display: flex; justify-content: space-between; gap: 10px; align-items: flex-start">
      <div>
        <div class="eyebrow" style="color: var(--prim-2); font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: .1em">
          {{ sorteo.loteria?.nombre ?? 'Lotería' }}
        </div>
        <h3 style="font-size: 17px; margin-top: 5px">{{ sorteo.titulo }}</h3>
      </div>
      <span class="badge" :class="estado.clase">{{ estado.texto }}</span>
    </div>

    <p class="texto-muted" style="font-size: 13px; margin: 10px 0 0">
      {{ new Intl.DateTimeFormat('es-CR', { day: '2-digit', month: 'long', hour: '2-digit', minute: '2-digit' }).format(new Date(sorteo.fecha)) }}
    </p>

    <div v-if="sorteo.estado === 'abierto'" style="margin-top: 16px; display: flex; justify-content: space-between; align-items: center; gap: 10px">
      <span class="hint">
        {{ colones(sorteo.loteria?.costo_numero ?? 0) }} por número
      </span>
      <button class="btn btn-primary btn-sm" @click="emit('jugar', sorteo)">Jugar</button>
    </div>

    <div v-else style="margin-top: 16px">
      <div class="hint" style="margin-bottom: 7px">Números ganadores</div>
      <div class="numeros">
        <span v-for="n in sorteo.numeros_ganadores" :key="n" class="num num-ganador">{{ String(n).padStart(2, '0') }}</span>
      </div>
    </div>
  </div>
</template>
