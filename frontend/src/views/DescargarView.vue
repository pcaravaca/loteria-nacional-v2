<script setup lang="ts">
/**
 * Página pública de descarga de la app Android nativa.
 * El APK y la versión se sirven como archivos estáticos desde /descargar/,
 * así que la app sabe cuándo hay una versión nueva (version.json).
 */
import { onMounted, ref } from 'vue'

const apkUrl = `${import.meta.env.VITE_API_BASE || ''}/descargar/LoteriaNacional.apk`
const version = ref('')

onMounted(async () => {
  try {
    const r = await fetch('/descargar/version.json', { cache: 'no-store' })
    if (r.ok) {
      const j = await r.json()
      version.value = `${j.versionName} (${j.versionCode})`
    }
  } catch {
    /* sin version.json no pasa nada: el botón igual descarga el APK */
  }
})
</script>

<template>
  <div class="min-h-screen bg-[#07211a] text-[#f4f8f6] flex flex-col items-center justify-center px-6 py-16">
    <div class="text-5xl font-extrabold text-[#c9a227]">LN</div>
    <h1 class="mt-3 text-2xl font-bold">App Android de Lotería Nacional</h1>
    <p class="mt-2 text-sm text-[#9dbbad] text-center max-w-md">
      App nativa en Kotlin: mismo usuario, mismo saldo y los mismos sorteos.
      Se actualiza sola preguntando acá cada vez que se abre.
    </p>

    <a
      :href="apkUrl"
      class="mt-8 inline-flex items-center gap-2 rounded-2xl bg-[#c9a227] px-8 py-4 text-base font-bold text-[#1a1405] shadow-lg hover:brightness-110"
    >
      ⬇ Descargar para Android
    </a>

    <p v-if="version" class="mt-3 text-xs text-[#9dbbad]">Versión publicada: {{ version }}</p>

    <ol class="mt-10 max-w-md text-sm space-y-2 text-[#9dbbad]">
      <li>1. Instalá el APK (autorizá "orígenes desconocidos" si el teléfono lo pide).</li>
      <li>2. Entrá con tu correo y clave de siempre.</li>
      <li>3. La app revisa esta página y se actualiza sola cuando haya versión nueva.</li>
    </ol>
  </div>
</template>
