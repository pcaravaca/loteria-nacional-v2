<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AppShell from '@/components/AppShell.vue'

const route = useRoute()
const auth = useAuthStore()

const esPublica = computed(() => !!route.meta.publica)
const usarShell = computed(() => auth.isAutenticado && !esPublica.value)
</script>

<template>
  <AppShell v-if="usarShell">
    <router-view />
  </AppShell>
  <router-view v-else />
</template>
