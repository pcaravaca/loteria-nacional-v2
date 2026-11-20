<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { api } from '@/api/client'
import type { AccesoContenido, AmbitoAcceso, CasinoGame, GrupoJugadores, Loteria, TipoContenido, User } from '@/types'
import Modal from '@/components/Modal.vue'

interface RifaAdmin {
  id: number
  titulo: string
  estado: string
}

const cargando = ref(true)
const error = ref('')
const mensaje = ref('')

const grupos = ref<GrupoJugadores[]>([])
const accesos = ref<AccesoContenido[]>([])
const usuarios = ref<User[]>([])
const loterias = ref<Loteria[]>([])
const juegos = ref<CasinoGame[]>([])
const rifas = ref<RifaAdmin[]>([])

async function cargar() {
  cargando.value = true
  error.value = ''
  try {
    const [g, a, u, l, j, r] = await Promise.all([
      api.get<GrupoJugadores[]>('/admin/grupos'),
      api.get<AccesoContenido[]>('/admin/accesos'),
      api.get<User[]>('/admin/usuarios'),
      api.get<Loteria[]>('/loterias', { params: { todas: 1 } }),
      api.get<CasinoGame[]>('/casino/games', { params: { todas: 1 } }),
      api.get<RifaAdmin[]>('/admin/rifas'),
    ])
    grupos.value = g.data
    accesos.value = a.data
    usuarios.value = u.data
    loterias.value = l.data
    juegos.value = j.data ?? []
    rifas.value = r.data ?? []
  } catch (e: any) {
    error.value = e.response?.data?.error ?? e.message
  } finally {
    cargando.value = false
  }
}

onMounted(cargar)

// ------------------------------------------------------------ paquetes

const grupoModal = reactive({ abierto: false, id: 0, nombre: '', descripcion: '' })
const guardandoGrupo = ref(false)

function nuevoGrupo() {
  grupoModal.abierto = true
  grupoModal.id = 0
  grupoModal.nombre = ''
  grupoModal.descripcion = ''
}

function editarGrupo(g: GrupoJugadores) {
  grupoModal.abierto = true
  grupoModal.id = g.id
  grupoModal.nombre = g.nombre
  grupoModal.descripcion = g.descripcion ?? ''
}

async function guardarGrupo() {
  if (!grupoModal.nombre.trim()) {
    error.value = 'El paquete necesita un nombre'
    return
  }
  guardandoGrupo.value = true
  error.value = ''
  try {
    const payload = { nombre: grupoModal.nombre.trim(), descripcion: grupoModal.descripcion }
    if (grupoModal.id) {
      const { data } = await api.put<GrupoJugadores>(`/admin/grupos/${grupoModal.id}`, payload)
      const i = grupos.value.findIndex((g) => g.id === data.id)
      if (i >= 0) grupos.value[i] = { ...grupos.value[i], ...data }
      mensaje.value = 'Paquete actualizado'
    } else {
      const { data } = await api.post<GrupoJugadores>('/admin/grupos', payload)
      grupos.value.push({ ...data, miembros: 0 })
      mensaje.value = 'Paquete creado'
    }
    grupoModal.abierto = false
  } catch (e: any) {
    error.value = e.response?.data?.error ?? e.message
  } finally {
    guardandoGrupo.value = false
  }
}

async function eliminarGrupo(g: GrupoJugadores) {
  if (!confirm(`¿Eliminar el paquete "${g.nombre}"? Quedarán sin efecto los accesos que tenían por paquete.`)) return
  error.value = ''
  try {
    await api.delete(`/admin/grupos/${g.id}`)
    grupos.value = grupos.value.filter((x) => x.id !== g.id)
    mensaje.value = 'Paquete eliminado'
  } catch (e: any) {
    error.value = e.response?.data?.error ?? e.message
  }
}

const miembros = reactive({ abierto: false, grupoId: 0, nombre: '', actuales: [] as number[], seleccion: [] as number[] })
const guardandoMiembros = ref(false)

async function abrirMiembros(g: GrupoJugadores) {
  error.value = ''
  try {
    const { data } = await api.get<User[]>(`/admin/grupos/${g.id}/jugadores`)
    miembros.grupoId = g.id
    miembros.nombre = g.nombre
    miembros.actuales = data.map((u) => u.id)
    miembros.seleccion = [...miembros.actuales]
    miembros.abierto = true
  } catch (e: any) {
    error.value = e.response?.data?.error ?? e.message
  }
}

function alternarMiembro(id: number) {
  const i = miembros.seleccion.indexOf(id)
  if (i >= 0) miembros.seleccion.splice(i, 1)
  else miembros.seleccion.push(id)
}

async function guardarMiembros() {
  guardandoMiembros.value = true
  error.value = ''
  try {
    const agregar = miembros.seleccion.filter((id) => !miembros.actuales.includes(id))
    const quitar = miembros.actuales.filter((id) => !miembros.seleccion.includes(id))
    if (agregar.length) {
      await api.post(`/admin/grupos/${miembros.grupoId}/jugadores`, { user_ids: agregar })
    }
    for (const uid of quitar) {
      await api.delete(`/admin/grupos/${miembros.grupoId}/jugadores/${uid}`)
    }
    miembros.abierto = false
    mensaje.value = 'Paquete actualizado'
    await cargar()
  } catch (e: any) {
    error.value = e.response?.data?.error ?? e.message
  } finally {
    guardandoMiembros.value = false
  }
}

// ------------------------------------------------------------ accesos

const ambitos: { id: AmbitoAcceso; etiqueta: string }[] = [
  { id: 'grupo', etiqueta: 'Paquete de jugadores' },
  { id: 'usuario', etiqueta: 'Jugador individual' },
  { id: 'global', etiqueta: 'Global (todos)' },
]

const tipos: { id: TipoContenido; etiqueta: string }[] = [
  { id: 'juego', etiqueta: 'Juego del casino' },
  { id: 'loteria', etiqueta: 'Lotería' },
  { id: 'rifa', etiqueta: 'Rifa' },
]

const form = reactive({
  ambito: 'grupo' as AmbitoAcceso,
  grupoId: 0,
  usuarioId: 0,
  tipo: 'juego' as TipoContenido,
  codigo: '',
  activo: true,
})
const guardandoAcceso = ref(false)

const opcionesContenido = computed(() => {
  if (form.tipo === 'juego') {
    return juegos.value.map((j) => ({ codigo: j.codigo, etiqueta: j.nombre }))
  }
  if (form.tipo === 'loteria') {
    return loterias.value.map((l) => ({ codigo: l.codigo, etiqueta: `${l.codigo} · ${l.nombre}` }))
  }
  return rifas.value.map((r) => ({ codigo: String(r.id), etiqueta: `#${r.id} · ${r.titulo}` }))
})

function cambiarTipo() {
  form.codigo = ''
}

const descripcionAmbito = computed(() => {
  if (form.ambito === 'global') return 'Se aplica a todos los jugadores.'
  if (form.ambito === 'grupo') {
    const g = grupos.value.find((x) => x.id === form.grupoId)
    return g ? `Se aplica a los ${g.miembros ?? 0} jugadores del paquete "${g.nombre}".` : 'Elegí un paquete.'
  }
  const u = usuarios.value.find((x) => x.id === form.usuarioId)
  return u ? `Se aplica solo a ${u.nombre}.` : 'Elegí un jugador.'
})

async function aplicarAcceso() {
  error.value = ''
  if (form.ambito === 'grupo' && !form.grupoId) {
    error.value = 'Elegí un paquete de jugadores'
    return
  }
  if (form.ambito === 'usuario' && !form.usuarioId) {
    error.value = 'Elegí un jugador'
    return
  }
  if (!form.codigo) {
    error.value = 'Elegí el contenido'
    return
  }
  guardandoAcceso.value = true
  try {
    const payload: Record<string, unknown> = {
      ambito: form.ambito,
      tipo: form.tipo,
      codigo: form.codigo,
      activo: form.activo,
    }
    if (form.ambito === 'grupo') payload.ambito_id = form.grupoId
    if (form.ambito === 'usuario') payload.ambito_id = form.usuarioId
    await api.post('/admin/accesos', payload)
    mensaje.value = form.activo ? 'Contenido activado' : 'Contenido desactivado'
    await cargar()
  } catch (e: any) {
    error.value = e.response?.data?.error ?? e.message
  } finally {
    guardandoAcceso.value = false
  }
}

async function alternarAcceso(a: AccesoContenido) {
  error.value = ''
  try {
    await api.put(`/admin/accesos/${a.id}`, { activo: !a.activo })
    a.activo = !a.activo
  } catch (e: any) {
    error.value = e.response?.data?.error ?? e.message
  }
}

async function eliminarAcceso(a: AccesoContenido) {
  if (!confirm('¿Quitar esta regla? El contenido volverá a regirse por el ámbito siguiente (paquete, global o el estado por defecto).')) return
  error.value = ''
  try {
    await api.delete(`/admin/accesos/${a.id}`)
    accesos.value = accesos.value.filter((x) => x.id !== a.id)
    mensaje.value = 'Regla eliminada'
  } catch (e: any) {
    error.value = e.response?.data?.error ?? e.message
  }
}

const etiquetaAmbito = (a: AccesoContenido) =>
  a.ambito === 'global' ? 'Global' : a.ambito === 'grupo' ? `Paquete: ${a.ambito_nombre ?? a.ambito_id}` : `Jugador: ${a.ambito_nombre ?? a.ambito_id}`

const etiquetaTipo = (a: AccesoContenido) =>
  a.tipo === 'juego' ? 'Juego' : a.tipo === 'loteria' ? 'Lotería' : 'Rifa'
</script>

<template>
  <div>
    <p v-if="error" class="error-text">{{ error }}</p>
    <p v-if="mensaje" class="hint" style="color: var(--prim-2)">{{ mensaje }}</p>
    <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando…</div>

    <!-- -------------------------------------------------- paquetes -->
    <div class="card" style="margin-bottom: 20px">
      <div class="acciones-fila" style="justify-content: space-between; margin-bottom: 12px">
        <div>
          <h3 style="font-size: 16px; margin: 0">Paquetes de jugadores</h3>
          <p class="hint" style="margin: 4px 0 0">
            Agrupá jugadores para activar o desactivar contenidos de una vez. Si un jugador
            está en varios paquetes, vale la primera regla que encuentre.
          </p>
        </div>
        <button class="btn btn-primary btn-sm" @click="nuevoGrupo">+ Nuevo paquete</button>
      </div>

      <div v-if="grupos.length" class="table-wrap">
        <table class="table">
          <thead>
            <tr><th>Nombre</th><th>Descripción</th><th>Jugadores</th><th></th></tr>
          </thead>
          <tbody>
            <tr v-for="g in grupos" :key="g.id">
              <td class="fuerte">{{ g.nombre }}</td>
              <td class="hint">{{ g.descripcion || '—' }}</td>
              <td class="mono">{{ g.miembros ?? 0 }}</td>
              <td style="display: flex; gap: 6px; flex-wrap: wrap">
                <button class="btn btn-ghost btn-sm" @click="abrirMiembros(g)">Jugadores</button>
                <button class="btn btn-ghost btn-sm" @click="editarGrupo(g)">Renombrar</button>
                <button class="btn btn-danger btn-sm" @click="eliminarGrupo(g)">Eliminar</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="hint">Todavía no hay paquetes. Creá uno para asignar accesos en grupo.</p>
    </div>

    <!-- -------------------------------------------------- accesos -->
    <div class="card" style="margin-bottom: 20px">
      <h3 style="font-size: 16px; margin-bottom: 6px">Accesos por paquete o jugador</h3>
      <p class="hint" style="margin-bottom: 14px">
        La resolución es: jugador &gt; paquete &gt; global. Si no hay ninguna regla, vale el estado
        global del contenido.
      </p>

      <div class="acceso-form">
        <label class="campo">
          <span>Ámbito</span>
          <select v-model="form.ambito" class="input">
            <option v-for="a in ambitos" :key="a.id" :value="a.id">{{ a.etiqueta }}</option>
          </select>
        </label>

        <label v-if="form.ambito === 'grupo'" class="campo">
          <span>Paquete</span>
          <select v-model.number="form.grupoId" class="input">
            <option :value="0" disabled>Elegí un paquete</option>
            <option v-for="g in grupos" :key="g.id" :value="g.id">{{ g.nombre }} ({{ g.miembros ?? 0 }})</option>
          </select>
        </label>

        <label v-if="form.ambito === 'usuario'" class="campo">
          <span>Jugador</span>
          <select v-model.number="form.usuarioId" class="input">
            <option :value="0" disabled>Elegí un jugador</option>
            <option v-for="u in usuarios" :key="u.id" :value="u.id">{{ u.nombre }} ({{ u.email }})</option>
          </select>
        </label>

        <label class="campo">
          <span>Tipo de contenido</span>
          <select v-model="form.tipo" class="input" @change="cambiarTipo">
            <option v-for="t in tipos" :key="t.id" :value="t.id">{{ t.etiqueta }}</option>
          </select>
        </label>

        <label class="campo campo-wide">
          <span>Contenido</span>
          <select v-model="form.codigo" class="input">
            <option value="" disabled>Elegí el contenido</option>
            <option v-for="o in opcionesContenido" :key="o.codigo" :value="o.codigo">{{ o.etiqueta }}</option>
          </select>
        </label>

        <label class="campo campo-check switch-campo">
          <span>Activo</span>
          <span class="switch">
            <input v-model="form.activo" type="checkbox" />
            <span class="slider"></span>
          </span>
        </label>
      </div>

      <p class="hint" style="margin: 4px 0 12px">{{ descripcionAmbito }}</p>
      <button class="btn btn-primary" :disabled="guardandoAcceso" @click="aplicarAcceso">
        {{ guardandoAcceso ? 'Aplicando…' : 'Aplicar regla' }}
      </button>
    </div>

    <div class="card">
      <h3 style="font-size: 16px; margin-bottom: 12px">Reglas vigentes</h3>
      <div v-if="accesos.length" class="table-wrap">
        <table class="table">
          <thead>
            <tr><th>Ámbito</th><th>Contenido</th><th>Tipo</th><th>Estado</th><th></th></tr>
          </thead>
          <tbody>
            <tr v-for="a in accesos" :key="a.id">
              <td><span class="badge" :class="a.ambito === 'global' ? 'badge-info' : a.ambito === 'grupo' ? 'badge-ok' : 'badge-violet'">{{ etiquetaAmbito(a) }}</span></td>
              <td class="fuerte">{{ a.contenido_nombre ?? a.codigo }}</td>
              <td>{{ etiquetaTipo(a) }}</td>
              <td>
                <span class="switch">
                  <input type="checkbox" :checked="a.activo" @change="alternarAcceso(a)" />
                  <span class="slider"></span>
                </span>
              </td>
              <td><button class="btn btn-ghost btn-sm" @click="eliminarAcceso(a)">Quitar</button></td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="hint">Sin reglas: cada contenido se rige por su estado global.</p>
    </div>

    <!-- -------------------------------------------------- modales -->
    <Modal v-if="grupoModal.abierto" titulo="Paquete de jugadores" :sub="grupoModal.id ? 'Renombrar' : 'Nuevo paquete'" @cerrar="grupoModal.abierto = false">
      <div class="form-grid">
        <label class="campo"><span>Nombre</span><input v-model="grupoModal.nombre" class="input" placeholder="VIP, Agentes, Pruebas…" /></label>
        <label class="campo campo-wide"><span>Descripción</span><input v-model="grupoModal.descripcion" class="input" placeholder="Para qué se usa el paquete" /></label>
      </div>
      <template #acciones>
        <button class="btn btn-ghost" @click="grupoModal.abierto = false">Cancelar</button>
        <button class="btn btn-primary" :disabled="guardandoGrupo" @click="guardarGrupo">{{ guardandoGrupo ? 'Guardando…' : 'Guardar' }}</button>
      </template>
    </Modal>

    <Modal v-if="miembros.abierto" titulo="Jugadores del paquete" :sub="miembros.nombre" @cerrar="miembros.abierto = false">
      <p class="hint" style="margin-bottom: 10px">Marcá los jugadores que deben pertenecer al paquete.</p>
      <div class="lista-miembros">
        <label v-for="u in usuarios" :key="u.id" class="miembro">
          <input type="checkbox" :checked="miembros.seleccion.includes(u.id)" @change="alternarMiembro(u.id)" />
          <span>{{ u.nombre }} <span class="hint">{{ u.email }}</span></span>
        </label>
      </div>
      <template #acciones>
        <button class="btn btn-ghost" @click="miembros.abierto = false">Cancelar</button>
        <button class="btn btn-primary" :disabled="guardandoMiembros" @click="guardarMiembros">{{ guardandoMiembros ? 'Guardando…' : 'Guardar' }}</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.acceso-form {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 10px;
  align-items: end;
}
.campo {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  color: var(--muted);
}
.campo-wide {
  grid-column: span 2;
}
@media (max-width: 640px) {
  .campo-wide {
    grid-column: span 1;
  }
}
.switch-campo {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.switch {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
  flex: none;
}
.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}
.slider {
  position: absolute;
  inset: 0;
  background: var(--borde, #3a3a3a);
  border-radius: 999px;
  transition: background 0.15s ease;
  cursor: pointer;
}
.slider::before {
  content: '';
  position: absolute;
  width: 18px;
  height: 18px;
  left: 3px;
  top: 3px;
  background: #fff;
  border-radius: 50%;
  transition: transform 0.15s ease;
}
.switch input:checked + .slider {
  background: var(--prim, #2f9e6e);
}
.switch input:checked + .slider::before {
  transform: translateX(20px);
}
.lista-miembros {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 6px 16px;
  max-height: 320px;
  overflow-y: auto;
}
.miembro {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
</style>
