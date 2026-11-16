<script setup lang="ts">
withDefaults(defineProps<{ titulo: string; sub?: string; ancho?: 'md' | 'lg'; mostrarCerrar?: boolean }>(), {
  ancho: 'md',
  mostrarCerrar: false,
})
const emit = defineEmits<{ cerrar: [] }>()
</script>

<template>
  <div class="modal-backdrop" @click.self="emit('cerrar')">
    <div class="modal" :class="ancho === 'lg' ? 'modal-lg' : ''">
      <button v-if="mostrarCerrar" type="button" class="modal-x" aria-label="Cerrar" @click="emit('cerrar')">×</button>
      <h3>{{ titulo }}</h3>
      <p v-if="sub" class="modal-sub">{{ sub }}</p>
      <slot />
      <div class="modal-acciones">
        <slot name="acciones" />
      </div>
    </div>
  </div>
</template>
