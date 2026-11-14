<script setup lang="ts">
const props = defineProps<{
  etiqueta: string
  maxSeleccion: number
  maximo: number
  modelValue: number[]
}>()

const emit = defineEmits<{ 'update:modelValue': [number[]] }>()

function toggle(n: number) {
  const actual = props.modelValue
  if (actual.includes(n)) {
    emit('update:modelValue', actual.filter((x) => x !== n))
    return
  }
  if (actual.length >= props.maxSeleccion) return
  emit('update:modelValue', [...actual, n].sort((a, b) => a - b))
}
</script>

<template>
  <div>
    <p class="hint">
      Selecciona hasta {{ maxSeleccion }} número{{ maxSeleccion > 1 ? 's' : '' }}
      ({{ modelValue.length }}/{{ maxSeleccion }}).
    </p>
    <div class="numpad">
      <button
        v-for="n in maximo + 1"
        :key="n - 1"
        type="button"
        :class="{ activo: modelValue.includes(n - 1) }"
        :disabled="!modelValue.includes(n - 1) && modelValue.length >= maxSeleccion"
        @click="toggle(n - 1)"
      >
        {{ String(n - 1).padStart(2, '0') }}
      </button>
    </div>
  </div>
</template>
