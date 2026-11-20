import { defineStore } from 'pinia'
import { api } from '@/api/client'

interface MarcaState {
  nombre: string
  logo: string
  icono: string
  cargada: boolean
}

// Marca de la plataforma: el mismo producto se vende con el nombre del negocio
// del cliente (backend: tabla ajustes, claves brand_*).
export const useMarcaStore = defineStore('marca', {
  state: (): MarcaState => ({
    nombre: 'Lotería Nacional',
    logo: '',
    icono: '',
    cargada: false,
  }),
  actions: {
    async cargar() {
      if (this.cargada) return
      try {
        const { data } = await api.get('/config')
        this.nombre = data.brand_nombre || this.nombre
        this.logo = data.brand_logo || ''
        this.icono = data.brand_icono || ''
      } catch {
        // Sin config queda el nombre por defecto.
      } finally {
        this.cargada = true
      }
    },
  },
})
