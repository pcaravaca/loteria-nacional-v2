// Temas visuales de toda la aplicacion (no solo el casino). Cada tema redefine
// las variables base de la hoja de estilos; la preferencia se guarda en este
// dispositivo y se sincroniza con el perfil (users.tema_app).
export type TemaAppId = 'esmeralda' | 'zafiro' | 'rubi' | 'amatista' | 'grafito'

export interface TemaApp {
  id: TemaAppId
  nombre: string
  muestra: string // color de muestra para el selector
}

export const TEMAS_APP: TemaApp[] = [
  { id: 'esmeralda', nombre: 'Esmeralda', muestra: '#10b981' },
  { id: 'zafiro', nombre: 'Zafiro', muestra: '#3b82f6' },
  { id: 'rubi', nombre: 'Rubí', muestra: '#e11d48' },
  { id: 'amatista', nombre: 'Amatista', muestra: '#a78bfa' },
  { id: 'grafito', nombre: 'Grafito', muestra: '#94a3b8' },
]

const KEY = 'app-tema'

export function temaAppValido(tema: string): tema is TemaAppId {
  return TEMAS_APP.some((t) => t.id === tema)
}

export function leerTemaApp(): TemaAppId {
  try {
    const v = localStorage.getItem(KEY)
    if (v && temaAppValido(v)) return v
  } catch {
    /* sin localStorage */
  }
  return 'esmeralda'
}

export function aplicarTemaApp(tema: TemaAppId) {
  document.documentElement.dataset.temaApp = tema
}

export function fijarTemaAppLocal(tema: TemaAppId) {
  try {
    localStorage.setItem(KEY, tema)
  } catch {
    /* sin localStorage */
  }
  aplicarTemaApp(tema)
}
