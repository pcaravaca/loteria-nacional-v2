// Sonidos del casino sintetizados con Web Audio API: sin archivos, sin dependencias.
// Se crean bajo demanda (el navegador exige un gesto del usuario antes de sonar).

let ctx: AudioContext | null = null
let habilitado = typeof localStorage !== 'undefined' && localStorage.getItem('casino-sfx') !== 'off'

export function sfxHabilitado() {
  return habilitado
}

export function alternarSfx(): boolean {
  habilitado = !habilitado
  try {
    localStorage.setItem('casino-sfx', habilitado ? 'on' : 'off')
  } catch {
    /* sin localStorage (modo privado): se queda en memoria */
  }
  if (habilitado) reproducir('click')
  return habilitado
}

function audio(): AudioContext | null {
  if (!habilitado) return null
  try {
    if (!ctx) ctx = new (window.AudioContext || (window as any).webkitAudioContext)()
    if (ctx.state === 'suspended') void ctx.resume()
    return ctx
  } catch {
    return null
  }
}

// Un tono con envolvente corta: ataque rapido y caida exponencial.
function tono(
  c: AudioContext,
  freq: number,
  dur: number,
  tipo: OscillatorType = 'sine',
  vol = 0.12,
  atraso = 0,
) {
  const t0 = c.currentTime + atraso
  const osc = c.createOscillator()
  const gain = c.createGain()
  osc.type = tipo
  osc.frequency.setValueAtTime(freq, t0)
  gain.gain.setValueAtTime(0.0001, t0)
  gain.gain.exponentialRampToValueAtTime(vol, t0 + 0.012)
  gain.gain.exponentialRampToValueAtTime(0.0001, t0 + dur)
  osc.connect(gain).connect(c.destination)
  osc.start(t0)
  osc.stop(t0 + dur + 0.02)
}

export type SfxNombre = 'click' | 'ficha' | 'ganar' | 'perder' | 'rodillo' | 'carta'

export function reproducir(nombre: SfxNombre) {
  const c = audio()
  if (!c) return
  switch (nombre) {
    case 'click':
      tono(c, 660, 0.07, 'square', 0.06)
      break
    case 'ficha':
      tono(c, 980, 0.06, 'triangle', 0.1)
      tono(c, 1320, 0.07, 'triangle', 0.08, 0.05)
      break
    case 'rodillo':
      tono(c, 240, 0.05, 'square', 0.07)
      tono(c, 300, 0.05, 'square', 0.07, 0.06)
      break
    case 'carta':
      tono(c, 520, 0.08, 'sine', 0.09)
      break
    case 'ganar':
      // Arpegio corto: sol, do, mi, sol.
      tono(c, 523, 0.12, 'triangle', 0.12)
      tono(c, 659, 0.12, 'triangle', 0.12, 0.09)
      tono(c, 784, 0.14, 'triangle', 0.12, 0.18)
      tono(c, 1046, 0.22, 'triangle', 0.1, 0.27)
      break
    case 'perder':
      tono(c, 220, 0.18, 'sine', 0.1)
      tono(c, 165, 0.26, 'sine', 0.09, 0.12)
      break
  }
}

export const sfx = {
  click: () => reproducir('click'),
  ficha: () => reproducir('ficha'),
  ganar: () => reproducir('ganar'),
  perder: () => reproducir('perder'),
  rodillo: () => reproducir('rodillo'),
  carta: () => reproducir('carta'),
}
