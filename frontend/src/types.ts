export type Rol = 'cliente' | 'agente' | 'admin'
export type TipoJuego = 'loteria' | 'quiniela'
export type EstadoSorteo = 'abierto' | 'cerrado' | 'pagado'
export type EstadoApuesta = 'pendiente' | 'ganadora' | 'perdedora' | 'pagada'

export interface User {
  id: number
  nombre: string
  email: string
  rol: Rol
  telefono: string
  saldo: number
  bono: number
  tipo_documento?: string
  numero_documento?: string
  documento_verificado?: boolean
  fecha_nacimiento?: string | null
  autoexcluido_hasta?: string | null
  tiene_foto?: boolean
  tema_casino?: string
  created_at: string
}

// ---------------- Cumplimiento (juego responsable y fiscalizacion) ----------

export interface Limites {
  user_id: number
  deposito_diario: number
  deposito_semanal: number
  deposito_mensual: number
  gasto_diario: number
  gasto_semanal: number
  gasto_mensual: number
  actualizado_en: string
}

export interface AuditEntry {
  id: number
  actor_id: number
  actor_nombre: string
  accion: string
  entidad: string
  entidad_id: string
  detalle: string
  ip: string
  created_at: string
}

export const TIPOS_DOCUMENTO: Record<string, string> = {
  CEDULA_FISICA: 'Cédula física',
  CEDULA_JURIDICA: 'Cédula jurídica',
  DIMEX: 'DIMEX',
  PASAPORTE: 'Pasaporte',
  CONSUMIDOR_FINAL: 'Consumidor final (sin documento)',
}

export interface Loteria {
  id: number
  codigo: string
  nombre: string
  tipo: TipoJuego
  descripcion: string
  costo_numero: number
  longitud_num: number
  max_numeros: number
  multiplicador: number
  tabla_premios?: string
  activa: boolean
}

export interface Sorteo {
  id: number
  loteria_id: number
  titulo: string
  fecha: string
  estado: EstadoSorteo
  numeros_ganadores: number[]
  created_at: string
  loteria?: Loteria
}

export interface Apuesta {
  id: number
  user_id: number
  sorteo_id: number
  numeros: number[]
  monto: number
  estado: EstadoApuesta
  aciertos: number
  premio: number
  created_at: string
  sorteo?: Sorteo
}

export interface Dashboard {
  saldo: number
  bono: number
  pendiente: number
  premios: number
  total_apostado: number
  apuestas_activas: number
  ultimas_apuestas: Apuesta[]
}

export interface Movimiento {
  id: number
  user_id: number
  tipo: string
  monto: number
  descripcion: string
  created_at: string
}

export interface Resumen {
  por_loteria: {
    loteria_id: number
    nombre: string
    tipo: TipoJuego
    apostado: number
    premios: number
    apuestas: number
    sorteos: number
  }[]
  total_apostado: number
  total_premios: number
  total_apuestas: number
  loterias_activas: number
  usuarios: number
}

// ---------------- Mini casino ----------------

export interface CasinoGame {
  codigo: string
  nombre: string
  descripcion: string
  min_apuesta: number
  max_apuesta: number
  rtp: number
  max_multiplicador: number
  max_premio: number
  apuesta_max?: number
  orden: number
  activo: boolean
  limite_diario: number
}

export interface CasinoStats {
  juego: string
  jugadas: number
  apostado: number
  premios: number
  beneficio: number
}

export interface CasinoRound {
  id: number
  game: string
  monto: number
  multiplicador: number
  premio: number
  client_seed: string
  server_seed_hash: string
  nonce: number
  resultado: any
  created_at: string
}

export interface PlayResponse {
  saldo: number
  premio: number
  multiplicador: number
  resultado: any
  server_seed_hash: string
  nonce: number
  ronda_id: number
}

export interface SemillaResponse {
  hash: string
  nonce: number
  client_seed: string
}
