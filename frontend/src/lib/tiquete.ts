import { api } from '@/api/client'
import { anchoPapelMM, imprimirTicket } from './impresion'

/**
 * Pide al backend el comprobante y lo manda a la impresora.
 * En la Sunmi V2 lo recibe el puente AndroidPOS; en el PC abre la ventana de
 * impresión del navegador (POS-80).
 */
export async function imprimirTiqueteApuesta(id: number | string): Promise<void> {
  const ancho = anchoPapelMM()
  const { data } = await api.get<string>(`/apuestas/${id}/ticket?ancho=${ancho}`, {
    responseType: 'text',
  })
  const html = typeof data === 'string' ? data : String(data || '')
  if (!html.includes('<html')) {
    throw new Error('No se pudo generar el comprobante')
  }
  if (!imprimirTicket(html, ancho)) {
    throw new Error('El navegador bloqueó la ventana de impresión')
  }
}
