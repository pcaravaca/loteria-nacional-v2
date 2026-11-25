/**
 * Impresión de comprobantes térmicos.
 *
 * Dos caminos, un mismo HTML generado por el backend:
 *  - Sunmi V2: dentro de la app nativa el WebView inyecta `window.AndroidPOS`;
 *    el bridge renderiza el HTML y lo manda a la impresora integrada (58 mm).
 *  - PC con POS-80: se abre el HTML en una ventana y se llama a window.print().
 *
 * Sintaxis compatible con Chromium 62 (WebView de la Sunmi V2): sin `?.`, `??`
 * ni `.at()`. Vite ya compila con target chrome62.
 */

export interface PrintResult {
  ok: boolean
  code?: number
  message?: string
  error?: string
}

type PrintCallback = (id: string, result: PrintResult) => void

const ANCHO_KEY = 'loteria_ancho_papel'
const suscriptores: PrintCallback[] = []

function puente(): any {
  return (window as any).AndroidPOS
}

export function esImpresoraNativa(): boolean {
  const b = puente()
  return !!b && typeof b === 'object'
}

function despacharResultado(payload: string): void {
  let id = '0'
  let resultado: PrintResult = { ok: false, error: 'resultado de impresión inválido' }
  try {
    const dato = JSON.parse(payload)
    if (dato && dato.id !== undefined && dato.id !== null) id = String(dato.id)
    if (dato && dato.result !== undefined && dato.result !== null) resultado = dato.result as PrintResult
  } catch (e) {
    resultado = { ok: false, error: 'resultado de impresión inválido' }
  }
  for (let i = 0; i < suscriptores.length; i++) {
    try {
      suscriptores[i](id, resultado)
    } catch (e) {
      // un suscriptor roto no debe impedir que los demás reciban el resultado
    }
  }
}

/** Se llama una vez al arrancar la app, antes de imprimir. */
export function instalarPuenteImpresion(): void {
  if (!esImpresoraNativa()) return
  ;(window as any)._pmPrintDispatch = despacharResultado
  try {
    puente()._onPrintDone = despacharResultado
  } catch (e) {
    // algunos WebViews no permiten expandos sobre el objeto inyectado
  }
}

export function alTerminarImprimir(cb: PrintCallback): () => void {
  suscriptores.push(cb)
  return () => {
    const i = suscriptores.indexOf(cb)
    if (i >= 0) suscriptores.splice(i, 1)
  }
}

/** 58 mm es el papel de la Sunmi V2; 80 mm el de la POS-80 del mostrador. */
export function anchoPapelMM(): number {
  try {
    const v = Number(window.localStorage.getItem(ANCHO_KEY))
    if (v === 80) return 80
  } catch (e) {
    /* almacenamiento bloqueado */
  }
  return 58
}

export function fijarAnchoPapelMM(mm: number): void {
  try {
    window.localStorage.setItem(ANCHO_KEY, mm === 80 ? '80' : '58')
  } catch (e) {
    /* almacenamiento bloqueado */
  }
}

/**
 * Imprime el comprobante. Devuelve true si se encoló la impresión.
 * El resultado definitivo en Sunmi llega por `alTerminarImprimir`.
 */
export function imprimirTicket(html: string, anchoMM?: number): boolean {
  const ancho = anchoMM || anchoPapelMM()
  if (esImpresoraNativa()) {
    const payload = {
      id: 'ln-' + Date.now(),
      html: html,
      baseUrl: window.location.origin,
      widthMm: ancho,
    }
    try {
      puente().printHtml(JSON.stringify(payload))
      return true
    } catch (e) {
      return false
    }
  }
  const win = window.open('', '_blank', 'width=420,height=620')
  if (!win) return false
  win.document.write(html)
  win.document.write('<script>window.addEventListener("load",function(){window.print()})<\/script>')
  win.document.close()
  return true
}
