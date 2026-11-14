<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api/client'
import type { Loteria, Sorteo } from '@/types'
import { colones } from '@/utils/formato'
import SorteoCard from '@/components/SorteoCard.vue'

const loterias = ref<Loteria[]>([])
const sorteos = ref<Sorteo[]>([])
const cargando = ref(true)

onMounted(async () => {
  try {
    const [l, s] = await Promise.all([
      api.get<Loteria[]>('/loterias'),
      api.get<Sorteo[]>('/sorteos/proximos'),
    ])
    loterias.value = l.data
    sorteos.value = s.data
  } catch {
    /* la landing debe cargar aunque la API falle */
  } finally {
    cargando.value = false
  }
})
</script>

<template>
  <div>
    <nav class="nav-publica">
      <div class="brand">
        <div class="brand-mark">LN</div>
        <div>
          <div class="brand-name">Lotería Nacional</div>
          <div class="brand-sub">Costa Rica</div>
        </div>
      </div>
      <div class="acciones">
        <router-link to="/login" class="btn btn-ghost btn-sm">Ingresar</router-link>
        <router-link to="/registro" class="btn btn-primary btn-sm">Crear cuenta</router-link>
      </div>
    </nav>

    <section class="hero">
      <div class="wrap">
        <span class="badge badge-ok" style="margin-bottom: 18px">+ Juego responsable 18+</span>
        <h1>
          La suerte está<br />
          <span class="acento">a un número de distancia.</span>
        </h1>
        <p>
          Juega Lotería Nacional, Quinielas y más desde cualquier dispositivo.
          Apuestas desde {{ colones(500) }}, resultados en tiempo real y premios automáticos.
        </p>
        <div class="hero-cta">
          <router-link to="/registro" class="btn btn-primary" style="padding: 13px 24px">
            Empezar a jugar
          </router-link>
          <router-link to="/login" class="btn btn-ghost" style="padding: 13px 24px">
            Ya tengo cuenta
          </router-link>
        </div>

        <div class="hero-stats">
          <div class="item">
            <div class="n">{{ loterias.length || 5 }}</div>
            <div class="l">Juegos activos</div>
          </div>
          <div class="item">
            <div class="n">x50</div>
            <div class="l">Premio máximo</div>
          </div>
          <div class="item">
            <div class="n">100%</div>
            <div class="l">Pago automático</div>
          </div>
        </div>
      </div>
    </section>

    <section class="wrap" style="padding-top: 48px">
      <div class="page-head">
        <div class="eyebrow">Próximos sorteos</div>
        <h1 style="font-size: 30px">Juega ahora</h1>
        <p>Elige un sorteo abierto, marca tus números y listo. El premio se acredita solo al cerrar.</p>
      </div>

      <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando sorteos…</div>
      <div v-else-if="sorteos.length" class="grid grid-3">
        <SorteoCard v-for="s in sorteos" :key="s.id" :sorteo="s" @jugar="() => $router.push('/loterias')" />
      </div>
      <div v-else class="card empty">
        <div class="ico">🎟</div>
        <h3>No hay sorteos abiertos</h3>
        <p>Vuelve pronto: publicamos nuevos sorteos cada semana.</p>
      </div>
    </section>

    <section class="wrap" style="padding-top: 40px; padding-bottom: 20px">
      <div class="page-head">
        <div class="eyebrow">Catálogo</div>
        <h1 style="font-size: 30px">Nuestros juegos</h1>
      </div>
      <div class="grid grid-3">
        <div v-for="l in loterias" :key="l.id" class="card card-hover">
          <span class="badge" :class="l.tipo === 'quiniela' ? 'badge-violet' : 'badge-ok'">
            {{ l.tipo === 'quiniela' ? 'Quiniela' : 'Lotería' }}
          </span>
          <h3 style="font-size: 18px; margin-top: 10px">{{ l.nombre }}</h3>
          <p class="texto-muted" style="font-size: 13px; margin: 8px 0 0">{{ l.descripcion }}</p>
          <div style="margin-top: 14px; display: flex; justify-content: space-between; align-items: center">
            <span class="mono texto-gold">{{ colones(l.costo_numero) }}<span class="hint"> / número</span></span>
            <router-link to="/registro" class="btn btn-ghost btn-sm">Jugar</router-link>
          </div>
        </div>
      </div>
    </section>

    <footer class="footer">
      <div class="brand" style="justify-content: center; margin-bottom: 10px">
        <div class="brand-mark">LN</div>
        <div class="brand-name">Lotería Nacional CR</div>
      </div>
      <p>Juega con responsabilidad. +18. Aserrí, San José, Costa Rica.</p>
      <p style="margin-top: 6px; font-size: 12px">Plataforma de demostración — no es la lotería oficial de Costa Rica.</p>
    </footer>
  </div>
</template>
