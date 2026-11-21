<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { colones } from '@/utils/formato'
import { useMarcaStore } from '@/stores/marca'
import AppAndroidBanner from '@/components/AppAndroidBanner.vue'
import { esDispositivoAndroid, esAppNativa } from '@/utils/dispositivo'
import AudioControl from '@/components/AudioControl.vue'
import TemaAppControl from '@/components/TemaAppControl.vue'

const marca = useMarcaStore()

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const abierto = ref(false)
const esAndroid = esDispositivoAndroid()
// Dentro de la app Android nativa ya estan el encabezado, el saldo y las
// pestanas: la web se despeja (sin sidebar ni topbar de escritorio).
const modoApp = esAppNativa()

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

const titulo = computed(() => titulos[route.name as string] ?? marca.nombre)

const nav = computed(() => {
  const items: { to: any; label: string; ico: string; query?: Record<string, unknown> }[] = [
    { to: '/dashboard', label: 'Panel', ico: '▤' },
    { to: '/loterias', label: 'Loterías', ico: '▦' },
    { to: '/quinielas', label: 'Quinelas', ico: '⚽' },
    { to: '/mis-apuestas', label: 'Mis apuestas', ico: '🎟' },
    { to: '/resultados', label: 'Resultados', ico: '🏆' },
    { to: '/rifas', label: 'Rifas', ico: '🎁' },
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
  // Al cerrar sesion se vuelve a la landing, no al login.
  router.push({ name: 'inicio' }).catch(() => {
    window.location.hash = '#/'
  })
  window.setTimeout(() => {
    if (!auth.isAutenticado && location.hash.startsWith('#/login')) window.location.hash = '#/'
  }, 250)
}
</script>

<template>
  <div class="app-shell" :class="{ 'modo-app': modoApp }">
    <aside v-if="!modoApp" class="sidebar" :class="{ abierta: abierto }">
      <div class="brand">
        <img v-if="marca.icono" class="brand-img" :src="marca.icono" alt="" />
        <div v-else class="brand-mark">LN</div>
        <div>
          <div class="brand-name">{{ marca.nombre }}</div>
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

      <a v-if="esAndroid" class="app-cta" href="/descargar/">
        <span class="app-cta-ico">▣</span>
        <span><b>App Android</b><small>Descargala gratis</small></span>
      </a>

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
        <button v-if="!modoApp" class="btn btn-ghost btn-sm menu-btn" @click="abierto = !abierto">☰</button>
        <h2 v-if="!modoApp">{{ titulo }}</h2>
        <div class="spacer"></div>
        <AudioControl />
        <TemaAppControl />
        <div v-if="!modoApp" class="saldo-chip">
          <span class="etq">Saldo</span>
          <span class="valor">{{ colones(auth.user?.saldo ?? 0) }}</span>
        </div>
      </header>

      <main class="content">
        <AppAndroidBanner />
        <slot />
      </main>
    </div>
  </div>
</template>

<style scoped>
.brand-img { width: 36px; height: 36px; border-radius: 11px; object-fit: cover; display: block; }

.app-cta {
  display: flex; align-items: center; gap: 10px; margin-top: 14px; padding: 9px 11px;
  border-radius: 12px; text-decoration: none;
  background: linear-gradient(180deg, rgba(232,206,122,.16), rgba(232,206,122,.05));
  border: 1px solid rgba(232,206,122,.3);
}
.app-cta-ico {
  width: 30px; height: 30px; border-radius: 9px; display: grid; place-items: center;
  color: #2b1d05; font-size: 15px; background: linear-gradient(180deg,#f0d98a,#c9a227);
}
.app-cta b { display: block; color: #eef6f1; font-size: 12.5px; }
.app-cta small { color: #9db8ac; font-size: 11px; }
</style>
