<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import type { Sorteo } from '@/types'
import { colones } from '@/utils/formato'
import { imprimirTiqueteApuesta } from '@/lib/tiquete'
import Modal from '@/components/Modal.vue'
import NumPad from '@/components/NumPad.vue'

const props = defineProps<{ sorteo: Sorteo }>()
const emit = defineEmits<{ cerrar: []; ok: [] }>()

const auth = useAuthStore()
const numeros = ref<number[]>([])
const cargando = ref(false)
const error = ref('')
const idApuesta = ref<number | null>(null)
const imprimiendo = ref(false)

async function imprimir() {
  if (idApuesta.value === null) return
  imprimiendo.value = true
  error.value = ''
  try {
    await imprimirTiqueteApuesta(idApuesta.value)
    emit('ok')
  } catch (e: any) {
    error.value = e.message
  } finally {
    imprimiendo.value = false
  }
}

const maximo = computed(() => Math.pow(10, props.sorteo.loteria?.longitud_num ?? 2) - 1)
const monto = computed(() => (props.sorteo.loteria?.costo_numero ?? 0) * numeros.value.length)
const saldoDisponible = computed(() => auth.user?.saldo ?? 0)
const puedeJugar = computed(() => numeros.value.length > 0 && monto.value <= saldoDisponible.value)

async function confirmar() {
  error.value = ''
  cargando.value = true
  try {
    const res = await api.post('/apuestas', {
      sorteo_id: props.sorteo.id,
      numeros: numeros.value,
    })
    await auth.cargar()
    idApuesta.value = res.data?.id ?? null
    // Con id podemos ofrecer el comprobante; si el backend es viejo y no lo
    // devuelve, cerramos como antes para no dejar al usuario colgado.
    if (idApuesta.value === null) emit('ok')
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

    <div v-if="idApuesta !== null" class="tiquete-listo">
      <strong>¡Apuesta registrada!</strong>
      <span>¿Imprimís el comprobante?</span>
    </div>

    <template #acciones>
      <template v-if="idApuesta !== null">
        <button class="btn btn-ghost" @click="emit('cerrar'); emit('ok')">Ahora no</button>
        <button class="btn btn-primary" :disabled="imprimiendo" @click="imprimir">
          {{ imprimiendo ? 'Imprimiendo…' : '🖨 Imprimir comprobante' }}
        </button>
      </template>
      <template v-else>
        <button class="btn btn-ghost" @click="emit('cerrar')">Cancelar</button>
        <button class="btn btn-primary" :disabled="!puedeJugar || cargando" @click="confirmar">
          {{ cargando ? 'Registrando…' : `Jugar ${colones(monto)}` }}
        </button>
      </template>
    </template>
  </Modal>
</template>

<style scoped>
.tiquete-listo {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: center;
  text-align: center;
  padding: 14px;
  margin-top: 12px;
  border: 1px dashed var(--border-soft, #3d4a45);
  border-radius: 10px;
  color: var(--text, #e8f2ed);
}
.tiquete-listo span { font-size: 12px; opacity: 0.8; }
</style>
