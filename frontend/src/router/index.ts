import { createRouter, createWebHashHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes = [
  { path: '/', name: 'inicio', component: () => import('@/views/LandingPage.vue'), meta: { publica: true } },
  { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue'), meta: { publica: true } },
  { path: '/registro', name: 'registro', component: () => import('@/views/RegisterView.vue'), meta: { publica: true } },
  { path: '/dashboard', name: 'dashboard', component: () => import('@/views/DashboardView.vue') },
  { path: '/loterias', name: 'loterias', component: () => import('@/views/LoteriasView.vue') },
  { path: '/quinielas', name: 'quinielas', component: () => import('@/views/QuinelasView.vue') },
  { path: '/mis-apuestas', name: 'mis-apuestas', component: () => import('@/views/MisApuestasView.vue') },
  { path: '/resultados', name: 'resultados', component: () => import('@/views/ResultadosView.vue') },
  { path: '/reportes', name: 'reportes', component: () => import('@/views/ReportesView.vue') },
  { path: '/casino', name: 'casino', component: () => import('@/views/CasinoView.vue') },
  { path: '/rifas', name: 'rifas', component: () => import('@/views/RifasView.vue') },
  { path: '/perfil', name: 'perfil', component: () => import('@/views/PerfilView.vue') },
  { path: '/admin', name: 'admin', component: () => import('@/views/AdminView.vue'), meta: { rol: 'admin' } },
  { path: '/agente', name: 'agente', component: () => import('@/views/AgenteView.vue'), meta: { rol: 'agente' } },
  { path: '/:pathMatch(.*)*', name: 'no-encontrado', component: () => import('@/views/NotFoundView.vue') },
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()

  if (auth.token && !auth.user) {
    await auth.cargar()
  }

  if (to.meta.publica) {
    if (auth.isAutenticado && (to.name === 'login' || to.name === 'registro')) {
      return { name: 'dashboard' }
    }
    return true
  }

  if (!auth.isAutenticado) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  const rol = to.meta.rol as string | undefined
  if (rol === 'admin' && !auth.esAdmin) {
    return { name: 'dashboard' }
  }
  if (rol === 'agente' && !auth.esAgente) {
    return { name: 'dashboard' }
  }

  return true
})
