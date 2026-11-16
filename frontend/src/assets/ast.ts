// Catalogo de juegos inspirado en AST Casino (Casino Diamond).
// Los numeros (rodillos, pesos, pagos y lineas) viven en el backend
// (internal/casino/data/ast_*.json); aca solo esta la identidad visual.

export type AstSlotInfo = {
  codigo: string
  nombre: string
  emoji: string[]
  color: string
}

export type AstPokerInfo = {
  codigo: string
  nombre: string
  color: string
}

export const astSlots: AstSlotInfo[] = [
  { codigo: 'spy', nombre: 'Spy', emoji: ['🕵️', '🎯', '🔫', '💣', '🛩️', '🚁', '🏆', '💰', '⭐', '👑'], color: '#1b2a4a' },
  { codigo: 'tuttifruti', nombre: 'TuttiFruti', emoji: ['🍒', '🍋', '🍊', '🍉', '🍇', '🍓', '🥝', '🍍', '🔔', '7️⃣'], color: '#3a1d3f' },
  { codigo: 'venice', nombre: 'Venice', emoji: ['🎭', '⛲', '🏛️', '🕯️', '🍇', '💍', '🎶', '🌹', '⭐', '👑'], color: '#2a1e3f' },
  { codigo: 'viking', nombre: 'Viking', emoji: ['🪓', '🛡️', '⚔️', '🏹', '🍺', '🐺', '⚓', '🔥', '⭐', '👑'], color: '#24303d' },
  { codigo: 'winplaceshow', nombre: 'Win Place Show', emoji: ['🏇', '🐎', '🏆', '🎖️', '🥈', '🥇', '🏟️', '🎺', '⭐', '👑'], color: '#33251a' },
  { codigo: 'winterfest', nombre: 'WinterFest', emoji: ['⛄', '❄️', '🎿', '🛷', '🎁', '🕯️', '🍪', '🔔', '⭐', '🎄'], color: '#1d3242' },
  { codigo: 'witchhouse', nombre: 'Witch House', emoji: ['🧙', '🦇', '🕸️', '🧹', '🔮', '💀', '🌙', '🍬', '⭐', '🕷️'], color: '#2c1d38' },
  { codigo: 'wonderland', nombre: 'Wonderland', emoji: ['🎩', '🐇', '🫖', '♠️', '♥️', '♦️', '♣️', '🍄', '⭐', '👑'], color: '#1e3a2a' },
  { codigo: 'worldcup', nombre: 'World Cup', emoji: ['⚽', '🥅', '🏟️', '🎽', '🏆', '🥇', '🎺', '⭐', '👑', '🍿'], color: '#1e3326' },
  { codigo: 'xmasmobile', nombre: 'Xmas Mobile', emoji: ['🎄', '🎁', '⛄', '🔔', '🕯️', '🍪', '🦌', '⭐', '👑', '❄️'], color: '#331f24' },
  { codigo: 'xmas', nombre: 'Xmas', emoji: ['🎄', '🎁', '⛄', '🔔', '🕯️', '🍪', '🦌', '⭐', '👑', '❄️'], color: '#2a2417' },
  { codigo: 'zoomerzang', nombre: 'Zoomer Zang', emoji: ['🤖', '👾', '🛸', '🚀', '🪐', '⚡', '💎', '⭐', '👑', '🔮'], color: '#201f3a' },
]

export const astPokers: AstPokerInfo[] = [
  { codigo: 'acesandfaces', nombre: 'Aces and Faces', color: '#2a1e3f' },
  { codigo: 'allamerican', nombre: 'All American', color: '#33251a' },
  { codigo: 'deucewild', nombre: 'Deuce Wild', color: '#1d3242' },
  { codigo: 'eightsandaces', nombre: 'Eights and Aces', color: '#24303d' },
  { codigo: 'jacksorbetter', nombre: 'Jacks or Better', color: '#1e3a2a' },
  { codigo: 'jokerpoker', nombre: 'Joker Poker', color: '#2c1d38' },
  { codigo: 'tensorbetter', nombre: 'Tens or Better', color: '#3a1d3f' },
  { codigo: 'twowaysroyal', nombre: 'Two Ways Royal', color: '#1b2a4a' },
]

function escapar(svg: string): string {
  return encodeURIComponent(svg).replace(/'/g, '%27').replace(/"/g, '%22')
}

// astTile genera el arte de cada juego como SVG (data URI), con el mismo marco
// oscuro y dorado del resto del casino.
export function astTile(codigo: string, nombre: string, ancho = 512, alto = 512): string {
  const slot = astSlots.find((s) => s.codigo === codigo)
  const poker = astPokers.find((p) => p.codigo === codigo)
  const base = slot?.color ?? poker?.color ?? '#1b2a4a'
  const emoji = slot ? slot.emoji[9] : '🎰'
  const secundario = slot ? slot.emoji[6] : '⭐'
  const iniciales = nombre.slice(0, 2).toUpperCase()
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${ancho}" height="${alto}" viewBox="0 0 ${ancho} ${alto}">
  <defs>
    <linearGradient id="f" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="${base}"/>
      <stop offset="1" stop-color="#0a0f14"/>
    </linearGradient>
    <radialGradient id="b" cx="0.5" cy="0.35" r="0.9">
      <stop offset="0" stop-color="#ffffff" stop-opacity="0.12"/>
      <stop offset="1" stop-color="#ffffff" stop-opacity="0"/>
    </radialGradient>
  </defs>
  <rect width="${ancho}" height="${alto}" rx="28" fill="url(#f)"/>
  <rect width="${ancho}" height="${alto}" rx="28" fill="url(#b)"/>
  <rect x="10" y="10" width="${ancho - 20}" height="${alto - 20}" rx="22" fill="none" stroke="#c9a24b" stroke-opacity="0.55" stroke-width="3"/>
  <text x="${ancho / 2}" y="${alto * 0.46}" font-size="${alto * 0.3}" text-anchor="middle" dominant-baseline="middle">${emoji}</text>
  <text x="${ancho / 2}" y="${alto * 0.74}" font-size="${alto * 0.16}" text-anchor="middle" dominant-baseline="middle">${secundario}</text>
  <text x="${ancho / 2}" y="${alto * 0.9}" font-family="Georgia, serif" font-size="${alto * 0.09}" letter-spacing="3" text-anchor="middle" fill="#e8d9a8">${iniciales}</text>
</svg>`
  return `data:image/svg+xml,${escapar(svg)}`
}

export function esAst(codigo: string): boolean {
  return astSlots.some((s) => s.codigo === codigo) || astPokers.some((p) => p.codigo === codigo)
}

export function astSimbolo(codigo: string, clave: string): string {
  const info = astSlots.find((s) => s.codigo === codigo)
  if (!info) return clave
  const indice = Number(clave.replace(/\D/g, ''))
  if (!Number.isInteger(indice) || indice < 0 || indice >= info.emoji.length) return '⭐'
  return info.emoji[indice]
}
