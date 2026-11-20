<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { api } from '@/api/client'
import { TEMAS_APP, fijarTemaAppLocal, leerTemaApp, temaAppValido, type TemaAppId } from '@/utils/temasApp'

const abierto = ref(false)
const actual = ref<TemaAppId>(leerTemaApp())

function elegir(id: TemaAppId) {
  actual.value = id
  fijarTemaAppLocal(id)
  // Se guarda en el perfil si hay sesion; si falla, queda el valor local.
  api.put('/auth/tema-app', { tema: id }).catch(() => {})
}

watch(actual, (v) => document.documentElement.dataset.temaApp = v)

onMounted(() => {
  const delPerfil = document.documentElement.dataset.temaApp
  if (delPerfil && temaAppValido(delPerfil)) actual.value = delPerfil
})
</script>

<template>
  <div class="tema-ctrl">
    <button class="tema-btn" title="Cambiar el tema de la app" @click="abierto = !abierto">
      🎨
    </button>
    <div v-if="abierto" class="tema-panel">
      <div class="tema-titulo">Tema de la aplicación</div>
      <button
        v-for="t in TEMAS_APP"
        :key="t.id"
        class="tema-opcion"
        :class="{ on: actual === t.id }"
        @click="elegir(t.id)"
      >
        <span class="muestra" :style="{ background: t.muestra }"></span>
        {{ t.nombre }}
      </button>
      <p class="tema-nota">Se aplica a toda la app y se guarda en tu perfil.</p>
    </div>
  </div>
</template>

<style scoped>
.tema-ctrl { position: relative; }
.tema-btn {
  width: 32px; height: 32px; border-radius: 9px; border: 1px solid var(--border);
  background: var(--surface-2); cursor: pointer; font-size: 14px;
}
.tema-panel {
  position: absolute; top: 40px; right: 0; z-index: 80; width: 210px;
  background: var(--surface); border: 1px solid var(--border); border-radius: 14px;
  padding: 12px; box-shadow: var(--shadow); display: flex; flex-direction: column; gap: 6px;
}
.tema-titulo { font-size: 12px; color: var(--muted); font-weight: 700; margin-bottom: 2px; }
.tema-opcion {
  display: flex; align-items: center; gap: 8px; text-align: left;
  border: 1px solid var(--border); background: var(--bg-2); color: var(--text);
  border-radius: 10px; padding: 7px 9px; font-size: 12.5px; cursor: pointer;
}
.tema-opcion.on { border-color: var(--prim); color: var(--prim-2); font-weight: 700; }
.muestra { width: 14px; height: 14px; border-radius: 50%; display: inline-block; flex: none; }
.tema-nota { margin: 6px 0 0; font-size: 11px; color: var(--faint); }
</style>
