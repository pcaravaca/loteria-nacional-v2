<script setup lang="ts">
import type { Loteria } from '@/types'
import { colones, tipoLabel } from '@/utils/formato'

defineProps<{ loteria: Loteria }>()
</script>

<template>
  <div class="card card-hover">
    <div style="display: flex; justify-content: space-between; align-items: flex-start; gap: 10px">
      <div>
        <span class="badge" :class="loteria.tipo === 'quiniela' ? 'badge-violet' : 'badge-ok'">
          {{ tipoLabel(loteria.tipo) }}
        </span>
        <h3 style="font-size: 19px; margin-top: 10px">{{ loteria.nombre }}</h3>
      </div>
      <span class="mono texto-gold" style="font-size: 18px">{{ colones(loteria.costo_numero) }}</span>
    </div>

    <p class="texto-muted" style="font-size: 13px; margin: 10px 0 0">{{ loteria.descripcion }}</p>

    <div style="display: flex; gap: 18px; margin-top: 14px; padding-top: 14px; border-top: 1px solid var(--border-soft)">
      <div>
        <div class="hint">Premio x acierto</div>
        <div class="mono texto-prim">x{{ loteria.multiplicador }}</div>
      </div>
      <div>
        <div class="hint">Números</div>
        <div class="mono">1 - {{ loteria.max_numeros }}</div>
      </div>
      <div>
        <div class="hint">Dígitos</div>
        <div class="mono">{{ loteria.longitud_num }}</div>
      </div>
    </div>
  </div>
</template>
