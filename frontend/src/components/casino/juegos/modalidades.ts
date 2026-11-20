// Modalidades seleccionables de los juegos del casino. Es el espejo de lo que
// declara el backend (internal/casino/modalidades.go): el backend es la fuente
// de verdad para los pagos, y esta copia dibuja el tablero antes de jugar.
// La primera modalidad de cada juego es la clasica: la que se juega si el
// jugador no elige ninguna.
import { ref, type Ref } from 'vue'

export interface PagoModalidad {
  etiqueta: string
  mult: string
}

export interface ModalidadCfg {
  codigo: string
  nombre: string
  descripcion: string
  pagos: PagoModalidad[]
  // Ruleta
  casillas?: string[]
  rojos?: string[]
  // Keno
  bolillasTotales?: number
  elegidos?: number
  cantidad?: number
  tablaKeno?: Record<number, number>
  // Raspa
  simbolosTotales?: number
  // Cascada
  filas?: number
  tablaCascada?: number[]
  // Alto / Hilo
  tiros?: number
}

const RULETA_EUROPEA = Array.from({ length: 37 }, (_, i) => String(i))
const RULETA_RAPIDA = ['0', '00', '1', '2', '3', '4', '5', '6', '7', '8']

export const MODALIDADES: Record<string, ModalidadCfg[]> = {
  ruleta: [
    {
      codigo: 'europea',
      nombre: 'Ruleta Europea',
      descripcion: 'Rueda de 37 casillas (del 0 al 36): pleno x35 y color x2.',
      pagos: [
        { etiqueta: 'Pleno', mult: 'x35' },
        { etiqueta: 'Color', mult: 'x2' },
      ],
      casillas: RULETA_EUROPEA,
      rojos: ['1', '3', '5', '7', '9', '12', '14', '16', '18', '19', '21', '23', '25', '27', '30', '32', '34', '36'],
    },
    {
      codigo: 'rapida',
      nombre: 'Ruleta Rápida',
      descripcion: 'Rueda corta de 10 casillas (0, 00 y del 1 al 8): pleno x8 y color x2.',
      pagos: [
        { etiqueta: 'Pleno', mult: 'x8' },
        { etiqueta: 'Color', mult: 'x2' },
      ],
      casillas: RULETA_RAPIDA,
      rojos: ['1', '3', '5', '7'],
    },
  ],
  keno: [
    {
      codigo: 'clasico',
      nombre: 'Keno Clásico',
      descripcion: 'Elegí 5 números de 60 y salen 15 bolillas: 2 aciertos x1, 3 x3, 4 x8 y 5 x65.',
      pagos: [
        { etiqueta: '5 aciertos', mult: 'x65' },
        { etiqueta: '4 aciertos', mult: 'x8' },
        { etiqueta: '3 aciertos', mult: 'x3' },
        { etiqueta: '2 aciertos', mult: 'x1' },
      ],
      bolillasTotales: 60,
      elegidos: 5,
      cantidad: 15,
      tablaKeno: { 2: 1, 3: 3, 4: 8, 5: 65 },
    },
    {
      codigo: 'relampago',
      nombre: 'Keno Relámpago',
      descripcion: 'Elegí 4 números de 40 y salen 8 bolillas: paga desde el primer acierto (x1, x2, x5 y x8).',
      pagos: [
        { etiqueta: '4 aciertos', mult: 'x8' },
        { etiqueta: '3 aciertos', mult: 'x5' },
        { etiqueta: '2 aciertos', mult: 'x2' },
        { etiqueta: '1 acierto', mult: 'x1' },
      ],
      bolillasTotales: 40,
      elegidos: 4,
      cantidad: 8,
      tablaKeno: { 1: 1, 2: 2, 3: 5, 4: 8 },
    },
  ],
  raspa: [
    {
      codigo: 'clasica',
      nombre: 'Raspa Clásica',
      descripcion: 'Tres casillas con 6 símbolos: tres iguales x18 y cualquier par x1.',
      pagos: [
        { etiqueta: 'Tres iguales', mult: 'x18' },
        { etiqueta: 'Cualquier par', mult: 'x1' },
      ],
      simbolosTotales: 6,
    },
    {
      codigo: 'expres',
      nombre: 'Raspa Exprés',
      descripcion: 'Tres casillas con 4 símbolos: tres iguales x5 y cualquier par x1. Los premios salen más seguido.',
      pagos: [
        { etiqueta: 'Tres iguales', mult: 'x5' },
        { etiqueta: 'Cualquier par', mult: 'x1' },
      ],
      simbolosTotales: 4,
    },
  ],
  baccarat: [
    {
      codigo: 'estandar',
      nombre: 'Baccarat Estándar',
      descripcion: 'Jugador x2, banca x2 con 5% de comisión y empate x8. En empate se devuelve lo apostado a jugador o banca.',
      pagos: [
        { etiqueta: 'Jugador', mult: 'x2' },
        { etiqueta: 'Banca', mult: 'x2 (5% comisión)' },
        { etiqueta: 'Empate', mult: 'x8' },
      ],
    },
    {
      codigo: 'sin-comision',
      nombre: 'Baccarat Sin Comisión',
      descripcion: 'La banca paga x2 sin comisión, pero si la banca gana con 6 paga x1. Empate x8; en empate se pierde lo apostado a jugador o banca.',
      pagos: [
        { etiqueta: 'Jugador', mult: 'x2' },
        { etiqueta: 'Banca', mult: 'x2 (x1 si gana con 6)' },
        { etiqueta: 'Empate', mult: 'x8' },
      ],
    },
  ],
  alto: [
    {
      codigo: 'clasico',
      nombre: 'Bajo o Sobre Clásico',
      descripcion: 'Dado del 1 al 100: elegí el tope y acertá si cae abajo o arriba. Un tiro, un pago.',
      pagos: [
        { etiqueta: '2%', mult: 'x48' },
        { etiqueta: '5%', mult: 'x19' },
        { etiqueta: '10%', mult: 'x9' },
        { etiqueta: '25%', mult: 'x3' },
        { etiqueta: '48%', mult: 'x2' },
        { etiqueta: '75%', mult: 'x1' },
        { etiqueta: '90%', mult: 'x1' },
        { etiqueta: '95%', mult: 'x1' },
      ],
      tiros: 1,
    },
    {
      codigo: 'racha',
      nombre: 'Bajo o Sobre en Racha',
      descripcion: 'Dos tiros con el mismo tope: acertá los dos seguidos y cobrá x4. Si falla uno solo, perdés la apuesta.',
      pagos: [{ etiqueta: 'Dos aciertos seguidos', mult: 'x4' }],
      tiros: 2,
    },
  ],
  hilo: [
    {
      codigo: 'clasico',
      nombre: 'Mayor o Menor Clásico',
      descripcion: 'Adiviná si la próxima carta es más alta o más baja que la base. Si empata, perdés.',
      pagos: [{ etiqueta: 'Acierto', mult: 'x2' }],
      tiros: 1,
    },
    {
      codigo: 'racha',
      nombre: 'Mayor o Menor en Racha',
      descripcion: 'Tres cartas: acertá dos predicciones seguidas con el mismo sentido y cobrá x6.',
      pagos: [{ etiqueta: 'Dos aciertos seguidos', mult: 'x6' }],
      tiros: 2,
    },
  ],
  cascada: [
    {
      codigo: 'clasica',
      nombre: 'Cascada Clásica',
      descripcion: 'Nueve filas de pinchos y 10 casillas: paga del centro (x0) a los bordes (x10).',
      pagos: [
        { etiqueta: 'Bordes', mult: 'x10' },
        { etiqueta: 'Centro', mult: 'x0' },
      ],
      filas: 9,
      tablaCascada: [10, 5, 3, 1, 0, 0, 1, 3, 5, 10],
    },
    {
      codigo: 'rapida',
      nombre: 'Cascada Rápida',
      descripcion: 'Cinco filas de pinchos y 6 casillas: cae rapidísimo y paga x4 en los bordes.',
      pagos: [
        { etiqueta: 'Bordes', mult: 'x4' },
        { etiqueta: 'Centro', mult: 'x0' },
      ],
      filas: 5,
      tablaCascada: [4, 2, 0, 0, 2, 4],
    },
    {
      codigo: 'profunda',
      nombre: 'Cascada Profunda',
      descripcion: 'Doce filas de pinchos y 13 casillas: recorrido larguísimo con x10 en los bordes.',
      pagos: [
        { etiqueta: 'Bordes', mult: 'x10' },
        { etiqueta: 'Centro', mult: 'x0' },
      ],
      filas: 12,
      tablaCascada: [10, 9, 6, 3, 1, 0, 0, 0, 1, 3, 6, 9, 10],
    },
  ],
}

/** Modalidades de un juego; si no conoce el juego, lista vacia. */
export function modalidadesDe(codigo: string): ModalidadCfg[] {
  return MODALIDADES[codigo] ?? []
}

/** Modalidad por codigo; si no existe, devuelve la primera (la clasica). */
export function modalidadPorCodigo(codigoJuego: string, codigo: string): ModalidadCfg {
  const lista = modalidadesDe(codigoJuego)
  return lista.find((m) => m.codigo === codigo) ?? lista[0]
}

// La modalidad seleccionada se comparte entre el tablero y el modal: el
// encabezado del modal muestra el tope de la modalidad elegida, asi que tiene
// que leer el mismo estado que el componente del juego.
const seleccionModalidad: Record<string, Ref<string>> = {}

/** Modalidad seleccionada del juego; persiste entre aperturas del modal. */
export function modalidadCompartida(codigoJuego: string): Ref<string> {
  if (!seleccionModalidad[codigoJuego]) {
    const lista = modalidadesDe(codigoJuego)
    seleccionModalidad[codigoJuego] = ref(lista[0]?.codigo ?? '')
  }
  return seleccionModalidad[codigoJuego]
}

/** Color de una casilla de ruleta. */
export function colorDeCasilla(cfg: ModalidadCfg, casilla: string): 'rojo' | 'negro' | 'verde' {
  if (casilla === '0' || casilla === '00') return 'verde'
  if (cfg.codigo === 'rapida') {
    return ['1', '3', '5', '7'].includes(casilla) ? 'rojo' : 'negro'
  }
  return (cfg.rojos ?? []).includes(casilla) ? 'rojo' : 'negro'
}
