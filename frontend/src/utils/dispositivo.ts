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
