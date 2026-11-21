<script setup lang="ts">
import { computed, ref } from 'vue'

import { esDispositivoAndroid, esAppNativa } from '@/utils/dispositivo'

const esAndroid = ref(esDispositivoAndroid())
const cerrado = ref(sessionStorage.getItem('app-banner-cerrado') === '1')
// Dentro de la app nativa no tiene sentido autopromocionarse.
const mostrar = computed(() => esAndroid.value && !esAppNativa() && !cerrado.value)
function cerrar() {
  sessionStorage.setItem('app-banner-cerrado', '1')
  cerrado.value = true
}
</script>

<template>
  <aside v-if="mostrar" class="app-banner">
    <div class="app-banner-art">▣</div>
    <div class="app-banner-texto">
      <b>Usá la app de Android</b>
      <small>Más rápida, misma cuenta y saldo. Se actualiza sola.</small>
    </div>
    <a class="btn btn-gold btn-sm" href="/descargar/">Descargar</a>
    <button class="app-banner-x" aria-label="Cerrar aviso" @click="cerrar">×</button>
  </aside>
</template>

<style scoped>
.app-banner {
  display: flex; align-items: center; gap: 12px; padding: 10px 14px;
  background: linear-gradient(180deg, rgba(232, 206, 122, .14), transparent);
  background: linear-gradient(180deg, color-mix(in srgb, var(--tm-accent, #e8ce7a) 16%, transparent), transparent);
  border: 1px solid var(--line, rgba(232, 206, 122, .3));
  border-radius: 14px; margin-bottom: 14px;
}
.app-banner-art {
  width: 34px; height: 34px; border-radius: 9px; flex: none;
  display: grid; place-items: center; font-size: 16px; color: #2b1d05;
  background: linear-gradient(180deg, #f0d98a, #c9a227);
}
.app-banner-texto { flex: 1; min-width: 0; line-height: 1.25; }
.app-banner-texto b { display: block; font-size: 13.5px; }
.app-banner-texto small { color: #9db8ac; font-size: 12px; }
.btn-sm { padding: 7px 12px; font-size: 12.5px; }
.app-banner-x { background: none; border: 0; color: #9db8ac; cursor: pointer; font-size: 18px; line-height: 1; padding: 4px 6px; }
</style>
