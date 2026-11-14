<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import type { Sorteo } from '@/types'
import { colones } from '@/utils/formato'
import Modal from '@/components/Modal.vue'
import NumPad from '@/components/NumPad.vue'

const props = defineProps<{ sorteo: Sorteo }>()
const emit = defineEmits<{ cerrar: []; ok: [] }>()

const auth = useAuthStore()
const numeros = ref<number[]>([])
const cargando = ref(false)
const error = ref('')

const maximo = computed(() => Math.pow(10, props.sorteo.loteria?.longitud_num ?? 2) - 1)
const monto = computed(() => (props.sorteo.loteria?.costo_numero ?? 0) * numeros.value.length)
const saldoDisponible = computed(() => auth.user?.saldo ?? 0)
const puedeJugar = computed(() => numeros.value.length > 0 && monto.value <= saldoDisponible.value)

async function confirmar() {
  error.value = ''
  cargando.value = true
  try {
    await api.post('/apuestas', {
      sorteo_id: props.sorteo.id,
      numeros: numeros.value,
    })
    await auth.cargar()
    emit('ok')
  } catch (e: any) {
    error.value = e.message
  } finally {
    cargando.value = false
  }
}
</script>

<template>
  <Modal
    :titulo="sorteo.titulo"
    :sub="`${sorteo.loteria?.nombre} · ${colones(sorteo.loteria?.costo_numero ?? 0)} por número`"
    @cerrar="emit('cerrar')"
  >
    <NumPad
      :etiqueta="'Elige tus números'"
      :max-seleccion="sorteo.loteria?.max_numeros ?? 1"
      :maximo="maximo"
      v-model="numeros"
    />

    <div class="resumen-numeros">
      <span v-if="numeros.length === 0" class="vacio"> Tus números aparecerán aquí</span>
      <span v-for="n in numeros" :key="n" class="num">{{ String(n).padStart(2, '0') }}</span>
      <div style="flex: 1"></div>
      <div style="text-align: right">
        <div class="hint">Total a pagar</div>
        <div class="mono texto-prim" style="font-size: 19px">{{ colones(monto) }}</div>
      </div>
    </div>

    <p class="hint" style="margin-top: 10px">
      Saldo disponible: <span class="mono">{{ colones(saldoDisponible) }}</span>
    </p>

    <p v-if="error" class="error-text" style="margin-top: 12px">{{ error }}</p>

    <template #acciones>
      <button class="btn btn-ghost" @click="emit('cerrar')">Cancelar</button>
      <button class="btn btn-primary" :disabled="!puedeJugar || cargando" @click="confirmar">
        {{ cargando ? 'Registrando…' : `Jugar ${colones(monto)}` }}
      </button>
    </template>
  </Modal>
</template>
