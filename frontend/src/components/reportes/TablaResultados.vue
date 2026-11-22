<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api } from '@/api/client'
import type { Sorteo } from '@/types'
import { colones, estadoApuesta, fechaHora } from '@/utils/formato'
import Paginador from './Paginador.vue'

interface Fila {
  sorteo: Sorteo
  aposto: boolean
  numeros: number[]
  monto: number
  aciertos: number
  premio: number
  estado_apuesta: string
}

interface Pagina<T> {
  items: T[]
  total: number
  pagina: number
  por_pagina: number
  paginas: number
}

const filas = ref<Fila[]>([])
const pagina = ref(1)
const paginas = ref(1)
const total = ref(0)
const cargando = ref(true)
const error = ref('')

function ganadores(f: Fila): Set<number> {
  return new Set(f.sorteo?.numeros_ganadores ?? [])
}

function esAcierto(f: Fila, n: number): boolean {
  if (!f.aposto) return false
  if (f.estado_apuesta === 'pendiente') return false
  return ganadores(f).has(n)
}

const resumen = computed(() => {
  const jugados = filas.value.filter((f) => f.aposto)
  const ganados = jugados.filter((f) => f.premio > 0)
  return {
    jugados: jugados.length,
    ganados: ganados.length,
    premios: jugados.reduce((s, f) => s + (f.premio || 0), 0),
  }
})

async function cargar() {
  cargando.value = true
  error.value = ''
  try {
    const { data } = await api.get<Pagina<Fila>>(
      `/resultados?pagina=${pagina.value}&por_pagina=10`,
    )
    // La API ya ordena por fecha descendente; se reordena acá como red de
    // seguridad para que nunca se vean los sorteos viejos primero.
    filas.value = [...data.items].sort((a, b) => b.sorteo.fecha.localeCompare(a.sorteo.fecha))
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
    <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando resultados…</div>
    <p v-else-if="error" class="error-text">{{ error }}</p>

    <template v-else>
      <div class="grid grid-3 resumen">
        <div class="card stat">
          <div class="etq">Sorteos con tu jugada</div>
          <div class="valor">{{ resumen.jugados }}</div>
        </div>
        <div class="card stat">
          <div class="etq">Con premio</div>
          <div class="valor texto-gold">{{ resumen.ganados }}</div>
        </div>
        <div class="card stat">
          <div class="etq">Premios de esta página</div>
          <div class="valor">{{ colones(resumen.premios) }}</div>
        </div>
      </div>

      <div class="grid" style="gap: 14px; margin-top: 16px">
        <div v-for="f in filas" :key="f.sorteo.id" class="card">
          <div class="fila-head">
            <div>
              <div class="eyebrow" style="color: var(--prim-2); font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: .1em">
                {{ f.sorteo.loteria?.nombre }}
              </div>
              <h3 style="font-size: 17px; margin-top: 5px">{{ f.sorteo.titulo }}</h3>
              <p class="hint" style="margin-top: 4px">{{ fechaHora(f.sorteo.fecha) }}</p>
            </div>
            <div class="lado-der">
              <span
                v-if="!f.aposto"
                class="badge"
                title="No compraste número en este sorteo"
              >No compraste</span>
              <template v-else>
                <span class="badge" :class="(estadoApuesta[f.estado_apuesta] ?? {}).clase">
                  {{ (estadoApuesta[f.estado_apuesta] ?? {}).texto ?? f.estado_apuesta }}
                </span>
                <div class="mono" :class="f.aciertos > 0 ? 'texto-prim' : 'texto-muted'">
                  {{ f.aciertos }} acierto{{ f.aciertos === 1 ? '' : 's' }}
                </div>
                <div v-if="f.premio" class="mono texto-gold">{{ colones(f.premio) }}</div>
              </template>
            </div>
          </div>

          <div class="bloque">
            <div class="hint" style="margin-bottom: 7px">Números ganadores</div>
            <div class="numeros">
              <span v-for="n in f.sorteo.numeros_ganadores" :key="n" class="num num-ganador">
                {{ String(n).padStart(2, '0') }}
              </span>
            </div>
          </div>

          <div class="bloque">
            <div class="hint" style="margin-bottom: 7px">Tus números</div>
            <template v-if="f.aposto">
              <div class="numeros">
                <span
                  v-for="n in f.numeros"
                  :key="n"
                  class="num"
                  :class="{ 'num-acierto': esAcierto(f, n) }"
                >{{ String(n).padStart(2, '0') }}</span>
              </div>
              <p class="hint" style="margin-top: 8px">
                Monto {{ colones(f.monto) }}
                <template v-if="f.aciertos === 0"> · ninguno acertó</template>
              </p>
            </template>
            <p v-else class="hint">No compraste número en este sorteo.</p>
          </div>
        </div>
      </div>

      <Paginador
        :pagina="pagina"
        :paginas="paginas"
        :total="total"
        @cambiar="pagina = $event"
      />
    </template>
  </div>
</template>

<style scoped>
.resumen { margin-bottom: 0; }
.fila-head { display: flex; justify-content: space-between; gap: 12px; flex-wrap: wrap; align-items: flex-start; }
.lado-der { text-align: right; display: flex; flex-direction: column; gap: 5px; align-items: flex-end; }
.bloque { margin-top: 14px; padding-top: 14px; border-top: 1px solid var(--border-soft); }
.hint { font-size: 12px; color: var(--faint); }
</style>
