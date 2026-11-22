import type { CasinoGame } from '@/types'
import { astSlots, astPokers } from '@/assets/ast'

export interface GrupoCasino {
  id: string
  titulo: string
  descripcion: string
  codigos: string[]
}

// Orden del lobby: primero lo de mesa (lo más "de casino"), después los
// clásicos, los rápidos y al final el catálogo AST (slots y video póker).
export const GRUPOS_CASINO: GrupoCasino[] = [
  {
    id: 'bingo',
    titulo: 'Bingo',
    descripcion: 'Cantá bolillas, armá línea y buscá el cartón lleno.',
    codigos: ['bingo', 'bingo90', 'bingo-esquinas', 'bingo-diamante', 'bingo-relampago'],
  },
  {
    id: 'mesa',
    titulo: 'Mesas',
    descripcion: 'Blackjack, Ruleta y baccará contra la casa.',
    codigos: ['blackjack', 'ruleta', 'baccarat'],
  },
  {
    id: 'mesa-nueva',
    titulo: 'Dominó y mesa',
    descripcion: 'Dominó, Ajedrez relámpago, backgammon y tetris por líneas.',
    codigos: ['domino', 'ajedrez', 'backgammon', 'tetris'],
  },
  {
    id: 'pinball',
    titulo: 'Pinball',
    descripcion: 'Tres mesas de bumpers y troneras: neón, clásica y plinko.',
    codigos: ['pinball-neon', 'pinball-clasico', 'plinko-premium'],
  },
  {
    id: 'clasicos',
    titulo: 'Clásicos',
    descripcion: 'Los juegos de siempre, con pago inmediato.',
    codigos: ['slots', 'dados', 'raspa', 'keno', 'gemas'],
  },
  {
    id: 'rapidos',
    titulo: 'Juegos rápidos',
    descripcion: 'Un toque, un sorteo y cobrás al instante.',
    codigos: ['cascada', 'hilo', 'alto', 'fortuna', 'moneda', 'suertudo', 'parley'],
  },
  {
    id: 'video-poker',
    titulo: 'Video póker',
    descripcion: 'Ocho variantes con tabla de pagos propia.',
    codigos: astPokers.map((p) => p.codigo),
  },
  {
    id: 'slots-ast',
    titulo: 'Slots',
    descripcion: 'Doce máquinas con líneas y multiplicadores.',
    codigos: astSlots.map((s) => s.codigo),
  },
]

export interface GrupoConJuegos extends GrupoCasino {
  juegos: CasinoGame[]
}

// agrupar deja los juegos en el orden de los grupos y, dentro de cada grupo,
// en el orden declarado. Los códigos sin grupo known van al final.
export function agruparJuegos(juegos: CasinoGame[]): GrupoConJuegos[] {
  const porCodigo = new Map(juegos.map((j) => [j.codigo, j]))
  const usados = new Set<string>()
  const out: GrupoConJuegos[] = []

  for (const g of GRUPOS_CASINO) {
    const lista = g.codigos
      .map((c) => porCodigo.get(c))
      .filter((j): j is CasinoGame => !!j)
    if (lista.length === 0) continue
    lista.forEach((j) => usados.add(j.codigo))
    out.push({ ...g, juegos: lista })
  }

  const sueltos = juegos.filter((j) => !usados.has(j.codigo))
  if (sueltos.length > 0) {
    out.push({
      id: 'otros',
      titulo: 'Otros juegos',
      descripcion: '',
      codigos: sueltos.map((j) => j.codigo),
      juegos: sueltos,
    })
  }
  return out
}
