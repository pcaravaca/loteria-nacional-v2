<script setup lang="ts">
defineProps<{ titulo: string; sub?: string }>()
const emit = defineEmits<{ cerrar: [] }>()
</script>

<template>
  <div class="modal-backdrop" @click.self="emit('cerrar')">
    <div class="modal">
      <h3>{{ titulo }}</h3>
      <p v-if="sub" class="modal-sub">{{ sub }}</p>
      <slot />
      <div class="modal-acciones">
        <slot name="acciones" />
      </div>
    </div>
  </div>
</template>
