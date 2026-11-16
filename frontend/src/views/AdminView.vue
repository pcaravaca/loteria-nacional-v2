<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { api } from '@/api/client'
import type { AuditEntry, CasinoGame, CasinoStats, Loteria, Sorteo, User } from '@/types'
import { colones, estadoSorteo, fechaHora, tipoLabel } from '@/utils/formato'
import Modal from '@/components/Modal.vue'
import EmptyState from '@/components/EmptyState.vue'

type Tab = 'usuarios' | 'loterias' | 'sorteas' | 'casino' | 'reportes'

const tabs: { id: Tab; label: string; ico: string }[] = [
  { id: 'usuarios', label: 'Usuarios', ico: '👥' },
  { id: 'loterias', label: 'Loterías', ico: '🎰' },
  { id: 'sorteas', label: 'Sorteos', ico: '📅' },
  { id: 'casino', label: 'Casino', ico: '🎲' },
  { id: 'reportes', label: 'Reportes', ico: '📊' },
]
const tab = ref<Tab>('usuarios')

const cargando = ref(true)
const error = ref('')
const usuarios = ref<User[]>([])
const loterias = ref<Loteria[]>([])
const sorteos = ref<Sorteo[]>([])
const juegos = ref<CasinoGame[]>([])
const stats = ref<CasinoStats[]>([])
const auditoria = ref<AuditEntry[]>([])
const filtrosAuditoria = reactive({ actor: '', accion: '', desde: '', hasta: '' })
const cargandoAuditoria = ref(false)

const reportesDisponibles = [
  { tipo: 'apuestas', etiqueta: 'Apuestas' },
  { tipo: 'sorteos', etiqueta: 'Sorteos' },
  { tipo: 'rondas', etiqueta: 'Rondas de casino' },
  { tipo: 'blackjack', etiqueta: 'Blackjack' },
  { tipo: 'movimientos', etiqueta: 'Movimientos' },
  { tipo: 'usuarios', etiqueta: 'Usuarios' },
  { tipo: 'auditoria', etiqueta: 'Auditoría' },
]

async function cargar() {
  cargando.value = true
  error.value = ''
  try {
    const [u, l, s, j, st] = await Promise.all([
      api.get<User[]>('/admin/usuarios'),
      api.get<Loteria[]>('/loterias', { params: { todas: 1 } }),
      api.get<Sorteo[]>('/sorteos'),
      api.get<CasinoGame[]>('/casino/games'),
      api.get<CasinoStats[]>('/admin/casino/stats'),
    ])
    usuarios.value = u.data
    loterias.value = l.data
    sorteos.value = s.data
    juegos.value = j.data
    stats.value = st.data
  } catch (e: any) {
    error.value = e.message
  } finally {
    cargando.value = false
  }
}

onMounted(cargar)

const guardandoJuego = ref('')
const mensaje = ref('')

function statsDe(codigo: string) {
  return stats.value.find((s) => s.juego === codigo)
}

async function guardarJuego(j: CasinoGame) {
  guardandoJuego.value = j.codigo
  mensaje.value = ''
  try {
    await api.patch(`/admin/casino/games/${j.codigo}`, {
      activo: j.activo,
      min_apuesta: j.min_apuesta,
      max_apuesta: j.max_apuesta,
      rtp: j.rtp,
      limite_diario: j.limite_diario,
      max_premio: j.max_premio,
    })
    mensaje.value = `${j.nombre} guardado`
    await cargar()
  } catch (e: any) {
    mensaje.value = e.message
  } finally {
    guardandoJuego.value = ''
  }
}

const ajuste = reactive({ abierto: false, userId: 0, nombre: '', monto: 0, descripcion: '' })
const guardando = ref(false)

function abrirAjuste(u: User) {
  ajuste.abierto = true
  ajuste.userId = u.id
  ajuste.nombre = u.nombre
  ajuste.monto = 0
  ajuste.descripcion = ''
}

function filtrosQuery() {
  const p = new URLSearchParams()
  if (filtrosAuditoria.desde) p.set('desde', filtrosAuditoria.desde)
  if (filtrosAuditoria.hasta) p.set('hasta', filtrosAuditoria.hasta)
  return p.toString() ? '?' + p.toString() : ''
}

async function descargarCSV(tipo: string) {
  mensaje.value = ''
  try {
    const { data } = await api.get(`/admin/reportes/csv/${tipo}${filtrosQuery()}`, { responseType: 'blob' })
    const url = URL.createObjectURL(new Blob([data], { type: 'text/csv;charset=utf-8' }))
    const a = document.createElement('a')
    a.href = url
    a.download = `${tipo}.csv`
    a.click()
    URL.revokeObjectURL(url)
    mensaje.value = `${tipo}.csv descargado`
  } catch (e: any) {
    mensaje.value = e.message
  }
}

async function cargarAuditoria() {
  cargandoAuditoria.value = true
  try {
    const params: Record<string, string> = {}
    if (filtrosAuditoria.actor) params.actor = filtrosAuditoria.actor
    if (filtrosAuditoria.accion) params.accion = filtrosAuditoria.accion
    if (filtrosAuditoria.desde) params.desde = filtrosAuditoria.desde
    if (filtrosAuditoria.hasta) params.hasta = filtrosAuditoria.hasta
    const { data } = await api.get<{ total: number; entradas: AuditEntry[] }>('/admin/auditoria', { params })
    auditoria.value = data.entradas
  } catch (e: any) {
    mensaje.value = e.message
  } finally {
    cargandoAuditoria.value = false
  }
}

async function verDocumento(u: User) {
  mensaje.value = ''
  try {
    const { data } = await api.get(`/admin/usuarios/${u.id}/documento`, { responseType: 'blob' })
    const url = URL.createObjectURL(data)
    window.open(url, '_blank')
  } catch (e: any) {
    mensaje.value = e.message
  }
}

async function anonimizar(u: User) {
  if (!confirm(`¿Anonimizar a ${u.nombre}? Se eliminarán sus datos personales (Ley 8968). No se puede deshacer.`)) return
  mensaje.value = ''
  try {
    await api.post(`/admin/usuarios/${u.id}/anonimizar`)
    mensaje.value = `${u.nombre} anonimizado`
    await cargar()
  } catch (e: any) {
    mensaje.value = e.message
  }
}

async function aplicarAjuste() {
  guardando.value = true
  try {
    const { data } = await api.post<User>(`/admin/usuarios/${ajuste.userId}/saldo`, {
      monto: ajuste.monto,
      descripcion: ajuste.descripcion || undefined,
    })
    const i = usuarios.value.findIndex((u) => u.id === data.id)
    if (i >= 0) usuarios.value[i] = data
    ajuste.abierto = false
  } catch (e: any) {
    error.value = e.message
  } finally {
    guardando.value = false
  }
}

const verificar = async (u: User) => {
  try {
    await api.post(`/admin/usuarios/${u.id}/verificar`)
    await cargar()
  } catch (e:any) {
    alert(e.message)
  }
}

const loteriaForm = reactive({
  codigo: '',
  nombre: '',
  tipo: 'loteria' as 'loteria' | 'quiniela',
  descripcion: '',
  costo_numero: 1000,
  longitud_num: 2,
  max_numeros: 1,
  multiplicador: 50,
  activa: true,
})
const loteriaEditando = ref<number | null>(null)
const loteriaModal = ref(false)

function nuevaLoteria() {
  loteriaEditando.value = null
  Object.assign(loteriaForm, {
    codigo: '', nombre: '', tipo: 'loteria', descripcion: '',
    costo_numero: 1000, longitud_num: 2, max_numeros: 1, multiplicador: 50, activa: true,
  })
  loteriaModal.value = true
}

function editarLoteria(l: Loteria) {
  loteriaEditando.value = l.id
  Object.assign(loteriaForm, {
    codigo: l.codigo, nombre: l.nombre, tipo: l.tipo, descripcion: l.descripcion,
    costo_numero: l.costo_numero, longitud_num: l.longitud_num,
    max_numeros: l.max_numeros, multiplicador: l.multiplicador, activa: l.activa,
  })
  loteriaModal.value = true
}

async function guardarLoteria() {
  guardando.value = true
  try {
    const payload = { ...loteriaForm }
    if (loteriaEditando.value) {
      const { data } = await api.put<Loteria>(`/admin/loterias/${loteriaEditando.value}`, payload)
      const i = loterias.value.findIndex((l) => l.id === data.id)
      if (i >= 0) loterias.value[i] = data
    } else {
      const { data } = await api.post<Loteria>('/admin/loterias', payload)
      loterias.value.unshift(data)
    }
    loteriaModal.value = false
  } catch (e: any) {
    error.value = e.message
  } finally {
    guardando.value = false
  }
}

const sorteoForm = reactive({ loteria_id: 0, titulo: '', fecha: '' })
const sorteoModal = ref(false)

function nuevoSorteo() {
  sorteoForm.loteria_id = loterias.value[0]?.id ?? 0
  sorteoForm.titulo = ''
  sorteoForm.fecha = ''
  sorteoModal.value = true
}

async function crearSorteo() {
  guardando.value = true
  try {
    const { data } = await api.post<Sorteo>('/admin/sorteos', {
      loteria_id: sorteoForm.loteria_id,
      titulo: sorteoForm.titulo,
      fecha: new Date(sorteoForm.fecha).toISOString(),
    })
    sorteos.value.unshift(data)
    sorteoModal.value = false
  } catch (e: any) {
    error.value = e.message
  } finally {
    guardando.value = false
  }
}

const cierre = reactive({ abierto: false, sorteoId: 0, titulo: '', numeros: '' })

function abrirCierre(s: Sorteo) {
  cierre.abierto = true
  cierre.sorteoId = s.id
  cierre.titulo = s.titulo
  cierre.numeros = ''
}

function parsearNumeros(texto: string): number[] {
  return texto
    .split(/[\s,]+/)
    .filter((t) => t !== '')
    .map((t) => Number(t))
    .filter((n) => Number.isInteger(n) && n >= 0)
}

async function cerrarSorteo() {
  const numeros = parsearNumeros(cierre.numeros)
  if (!numeros.length) {
    error.value = 'Ingresa al menos un número ganador'
    return
  }
  guardando.value = true
  try {
    const { data } = await api.post<Sorteo>(`/admin/sorteos/${cierre.sorteoId}/cerrar`, {
      numeros_ganadores: numeros,
    })
    const i = sorteos.value.findIndex((s) => s.id === data.id)
    if (i >= 0) sorteos.value[i] = data
    cierre.abierto = false
  } catch (e: any) {
    error.value = e.message
  } finally {
    guardando.value = false
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <div class="eyebrow">Administración</div>
      <h1>Panel de administración</h1>
      <p>Usuarios, juegos y sorteos.</p>
    </div>

    <div class="tabs">
      <button v-for="t in tabs" :key="t.id" class="tab" :class="{ activa: tab === t.id }" @click="tab = t.id">
        <span>{{ t.ico }}</span> {{ t.label }}
      </button>
    </div>

    <p v-if="error" class="error-text">{{ error }}</p>
    <div v-if="cargando" class="cargando"><span class="spinner"></span> Cargando…</div>

    <div v-show="tab === 'usuarios'">
      <div v-if="usuarios.length" class="table-wrap">
        <table class="table">
          <thead>
            <tr><th>Nombre</th><th>Email</th><th>Rol</th><th>Teléfono</th><th>Saldo</th><th>Bono</th><th>Identificación</th><th></th></tr>
          </thead>
          <tbody>
            <tr v-for="u in usuarios" :key="u.id">
              <td class="fuerte">{{ u.nombre }}</td>
              <td>{{ u.email }}</td>
              <td><span class="badge" :class="u.rol === 'admin' ? 'badge-violet' : u.rol === 'agente' ? 'badge-info' : ''">{{ u.rol }}</span></td>
              <td>{{ u.telefono || '—' }}</td>
              <td class="mono">{{ colones(u.saldo) }}</td>
              <td class="mono texto-gold">{{ colones(u.bono) }}</td>
              <td>
                <template v-if="u.tipo_documento">
                  <span class="hint" style="font-size: 12px">{{ u.tipo_documento }} {{ u.numero_documento }}</span>
                  <span v-if="u.documento_verificado" class="badge badge-ok" style="margin-left: 6px">Verificado</span>
                  <span v-if="!u.fecha_nacimiento" class="badge badge-gold" style="margin-left: 4px">Sin fecha</span>
                  <span v-if="!u.tiene_foto" class="badge badge-gold" style="margin-left: 4px">Sin foto</span>
                </template>
                <span v-else class="hint">Sin cargar</span>
              </td>
              <td style="display: flex; gap: 6px; flex-wrap: wrap">
                <button v-if="u.tipo_documento && !u.documento_verificado" class="btn btn-ghost btn-sm" @click="verificar(u)">
                  Verificar
                </button>
                <button class="btn btn-ghost btn-sm" title="Ver foto del documento" @click="verDocumento(u)">Doc.</button>
                <button class="btn btn-ghost btn-sm" title="Eliminar datos personales (Ley 8968)" @click="anonimizar(u)">Anonimizar</button>
                <button class="btn btn-ghost btn-sm" @click="abrirAjuste(u)">Ajustar saldo</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-else-if="!cargando" icono="👥" titulo="Sin usuarios" />
    </div>

    <div v-show="tab === 'loterias'">
      <div class="acciones-fila">
        <button class="btn btn-primary btn-sm" @click="nuevaLoteria">+ Nueva lotería</button>
      </div>
      <div v-if="loterias.length" class="table-wrap">
        <table class="table">
          <thead>
            <tr><th>Código</th><th>Nombre</th><th>Tipo</th><th>Costo</th><th>Premio x</th><th>Estado</th><th></th></tr>
          </thead>
          <tbody>
            <tr v-for="l in loterias" :key="l.id">
              <td class="mono">{{ l.codigo }}</td>
              <td class="fuerte">{{ l.nombre }}</td>
              <td><span class="badge" :class="l.tipo === 'quiniela' ? 'badge-violet' : 'badge-ok'">{{ tipoLabel(l.tipo) }}</span></td>
              <td class="mono">{{ colones(l.costo_numero) }}</td>
              <td class="mono">{{ l.multiplicador }}x</td>
              <td><span class="badge" :class="l.activa ? 'badge-ok' : 'badge-bad'">{{ l.activa ? 'Activa' : 'Inactiva' }}</span></td>
              <td><button class="btn btn-ghost btn-sm" @click="editarLoteria(l)">Editar</button></td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-else-if="!cargando" icono="🎰" titulo="Sin loterías" texto="Crea tu primer juego para empezar." />
    </div>

    <div v-show="tab === 'sorteas'">
      <div class="acciones-fila">
        <button class="btn btn-primary btn-sm" @click="nuevoSorteo">+ Nuevo sorteo</button>
      </div>
      <div v-if="sorteos.length" class="grid" style="gap: 12px">
        <div v-for="s in sorteos" :key="s.id" class="card sorteo-row">
          <div>
            <div class="eyebrow" style="color: var(--prim-2); font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: .1em">
              {{ s.loteria?.nombre }}
            </div>
            <h3 style="font-size: 16px; margin-top: 5px">{{ s.titulo }}</h3>
            <p class="hint" style="margin-top: 3px">{{ fechaHora(s.fecha) }}</p>
          </div>
          <div style="display: flex; align-items: center; gap: 12px">
            <span class="badge" :class="(estadoSorteo[s.estado] ?? {}).clase">{{ (estadoSorteo[s.estado] ?? {}).texto }}</span>
            <button v-if="s.estado === 'abierto'" class="btn btn-primary btn-sm" @click="abrirCierre(s)">Cerrar</button>
            <div v-else class="numeros">
              <span v-for="n in s.numeros_ganadores" :key="n" class="num num-ganador">{{ String(n).padStart(2, '0') }}</span>
            </div>
          </div>
        </div>
      </div>
      <EmptyState v-else-if="!cargando" icono="📅" titulo="Sin sorteos" />
    </div>

    <div v-show="tab === 'casino'">
      <p v-if="mensaje" class="hint" style="color: var(--prim-2)">{{ mensaje }}</p>
      <div v-if="juegos.length" class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>Juego</th><th>Activo</th><th>Mín</th><th>Máx</th><th>RTP %</th>
              <th>Tope premio</th><th>Tope diario</th><th>Jugadas</th><th>Beneficio</th><th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="j in juegos" :key="j.codigo">
              <td class="fuerte">{{ j.nombre }}</td>
              <td>
                <button
                  class="badge btn-sm"
                  :class="j.activo ? 'badge-ok' : 'badge-muted'"
                  @click="j.activo = !j.activo"
                >{{ j.activo ? 'activo' : 'pausado' }}</button>
              </td>
              <td><input v-model.number="j.min_apuesta" type="number" class="input input-sm" /></td>
              <td><input v-model.number="j.max_apuesta" type="number" class="input input-sm" /></td>
              <td title="Retorno teórico al jugador (%).">
                <input v-model.number="j.rtp" type="number" step="0.1" min="1" max="100" class="input input-sm" />
              </td>
              <td title="Premio máximo por jugada. 0 = sin techo.">
                <input v-model.number="j.max_premio" type="number" class="input input-sm" />
              </td>
              <td><input v-model.number="j.limite_diario" type="number" class="input input-sm" /></td>
              <td class="mono">{{ statsDe(j.codigo)?.jugadas ?? 0 }}</td>
              <td class="mono" :class="(statsDe(j.codigo)?.beneficio ?? 0) >= 0 ? 'texto-pos' : 'texto-neg'">
                {{ colones(statsDe(j.codigo)?.beneficio ?? 0) }}
              </td>
              <td>
                <button class="btn btn-primary btn-sm" :disabled="guardandoJuego === j.codigo" @click="guardarJuego(j)">
                  {{ guardandoJuego === j.codigo ? '…' : 'Guardar' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-else-if="!cargando" icono="🎲" titulo="Sin juegos" />
    </div>

    <div v-show="tab === 'reportes'">
      <p v-if="mensaje" class="hint" style="color: var(--prim-2)">{{ mensaje }}</p>
      <div class="card">
        <h3 style="font-size: 16px; margin-bottom: 6px">Exportaciones (CSV)</h3>
        <p class="hint" style="margin-bottom: 12px">
          Los archivos salen con BOM UTF-8 para abrirse directo en Excel. Si definís un rango
          de fechas en el panel de auditoría, ese mismo rango se aplica a todos los reportes.
        </p>
        <div style="display: flex; gap: 8px; flex-wrap: wrap">
          <button
            v-for="r in reportesDisponibles"
            :key="r.tipo"
            class="btn btn-ghost btn-sm"
            @click="descargarCSV(r.tipo)"
          >
            {{ r.etiqueta }}
          </button>
        </div>
      </div>

      <div class="card" style="margin-top: 16px">
        <h3 style="font-size: 16px; margin-bottom: 12px">Auditoría de acciones</h3>
        <div style="display: flex; gap: 8px; flex-wrap: wrap; align-items: center; margin-bottom: 12px">
          <input v-model="filtrosAuditoria.actor" class="input input-sm" placeholder="ID del actor" style="width: 120px" />
          <input v-model="filtrosAuditoria.accion" class="input input-sm" placeholder="acción" style="width: 190px" />
          <input v-model="filtrosAuditoria.desde" type="date" class="input input-sm" style="width: 150px" />
          <input v-model="filtrosAuditoria.hasta" type="date" class="input input-sm" style="width: 150px" />
          <button class="btn btn-primary btn-sm" @click="cargarAuditoria">Filtrar</button>
          <button class="btn btn-ghost btn-sm" @click="descargarCSV('auditoria')">Descargar CSV</button>
        </div>
        <div v-if="cargandoAuditoria" class="cargando"><span class="spinner"></span></div>
        <div v-else-if="auditoria.length" class="table-wrap">
          <table class="table">
            <thead>
              <tr><th>Fecha</th><th>Actor</th><th>Acción</th><th>Entidad</th><th>Detalle</th><th>IP</th></tr>
            </thead>
            <tbody>
              <tr v-for="e in auditoria" :key="e.id">
                <td class="mono">{{ fechaHora(e.created_at) }}</td>
                <td>{{ e.actor_nombre || '—' }}</td>
                <td><span class="badge badge-info">{{ e.accion }}</span></td>
                <td class="mono">{{ e.entidad }} {{ e.entidad_id }}</td>
                <td class="hint" style="font-size: 12px">{{ e.detalle }}</td>
                <td class="mono">{{ e.ip }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else class="hint">Sin movimientos en el registro con esos filtros.</p>
      </div>
    </div>

    <Modal v-if="ajuste.abierto" titulo="Ajustar saldo" :sub="ajuste.nombre" @cerrar="ajuste.abierto = false">
      <div class="form-grid">
        <label class="campo">
          <span>Monto (positivo recarga, negativo resta)</span>
          <input v-model.number="ajuste.monto" type="number" class="input" placeholder="1000" />
        </label>
        <label class="campo">
          <span>Descripción</span>
          <input v-model="ajuste.descripcion" class="input" placeholder="Recarga de saldo" />
        </label>
      </div>
      <template #acciones>
        <button class="btn btn-ghost" @click="ajuste.abierto = false">Cancelar</button>
        <button class="btn btn-primary" :disabled="guardando" @click="aplicarAjuste">{{ guardando ? 'Guardando…' : 'Aplicar' }}</button>
      </template>
    </Modal>

    <Modal v-if="loteriaModal" titulo="Lotería" :sub="loteriaEditando ? 'Editar juego' : 'Nuevo juego'" @cerrar="loteriaModal = false">
      <div class="form-grid">
        <label class="campo"><span>Código</span><input v-model="loteriaForm.codigo" class="input" placeholder="LN-01" /></label>
        <label class="campo"><span>Nombre</span><input v-model="loteriaForm.nombre" class="input" placeholder="Lotería Nacional" /></label>
        <label class="campo">
          <span>Tipo</span>
          <select v-model="loteriaForm.tipo" class="input">
            <option value="loteria">Lotería</option>
            <option value="quiniela">Quiniela</option>
          </select>
        </label>
        <label class="campo campo-wide"><span>Descripción</span><input v-model="loteriaForm.descripcion" class="input" /></label>
        <label class="campo"><span>Costo por número</span><input v-model.number="loteriaForm.costo_numero" type="number" class="input" /></label>
        <label class="campo"><span>Multiplicador</span><input v-model.number="loteriaForm.multiplicador" type="number" class="input" /></label>
        <label class="campo"><span>Dígitos por número</span><input v-model.number="loteriaForm.longitud_num" type="number" class="input" /></label>
        <label class="campo"><span>Máx. números por apuesta</span><input v-model.number="loteriaForm.max_numeros" type="number" class="input" /></label>
        <label class="campo campo-check"><input v-model="loteriaForm.activa" type="checkbox" /> <span>Activa</span></label>
      </div>
      <template #acciones>
        <button class="btn btn-ghost" @click="loteriaModal = false">Cancelar</button>
        <button class="btn btn-primary" :disabled="guardando" @click="guardarLoteria">{{ guardando ? 'Guardando…' : 'Guardar' }}</button>
      </template>
    </Modal>

    <Modal v-if="sorteoModal" titulo="Nuevo sorteo" @cerrar="sorteoModal = false">
      <div class="form-grid">
        <label class="campo">
          <span>Juego</span>
          <select v-model.number="sorteoForm.loteria_id" class="input">
            <option v-for="l in loterias" :key="l.id" :value="l.id">{{ l.nombre }}</option>
          </select>
        </label>
        <label class="campo"><span>Título</span><input v-model="sorteoForm.titulo" class="input" placeholder="Sorteo 14 de noviembre" /></label>
        <label class="campo"><span>Fecha</span><input v-model="sorteoForm.fecha" type="datetime-local" class="input" /></label>
      </div>
      <template #acciones>
        <button class="btn btn-ghost" @click="sorteoModal = false">Cancelar</button>
        <button class="btn btn-primary" :disabled="guardando" @click="crearSorteo">{{ guardando ? 'Creando…' : 'Crear' }}</button>
      </template>
    </Modal>

    <Modal v-if="cierre.abierto" titulo="Cerrar sorteo" :sub="cierre.titulo" @cerrar="cierre.abierto = false">
      <label class="campo">
        <span>Números ganadores (separados por coma o espacio)</span>
        <input v-model="cierre.numeros" class="input" placeholder="07, 19" autofocus />
      </label>
      <p class="hint" style="margin-top: 8px">Al cerrar se evalúan las apuestas pendientes y se pagan los premios automáticamente.</p>
      <template #acciones>
        <button class="btn btn-ghost" @click="cierre.abierto = false">Cancelar</button>
        <button class="btn btn-primary" :disabled="guardando" @click="cerrarSorteo">{{ guardando ? 'Cerrando…' : 'Cerrar y pagar' }}</button>
      </template>
    </Modal>
  </div>
</template>
