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
  created_at: string
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
