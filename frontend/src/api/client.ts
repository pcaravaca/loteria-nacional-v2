import axios, { AxiosError } from 'axios'

const TOKEN_KEY = 'loteria_token'
// Marca de que la pagina corrio dentro del WebView de la app Android nativa.
// Se guarda antes de limpiar el query, porque `getToken` lo elimina.
export const APP_KEY = 'loteria_app_nativa'

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
  timeout: 15000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error: AxiosError<{ error?: string }>) => {
    if (error.response?.status === 401 && localStorage.getItem(TOKEN_KEY)) {
      localStorage.removeItem(TOKEN_KEY)
      if (location.hash !== '#/login') {
        location.hash = '#/login'
      }
    }
    const message = error.response?.data?.error || error.message || 'Error de conexion'
    if (error.response?.status === 451) {
      return Promise.reject(new Error(message + ' (tu ubicacion no esta permitida para jugar)'))
    }
    return Promise.reject(new Error(message))
  },
)

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export function getToken() {
  // La app nativa abre la web con ?token=JWT&app=1 para heredar la sesion ya
  // iniciada y pedir el modo app (sin sidebar ni topbar propios).
  const params = new URLSearchParams(window.location.search)
  const porUrl = params.get('token')
  if (porUrl) {
    setToken(porUrl)
    if (params.get('app') === '1') sessionStorage.setItem(APP_KEY, '1')
    const limpia = window.location.pathname + window.location.hash
    window.history.replaceState(null, '', limpia)
  }
  return localStorage.getItem(TOKEN_KEY)
}
