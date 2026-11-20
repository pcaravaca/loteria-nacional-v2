import { defineStore } from 'pinia'
import { api, clearToken, getToken, setToken } from '@/api/client'
import type { User } from '@/types'
import { esTemaValido, TEMA_DEFAULT, type TemaCasino } from '@/assets/casino/temas'

const TEMA_KEY = 'loteria_tema_casino'

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
    // Ambiente del casino: el del perfil si esta validado; si no, el guardado
    // en este dispositivo; si no hay nada, el estilo de la casa.
    temaCasino: (state): TemaCasino => {
      if (esTemaValido(state.user?.tema_casino)) return state.user!.tema_casino as TemaCasino
      const local = localStorage.getItem(TEMA_KEY)
      return esTemaValido(local) ? (local as TemaCasino) : TEMA_DEFAULT
    },
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
    // Guarda el ambiente elegido: primero en este dispositivo (funciona tambien
    // sin sesion y como respaldo) y despues en el perfil.
    async fijarTema(tema: TemaCasino) {
      localStorage.setItem(TEMA_KEY, tema)
      if (this.user) {
        this.user = { ...this.user, tema_casino: tema }
      }
      if (!this.token) return
      try {
        const { data } = await api.put<{ tema_casino: string }>('/auth/tema', { tema })
        if (this.user) this.user = { ...this.user, tema_casino: data.tema_casino }
      } catch {
        // El valor local queda aplicado; el perfil se sincroniza en el proximo login.
      }
    },
    logout() {
      this.user = null
      this.token = null
      clearToken()
    },
  },
})
