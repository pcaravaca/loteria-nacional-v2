// Configuración local de las variantes de bingo. El backend es la autoridad en
// reglas, pagos y sorteo; este mapa solo alimenta la interfaz (tablero, letras,
// tope y tabla de pagos) antes de que el servidor confirme la ronda.

export interface BingoPago {
  nombre: string
  mult: number
}

export interface BingoVarianteCfg {
  codigo: string
  nombre: string
  descripcion: string
  bolillas: number
  filas: number
  columnas: number
  centroLibre: boolean
  tope: number
  letras: string[]
  pagos: BingoPago[]
}

export const BINGO_VARIANTES: Record<string, BingoVarianteCfg> = {
  bingo: {
    codigo: 'bingo',
    nombre: 'Bingo 75',
    descripcion: 'Cartón 5x5 de 75 bolillas: una línea x1, dos o más x3 y cartón lleno x25.',
    bolillas: 75,
    filas: 5,
    columnas: 5,
    centroLibre: true,
    tope: 42,
    letras: ['B', 'I', 'N', 'G', 'O'],
    pagos: [
      { nombre: 'Cartón lleno', mult: 25 },
      { nombre: 'Dos o más líneas', mult: 3 },
      { nombre: 'Una línea', mult: 1 },
    ],
  },
  bingo90: {
    codigo: 'bingo90',
    nombre: 'Bingo 90',
    descripcion: 'Ticket 3x9 de 90 bolillas: 1 línea x2, 2 líneas x5 y cartón lleno x40.',
    bolillas: 90,
    filas: 3,
    columnas: 9,
    centroLibre: false,
    tope: 60,
    letras: ['1', '2', '3', '4', '5', '6', '7', '8', '9'],
    pagos: [
      { nombre: 'Cartón lleno', mult: 40 },
      { nombre: 'Dos líneas', mult: 5 },
      { nombre: 'Una línea', mult: 2 },
    ],
  },
  'bingo-esquinas': {
    codigo: 'bingo-esquinas',
    nombre: 'Cuatro Esquinas',
    descripcion: 'Bingo 75: completá las cuatro esquinas (x9) o el cartón lleno (x25).',
    bolillas: 75,
    filas: 5,
    columnas: 5,
    centroLibre: true,
    tope: 42,
    letras: ['B', 'I', 'N', 'G', 'O'],
    pagos: [
      { nombre: 'Cartón lleno', mult: 25 },
      { nombre: 'Cuatro esquinas', mult: 9 },
    ],
  },
  'bingo-diamante': {
    codigo: 'bingo-diamante',
    nombre: 'Patrón Diamante',
    descripcion: 'Bingo 75: completá el diamante (x9) o el cartón lleno (x25).',
    bolillas: 75,
    filas: 5,
    columnas: 5,
    centroLibre: true,
    tope: 42,
    letras: ['B', 'I', 'N', 'G', 'O'],
    pagos: [
      { nombre: 'Cartón lleno', mult: 25 },
      { nombre: 'Diamante', mult: 9 },
    ],
  },
  'bingo-relampago': {
    codigo: 'bingo-relampago',
    nombre: 'Bingo Relámpago',
    descripcion: 'Bingo 75 con 20 cantadas: pago fijo por aciertos (6-8 x1, 9 x2, 10 x3, 11 x4, 12+ x5).',
    bolillas: 75,
    filas: 5,
    columnas: 5,
    centroLibre: true,
    tope: 20,
    letras: ['B', 'I', 'N', 'G', 'O'],
    pagos: [
      { nombre: '12 o más aciertos', mult: 5 },
      { nombre: '11 aciertos', mult: 4 },
      { nombre: '10 aciertos', mult: 3 },
      { nombre: '9 aciertos', mult: 2 },
      { nombre: '6 a 8 aciertos', mult: 1 },
    ],
  },
}

export function esBingo(codigo: string): boolean {
  return Object.prototype.hasOwnProperty.call(BINGO_VARIANTES, codigo)
}

export function bingoVariante(codigo: string): BingoVarianteCfg {
  return BINGO_VARIANTES[codigo] ?? BINGO_VARIANTES.bingo
}


// Jerga del bingo: cada bolilla cantada con su nombre tradicional (y un remate
// tico cuando aplica). El llamador la canta en voz alta y tambien se muestra.
const JERGA_BINGO: Record<number, string> = {
  2: 'los dos patitos',
  4: 'la cama, ¡el gato!',
  7: 'la muleta, el revólver',
  8: 'la señora gorda',
  9: '¡levántale el rabo a la burra y bebe!',
  11: 'las banderillas',
  12: 'la docena',
  13: 'la mala pata, ni te cases ni te embarques',
  15: 'la niña bonita',
  19: 'el correo para Cuba',
  22: 'los dos patitos en el lago',
  25: 'Navidad',
  27: 'la baraja, ¡pura vida!',
  33: 'la edad de Cristo',
  36: 'la mala pata, ¡aguás!',
  48: 'el borrego',
  55: 'los dos cinco, ¡tuanis!',
  63: 'la cebolla',
  70: 'el medio siglo',
  75: 'fin del juego',
  81: 'el matrimonio',
  88: 'las gordas',
  90: 'el abuelo',
}

export function jergaBolilla(n: number): string {
  return JERGA_BINGO[n] ?? ''
}

// Canto completo: la letra y el numero con su jerga.
export function cantoBolilla(cfg: BingoVarianteCfg, n: number): string {
  const base = etiquetaBolilla(cfg, n)
  const j = jergaBolilla(n)
  return j ? `${base}, ${j}` : base
}

// etiquetaBolilla deja el número con su letra (B-12, I-25…). En el ticket 90
// se usa la columna (1-12, 2-25…) porque no hay letras BINGO.
export function etiquetaBolilla(cfg: BingoVarianteCfg, n: number): string {
  if (n <= 0) return '—'
  if (cfg.codigo === 'bingo90') {
    const col = n <= 9 ? 1 : Math.min(9, Math.floor((n - 10) / 10) + 2)
    return `${col}-${n}`
  }
  const idx = Math.min(cfg.letras.length - 1, Math.floor((n - 1) / 15))
  return `${cfg.letras[idx]}-${n}`
}

// celdasGanadoras devuelve las casillas del cartón que formaron el patrón que
// pagó, para resaltarlas con .resalta-ganador. El relámpago no tiene patrón
// fijo: paga por aciertos y ya quedan marcadas.
export function celdasGanadoras(
  cfg: BingoVarianteCfg,
  grid: number[][],
  marcadas: boolean[][],
): Set<string> {
  const out = new Set<string>()
  const filas = grid.length
  if (!filas || !grid[0]) return out
  const cols = grid[0].length
  const cm = Math.floor(filas / 2)
  const cc = Math.floor(cols / 2)
  const libre = (r: number, c: number) => cfg.centroLibre && r === cm && c === cc
  const llena = (r: number, c: number) => libre(r, c) || Boolean(marcadas[r]?.[c])
  const suma = (lista: Array<[number, number]>) => lista.forEach(([r, c]) => out.add(`${r},${c}`))

  if (cfg.codigo === 'bingo-relampago') {
    // Paga por aciertos: no hay patrón fijo que resaltar.
    return out
  }

  if (cfg.codigo === 'bingo-esquinas') {
    if (llena(0, 0) && llena(0, cols - 1) && llena(filas - 1, 0) && llena(filas - 1, cols - 1)) {
      suma([
        [0, 0],
        [0, cols - 1],
        [filas - 1, 0],
        [filas - 1, cols - 1],
      ])
    }
    return out
  }

  if (cfg.codigo === 'bingo-diamante') {
    const diamante: Array<[number, number]> = [
      [cm, cc],
      [cm - 1, cc],
      [cm, cc - 1],
      [cm, cc + 1],
      [cm + 1, cc],
    ]
    if (diamante.every(([r, c]) => llena(r, c))) suma(diamante)
    return out
  }

  // Líneas completas: filas, columnas y (en el tablero cuadrado) diagonales.
  for (let r = 0; r < filas; r++) {
    const fila: Array<[number, number]> = []
    for (let c = 0; c < cols; c++) if (grid[r][c] !== 0) fila.push([r, c])
    if (fila.length && fila.every(([, c]) => llena(r, c))) suma(fila)
  }
  if (filas === cols) {
    for (let c = 0; c < cols; c++) {
      const col: Array<[number, number]> = []
      for (let r = 0; r < filas; r++) if (grid[r][c] !== 0) col.push([r, c])
      if (col.length && col.every(([r]) => llena(r, c))) suma(col)
    }
    const d1: Array<[number, number]> = []
    const d2: Array<[number, number]> = []
    for (let i = 0; i < filas; i++) {
      d1.push([i, i])
      d2.push([i, cols - 1 - i])
    }
    if (d1.every(([r, c]) => llena(r, c))) suma(d1)
    if (d2.every(([r, c]) => llena(r, c))) suma(d2)
  }
  return out
}
