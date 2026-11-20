// Ambientes visuales del casino. Cada tema es solo presentacion: cambia la
// carcasa (marquesina, panel de apuesta, fichas y guia) y nunca la logica del
// juego. El backend valida que el codigo exista (models.TemaCasinoValido).
export type TemaCasino =
  | 'clasico'
  | 'neon'
  | 'premium'
  | 'tropical'
  | 'medianoche'
  | 'rubi'

export interface TemaInfo {
  codigo: TemaCasino
  nombre: string
  descripcion: string
  // Gradiente de la muestra en el selector.
  muestra: string
}

export const TEMA_DEFAULT: TemaCasino = 'premium'

export const TEMAS: TemaInfo[] = [
  {
    codigo: 'clasico',
    nombre: 'Clásico',
    descripcion: 'Dorado y crema, la máquina de siempre',
    muestra: 'linear-gradient(135deg,#f0c35a,#8a6524)',
  },
  {
    codigo: 'neon',
    nombre: 'Neón',
    descripcion: 'Cian sobre azul noche',
    muestra: 'linear-gradient(135deg,#38bdf8,#0b1a2b)',
  },
  {
    codigo: 'premium',
    nombre: 'Premium',
    descripcion: 'Negro esmeralda con línea dorada',
    muestra: 'linear-gradient(135deg,#0a100d 55%,#c9a227)',
  },
  {
    codigo: 'tropical',
    nombre: 'Tropical CR',
    descripcion: 'Verde bosque y sol, el de la marca',
    muestra: 'linear-gradient(135deg,#5ee8a5,#0b5c3c)',
  },
  {
    codigo: 'medianoche',
    nombre: 'Medianoche',
    descripcion: 'Violeta y plata, tono de sala VIP',
    muestra: 'linear-gradient(135deg,#c9b8ff,#1a1436)',
  },
  {
    codigo: 'rubi',
    nombre: 'Rubí',
    descripcion: 'Rojo casino con oro',
    muestra: 'linear-gradient(135deg,#e04a5c,#3a0a12)',
  },
]

export function esTemaValido(tema: string | undefined | null): tema is TemaCasino {
  return !!tema && TEMAS.some((t) => t.codigo === tema)
}

export function nombreTema(tema: string | undefined | null): string {
  return TEMAS.find((t) => t.codigo === tema)?.nombre ?? 'Premium'
}
