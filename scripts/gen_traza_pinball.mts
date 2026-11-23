// Genera una traza determinista, la simula en el motor TS y la vuelca en JSON
// para que el motor Go la reproduzca y se compare el puntaje (equivalencia
// extremo a extremo entre pinballFisica.ts y pinballfisica.go).
import { writeFileSync } from 'node:fs'
import { crearSim, SPEC } from '/mnt/c/Users/Peter/lotto-work/repo/frontend/src/components/casino/juegos/pinballFisica.ts'

let seed = 987654321
const rnd = () => {
  seed = (seed * 1103515245 + 12345) & 0x7fffffff
  return seed / 0x7fffffff
}

const N = Number(process.env.N || 200)
const casos = []
for (let i = 0; i < N; i++) {
  const sim = crearSim()
  const traza = []
  let f = 0
  while (sim.bolas() < SPEC.BOLAS && f < SPEC.MAX_FRAMES) {
    const evs = []
    const b = sim.bola()
    if (!b) {
      evs.push({ f, k: 'P', v: Math.round((0.35 + rnd() * 0.65) * 100) / 100 })
    } else {
      // jugador "semi hábil" con ruido
      if (rnd() < 0.25) evs.push({ f, k: 'L', v: rnd() < 0.5 ? 1 : 0 })
      if (rnd() < 0.25) evs.push({ f, k: 'R', v: rnd() < 0.5 ? 1 : 0 })
    }
    for (const e of evs) traza.push(e)
    sim.paso(f, evs)
    f++
  }
  sim.cerrar()
  casos.push({ n: i, eventos: traza, puntajeTS: sim.puntaje(), bolas: sim.bolas(), frames: sim.frames() })
}
writeFileSync('/home/peter/.hermes-data/cache/scratch/traza_equivalencia.json', JSON.stringify(casos))
console.log(`casos=${casos.length} primero: ${JSON.stringify(casos[0]).slice(0, 220)}`)
