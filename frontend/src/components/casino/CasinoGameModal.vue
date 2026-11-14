<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import type { CasinoGame, PlayResponse } from '@/types'
import { colones } from '@/utils/formato'
import Modal from '@/components/Modal.vue'
import { heroes, type CodigoJuego } from '@/assets/casino'
import DadosGame from './juegos/DadosGame.vue'
import SlotsGame from './juegos/SlotsGame.vue'
import RuletaGame from './juegos/RuletaGame.vue'
import RaspaGame from './juegos/RaspaGame.vue'
import KenoGame from './juegos/KenoGame.vue'

const props = defineProps<{ juego: CasinoGame; clientSeed: string }>()
const emit = defineEmits<{ cerrar: [] }>()

const auth = useAuthStore()

const monto = ref<number>(props.juego.min_apuesta)
const cargando = ref(false)
const error = ref('')
const respuesta = ref<PlayResponse | null>(null)
const rondaKey = ref(0)

const rapidos = [500, 1000, 5000, 10000]
const saldo = computed(() => auth.user?.saldo ?? 0)

function ajustar(delta: number) {
  monto.value = Math.max(props.juego.min_apuesta, Math.min(props.juego.max_apuesta, monto.value + delta))
}

async function jugar(params: Record<string, unknown>) {
  error.value = ''
  cargando.value = true
  try {
    const { data } = await api.post<PlayResponse>('/casino/play', {
      game: props.juego.codigo,
      monto: monto.value,
      client_seed: props.clientSeed,
      params,
    })
    respuesta.value = data
    rondaKey.value++
    await auth.cargar()
  } catch (e: any) {
    error.value = e.message
  } finally {
    cargando.value = false
  }
}

function repetir() {
  respuesta.value = null
  error.value = ''
}
</script>

<template>
  <Modal :titulo="juego.nombre" :sub="juego.descripcion" @cerrar="emit('cerrar')">
    <div class="casino-modal">
      <div class="hero">
        <img :src="heroes[juego.codigo as CodigoJuego]" alt="" />
      </div>

      <div class="apuesta">
        <button class="step" @click="ajustar(-juego.min_apuesta)">−</button>
        <div class="monto">
          <span class="etq">Apuesta</span>
          <strong>{{ colones(monto) }}</strong>
        </div>
        <button class="step" @click="ajustar(juego.min_apuesta)">+</button>
      </div>
      <div class="rapidos">
        <button
          v-for="r in rapidos"
          :key="r"
          class="chip"
          :class="{ activa: monto === r }"
          @click="monto = Math.max(juego.min_apuesta, Math.min(juego.max_apuesta, r))"
        >{{ colones(r) }}</button>
        <span class="saldo">Saldo: <b>{{ colones(saldo) }}</b></span>
      </div>

      <p v-if="error" class="error-text">{{ error }}</p>

      <DadosGame
        v-if="juego.codigo === 'dados'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        @jugar="jugar"
      />
      <SlotsGame
        v-else-if="juego.codigo === 'slots'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        :ronda-key="rondaKey"
        @jugar="jugar"
      />
      <RuletaGame
        v-else-if="juego.codigo === 'ruleta'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        @jugar="jugar"
      />
      <RaspaGame
        v-else-if="juego.codigo === 'raspa'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        @jugar="jugar"
      />
      <KenoGame
        v-else-if="juego.codigo === 'keno'"
        :resultado="respuesta?.resultado ?? null"
        :premio="respuesta?.premio ?? 0"
        :cargando="cargando"
        @jugar="jugar"
      />

      <div v-if="respuesta" class="acciones-resultado">
        <button class="btn btn-ghost" @click="repetir">Jugar de nuevo</button>
        <button class="btn btn-primary" @click="emit('cerrar')">Cerrar</button>
      </div>

      <p class="nota">
        Resultado generado en el servidor · provably fair ·
        <span class="hash">{{ respuesta?.server_seed_hash?.slice(0, 16) ?? '' }}</span>
      </p>
    </div>
  </Modal>
</template>

<style scoped>
.casino-modal { display: flex; flex-direction: column; gap: 14px; }
.hero { height: 150px; border-radius: 12px; overflow: hidden; border: 1px solid var(--border-soft); }
.hero img { width: 100%; height: 100%; object-fit: cover; }
.apuesta { display: flex; align-items: center; gap: 10px; }
.step { width: 40px; height: 40px; border-radius: 10px; border: 1px solid var(--border); background: var(--surface); color: var(--text); font-size: 18px; cursor: pointer; }
.monto { flex: 1; text-align: center; background: #05080c; border: 1px solid var(--border); border-radius: 10px; padding: 8px; }
.monto .etq { display: block; font-size: 10px; text-transform: uppercase; letter-spacing: .09em; color: var(--faint); }
.monto strong { font-size: 20px; color: var(--text); }
.rapidos { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.chip { padding: 6px 10px; border-radius: 999px; border: 1px solid var(--border-soft); background: var(--surface); color: var(--muted); font-size: 12px; cursor: pointer; }
.chip.activa { border-color: var(--prim); color: var(--prim-2); background: var(--prim-dim); }
.saldo { margin-left: auto; font-size: 12px; color: var(--faint); }
.saldo b { color: var(--prim-2); }
.acciones-resultado { display: flex; gap: 10px; justify-content: flex-end; }
.nota { font-size: 11px; color: var(--faint); text-align: center; }
.hash { font-family: ui-monospace, monospace; }
</style>
