// Voz del bingo: la bolilla cantada en voz alta, con presets configurables
// (estilo Waze). Usa la Web Speech API del navegador/app: en Android salen las
// voces instaladas del sistema, sin archivos de audio ni dependencias.
import { silenciado } from './sfx'

export interface PresetVoz {
  id: string
  nombre: string
  descripcion: string
  pitch: number
  rate: number
  // Nombre (o parte) de la voz que prefiere este preset; si no matchea, se
  // usa la primera voz en español disponible.
  prefiere?: string
}

export const PRESETS_VOZ: PresetVoz[] = [
  { id: 'clasica', nombre: 'Clásica', descripcion: 'Voz del sistema, ritmo normal.', pitch: 1, rate: 1 },
  { id: 'tiro', nombre: 'De tiro', descripcion: 'Más grave y firme, se entiende en salón lleno.', pitch: 0.78, rate: 0.96, prefiere: 'masculina' },
  { id: 'animada', nombre: 'Animada', descripcion: 'Aguda y rapidita, para bingo relámpago.', pitch: 1.28, rate: 1.14 },
  { id: 'severa', nombre: 'Severa', descripcion: 'Lenta y clara,ideal para cartones 90.', pitch: 0.9, rate: 0.86 },
  { id: 'radio', nombre: 'Locutor', descripcion: 'Con énfasis de locución deportiva.', pitch: 1.12, rate: 1.24 },
]

const KEY_VOZ = 'bingo-voz'

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

export function presetVoz(): string {
  const id = leerClave(KEY_VOZ)
  return PRESETS_VOZ.some((p) => p.id === id) ? (id as string) : 'clasica'
}

export function fijarPresetVoz(id: string) {
  if (!PRESETS_VOZ.some((p) => p.id === id)) return
  escribirClave(KEY_VOZ, id)
}

export function presetPorId(id: string): PresetVoz {
  return PRESETS_VOZ.find((p) => p.id === id) ?? PRESETS_VOZ[0]
}

function soportado(): boolean {
  // Se comprueba el valor y no solo la presencia: hay WebViews donde la
  // propiedad existe pero esta indefinida, y ahi el canto tiene que caer en la
  // voz nativa de la app.
  return typeof window !== 'undefined' && typeof window.speechSynthesis !== 'undefined'
}

// La app Android abre los juegos en un WebView que NO trae Web Speech API: ahí la
// voz la pone el TextToSpeech del sistema, inyectado como window.LottoVoz.
declare global {
  interface Window {
    LottoVoz?: {
      hablar: (texto: string, pitch: number, rate: number) => void
      callar: () => void
    }
  }
}

function nativo(): boolean {
  return typeof window !== 'undefined' && typeof window.LottoVoz?.hablar === 'function'
}

let vocesCache: SpeechSynthesisVoice[] = []

function cargarVoces(): SpeechSynthesisVoice[] {
  if (!soportado()) return []
  const lista = window.speechSynthesis.getVoices()
  if (lista.length) vocesCache = lista
  return vocesCache
}

if (soportado()) {
  cargarVoces()
  window.speechSynthesis.onvoiceschanged = () => cargarVoces()
}

function esEspanol(v: SpeechSynthesisVoice): boolean {
  return /^es([-_]|$)/i.test(v.lang)
}

// La voz del preset: primero la que pide por nombre, si no la primera en
// español, si no la del sistema.
export function elegirVoz(preset: PresetVoz): SpeechSynthesisVoice | null {
  const voces = cargarVoces()
  if (!voces.length) return null
  const espanolas = voces.filter(esEspanol)
  const pool = espanolas.length ? espanolas : voces
  if (preset.prefiere) {
    const hallada = pool.find(
      (v) =>
        v.name.toLowerCase().includes(preset.prefiere!.toLowerCase()) ||
        v.lang.toLowerCase().includes(preset.prefiere!.toLowerCase()),
    )
    if (hallada) return hallada
  }
  return pool[0]
}

export function vocesDisponibles(): SpeechSynthesisVoice[] {
  return cargarVoces().filter(esEspanol)
}

/** Cantar una bolilla. Devuelve true si se pudo encolar la voz. */
export function cantarBolilla(texto: string, presetId: string, forzar = false): boolean {
  if (!forzar && silenciado()) return false
  const preset = presetPorId(presetId)
  if (!soportado()) {
    // WebView de la app Android: no hay Web Speech API, se usa el TTS nativo.
    if (nativo()) {
      window.LottoVoz!.hablar(texto, preset.pitch, preset.rate)
      return true
    }
    return false
  }
  const voz = elegirVoz(preset)
  window.speechSynthesis.cancel()
  const u = new SpeechSynthesisUtterance(texto)
  if (voz) u.voice = voz
  u.lang = voz?.lang || 'es-CR'
  u.pitch = preset.pitch
  u.rate = preset.rate
  u.volume = 1
  window.speechSynthesis.speak(u)
  return true
}

export function callar() {
  if (nativo()) window.LottoVoz!.callar()
  if (!soportado()) return
  window.speechSynthesis.cancel()
}
