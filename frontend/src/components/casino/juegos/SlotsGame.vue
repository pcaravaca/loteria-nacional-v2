<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { colones } from '@/utils/formato'
import { ordenSimbolos, paytable, simbolos } from '@/assets/casino'
import { reproducir } from '@/utils/sfx'

const props = defineProps<{
  resultado: any
  premio: number
  cargando: boolean
  rondaKey: number
}>()
const emit = defineEmits<{ jugar: [Record<string, unknown>] }>()

// Que rodillos pagaron: si salieron tres iguales marcan los tres; si solo
// coinciden los dos primeros, marcan esos dos.
const ganadores = computed<number[]>(() => {
  const r = props.resultado?.rodillos
  if (!Array.isArray(r) || r.length !== 3 || props.premio <= 0) return []
  if (r[0] === r[1] && r[1] === r[2]) return [0, 1, 2]
  if (r[0] === r[1]) return [0, 1]
  return []
})
const motivo = computed(() => {
  const r = props.resultado?.rodillos
  if (!r || !ganadores.value.length) return ''
  if (ganadores.value.length === 3) return `Tres ${r[0]}`
  return `Dos ${r[0]}`
})

const visibles = ref<string[]>(['cereza', 'campana', 'siete'])
const girando = ref(false)
let timer: number | undefined

function aleatorio() {
  return ordenSimbolos[Math.floor(Math.random() * ordenSimbolos.length)]
}

function girar() {
  girando.value = true
  reproducir('rodillo')
  visibles.value = [aleatorio(), aleatorio(), aleatorio()]
  timer = window.setInterval(() => {
    visibles.value = [aleatorio(), aleatorio(), aleatorio()]
  }, 90)
}

watch(
  () => props.rondaKey,
  (k) => {
    if (k === 0) return
    girar()
    const retardo = [900, 1350, 1800]
    retardo.forEach((ms, i) => {
      window.setTimeout(() => {
        if (props.resultado?.rodillos) {
          visibles.value[i] = props.resultado.rodillos[i]
        }
        if (i === 2) {
          window.clearInterval(timer)
          girando.value = false
        }
      }, ms)
    })
  },
)

onUnmounted(() => window.clearInterval(timer))

function jugar() {
  emit('jugar', {})
}
</script>

<template>
  <div class="slots">
    <div class="cabina">
      <div class="marquesina"><span class="luces">● ● ●</span><b>Tragamonedas</b><span class="luces">● ● ●</span></div>
      <div class="rieles-marco">
        <div class="rieles-railes"></div>
    <div class="reels" :class="{ girando }">
      <div v-for="(s, i) in visibles" :key="i" class="reel" :class="{ ganador: ganadores.includes(i) }">
        <img :src="simbolos[s]" :alt="s" draggable="false" />
      </div>
    </div>
        <div class="rieles-railes der"></div>
      </div>
      <div class="bandeja"><span class="ranura"></span><span class="monedas">◉ ◉ ◉</span></div>
    </div>

    <div v-if="resultado" class="estado" :class="{ gana: premio > 0 }">
      <template v-if="premio > 0">
        <span>Ganaste{{ motivo ? ' · ' + motivo : '' }}</span><strong>{{ colones(premio) }}</strong>
      </template>
      <span v-else class="neutro">Sin premio</span>
    </div>

    <div class="paytable">
      <div v-for="f in paytable.slice(0, 6)" :key="f.etiqueta" class="fila">
        <span class="combo">
          <img v-for="(s, i) in f.simbolos" :key="i" :src="simbolos[s]" :alt="s" />
        </span>
        <span class="x">x{{ f.multipl }}</span>
      </div>
    </div>

    <button class="btn btn-primary btn-block" :disabled="cargando || girando" @click="jugar">
      {{ cargando || girando ? 'Girando…' : 'Girar' }}
    </button>
  </div>
</template>

<style scoped>.reel.ganador {
  border-color: var(--gold);
  box-shadow: 0 0 0 2px var(--gold), 0 0 24px rgba(245, 185, 66, .55);
  animation: brillo 1.1s ease-in-out infinite alternate;
}
.reel.ganador img { animation: rebote .5s ease 2; }
@keyframes brillo {
  from { box-shadow: 0 0 0 2px var(--gold), 0 0 10px rgba(245, 185, 66, .35); }
  to { box-shadow: 0 0 0 3px #ffe9a8, 0 0 30px rgba(245, 185, 66, .75); }
}
@keyframes rebote {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-10px); }
}

/* Carcasa de maquina tragamonedas: se reconoce aunque este quieta. */
.cabina {
  border-radius: 18px; padding: 10px; display: flex; flex-direction: column; gap: 8px;
  background: linear-gradient(180deg, #132a21, #061510);
  border: 2px solid var(--gold); box-shadow: inset 0 12px 24px rgba(0,0,0,.5), 0 14px 30px rgba(0,0,0,.35);
}
.marquesina {
  display: flex; align-items: center; justify-content: space-between; gap: 8px;
  border-radius: 10px; padding: 6px 10px;
  background: linear-gradient(180deg, #f0d98a, #c9a227); color: #2b1d05;
  font-size: 11px; letter-spacing: .2em; text-transform: uppercase; font-weight: 900;
}
.marquesina .luces { color: #7a1414; letter-spacing: .1em; font-size: 9px; }
.rieles-marco { position: relative; padding: 0 8px; }
.rieles-railes {
  position: absolute; top: -4px; bottom: -4px; left: 0; width: 6px; border-radius: 6px;
  background: linear-gradient(180deg, #e8ce7a, #7a5c17);
}
.rieles-railes.der { left: auto; right: 0; }
.bandeja { display: flex; align-items: center; justify-content: center; gap: 12px; padding-top: 4px; }
.bandeja .ranura {
  width: 64px; height: 8px; border-radius: 4px; background: #02120c;
  border: 1px solid #2b3f35; box-shadow: inset 0 2px 4px rgba(0,0,0,.9);
}
.bandeja .monedas { color: #e8ce7a; font-size: 10px; letter-spacing: .3em; }

.slots { display: flex; flex-direction: column; gap: 12px; }
.reels { display: flex; gap: 10px; }
.reel {
  flex: 1; height: 118px; border-radius: 12px; background: #05080c;
  border: 1px solid var(--border); display: grid; place-items: center; overflow: hidden;
}
.reel img { width: 68px; height: 68px; object-fit: contain; }
.reels.girando .reel img { animation: vaiven .18s linear infinite; }
@keyframes vaiven {
  0% { transform: translateY(-14px); filter: blur(1px); }
  50% { transform: translateY(12px); }
  100% { transform: translateY(-14px); filter: blur(1px); }
}
.estado {
  display: flex; align-items: center; justify-content: center; gap: 10px;
  background: rgba(255, 255, 255, .02); border: 1px solid var(--border-soft);
  border-radius: 10px; padding: 10px; color: var(--muted);
}
.estado.gana { background: var(--prim-dim); border-color: rgba(16, 185, 129, .35); color: var(--prim-2); }
.estado strong { font-size: 20px; color: var(--gold); }
.estado .neutro { color: var(--muted); }
.paytable {
  display: grid; grid-template-columns: repeat(3, 1fr); gap: 5px;
  font-size: 11px; color: var(--muted);
}
.fila {
  display: flex; align-items: center; justify-content: space-between; gap: 6px;
  background: rgba(255, 255, 255, .02); border: 1px solid var(--border-soft);
  border-radius: 8px; padding: 4px 7px;
}
.combo { display: inline-flex; gap: 2px; }
.combo img { width: 18px; height: 18px; object-fit: contain; }
.x { color: var(--gold); font-weight: 700; }
</style>
