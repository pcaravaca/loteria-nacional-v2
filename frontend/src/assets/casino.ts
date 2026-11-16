// Assets del mini casino (generados, paleta oscura + esmeralda + dorado).
import dadosCard from './casino/cards/dados.webp'
import slotsCard from './casino/cards/slots.webp'
import ruletaCard from './casino/cards/ruleta.webp'
import blackjackCard from './casino/cards/blackjack.webp'
import raspaCard from './casino/cards/raspa.webp'
import kenoCard from './casino/cards/keno.webp'
import cascadaCard from './casino/cards/cascada.webp'
import hiloCard from './casino/cards/hilo.webp'
import altoCard from './casino/cards/alto.webp'
import fortunaCard from './casino/cards/fortuna.webp'
import monedaCard from './casino/cards/moneda.webp'
import suertudoCard from './casino/cards/suertudo.webp'
import parleyCard from './casino/cards/parley.webp'
import baccaratCard from './casino/cards/baccarat.webp'

import dadosHero from './casino/hero/dados.webp'
import slotsHero from './casino/hero/slots.webp'
import ruletaHero from './casino/hero/ruleta.webp'
import blackjackHero from './casino/hero/blackjack.webp'
import raspaHero from './casino/hero/raspa.webp'
import kenoHero from './casino/hero/keno.webp'
import cascadaHero from './casino/hero/cascada.webp'
import hiloHero from './casino/hero/hilo.webp'
import altoHero from './casino/hero/alto.webp'
import fortunaHero from './casino/hero/fortuna.webp'
import monedaHero from './casino/hero/moneda.webp'
import suertudoHero from './casino/hero/suertudo.webp'
import parleyHero from './casino/hero/parley.webp'
import baccaratHero from './casino/hero/baccarat.webp'

import cereza from './casino/symbols/cereza.webp'
import campana from './casino/symbols/campana.webp'
import siete from './casino/symbols/siete.webp'
import diamante from './casino/symbols/diamante.webp'
import herradura from './casino/symbols/herradura.webp'
import barra from './casino/symbols/barra.webp'

export type CodigoJuego =
  | 'dados'
  | 'slots'
  | 'ruleta'
  | 'raspa'
  | 'keno'
  | 'blackjack'
  | 'cascada'
  | 'hilo'
  | 'alto'
  | 'fortuna'
  | 'moneda'
  | 'suertudo'
  | 'parley'
  | 'baccarat'

export const cartas: Record<CodigoJuego, string> = {
  dados: dadosCard,
  slots: slotsCard,
  ruleta: ruletaCard,
  blackjack: blackjackCard,
  raspa: raspaCard,
  keno: kenoCard,
  cascada: cascadaCard,
  hilo: hiloCard,
  alto: altoCard,
  fortuna: fortunaCard,
  moneda: monedaCard,
  suertudo: suertudoCard,
  parley: parleyCard,
  baccarat: baccaratCard,
}

export const heroes: Record<CodigoJuego, string> = {
  dados: dadosHero,
  slots: slotsHero,
  ruleta: ruletaHero,
  blackjack: blackjackHero,
  raspa: raspaHero,
  keno: kenoHero,
  cascada: cascadaHero,
  hilo: hiloHero,
  alto: altoHero,
  fortuna: fortunaHero,
  moneda: monedaHero,
  suertudo: suertudoHero,
  parley: parleyHero,
  baccarat: baccaratHero,
}

export const simbolos: Record<string, string> = {
  cereza,
  campana,
  siete,
  diamante,
  herradura,
  barra,
}

// Orden de los simbolos en el rodillo (el backend usa el mismo orden).
export const ordenSimbolos = ['cereza', 'campana', 'herradura', 'barra', 'diamante', 'siete'] as const

// Paytable de la tragamonedas: tres iguales o los dos primeros.
export const paytable: { simbolos: string[]; multipl: number; etiqueta: string }[] = [
  { simbolos: ['siete', 'siete', 'siete'], multipl: 50, etiqueta: 'Tres sietes' },
  { simbolos: ['diamante', 'diamante', 'diamante'], multipl: 25, etiqueta: 'Tres diamantes' },
  { simbolos: ['campana', 'campana', 'campana'], multipl: 18, etiqueta: 'Tres campanas' },
  { simbolos: ['cereza', 'cereza', 'cereza'], multipl: 8, etiqueta: 'Tres cerezas' },
  { simbolos: ['herradura', 'herradura', 'herradura'], multipl: 5, etiqueta: 'Tres herraduras' },
  { simbolos: ['barra', 'barra', 'barra'], multipl: 2, etiqueta: 'Tres barras' },
  { simbolos: ['siete', 'siete'], multipl: 20, etiqueta: 'Dos sietes' },
  { simbolos: ['diamante', 'diamante'], multipl: 8, etiqueta: 'Dos diamantes' },
  { simbolos: ['campana', 'campana'], multipl: 3, etiqueta: 'Dos campanas' },
  { simbolos: ['herradura', 'herradura'], multipl: 5, etiqueta: 'Dos herraduras' },
  { simbolos: ['barra', 'barra'], multipl: 2, etiqueta: 'Dos barras' },
  { simbolos: ['cereza', 'cereza'], multipl: 1, etiqueta: 'Dos cerezas' },
]
