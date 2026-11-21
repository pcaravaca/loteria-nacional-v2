import { APP_KEY } from '@/api/client'

/** Detección de Android (teléfono o tablet) para sugerir la app nativa. */
export function esDispositivoAndroid(): boolean {
  if (typeof navigator === 'undefined') return false
  const ua = navigator.userAgent || ''
  if (/Android/i.test(ua)) return true
  // Chromium expone el platform real; algunos tablets reportan solo esto.
  const data = (navigator as unknown as { userAgentData?: { platform?: string } }).userAgentData
  if (data?.platform && /Android/i.test(data.platform)) return true
  // Tablets antiguos con UA sin "Android" pero táctiles y Linux.
  if (/Linux/i.test(navigator.platform || '') && window.matchMedia?.('(pointer: coarse)').matches) return true
  return false
}

/**
 * Verdadero cuando la pagina corre dentro del WebView de la app Android nativa.
 * La app agrega `?app=1` al cargar un modulo; asi la web sabe que ya tiene
 * encabezado, saldo y navegacion propios y no debe duplicar la estructura de
 * escritorio (sidebar + topbar) ni mostrarse a si misma como banner.
 *
 * Ojo: `getToken()` limpia el query con replaceState antes de que la app se
 * monte, por eso la bandera se guarda en sessionStorage ahi.
 */
export function esAppNativa(): boolean {
  if (typeof window === 'undefined') return false
  try {
    if (window.sessionStorage.getItem(APP_KEY) === '1') return true
  } catch {
    /* sessionStorage puede estar bloqueado; se ignora */
  }
  const q = window.location.search || ''
  const h = window.location.hash || ''
  return /[?&]app=1\b/.test(q) || /[?&]app=1\b/.test(h)
}
