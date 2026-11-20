
.brand-img { width: 40px; height: 40px; border-radius: 12px; object-fit: cover; display: block; }<script setup lang="ts">
import AppAndroidBanner from '@/components/AppAndroidBanner.vue'
import { useMarcaStore } from '@/stores/marca'

const marca = useMarcaStore()
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const email = ref('')
const password = ref('')
const error = ref('')

async function enviar() {
  error.value = ''
  try {
    await auth.login(email.value, password.value)
    const redirect = (route.query.redirect as string) || '/dashboard'
    router.push(redirect)
  } catch (e: any) {
    error.value = e.message
  }
}
</script>

<template>
  <div class="auth-layout">
    <aside class="auth-side">
      <div class="brand">
        <img v-if="marca.icono" class="brand-img" :src="marca.icono" alt="" />
        <div v-else class="brand-mark">LN</div>
        <div>
          <div class="brand-name">{{ marca.nombre }}</div>
          <div class="brand-sub">Costa Rica</div>
        </div>
      </div>
      <div>
        <h2>La suerte está a un número de distancia.</h2>
        <p>
          Lotería Nacional, Quinielas y más juegos de azar de Costa Rica.
          Premios automáticos, resultados verificables y saldo en tiempo real.
        </p>
      </div>
      <p class="hint">Juega con responsabilidad · +18</p>
    </aside>

    <div class="auth-form">
      <div class="auth-card">
        <AppAndroidBanner />
        <h1>Bienvenido de nuevo</h1>
        <p class="sub">Ingresa tus credenciales para continuar</p>

        <form @submit.prevent="enviar">
          <div class="field">
            <label for="email">Correo electrónico</label>
            <input id="email" v-model="email" class="input" type="email" placeholder="tu@correo.com" required />
          </div>
          <div class="field">
            <label for="password">Clave</label>
            <input id="password" v-model="password" class="input" type="password" placeholder="••••••••" required />
          </div>

          <p v-if="error" class="error-text">{{ error }}</p>

          <button class="btn btn-primary btn-block" style="margin-top: 14px" :disabled="auth.loading">
            {{ auth.loading ? 'Ingresando…' : 'Ingresar' }}
          </button>
        </form>

        <p class="auth-alt">
          ¿No tienes cuenta? <router-link to="/registro">Regístrate</router-link>
        </p>
        <p class="auth-alt" style="margin-top: 10px">
          <router-link to="/">← Volver al inicio</router-link>
        </p>

        <div class="card" style="margin-top: 22px; padding: 14px; font-size: 12px">
          <div class="hint" style="margin-bottom: 6px">Usuarios de demostración</div>
          <div class="texto-muted">cliente@loteria.test / cliente123</div>
          <div class="texto-muted">agente@loteria.test / agente123</div>
          <div class="texto-muted">admin@loteria.test / admin123</div>
        </div>
      </div>
    </div>
  </div>
</template>
