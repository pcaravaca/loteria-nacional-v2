<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import type { Limites } from '@/types'
import { TIPOS_DOCUMENTO } from '@/types'
import { colones, fechaHora } from '@/utils/formato'

const auth = useAuthStore()
const cargando = ref(true)

// Juego responsable e identificacion (KYC basico).
const autoexclusion = ref<{ activa: boolean; hasta: string | null }>({ activa: false, hasta: null })
const mesesAutoexclusion = ref(6)
const confirmoIrreversible = ref(false)
const tipoDocumento = ref('')
const numeroDocumento = ref('')
const fechaNacimiento = ref('')
const fotoseleccionada = ref<File | null>(null)
const guardandoIdentidad = ref(false)
const errorIdentidad = ref('')

// Limites voluntarios de deposito y gasto.
const limites = ref<Limites | null>(null)
const guardandoLimites = ref(false)
const mensajeLimites = ref('')

const autoexcluido = computed(() => auth.user?.autoexcluido_hasta != null)
const documentoVerificado = computed(() => !!auth.user?.documento_verificado)
const tieneFoto = computed(() => !!auth.user?.tiene_foto)

async function cargar() {
  try {
    const { data: estado } = await api.get<{ activa: boolean; hasta: string | null }>('/cuenta/autoexclusion')
    autoexclusion.value = estado
    tipoDocumento.value = auth.user?.tipo_documento ?? ''
    numeroDocumento.value = auth.user?.numero_documento ?? ''
    fechaNacimiento.value = auth.user?.fecha_nacimiento ?? ''
    const limitesData = await api.get<Limites>('/cuenta/limites')
    limites.value = limitesData.data
  } catch {
    // Si falla el estado de autoexclusion, el perfil sigue siendo usable.
  } finally {
    cargando.value = false
  }
}

onMounted(cargar)

async function guardarIdentificacion() {
  errorIdentidad.value = ''
  guardandoIdentidad.value = true
  try {
    await api.put('/cuenta/identificacion', {
      tipo: tipoDocumento.value,
      numero: numeroDocumento.value,
      fecha_nacimiento: fechaNacimiento.value || undefined,
    })
    await auth.cargar()
  } catch (e: any) {
    errorIdentidad.value = e.message
  } finally {
    guardandoIdentidad.value = false
  }
}

async function autoexcluirme() {
  errorIdentidad.value = ''
  try {
    await api.post('/cuenta/autoexclusion', { meses: mesesAutoexclusion.value })
    await auth.cargar()
    await cargar()
    confirmoIrreversible.value = false
  } catch (e: any) {
    errorIdentidad.value = e.message
  }
}

function elegirFoto(e: Event) {
  fotoseleccionada.value = (e.target as HTMLInputElement).files?.[0] ?? null
}

async function subirFoto() {
  if (!fotoseleccionada.value) return
  errorIdentidad.value = ''
  guardandoIdentidad.value = true
  try {
    const fd = new FormData()
    fd.append('archivo', fotoseleccionada.value)
    await api.post('/cuenta/identificacion/foto', fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    fotoseleccionada.value = null
    await auth.cargar()
  } catch (e: any) {
    errorIdentidad.value = e.message
  } finally {
    guardandoIdentidad.value = false
  }
}

async function guardarLimites() {
  if (!limites.value) return
  mensajeLimites.value = ''
  guardandoLimites.value = true
  try {
    const { data } = await api.put<Limites>('/cuenta/limites', {
      deposito_diario: limites.value.deposito_diario,
      deposito_semanal: limites.value.deposito_semanal,
      deposito_mensual: limites.value.deposito_mensual,
      gasto_diario: limites.value.gasto_diario,
      gasto_semanal: limites.value.gasto_semanal,
      gasto_mensual: limites.value.gasto_mensual,
    })
    limites.value = data
    mensajeLimites.value = 'Límites guardados'
  } catch (e: any) {
    mensajeLimites.value = e.message
  } finally {
    guardandoLimites.value = false
  }
}

const etiquetas: Record<string, { texto: string; clase: string }> = {
  recarga: { texto: 'Recarga', clase: 'badge-ok' },
  apuesta: { texto: 'Apuesta', clase: 'badge-gold' },
  premio: { texto: 'Premio', clase: 'badge-info' },
  bono: { texto: 'Bono', clase: 'badge-violet' },
  retiro: { texto: 'Retiro', clase: 'badge-bad' },
  ajuste: { texto: 'Ajuste', clase: '' },
}
</script>

<template>
  <div>
    <div class="page-head">
      <div class="eyebrow">Tu cuenta</div>
      <h1>Mi perfil</h1>
    </div>

    <div class="grid" style="align-items: start; max-width: 620px">
      <div class="card">
        <div style="display: flex; align-items: center; gap: 14px">
          <div class="avatar" style="width: 52px; height: 52px; font-size: 20px">
            {{ (auth.user?.nombre ?? 'U').slice(0, 1).toUpperCase() }}
          </div>
          <div>
            <h3 style="font-size: 19px">{{ auth.user?.nombre }}</h3>
            <p class="texto-muted" style="margin: 2px 0 0; font-size: 13px">{{ auth.user?.email }}</p>
            <span class="badge badge-ok" style="margin-top: 6px">{{ auth.user?.rol }}</span>
          </div>
        </div>

        <div style="display: flex; gap: 20px; margin-top: 20px; padding-top: 18px; border-top: 1px solid var(--border-soft)">
          <div>
            <div class="hint">Saldo disponible</div>
            <div class="mono texto-prim" style="font-size: 22px">{{ colones(auth.user?.saldo ?? 0) }}</div>
          </div>
          <div>
            <div class="hint">Bono</div>
            <div class="mono texto-gold" style="font-size: 22px">{{ colones(auth.user?.bono ?? 0) }}</div>
          </div>
          <div>
            <div class="hint">Teléfono</div>
            <div class="mono" style="font-size: 16px">{{ auth.user?.telefono || '—' }}</div>
          </div>
        </div>

        <p class="hint" style="margin-top: 16px">
          ¿Necesitas saldo? Contacta a un agente para recargar tu cuenta.
        </p>
        <p class="hint" style="margin-top: 6px">
          Tus movimientos de saldo y tus jugadas están en
          <router-link class="link" to="/reportes?tab=movimientos">Reportes</router-link>.
        </p>
      </div>

    </div>

    <div class="card" style="margin-top: 20px">
      <h3 style="font-size: 16px; margin-bottom: 6px">Identificación y juego responsable</h3>
      <p class="hint" style="margin-bottom: 14px">
        El operador puede exigir identificación verificada antes de permitir jugar, y la
        autoexclusión bloquea tus jugadas durante el periodo que elijas (sin reversa).
      </p>

      <div style="display: flex; gap: 8px; flex-wrap: wrap; align-items: center; margin-bottom: 12px">
        <select v-model="tipoDocumento" class="input" style="min-width: 220px">
          <option value="">Tipo de documento…</option>
          <option v-for="(nombre, codigo) in TIPOS_DOCUMENTO" :key="codigo" :value="codigo">
            {{ nombre }}
          </option>
        </select>
        <input
          v-model="numeroDocumento"
          class="input"
          style="min-width: 180px"
          placeholder="Número de documento"
          :disabled="tipoDocumento === 'CONSUMIDOR_FINAL'"
        />
        <button
          class="btn btn-primary"
          :disabled="guardandoIdentidad || !tipoDocumento"
          @click="guardarIdentificacion"
        >
          {{ guardandoIdentidad ? 'Guardando…' : 'Guardar identificación' }}
        </button>
        <span v-if="documentoVerificado" class="badge badge-ok">Identificación verificada</span>
        <span v-else-if="tipoDocumento" class="badge badge-gold">Pendiente de verificación</span>
      </div>
      <div style="display: flex; gap: 8px; flex-wrap: wrap; align-items: center; margin-bottom: 12px">
        <input v-model="fechaNacimiento" type="date" class="input" style="min-width: 160px" />
        <span class="hint" style="font-size: 12px">Fecha de nacimiento (requerida para verificar)</span>
      </div>
      <div style="display: flex; gap: 8px; flex-wrap: wrap; align-items: center; margin-bottom: 12px">
        <label class="hint" style="font-size: 12px; display: flex; gap: 6px; align-items: center">
          <input type="file" accept="image/png,image/jpeg" @change="elegirFoto" />
          Foto del documento (JPG o PNG, hasta 5 MB)
        </label>
        <button
          class="btn btn-ghost btn-sm"
          :disabled="guardandoIdentidad || !fotoseleccionada"
          @click="subirFoto"
        >
          Subir foto
        </button>
        <span v-if="tieneFoto" class="badge badge-ok">Documento subido</span>
      </div>
      <p class="hint" style="font-size: 11px">
        Si cambias tu documento, hay que verificarlo de nuevo.
      </p>

      <hr style="border: 0; border-top: 1px solid var(--border-soft); margin: 16px 0" />

      <template v-if="autoexcluido">
        <p class="badge badge-bad" style="margin-bottom: 8px">
          Cuenta autoexcluida
        </p>
        <p class="hint">
          No puedes jugar hasta el {{ fechaHora(auth.user?.autoexcluido_hasta as string) }}.
          La autoexclusión no tiene reversa.
        </p>
      </template>
      <template v-else>
        <div style="display: flex; gap: 8px; flex-wrap: wrap; align-items: center">
          <select v-model.number="mesesAutoexclusion" class="input" style="min-width: 150px">
            <option :value="6">6 meses</option>
            <option :value="12">12 meses</option>
            <option :value="24">24 meses</option>
          </select>
          <button
            class="btn btn-ghost"
            :disabled="!confirmoIrreversible"
            @click="autoexcluirme"
          >
            Autoexcluirme
          </button>
        </div>
        <label style="display: flex; gap: 8px; align-items: flex-start; margin-top: 10px">
          <input v-model="confirmoIrreversible" type="checkbox" />
          <span class="hint" style="font-size: 12px">
            Entiendo que la autoexclusión es irreversible durante el periodo elegido y que
            bloqueará mis jugadas (lotería y casino) hasta la fecha indicada.
          </span>
        </label>
      </template>

      <p v-if="errorIdentidad" class="error-text" style="margin-top: 12px">{{ errorIdentidad }}</p>
    </div>

    <div class="card" style="margin-top: 20px">
      <h3 style="font-size: 16px; margin-bottom: 6px">Límites de depósito y gasto</h3>
      <p class="hint" style="margin-bottom: 14px">
        Puedes autoimponerte topes para mantener el control. 0 significa sin límite. El gasto
        cuenta lo apostado en lotería y casino; los depósitos son las recargas que te hacen.
      </p>
      <div v-if="limites" class="form-grid">
        <label class="campo"><span>Depósito diario</span><input v-model.number="limites.deposito_diario" type="number" min="0" class="input" /></label>
        <label class="campo"><span>Depósito semanal</span><input v-model.number="limites.deposito_semanal" type="number" min="0" class="input" /></label>
        <label class="campo"><span>Depósito mensual</span><input v-model.number="limites.deposito_mensual" type="number" min="0" class="input" /></label>
        <label class="campo"><span>Gasto diario</span><input v-model.number="limites.gasto_diario" type="number" min="0" class="input" /></label>
        <label class="campo"><span>Gasto semanal</span><input v-model.number="limites.gasto_semanal" type="number" min="0" class="input" /></label>
        <label class="campo"><span>Gasto mensual</span><input v-model.number="limites.gasto_mensual" type="number" min="0" class="input" /></label>
      </div>
      <div style="display: flex; gap: 10px; align-items: center; margin-top: 14px">
        <button class="btn btn-primary" :disabled="guardandoLimites" @click="guardarLimites">
          {{ guardandoLimites ? 'Guardando…' : 'Guardar límites' }}
        </button>
        <span v-if="mensajeLimites" class="hint" style="color: var(--prim-2)">{{ mensajeLimites }}</span>
      </div>
    </div>
  </div>
</template>
