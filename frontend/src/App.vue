<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useMarcaStore } from '@/stores/marca'
import AppShell from '@/components/AppShell.vue'

const route = useRoute()
const auth = useAuthStore()
const marca = useMarcaStore()

onMounted(() => marca.cargar())

const esPublica = computed(() => !!route.meta.publica)
const usarShell = computed(() => auth.isAutenticado && !esPublica.value)
</script>

<template>
  <AppShell v-if="usarShell">
    <router-view />
  </AppShell>
  <router-view v-else />
</template>
