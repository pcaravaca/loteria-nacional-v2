<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  alternarMusica,
  alternarSilencio,
  fijarVolumen,
  musicaActivada,
  silenciado,
  volumen,
} from '@/utils/sfx'

const abierto = ref(false)
const mute = ref(silenciado())
const musica = ref(musicaActivada())
const vol = ref(volumen())

function toggleMute() {
  mute.value = alternarSilencio()
}
function toggleMusica() {
  musica.value = alternarMusica()
}
function cambiarVolumen(event: Event) {
  vol.value = fijarVolumen(Number((event.target as HTMLInputElement).value))
}
onMounted(() => {
  // Si el usuario pidio musica antes, arranca con el primer gesto (el click en este panel).
  if (musica.value && !mute.value) return
})
</script>

<template>
  <div class="audioctrl">
    <button class="audio-btn" :title="mute ? 'Activar sonido' : 'Silenciar'" @click="toggleMute">
      {{ mute ? '🔇' : '🔊' }}
    </button>
    <div v-if="abierto" class="audio-panel">
      <label class="fila">
        <span>Volumen</span>
        <input type="range" min="0" max="100" step="5" :value="vol" @input="cambiarVolumen" />
        <b>{{ vol }}</b>
      </label>
      <label class="fila">
        <span>Música</span>
        <button class="mini" :class="{ on: musica }" @click="toggleMusica">{{ musica ? 'Sí' : 'No' }}</button>
      </label>
      <button class="mini cerrar" @click="abierto = false">Cerrar</button>
    </div>
    <button class="audio-btn mas" title="Ajustes de sonido y música" @click="abierto = !abierto">♪</button>
  </div>
</template>

<style scoped>
.audioctrl { position: relative; display: flex; align-items: center; gap: 4px; }
.audio-btn {
  width: 32px; height: 32px; border-radius: 9px; border: 1px solid var(--border);
  background: var(--surface-2); cursor: pointer; font-size: 14px;
}
.audio-btn.mas { color: var(--prim-2); font-weight: 700; }
.audio-panel {
  position: absolute; top: 40px; right: 0; z-index: 80; width: 210px;
  background: var(--surface); border: 1px solid var(--border); border-radius: 14px;
  padding: 12px; box-shadow: var(--shadow); display: flex; flex-direction: column; gap: 10px;
}
.fila { display: flex; align-items: center; gap: 8px; font-size: 12px; color: var(--muted); }
.fila span { width: 52px; flex: none; }
.fila input { flex: 1; accent-color: #c9a227; }
.fila b { width: 26px; text-align: right; color: var(--text); }
.mini {
  border: 1px solid var(--border); background: var(--bg-2); color: var(--muted);
  border-radius: 8px; padding: 3px 10px; font-size: 12px; cursor: pointer;
}
.mini.on { color: var(--prim-2); border-color: var(--prim); }
.mini.cerrar { align-self: flex-end; }
</style>
