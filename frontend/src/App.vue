<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useMarcaStore } from '@/stores/marca'
import { aplicarTemaApp, fijarTemaAppLocal, leerTemaApp, temaAppValido } from '@/utils/temasApp'
import AppShell from '@/components/AppShell.vue'

const route = useRoute()
const auth = useAuthStore()
const marca = useMarcaStore()

const temaPerfil = computed(() => (auth.user?.tema_app as string) || '')
const temaActivo = leerTemaApp()
aplicarTemaApp(temaAppValido(temaActivo) ? temaActivo : 'esmeralda')

onMounted(() => marca.cargar())

// Si el perfil trae un tema elegido en otro dispositivo, se adopta.
if (temaPerfil.value && temaAppValido(temaPerfil.value)) {
  fijarTemaAppLocal(temaPerfil.value)
}

const esPublica = computed(() => !!route.meta.publica)
const usarShell = computed(() => auth.isAutenticado && !esPublica.value)
</script>

<template>
  <AppShell v-if="usarShell">
    <router-view />
  </AppShell>
  <router-view v-else />
</template>
