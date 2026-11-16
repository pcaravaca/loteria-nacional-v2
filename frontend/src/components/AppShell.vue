<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { colones } from '@/utils/formato'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const abierto = ref(false)

const titulos: Record<string, string> = {
  dashboard: 'Panel de control',
  loterias: 'Loterías',
  quinielas: 'Quinelas',
  'mis-apuestas': 'Mis apuestas',
  resultados: 'Resultados',
  reportes: 'Reportes',
  casino: 'Casino',
  perfil: 'Mi perfil',
  admin: 'Administración',
  agente: 'Panel de agente',
}

const titulo = computed(() => titulos[route.name as string] ?? 'Lotería Nacional CR')

const nav = computed(() => {
  const items: { to: any; label: string; ico: string; query?: Record<string, unknown> }[] = [
    { to: '/dashboard', label: 'Panel', ico: '▤' },
    { to: '/loterias', label: 'Loterías', ico: '▦' },
    { to: '/quinielas', label: 'Quinelas', ico: '⚽' },
    { to: '/mis-apuestas', label: 'Mis apuestas', ico: '🎟' },
    { to: '/resultados', label: 'Resultados', ico: '🏆' },
    { to: '/reportes', label: 'Reportes', ico: '📊' },
  ]
  // El admin administra el casino desde su panel; los demas juegan en el lobby.
  if (auth.esAdmin) {
    items.push({ to: { path: '/admin', query: { tab: 'casino' } }, label: 'Casino', ico: '🎲' })
  } else {
    items.push({ to: '/casino', label: 'Casino', ico: '🎰' })
  }
  if (auth.esAgente) {
    items.push({ to: '/agente', label: 'Agente', ico: '🧾' })
  }
  if (auth.esAdmin) {
    items.push({ to: '/admin', label: 'Admin', ico: '⚙' })
  }
  items.push({ to: '/perfil', label: 'Mi perfil', ico: '👤' })
  return items
})

function salir() {
  auth.logout()
  router.push({ name: 'inicio' })
}
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar" :class="{ abierta: abierto }">
      <div class="brand">
        <div class="brand-mark">LN</div>
        <div>
          <div class="brand-name">Lotería Nacional</div>
          <div class="brand-sub">Costa Rica</div>
        </div>
      </div>

      <nav class="nav">
        <div class="nav-label">Juegos</div>
        <router-link
          v-for="item in nav"
          :key="item.to"
          :to="item.to"
          class="nav-link"
          @click="abierto = false"
        >
          <span class="nav-ico">{{ item.ico }}</span>
          {{ item.label }}
        </router-link>
      </nav>

      <div class="sidebar-foot">
        <div class="user-chip">
          <div class="avatar">{{ (auth.user?.nombre ?? 'U').slice(0, 1).toUpperCase() }}</div>
          <div>
            <div class="nombre">{{ auth.user?.nombre ?? 'Usuario' }}</div>
            <div class="rol">{{ auth.user?.rol }}</div>
          </div>
        </div>
        <button class="btn btn-ghost btn-sm" @click="salir">Cerrar sesión</button>
      </div>
    </aside>

    <div class="main">
      <header class="topbar">
        <button class="btn btn-ghost btn-sm menu-btn" @click="abierto = !abierto">☰</button>
        <h2>{{ titulo }}</h2>
        <div class="spacer"></div>
        <div class="saldo-chip">
          <span class="etq">Saldo</span>
          <span class="valor">{{ colones(auth.user?.saldo ?? 0) }}</span>
        </div>
      </header>

      <main class="content">
        <slot />
      </main>
    </div>
  </div>
</template>
