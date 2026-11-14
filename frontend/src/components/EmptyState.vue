<script setup lang="ts">
defineProps<{ icono?: string; titulo: string; texto?: string }>()
</script>

<template>
  <div class="empty">
    <div class="ico">{{ icono ?? '🎲' }}</div>
    <h3>{{ titulo }}</h3>
    <p v-if="texto">{{ texto }}</p>
    <slot />
  </div>
</template>
