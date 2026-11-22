<script setup lang="ts">
const props = defineProps<{
  pagina: number
  paginas: number
  total?: number
  porPagina?: number
}>()
const emit = defineEmits<{ cambiar: [pagina: number] }>()

// Ventana de numeros de pagina: nunca mas de 7 botones, con la actual al medio.
function ventana(): number[] {
  const total = Math.max(1, props.paginas)
  const actual = Math.min(Math.max(1, props.pagina), total)
  let inicio = Math.max(1, actual - 3)
  const fin = Math.min(total, inicio + 6)
  inicio = Math.max(1, fin - 6)
  const out: number[] = []
  for (let i = inicio; i <= fin; i++) out.push(i)
  return out
}
</script>

<template>
  <div v-if="paginas > 1 || (total ?? 0) > 0" class="paginador">
    <span class="info">
      <template v-if="total != null && total > 0">
        {{ total }} en total · página {{ pagina }} de {{ paginas }}
      </template>
      <template v-else>Sin resultados</template>
    </span>
    <div v-if="paginas > 1" class="botones">
      <button class="pg" :disabled="pagina <= 1" @click="emit('cambiar', pagina - 1)">Anterior</button>
      <button
        v-for="p in ventana()"
        :key="p"
        class="pg num"
        :class="{ on: p === pagina }"
        @click="emit('cambiar', p)"
      >{{ p }}</button>
      <button class="pg" :disabled="pagina >= paginas" @click="emit('cambiar', pagina + 1)">Siguiente</button>
    </div>
  </div>
</template>

<style scoped>
.paginador {
  display: flex; justify-content: space-between; align-items: center; gap: 12px;
  flex-wrap: wrap; margin-top: 14px;
}
.info { font-size: 12px; color: var(--faint); }
.botones { display: flex; gap: 6px; flex-wrap: wrap; }
.pg {
  padding: 7px 12px; border-radius: 9px; border: 1px solid var(--border-soft);
  background: var(--surface); color: var(--muted); cursor: pointer; font-size: 12.5px;
}
.pg:hover:not(:disabled) { color: var(--text); border-color: var(--border); }
.pg:disabled { opacity: .45; cursor: not-allowed; }
.pg.num { min-width: 34px; }
.pg.on { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
</style>
