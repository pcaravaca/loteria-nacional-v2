<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api/client'
import type { Resumen } from '@/types'
import { colones } from '@/utils/formato'

const datos = ref<Resumen | null>(null)
const cargando = ref(true)
const error = ref('')

onMounted(async () => {
  try {
    const { data } = await api.get<Resumen>('/reportes/resumen')
    datos.value = data
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
      <div class="eyebrow">Estadísticas</div>
      <h1>Reportes</h1>
      <p>Comportamiento de las apuestas por juego.</p>
    </div>

    <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando reportes…</div>
    <p v-else-if="error" class="error-text">{{ error }}</p>

    <template v-else-if="datos">
      <div class="grid grid-4">
        <div class="card stat">
          <div class="etq">Total apostado</div>
          <div class="valor">{{ colones(datos.total_apostado) }}</div>
        </div>
        <div class="card stat">
          <div class="etq">Premios pagados</div>
          <div class="valor texto-gold">{{ colones(datos.total_premios) }}</div>
        </div>
        <div class="card stat">
          <div class="etq">Apuestas</div>
          <div class="valor">{{ datos.total_apuestas }}</div>
        </div>
        <div class="card stat">
          <div class="etq">Juegos activos</div>
          <div class="valor">{{ datos.loterias_activas }}</div>
        </div>
      </div>

      <div class="section-title">Por juego</div>
      <div class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>Juego</th>
              <th>Tipo</th>
              <th>Apostado</th>
              <th>Premios</th>
              <th>Apuestas</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="f in datos.por_loteria" :key="f.loteria_id">
              <td class="fuerte">{{ f.nombre }}</td>
              <td>
                <span class="badge" :class="f.tipo === 'quiniela' ? 'badge-violet' : 'badge-ok'">
                  {{ f.tipo === 'quiniela' ? 'Quiniela' : 'Lotería' }}
                </span>
              </td>
              <td class="mono">{{ colones(f.apostado) }}</td>
              <td class="mono texto-gold">{{ colones(f.premios) }}</td>
              <td class="mono">{{ f.apuestas }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <p v-if="datos.usuarios" class="hint" style="margin-top: 14px">
        Usuarios registrados: {{ datos.usuarios }}
      </p>
    </template>
  </div>
</template>
