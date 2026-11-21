// El producto es de Costa Rica: las fechas se muestran siempre en hora CR,
// sin importar la zona horaria del navegador (el sorteo no se mueve).
const ZONA_CR = 'America/Costa_Rica'

export function colones(valor: number): string {
  return '¢' + new Intl.NumberFormat('es-CR', { maximumFractionDigits: 0 }).format(valor || 0)
}

export function fechaCorta(iso: string): string {
  if (!iso) return '—'
  return new Intl.DateTimeFormat('es-CR', {
    day: '2-digit', month: 'short', timeZone: ZONA_CR,
  }).format(new Date(iso))
}

export function fechaHora(iso: string): string {
  if (!iso) return '—'
  return new Intl.DateTimeFormat('es-CR', {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
    timeZone: ZONA_CR,
  }).format(new Date(iso))
}

export const estadoApuesta: Record<string, { texto: string; clase: string }> = {
  pendiente: { texto: 'Pendiente', clase: 'badge-gold' },
  ganadora: { texto: 'Ganadora', clase: 'badge-ok' },
  perdedora: { texto: 'Perdedora', clase: 'badge-bad' },
  pagada: { texto: 'Pagada', clase: 'badge-info' },
  cancelada: { texto: 'Cancelada', clase: 'badge-info' },
}

export const estadoSorteo: Record<string, { texto: string; clase: string }> = {
  abierto: { texto: 'Abierto', clase: 'badge-ok' },
  cerrado: { texto: 'Cerrado', clase: 'badge-gold' },
  pagado: { texto: 'Pagado', clase: 'badge-info' },
}

export function tipoLabel(tipo: string): string {
  return tipo === 'quiniela' ? 'Quiniela' : 'Lotería'
}

export function pad(n: number, largo: number): string {
  return String(n).padStart(largo, '0')
}
