<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api/client'
import type { Loteria, Sorteo } from '@/types'
import LoteriaCard from '@/components/LoteriaCard.vue'
import SorteoCard from '@/components/SorteoCard.vue'
import ApuestaModal from '@/components/ApuestaModal.vue'
import EmptyState from '@/components/EmptyState.vue'

const props = defineProps<{ soloQuinielas?: boolean }>()

const loterias = ref<Loteria[]>([])
const sorteos = ref<Sorteo[]>([])
const cargando = ref(true)
const sorteoParaJugar = ref<Sorteo | null>(null)
const filtro = ref('')

const filtradas = computed(() =>
  loterias.value.filter((l) =>
    filtro.value ? l.nombre.toLowerCase().includes(filtro.value.toLowerCase()) : true,
  ),
)

const sorteosAbiertos = computed(() =>
  sorteos.value.filter(
    (s) => s.estado === 'abierto' && (!props.soloQuinielas || s.loteria?.tipo === 'quiniela'),
  ),
)

onMounted(async () => {
  try {
    const params = props.soloQuinielas ? { tipo: 'quiniela' } : {}
    const [l, s] = await Promise.all([
      api.get<Loteria[]>('/loterias', { params }),
      api.get<Sorteo[]>('/sorteos', { params: { estado: 'abierto' } }),
    ])
    loterias.value = l.data
    sorteos.value = s.data
  } finally {
    cargando.value = false
  }
})
</script>

<template>
  <div>
    <div class="page-head">
      <div class="eyebrow">{{ soloQuinielas ? 'Quinielas' : 'Loterías' }}</div>
      <h1>{{ soloQuinielas ? 'Quinelas deportivas y numéricas' : 'Loterías de Costa Rica' }}</h1>
      <p>
        {{
          soloQuinielas
            ? 'Pronostica los números y resultados. Premios automáticos al cerrar la jornada.'
            : 'Elige tus números, paga por jugada y el premio se acredita solo al cerrar el sorteo.'
        }}
      </p>
    </div>

    <div class="acciones-fila" style="margin-bottom: 18px">
      <input v-model="filtro" class="input" style="max-width: 300px" placeholder="Buscar por nombre…" />
      <span class="badge">{{ filtradas.length }} juegos</span>
      <span class="badge badge-ok">{{ sorteosAbiertos.length }} sorteos abiertos</span>
    </div>

    <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando catálogo…</div>

    <template v-else>
      <div v-if="sorteosAbiertos.length" class="section-title">Sorteos abiertos</div>
      <div v-if="sorteosAbiertos.length" class="grid grid-3">
        <SorteoCard v-for="s in sorteosAbiertos" :key="s.id" :sorteo="s" @jugar="sorteoParaJugar = s" />
      </div>

      <div class="section-title">Catálogo de juegos</div>
      <div v-if="filtradas.length" class="grid grid-3">
        <LoteriaCard v-for="l in filtradas" :key="l.id" :loteria="l" />
      </div>
      <div v-else>
        <EmptyState icono="🔍" titulo="Sin resultados" texto="Ningún juego coincide con tu búsqueda." />
      </div>
    </template>

    <ApuestaModal
      v-if="sorteoParaJugar"
      :sorteo="sorteoParaJugar"
      @cerrar="sorteoParaJugar = null"
      @ok="sorteoParaJugar = null"
    />
  </div>
</template>
