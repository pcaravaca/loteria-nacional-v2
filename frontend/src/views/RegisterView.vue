<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

const nombre = ref('')
const email = ref('')
const telefono = ref('')
const password = ref('')
const confirmar = ref('')
const error = ref('')

async function enviar() {
  error.value = ''
  if (password.value !== confirmar.value) {
    error.value = 'Las claves no coinciden'
    return
  }
  try {
    await auth.register(nombre.value, email.value, password.value, telefono.value)
    router.push('/dashboard')
  } catch (e: any) {
    error.value = e.message
  }
}
</script>

<template>
  <div class="auth-layout">
    <aside class="auth-side">
      <div class="brand">
        <div class="brand-mark">LN</div>
        <div>
          <div class="brand-name">Lotería Nacional</div>
          <div class="brand-sub">Costa Rica</div>
        </div>
      </div>
      <div>
        <h2>Crea tu cuenta en menos de un minuto.</h2>
        <p>Regístrate, recarga tu saldo y empieza a jugar tus loterías y quinielas favoritas.</p>
      </div>
      <p class="hint">Juega con responsabilidad · +18</p>
    </aside>

    <div class="auth-form">
      <div class="auth-card">
        <h1>Crear cuenta</h1>
        <p class="sub">Solo necesitamos unos datos para empezar</p>

        <form @submit.prevent="enviar">
          <div class="field">
            <label for="nombre">Nombre completo</label>
            <input id="nombre" v-model="nombre" class="input" placeholder="Juan Pérez" required />
          </div>
          <div class="field">
            <label for="email">Correo electrónico</label>
            <input id="email" v-model="email" class="input" type="email" placeholder="tu@correo.com" required />
          </div>
          <div class="field">
            <label for="telefono">Teléfono (opcional)</label>
            <input id="telefono" v-model="telefono" class="input" placeholder="8888 8888" />
          </div>
          <div class="field">
            <label for="password">Clave</label>
            <input id="password" v-model="password" class="input" type="password" placeholder="Mínimo 6 caracteres" required />
          </div>
          <div class="field">
            <label for="confirmar">Confirmar clave</label>
            <input id="confirmar" v-model="confirmar" class="input" type="password" placeholder="Repite tu clave" required />
          </div>

          <p v-if="error" class="error-text">{{ error }}</p>

          <button class="btn btn-primary btn-block" style="margin-top: 14px" :disabled="auth.loading">
            {{ auth.loading ? 'Creando cuenta…' : 'Crear cuenta' }}
          </button>
        </form>

        <p class="auth-alt">
          ¿Ya tienes cuenta? <router-link to="/login">Ingresa</router-link>
        </p>
      </div>
    </div>
  </div>
</template>
