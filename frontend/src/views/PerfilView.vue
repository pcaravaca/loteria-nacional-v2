<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import type { Movimiento } from '@/types'
import { colones, fechaHora } from '@/utils/formato'

const auth = useAuthStore()
const movimientos = ref<Movimiento[]>([])
const cargando = ref(true)

onMounted(async () => {
  try {
    const { data } = await api.get<Movimiento[]>('/movimientos')
    movimientos.value = data
  } finally {
    cargando.value = false
  }
})

const etiquetas: Record<string, { texto: string; clase: string }> = {
  recarga: { texto: 'Recarga', clase: 'badge-ok' },
  apuesta: { texto: 'Apuesta', clase: 'badge-gold' },
  premio: { texto: 'Premio', clase: 'badge-info' },
  bono: { texto: 'Bono', clase: 'badge-violet' },
  retiro: { texto: 'Retiro', clase: 'badge-bad' },
  ajuste: { texto: 'Ajuste', clase: '' },
}
</script>

<template>
  <div>
    <div class="page-head">
      <div class="eyebrow">Tu cuenta</div>
      <h1>Mi perfil</h1>
    </div>

    <div class="grid grid-2" style="align-items: start">
      <div class="card">
        <div style="display: flex; align-items: center; gap: 14px">
          <div class="avatar" style="width: 52px; height: 52px; font-size: 20px">
            {{ (auth.user?.nombre ?? 'U').slice(0, 1).toUpperCase() }}
          </div>
          <div>
            <h3 style="font-size: 19px">{{ auth.user?.nombre }}</h3>
            <p class="texto-muted" style="margin: 2px 0 0; font-size: 13px">{{ auth.user?.email }}</p>
            <span class="badge badge-ok" style="margin-top: 6px">{{ auth.user?.rol }}</span>
          </div>
        </div>

        <div style="display: flex; gap: 20px; margin-top: 20px; padding-top: 18px; border-top: 1px solid var(--border-soft)">
          <div>
            <div class="hint">Saldo disponible</div>
            <div class="mono texto-prim" style="font-size: 22px">{{ colones(auth.user?.saldo ?? 0) }}</div>
          </div>
          <div>
            <div class="hint">Bono</div>
            <div class="mono texto-gold" style="font-size: 22px">{{ colones(auth.user?.bono ?? 0) }}</div>
          </div>
          <div>
            <div class="hint">Teléfono</div>
            <div class="mono" style="font-size: 16px">{{ auth.user?.telefono || '—' }}</div>
          </div>
        </div>

        <p class="hint" style="margin-top: 16px">
          ¿Necesitas saldo? Contacta a un agente para recargar tu cuenta.
        </p>
      </div>

      <div class="card">
        <h3 style="font-size: 16px; margin-bottom: 12px">Movimientos de saldo</h3>
        <div v-if="cargando" class="cargando"><span class="spinner"></span></div>
        <div v-else-if="movimientos.length" class="grid" style="gap: 8px">
          <div
            v-for="m in movimientos"
            :key="m.id"
            style="display: flex; justify-content: space-between; gap: 10px; padding: 10px 0; border-bottom: 1px solid var(--border-soft)"
          >
            <div>
              <span class="badge" :class="(etiquetas[m.tipo] ?? {}).clase">
                {{ (etiquetas[m.tipo] ?? {}).texto ?? m.tipo }}
              </span>
              <div class="hint" style="margin-top: 4px">{{ m.descripcion }}</div>
              <div class="hint" style="font-size: 11px">{{ fechaHora(m.created_at) }}</div>
            </div>
            <div
              class="mono"
              :style="{ color: m.monto >= 0 ? 'var(--prim-2)' : '#ff9b9b' }"
            >
              {{ m.monto >= 0 ? '+' : '' }}{{ colones(m.monto) }}
            </div>
          </div>
        </div>
        <p v-else class="hint">Sin movimientos todavía.</p>
      </div>
    </div>
  </div>
</template>
