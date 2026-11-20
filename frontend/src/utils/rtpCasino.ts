// RTP declarado por juego. Los valores habiles y casuales salen de la
// simulacion sobre el algoritmo de cada juego y se cargan en
// backend/internal/database/seed.go (SeedCasino, campo rtp_casual). Mantener
// sincronizado con el seed: el admin y el catalogo lo leen de ahi.
//
// - rtp: retorno jugando bien (la jugada hábil que simula el backend).
// - rtp_casual: retorno jugando al azar, sin estrategia. Solo aplica a los
//   juegos con decisiones; el resto queda con 0 ("no aplica").

export interface RtpJuego {
  rtp: number
  rtp_casual: number
}

export const rtpJuegos: Record<string, RtpJuego> = {
  domino: { rtp: 89.43, rtp_casual: 79.08 },
  ajedrez: { rtp: 86.59, rtp_casual: 86.59 },
  backgammon: { rtp: 89.21, rtp_casual: 19.17 },
  tetris: { rtp: 82.95, rtp_casual: 4.51 },
}

// rtpDe devuelve el par de RTP declarado de un juego, o undefined si el juego
// no tiene doble declaracion (azar puro).
export function rtpDe(codigo: string): RtpJuego | undefined {
  return rtpJuegos[codigo]
}

// rtpTexto formatea un RTP con dos decimales, como se declara en el seed.
export function rtpTexto(valor: number): string {
  return `${valor.toFixed(2)}%`
}
