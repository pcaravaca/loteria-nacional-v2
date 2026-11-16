<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api/client'
import type { Loteria, Sorteo } from '@/types'
import { colones } from '@/utils/formato'
import SorteoCard from '@/components/SorteoCard.vue'
import heroArt from '@/assets/hero-landing.webp'
import { cartas } from '@/assets/casino'

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

const casinoJuegos = [
  { nombre: 'Dados del Sur', img: cartas.dados },
  { nombre: 'Fortuna CR', img: cartas.slots },
  { nombre: 'Rueda Fortuna', img: cartas.ruleta },
  { nombre: 'Raspa y Gana', img: cartas.raspa },
  { nombre: 'Keno CR', img: cartas.keno },
  { nombre: '21 Blackjack', img: cartas.blackjack },
  { nombre: 'Cascada Fortuna', img: cartas.cascada },
  { nombre: 'Mayor o Menor', img: cartas.hilo },
  { nombre: 'Bajo o Sobre', img: cartas.alto },
  { nombre: 'Fortuna Rápida', img: cartas.fortuna },
]
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
      <div class="wrap landing-grid">
        <div>
          <span class="badge badge-ok" style="margin-bottom: 18px">+ Juego responsable 18+</span>
          <h1>La suerte está<br /><span class="acento">a un número de distancia.</span></h1>
          <p>
            Juega Lotería Nacional, Nuevos Tiempos, Quinielas, Chaves y el mini casino desde
            cualquier dispositivo. Apuestas desde {{ colones(500) }}, resultados en tiempo real
            y premios automáticos.
          </p>
          <div class="hero-cta" style="justify-content: flex-start">
            <router-link to="/registro" class="btn btn-primary" style="padding: 13px 24px">
              Empezar a jugar
            </router-link>
            <router-link to="/login" class="btn btn-ghost" style="padding: 13px 24px">
              Ya tengo cuenta
            </router-link>
          </div>
          <div class="hero-stats" style="justify-content: flex-start">
            <div class="item">
              <div class="n">{{ loterias.length + casinoJuegos.length || 8 }}</div>
              <div class="l">Juegos activos</div>
            </div>
            <div class="item">
              <div class="n">x75</div>
              <div class="l">Premio por número</div>
            </div>
            <div class="item">
              <div class="n">100%</div>
              <div class="l">Pago automático</div>
            </div>
          </div>
        </div>
        <div class="landing-art">
          <img :src="heroArt" alt="Bolas de lotería en tonos esmeralda y dorado" />
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

    <section class="wrap" style="padding-top: 40px">
      <div class="page-head">
        <div class="eyebrow">Catálogo</div>
        <h1 style="font-size: 30px">Nuestros juegos</h1>
        <p>
          Los clásicos costarricenses. Cada juego define su apuesta mínima, multiplicador y
          cantidad de números.
        </p>
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

    <section class="wrap" style="padding-top: 20px; padding-bottom: 24px">
      <div class="page-head">
        <div class="eyebrow">Mini casino</div>
        <h1 style="font-size: 30px">Diez juegos, un solo saldo</h1>
        <p>Provably fair: podés verificar cada ronda con la semilla del servidor.</p>
      </div>
      <div class="landing-strip">
        <router-link
          v-for="j in casinoJuegos"
          :key="j.nombre"
          to="/casino"
          class="landing-mini card"
          :title="j.nombre"
        >
          <img :src="j.img" :alt="j.nombre" loading="lazy" />
          <div class="t">{{ j.nombre }}</div>
        </router-link>
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

<style scoped>
/* Hero a dos columnas: texto a la izquierda, arte generado a la derecha. */
.hero {
  text-align: left;
  padding: 0;
}
.landing-grid {
  display: grid;
  grid-template-columns: 1.05fr 0.95fr;
  gap: 44px;
  align-items: center;
  padding: 64px 26px 56px;
}
.landing-art {
  position: relative;
}
.landing-art img {
  width: 100%;
  height: 430px;
  object-fit: cover;
  border-radius: 18px;
  border: 1px solid var(--border);
  box-shadow: 0 34px 90px rgba(0, 0, 0, 0.6), 0 0 70px rgba(16, 185, 129, 0.14);
}
.landing-art::after {
  content: '';
  position: absolute;
  inset: auto 0 0 0;
  height: 110px;
  border-radius: 0 0 18px 18px;
  background: linear-gradient(180deg, transparent, rgba(8, 11, 16, 0.55));
  pointer-events: none;
}

/* Tira del mini casino. */
.landing-strip {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
}
.landing-mini {
  overflow: hidden;
  padding: 0;
  transition: transform 0.16s, border-color 0.16s, box-shadow 0.16s;
}
.landing-mini:hover {
  transform: translateY(-3px);
  border-color: var(--border);
  box-shadow: var(--shadow);
}
.landing-mini img {
  width: 100%;
  height: 96px;
  object-fit: cover;
  display: block;
}
.landing-mini .t {
  padding: 10px 12px;
  font-size: 13px;
  font-weight: 600;
}

@media (max-width: 980px) {
  .landing-grid {
    grid-template-columns: 1fr;
    gap: 28px;
    padding: 44px 26px 40px;
  }
  .landing-art img {
    height: 320px;
  }
  .landing-strip {
    grid-template-columns: repeat(3, 1fr);
  }
}
</style>
