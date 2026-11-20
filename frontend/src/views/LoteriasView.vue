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

// Los sorteos se agrupan por juego y se muestran solo los proximos de cada uno:
// el calendario genera muchos sorteos por dia y una lista plana se vuelve ilegible.
interface GrupoSorteos {
  codigo: string
  nombre: string
  items: Sorteo[]
  total: number
}
const agrupados = computed<GrupoSorteos[]>(() => {
  const porCodigo = new Map<string, Sorteo[]>()
  for (const s of sorteosAbiertos.value) {
    const cod = s.loteria?.codigo ?? 'OTROS'
    const lista = porCodigo.get(cod) ?? []
    lista.push(s)
    porCodigo.set(cod, lista)
  }
  const out: GrupoSorteos[] = []
  for (const [codigo, lista] of porCodigo) {
    const nombre =
      loterias.value.find((l) => l.codigo === codigo)?.nombre ??
      (lista[0]?.loteria?.nombre === undefined ? 'Otros juegos' : lista[0].loteria.nombre)
    out.push({ codigo, nombre, items: lista.slice(0, 6), total: lista.length })
  }
  // primero las loterias del catalogo, en el orden en que aparecen ahi
  const orden = new Map(loterias.value.map((l, i) => [l.codigo, i]))
  out.sort((a, b) => (orden.get(a.codigo) ?? 999) - (orden.get(b.codigo) ?? 999))
  return out
})

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
      <template v-if="sorteosAbiertos.length">
        <div class="section-title">Sorteos abiertos</div>
        <section v-for="g in agrupados" :key="g.codigo" class="grupo-sorteos">
          <div class="grupo-head">
            <h2>{{ g.nombre }}</h2>
            <span>{{ g.total }} abiertos</span>
          </div>
          <div class="grid grid-3">
            <SorteoCard v-for="s in g.items" :key="s.id" :sorteo="s" @jugar="sorteoParaJugar = s" />
          </div>
          <p v-if="g.total > g.items.length" class="mas-sorteos">
            y {{ g.total - g.items.length }} sorteos mas de este juego.
          </p>
        </section>
      </template>

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
