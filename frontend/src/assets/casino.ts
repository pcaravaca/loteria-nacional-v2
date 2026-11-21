// Assets del mini casino (generados, paleta oscura + esmeralda + dorado).
import dadosCard from './casino/cards/dados.webp'
import slotsCard from './casino/cards/slots.webp'
import RuletaCard from './casino/cards/ruleta.webp'
import BlackjackCard from './casino/cards/blackjack.webp'
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
import bingoCard from './casino/cards/bingo.svg'
import gemasCard from './casino/cards/gemas.svg'
import DominóCard from './casino/cards/domino.svg'
import AjedrezCard from './casino/cards/ajedrez.svg'
import backgammonCard from './casino/cards/backgammon.svg'
import tetrisCard from './casino/cards/tetris.svg'

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
import bingoHero from './casino/hero/bingo.svg'
import gemasHero from './casino/hero/gemas.svg'
import dominoHero from './casino/hero/domino.svg'
import ajedrezHero from './casino/hero/ajedrez.svg'
import backgammonHero from './casino/hero/backgammon.svg'
import tetrisHero from './casino/hero/tetris.svg'

// Arte del catalogo AST Casino (ilustraciones generadas con la misma paleta).
import astTuttifruti from './casino/ast/tuttifruti.webp'
import astVenice from './casino/ast/venice.webp'
import astSpy from './casino/ast/spy.webp'
import astViking from './casino/ast/viking.webp'
import astWinplaceshow from './casino/ast/winplaceshow.webp'
import astWinterfest from './casino/ast/winterfest.webp'
import astWitchhouse from './casino/ast/witchhouse.webp'
import astWonderland from './casino/ast/wonderland.webp'
import astWorldcup from './casino/ast/worldcup.webp'
import astXmas from './casino/ast/xmas.webp'
import astXmasmobile from './casino/ast/xmasmobile.webp'
import astZoomerzang from './casino/ast/zoomerzang.webp'
import astAcesandfaces from './casino/ast/acesandfaces.webp'
import astAllamerican from './casino/ast/allamerican.webp'
import astDeucewild from './casino/ast/deucewild.webp'
import astEightsandaces from './casino/ast/eightsandaces.webp'
import astJacksorbetter from './casino/ast/jacksorbetter.webp'
import astJokerpoker from './casino/ast/jokerpoker.webp'
import astTensorbetter from './casino/ast/tensorbetter.webp'
import astTwowaysroyal from './casino/ast/twowaysroyal.webp'

const astArte: Record<string, string> = {
  spy: astSpy,
  tuttifruti: astTuttifruti,
  venice: astVenice,
  viking: astViking,
  winplaceshow: astWinplaceshow,
  winterfest: astWinterfest,
  witchhouse: astWitchhouse,
  wonderland: astWonderland,
  worldcup: astWorldcup,
  xmas: astXmas,
  xmasmobile: astXmasmobile,
  zoomerzang: astZoomerzang,
  acesandfaces: astAcesandfaces,
  allamerican: astAllamerican,
  deucewild: astDeucewild,
  eightsandaces: astEightsandaces,
  jacksorbetter: astJacksorbetter,
  jokerpoker: astJokerpoker,
  tensorbetter: astTensorbetter,
  twowaysroyal: astTwowaysroyal,
}

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
  | 'bingo'
  | 'gemas'
  | 'domino'
  | 'ajedrez'
  | 'backgammon'
  | 'tetris'
  // Catalogo AST Casino
  | 'spy'
  | 'tuttifruti'
  | 'venice'
  | 'viking'
  | 'winplaceshow'
  | 'winterfest'
  | 'witchhouse'
  | 'wonderland'
  | 'worldcup'
  | 'xmasmobile'
  | 'xmas'
  | 'zoomerzang'
  | 'acesandfaces'
  | 'allamerican'
  | 'deucewild'
  | 'eightsandaces'
  | 'jacksorbetter'
  | 'jokerpoker'
  | 'tensorbetter'
  | 'twowaysroyal'

export const cartas: Record<CodigoJuego, string> = {
  dados: dadosCard,
  slots: slotsCard,
  ruleta: RuletaCard,
  blackjack: BlackjackCard,
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
  bingo: bingoCard,
  gemas: gemasCard,
  domino: DominóCard,
  ajedrez: AjedrezCard,
  backgammon: backgammonCard,
  tetris: tetrisCard,
  ...astArte,
} as Record<CodigoJuego, string>

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
  bingo: bingoHero,
  gemas: gemasHero,
  domino: dominoHero,
  ajedrez: ajedrezHero,
  backgammon: backgammonHero,
  tetris: tetrisHero,
  ...astArte,
} as Record<CodigoJuego, string>

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
