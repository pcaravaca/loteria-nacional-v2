<script setup lang="ts">
defineProps<{
  etiqueta: string
  valor: string
  nota?: string
  color?: string
  icono?: string
}>()
</script>

<template>
  <div class="card stat" :style="{ '--stat-c': color }">
    <div class="etq"><span v-if="icono">{{ icono }}</span>{{ etiqueta }}</div>
    <div class="valor">{{ valor }}</div>
    <div v-if="nota" class="nota">{{ nota }}</div>
  </div>
</template>
