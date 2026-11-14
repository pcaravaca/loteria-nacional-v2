import { defineStore } from 'pinia'
import { api, clearToken, getToken, setToken } from '@/api/client'
import type { User } from '@/types'

interface AuthState {
  user: User | null
  token: string | null
  loading: boolean
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    user: null,
    token: getToken(),
    loading: false,
  }),
  getters: {
    isAutenticado: (state) => !!state.token,
    esAdmin: (state) => state.user?.rol === 'admin',
    esAgente: (state) => state.user?.rol === 'agente' || state.user?.rol === 'admin',
  },
  actions: {
    async login(email: string, password: string) {
      this.loading = true
      try {
        const { data } = await api.post('/auth/login', { email, password })
        this.token = data.token
        this.user = data.user
        setToken(data.token)
      } finally {
        this.loading = false
      }
    },
    async register(nombre: string, email: string, password: string, telefono: string) {
      this.loading = true
      try {
        await api.post('/auth/register', { nombre, email, password, telefono })
        await this.login(email, password)
      } finally {
        this.loading = false
      }
    },
    async cargar() {
      if (!this.token) return
      try {
        const { data } = await api.get<User>('/auth/me')
        this.user = data
      } catch {
        this.logout()
      }
    },
    logout() {
      this.user = null
      this.token = null
      clearToken()
    },
  },
})
