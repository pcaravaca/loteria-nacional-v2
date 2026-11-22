<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { api } from '@/api/client'
import type { CasinoRound } from '@/types'
import { colones } from '@/utils/formato'
import Modal from '@/components/Modal.vue'
import Paginador from './Paginador.vue'

interface Pagina<T> {
  items: T[]
  total: number
  pagina: number
  por_pagina: number
  paginas: number
}

const items = ref<CasinoRound[]>([])
const pagina = ref(1)
const paginas = ref(1)
const total = ref(0)
const cargando = ref(true)
const error = ref('')
const verificando = ref<any | null>(null)

function fmtMult(valor: number | undefined): string {
  if (valor === undefined || valor === null || valor <= 0) return '—'
  const r = Math.round(valor * 100) / 100
  return Number.isInteger(r) ? 'x' + r : 'x' + r.toFixed(2).replace(/0+$/, '').replace(/\.$/, '')
}

function detalleMultiplicador(r: CasinoRound): string {
  const mult = fmtMult(r.multiplicador)
  if (r.source === 'gemas' && r.puntos) return `${mult} · ${r.puntos} pts`
  if (r.source === 'bingo' && r.lineas) return `${mult} · ${r.lineas} línea${r.lineas === 1 ? '' : 's'}`
  return mult
}

async function cargar() {
  cargando.value = true
  error.value = ''
  try {
    const { data } = await api.get<Pagina<CasinoRound>>(
      `/casino/rounds?pagina=${pagina.value}&por_pagina=20`,
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

async function verificar(ronda: CasinoRound) {
  try {
    const source = ronda.source || 'casino'
    const { data } = await api.get(`/casino/rounds/${ronda.id}/verify?source=${source}`)
    verificando.value = data
  } catch (e: any) {
    error.value = e.message
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
          <tr><th>Juego</th><th>Monto</th><th>Multipl</th><th>Premio</th><th>Fecha</th><th></th></tr>
        </thead>
        <tbody>
          <tr v-for="r in items" :key="(r.source || 'casino') + '-' + r.id">
            <td class="fuerte">{{ r.nombre || r.game }}</td>
            <td class="mono">{{ colones(r.monto) }}</td>
            <td class="mono">{{ detalleMultiplicador(r) }}</td>
            <td class="mono" :style="{ color: r.premio > 0 ? 'var(--gold)' : 'var(--muted)' }">
              {{ r.premio ? colones(r.premio) : '—' }}
            </td>
            <td>{{ new Date(r.created_at).toLocaleString('es-CR', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' }) }}</td>
            <td><button class="btn btn-ghost btn-sm" @click="verificar(r)">Verificar</button></td>
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

    <p v-else class="hint">Todavía no jugaste en el casino.</p>

    <Modal v-if="verificando" titulo="Verificacion provably fair" @cerrar="verificando = null">
      <div class="verificacion">
        <p class="linea"><span>Ronda</span><b>#{{ verificando.ronda_id }}</b></p>
        <p class="linea"><span>Juego</span><b>{{ verificando.nombre || verificando.juego }}</b></p>
        <p v-if="verificando.puntos" class="linea"><span>Puntos</span><b>{{ verificando.puntos }}</b></p>
        <p v-if="verificando.lineas" class="linea"><span>Líneas</span><b>{{ verificando.lineas }}</b></p>
        <p class="linea"><span>Nonce</span><b>{{ verificando.nonce }}</b></p>
        <p class="linea"><span>Client seed</span><b class="mono">{{ verificando.client_seed }}</b></p>
        <p class="linea"><span>Server seed</span><b class="mono">{{ verificando.server_seed }}</b></p>
        <p class="linea"><span>Hash</span><b class="mono">{{ verificando.server_seed_hash }}</b></p>
        <p class="linea">
          <span>Verificado</span>
          <b :style="{ color: verificando.verificado ? 'var(--prim-2)' : '#ff9b9b' }">
            {{ verificando.verificado ? 'SI' : 'NO' }}
          </b>
        </p>
      </div>
      <button class="btn btn-primary" @click="verificando = null">Cerrar</button>
    </Modal>
  </div>
</template>

<style scoped>
.hint { color: var(--faint); }
.verificacion { display: flex; flex-direction: column; gap: 8px; margin-bottom: 14px; }
.linea { display: flex; justify-content: space-between; gap: 16px; font-size: 13px; color: var(--muted); }
.linea b { color: var(--text); text-align: right; word-break: break-all; }
</style>
