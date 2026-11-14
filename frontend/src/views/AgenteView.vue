<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { api } from '@/api/client'
import type { Sorteo } from '@/types'
import { colones, estadoSorteo, fechaHora } from '@/utils/formato'
import Modal from '@/components/Modal.vue'
import EmptyState from '@/components/EmptyState.vue'

const cargando = ref(true)
const error = ref('')
const sorteos = ref<Sorteo[]>([])

async function cargar() {
  cargando.value = true
  error.value = ''
  try {
    const { data } = await api.get<Sorteo[]>('/sorteos')
    sorteos.value = data
  } catch (e: any) {
    error.value = e.message
  } finally {
    cargando.value = false
  }
}

onMounted(cargar)

const abiertos = computed(() => sorteos.value.filter((s) => s.estado === 'abierto'))
const cerrados = computed(() => sorteos.value.filter((s) => s.estado !== 'abierto'))

const cierre = reactive({ abierto: false, sorteoId: 0, titulo: '', numeros: '' })
const guardando = ref(false)

function abrirCierre(s: Sorteo) {
  cierre.abierto = true
  cierre.sorteoId = s.id
  cierre.titulo = s.titulo
  cierre.numeros = ''
}

function parsearNumeros(texto: string): number[] {
  return texto
    .split(/[\s,]+/)
    .filter((t) => t !== '')
    .map((t) => Number(t))
    .filter((n) => Number.isInteger(n) && n >= 0)
}

async function cerrarSorteo() {
  const numeros = parsearNumeros(cierre.numeros)
  if (!numeros.length) {
    error.value = 'Ingresa al menos un número ganador'
    return
  }
  guardando.value = true
  try {
    const { data } = await api.post<Sorteo>(`/admin/sorteos/${cierre.sorteoId}/cerrar`, {
      numeros_ganadores: numeros,
    })
    sorteos.value = sorteos.value.map((s) => (s.id === data.id ? data : s))
    cierre.abierto = false
  } catch (e: any) {
    error.value = e.message
  } finally {
    guardando.value = false
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <div class="eyebrow">Agente</div>
      <h1>Panel de agente</h1>
      <p>Cierra sorteos cargando los números ganadores.</p>
    </div>

    <div class="grid grid-3">
      <div class="card stat">
        <div class="etq">Sorteos abiertos</div>
        <div class="valor">{{ abiertos.length }}</div>
      </div>
      <div class="card stat">
        <div class="etq">Ya cerrados</div>
        <div class="valor">{{ cerrados.length }}</div>
      </div>
      <div class="card stat">
        <div class="etq">Total</div>
        <div class="valor">{{ sorteos.length }}</div>
      </div>
    </div>

    <p v-if="error" class="error-text" style="margin-top: 14px">{{ error }}</p>
    <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando sorteos…</div>

    <template v-else>
      <div class="section-title">Abiertos</div>
      <div v-if="abiertos.length" class="grid" style="gap: 12px">
        <div v-for="s in abiertos" :key="s.id" class="card sorteo-row">
          <div>
            <div class="eyebrow" style="color: var(--prim-2); font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: .1em">
              {{ s.loteria?.nombre }}
            </div>
            <h3 style="font-size: 16px; margin-top: 5px">{{ s.titulo }}</h3>
            <p class="hint" style="margin-top: 3px">{{ fechaHora(s.fecha) }}</p>
          </div>
          <button class="btn btn-primary btn-sm" @click="abrirCierre(s)">Cerrar sorteo</button>
        </div>
      </div>
      <EmptyState v-else icono="✅" titulo="No hay sorteos abiertos" texto="Cuando se creen sorteos nuevos aparecerán aquí." />

      <div v-if="cerrados.length" class="section-title" style="margin-top: 24px">Cerrados</div>
      <div v-if="cerrados.length" class="grid" style="gap: 12px">
        <div v-for="s in cerrados" :key="s.id" class="card sorteo-row">
          <div>
            <div class="eyebrow" style="color: var(--prim-2); font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: .1em">
              {{ s.loteria?.nombre }}
            </div>
            <h3 style="font-size: 16px; margin-top: 5px">{{ s.titulo }}</h3>
            <p class="hint" style="margin-top: 3px">{{ fechaHora(s.fecha) }}</p>
          </div>
          <div style="display: flex; align-items: center; gap: 12px">
            <span class="badge" :class="(estadoSorteo[s.estado] ?? {}).clase">{{ (estadoSorteo[s.estado] ?? {}).texto }}</span>
            <div class="numeros">
              <span v-for="n in s.numeros_ganadores" :key="n" class="num num-ganador">{{ String(n).padStart(2, '0') }}</span>
            </div>
          </div>
        </div>
      </div>
    </template>

    <Modal v-if="cierre.abierto" titulo="Cerrar sorteo" :sub="cierre.titulo" @cerrar="cierre.abierto = false">
      <label class="campo">
        <span>Números ganadores (separados por coma o espacio)</span>
        <input v-model="cierre.numeros" class="input" placeholder="07, 19" autofocus />
      </label>
      <p class="hint" style="margin-top: 8px">Se evalúan las apuestas pendientes y se pagan los premios al instante.</p>
      <template #acciones>
        <button class="btn btn-ghost" @click="cierre.abierto = false">Cancelar</button>
        <button class="btn btn-primary" :disabled="guardando" @click="cerrarSorteo">{{ guardando ? 'Cerrando…' : 'Cerrar y pagar' }}</button>
      </template>
    </Modal>
  </div>
</template>
