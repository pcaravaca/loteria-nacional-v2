// Sonido del casino sintetizado con Web Audio API: sin archivos, sin dependencias.
// Incluye volumen ajustable, silencio y una musica ambiental suave, todos
// persistidos en localStorage para que el usuario no los tenga que re-poner.

let ctx: AudioContext | null = null
let master: GainNode | null = null
let musicaNodes: { stop: () => void } | null = null

const KEY_VOL = 'casino-volumen'
const KEY_SIL = 'casino-silencio'
const KEY_MUS = 'casino-musica'

const VOL_DEFECTO = 60

function leerClave(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function escribirClave(key: string, valor: string) {
  try {
    localStorage.setItem(key, valor)
  } catch {
    /* modo privado: queda en memoria */
  }
}

export function volumen(): number {
  const v = Number(leerClave(KEY_VOL))
  return Number.isFinite(v) && v >= 0 && v <= 100 ? v : VOL_DEFECTO
}

export function fijarVolumen(valor: number) {
  const v = Math.max(0, Math.min(100, Math.round(valor)))
  escribirClave(KEY_VOL, String(v))
  aplicarGanancia()
  return v
}

export function silenciado(): boolean {
  return leerClave(KEY_SIL) === '1'
}

export function alternarSilencio(): boolean {
  const ahora = !silenciado()
  escribirClave(KEY_SIL, ahora ? '1' : '0')
  aplicarGanancia()
  if (ahora) detenerMusica()
  else if (musicaActivada()) iniciarMusica()
  return ahora
}

export function musicaActivada(): boolean {
  return leerClave(KEY_MUS) !== 'off'
}

export function alternarMusica(): boolean {
  const ahora = !musicaActivada()
  escribirClave(KEY_MUS, ahora ? 'on' : 'off')
  if (ahora) iniciarMusica()
  else detenerMusica()
  return ahora
}

/** Para compatibilidad con el codigo viejo: el SFX deja de sonar solo si esta silenciado. */
export function sfxHabilitado() {
  return !silenciado()
}

export function alternarSfx(): boolean {
  return !alternarSilencio()
}

function ganancia(): GainNode | null {
  const c = audio()
  if (!c) return null
  if (!master) {
    master = c.createGain()
    master.connect(c.destination)
  }
  return master
}

function aplicarGanancia() {
  const c = ctx
  if (!c || !master) return
  const target = silenciado() ? 0 : Math.pow(volumen() / 100, 1.6) * 0.9
  master.gain.cancelScheduledValues(c.currentTime)
  master.gain.setTargetAtTime(target, c.currentTime, 0.03)
}

function audio(): AudioContext | null {
  try {
    if (!ctx) {
      const AC =
        window.AudioContext ||
        (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext
      ctx = new AC()
      master = null
      aplicarGanancia()
    }
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
  const salida = ganancia()
  if (!salida) return
  const t0 = c.currentTime + atraso
  const osc = c.createOscillator()
  const gain = c.createGain()
  osc.type = tipo
  osc.frequency.setValueAtTime(freq, t0)
  gain.gain.setValueAtTime(0.0001, t0)
  gain.gain.exponentialRampToValueAtTime(Math.max(0.0002, vol), t0 + 0.012)
  gain.gain.exponentialRampToValueAtTime(0.0001, t0 + dur)
  osc.connect(gain).connect(salida)
  osc.start(t0)
  osc.stop(t0 + dur + 0.02)
}

export type SfxNombre =
  | 'click'
  | 'ficha'
  | 'ganar'
  | 'perder'
  | 'rodillo'
  | 'carta'
  | 'bolilla'
  | 'tecla'
  | 'giro'
  | 'caida'
  | 'acierto'
  | 'error'
  | 'bingo'
  | 'tambor'
  | 'ruleta'
  | 'deflector'

export function reproducir(nombre: SfxNombre) {
  if (silenciado()) return
  const c = audio()
  if (!c) return
  switch (nombre) {
    case 'click':
      tono(c, 660, 0.07, 'square', 0.06)
      break
    case 'tecla':
      tono(c, 880, 0.05, 'triangle', 0.07)
      break
    case 'ficha':
      tono(c, 980, 0.06, 'triangle', 0.1)
      tono(c, 1320, 0.07, 'triangle', 0.08, 0.05)
      break
    case 'rodillo':
      tono(c, 240, 0.05, 'square', 0.07)
      tono(c, 300, 0.05, 'square', 0.07, 0.06)
      break
    case 'giro':
      for (let i = 0; i < 6; i++) tono(c, 200 + i * 60, 0.05, 'square', 0.05, i * 0.055)
      break
    case 'carta':
      tono(c, 520, 0.08, 'sine', 0.09)
      break
    case 'bolilla':
      // Golpe seco de bombo de bingo con una nota corta.
      tono(c, 180, 0.09, 'square', 0.12)
      tono(c, 1200, 0.05, 'triangle', 0.06, 0.02)
      break
    case 'caida':
      tono(c, 900, 0.16, 'sine', 0.08)
      tono(c, 620, 0.2, 'sine', 0.07, 0.1)
      break
    case 'acierto':
      tono(c, 780, 0.09, 'triangle', 0.1)
      tono(c, 1170, 0.12, 'triangle', 0.09, 0.07)
      break
    case 'error':
      tono(c, 180, 0.14, 'sawtooth', 0.08)
      tono(c, 140, 0.18, 'sawtooth', 0.07, 0.1)
      break
    case 'tambor':
      for (let i = 0; i < 3; i++) tono(c, 160, 0.08, 'square', 0.09, i * 0.1)
      break
    case 'ruleta': {
      // Friccion del aro: ruido filtrado que va bajando de frecuencia mientras
      // la bola pierde velocidad por el carril.
      const salida = ganancia()
      if (!salida) break
      const dur = 3.4
      const buf = c.createBuffer(1, Math.floor(c.sampleRate * dur), c.sampleRate)
      const datos = buf.getChannelData(0)
      for (let i = 0; i < datos.length; i++) datos[i] = Math.random() * 2 - 1
      const ruido = c.createBufferSource()
      ruido.buffer = buf
      const filtro = c.createBiquadFilter()
      filtro.type = 'bandpass'
      filtro.Q.value = 1.4
      filtro.frequency.setValueAtTime(1500, c.currentTime)
      filtro.frequency.exponentialRampToValueAtTime(320, c.currentTime + dur)
      const g = c.createGain()
      g.gain.setValueAtTime(0.0001, c.currentTime)
      g.gain.exponentialRampToValueAtTime(0.07, c.currentTime + 0.12)
      g.gain.exponentialRampToValueAtTime(0.0001, c.currentTime + dur)
      ruido.connect(filtro).connect(g).connect(salida)
      ruido.start()
      ruido.stop(c.currentTime + dur + 0.05)
      break
    }
    case 'deflector':
      // Golpe seco de la bolilla contra un deflector metalico.
      tono(c, 2300, 0.035, 'square', 0.05)
      tono(c, 1560, 0.06, 'triangle', 0.05, 0.008)
      break
    case 'bingo':
      // Campanilla de bingo: arpegio brillante con campanas.
      tono(c, 1046, 0.3, 'triangle', 0.12)
      tono(c, 1318, 0.3, 'triangle', 0.11, 0.1)
      tono(c, 1568, 0.42, 'triangle', 0.11, 0.2)
      tono(c, 2093, 0.5, 'sine', 0.08, 0.3)
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

// Musica ambiental: dos osciladores graves con vaiven lento, bien bajito.
function iniciarMusica() {
  if (silenciado() || musicaNodes) return
  const c = audio()
  const salida = ganancia()
  if (!c || !salida) return
  const oscs: OscillatorNode[] = []
  const gains: GainNode[] = []
  const lfo = c.createOscillator()
  const lfoGain = c.createGain()
  lfo.frequency.value = 0.08
  lfoGain.gain.value = 0.012
  lfo.connect(lfoGain)
  ;[110, 164.8, 220].forEach((f, i) => {
    const o = c.createOscillator()
    const g = c.createGain()
    o.type = i === 2 ? 'triangle' : 'sine'
    o.frequency.value = f
    g.gain.value = 0.0001
    g.gain.setTargetAtTime(i === 2 ? 0.02 : 0.035, c.currentTime, 1.2)
    lfoGain.connect(g.gain)
    o.connect(g).connect(salida)
    o.start()
    oscs.push(o)
    gains.push(g)
  })
  lfo.start()
  oscs.push(lfo)
  musicaNodes = {
    stop: () => {
      const t = c.currentTime
      gains.forEach((g) => g.gain.setTargetAtTime(0.0001, t, 0.25))
      setTimeout(
        () =>
          oscs.forEach((o) => {
            try {
              o.stop()
            } catch {
              /* ya detenido */
            }
          }),
        900,
      )
    },
  }
}

function detenerMusica() {
  if (!musicaNodes) return
  musicaNodes.stop()
  musicaNodes = null
}

export const sfx = {
  click: () => reproducir('click'),
  ficha: () => reproducir('ficha'),
  ganar: () => reproducir('ganar'),
  perder: () => reproducir('perder'),
  rodillo: () => reproducir('rodillo'),
  carta: () => reproducir('carta'),
  bolilla: () => reproducir('bolilla'),
  tecla: () => reproducir('tecla'),
  giro: () => reproducir('giro'),
  caida: () => reproducir('caida'),
  acierto: () => reproducir('acierto'),
  error: () => reproducir('error'),
  bingo: () => reproducir('bingo'),
  tambor: () => reproducir('tambor'),
  ruleta: () => reproducir('ruleta'),
  deflector: () => reproducir('deflector'),
}
