// Tamano de los juegos: el usuario puede agrandar o achicar todos los tableros
// (50% a 150%) y la preferencia queda guardada en este dispositivo.
const KEY = 'casino-zoom'

export function zoomJuego(): number {
  try {
    const v = Number(localStorage.getItem(KEY))
    if (Number.isFinite(v) && v >= 50 && v <= 150) return v
  } catch {
    /* sin localStorage */
  }
  return 100
}

export function fijarZoom(valor: number): number {
  const v = Math.max(50, Math.min(150, Math.round(valor / 5) * 5))
  try {
    localStorage.setItem(KEY, String(v))
  } catch {
    /* sin localStorage */
  }
  return v
}
