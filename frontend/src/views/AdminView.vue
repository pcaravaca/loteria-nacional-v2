<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/api/client'
import type { AuditEntry, CasinoGame, CasinoStats, Loteria, Sorteo, User } from '@/types'
import { colones, estadoSorteo, fechaHora, tipoLabel } from '@/utils/formato'
import Modal from '@/components/Modal.vue'
import EmptyState from '@/components/EmptyState.vue'
import AccesosAdmin from '@/components/admin/AccesosAdmin.vue'

type Tab = 'usuarios' | 'loterias' | 'sorteos' | 'casino' | 'rifas' | 'accesos' | 'reportes' | 'cs'

const tabs: { id: Tab; label: string; ico: string }[] = [
  { id: 'usuarios', label: 'Usuarios', ico: '👥' },
  { id: 'loterias', label: 'Loterías', ico: '🎰' },
  { id: 'sorteos', label: 'Sorteos', ico: '📅' },
  { id: 'casino', label: 'Casino', ico: '🎲' },
  { id: 'rifas', label: 'Rifas', ico: '🎁' },
  { id: 'accesos', label: 'Accesos', ico: '🔑' },
  { id: 'reportes', label: 'Reportes', ico: '📊' },
  { id: 'cs', label: 'Servicio al cliente', ico: '🎧' },
]
const route = useRoute()
const tab = ref<Tab>((route.query.tab as Tab) || 'usuarios')

function irA(id: Tab) {
  tab.value = id
  if (id === 'cs') void cargarAlertas()
}

const cargando = ref(true)
const error = ref('')
const usuarios = ref<User[]>([])
const loterias = ref<Loteria[]>([])
const sorteos = ref<Sorteo[]>([])
const juegos = ref<CasinoGame[]>([])
const stats = ref<CasinoStats[]>([])
const auditoria = ref<AuditEntry[]>([])
const rifas = ref<any[]>([])
const rifaForm = reactive({
  titulo: '', descripcion: '', premio: '', precio_numero: 5000,
  desde_numero: 0, hasta_numero: 99, fecha_sorteo: '', tipo_sorteo: 'azar', premio_monto: 0,
})
const guardandoRifa = ref(false)
const filtrosAuditoria = reactive({ actor: '', accion: '', desde: '', hasta: '' })
// Servicio al cliente: linea de tiempo de lo que un jugador hizo.
const csFiltros = reactive({ user_id: '', request_id: '', accion: '', desde: '', hasta: '' })
const csEntradas = ref<AuditEntry[]>([])
const csJugador = ref<Record<string, any> | null>(null)
const csTotal = ref(0)
const csCargando = ref(false)
interface AlertaCS {
  id: number
  tipo: string
  severidad: string
  user_id: number
  entidad: string
  entidad_id: string
  detalle: string
  revisada: boolean
  created_at: string
}
const csAlertas = ref<AlertaCS[]>([])
const csAlertasCargando = ref(false)

async function cargarAlertas() {
  csAlertasCargando.value = true
  try {
    const { data } = await api.get<{ total: number; alertas: AlertaCS[] }>('/admin/cs/alertas')
    csAlertas.value = data.alertas
  } catch (e: any) {
    mensaje.value = e.message
  } finally {
    csAlertasCargando.value = false
  }
}

async function revisarAlerta(a: AlertaCS) {
  try {
    await api.post(`/admin/cs/alertas/${a.id}/revisar`)
    a.revisada = true
    csAlertas.value = csAlertas.value.filter((x) => x.id !== a.id)
    mensaje.value = 'Alerta marcada como revisada'
  } catch (e: any) {
    mensaje.value = e.message
  }
}

const severidadClase: Record<string, string> = {
  alta: 'badge-bad',
  media: 'badge-gold',
  informativa: 'badge-info',
}
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
    const [u, l, s, j, st, rf] = await Promise.all([
      api.get<User[]>('/admin/usuarios'),
      api.get<Loteria[]>('/loterias', { params: { todas: 1 } }),
      api.get<Sorteo[]>('/sorteos'),
      api.get<CasinoGame[]>('/casino/games', { params: { todas: 1 } }),
      api.get<CasinoStats[]>('/admin/casino/stats'),
      api.get<any[]>('/admin/rifas'),
    ])
    usuarios.value = u.data
    loterias.value = l.data
    sorteos.value = s.data
    juegos.value = j.data
    stats.value = st.data ?? []
    rifas.value = rf.data ?? []
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
  return (stats.value ?? []).find((s) => s.juego === codigo)
}

async function crearRifa() {
  guardandoRifa.value = true
  error.value = ''
  try {
    const { data } = await api.post('/admin/rifas', {
      ...rifaForm,
      fecha_sorteo: new Date(rifaForm.fecha_sorteo).toISOString(),
    })
    mensaje.value = 'Rifa creada #' + data.id
    rifaForm.titulo = ''
    rifaForm.premio = ''
    rifaForm.descripcion = ''
    await cargar()
  } catch (e: any) {
    error.value = e.response?.data?.error ?? e.message
  } finally {
    guardandoRifa.value = false
  }
}

async function sortearRifa(id: number) {
  if (!confirm('¿Sortear esta rifa al azar entre los vendidos?')) return
  try {
    await api.post(`/admin/rifas/${id}/sortear`, {})
    mensaje.value = 'Rifa sorteada'
    await cargar()
  } catch (e: any) {
    error.value = e.response?.data?.error ?? e.message
  }
}

async function cancelarRifa(id: number) {
  if (!confirm('¿Cancelar la rifa y reembolsar todos los números?')) return
  try {
    await api.post(`/admin/rifas/${id}/cancelar`, {})
    mensaje.value = 'Rifa cancelada, reembolsos aplicados'
    await cargar()
  } catch (e: any) {
    error.value = e.response?.data?.error ?? e.message
  }
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

async function cargarCs() {
  csCargando.value = true
  void cargarAlertas()
  csJugador.value = null
  csEntradas.value = []
  try {
    const params: Record<string, string> = {}
    if (csFiltros.user_id) params.user_id = csFiltros.user_id
    if (csFiltros.request_id) params.request_id = csFiltros.request_id
    if (csFiltros.accion) params.accion = csFiltros.accion
    if (csFiltros.desde) params.desde = csFiltros.desde
    if (csFiltros.hasta) params.hasta = csFiltros.hasta
    const { data } = await api.get<{ total: number; entradas: AuditEntry[]; jugador?: Record<string, any> }>(
      '/admin/cs/timeline',
      { params },
    )
    csEntradas.value = data.entradas
    csTotal.value = data.total
    csJugador.value = data.jugador ?? null
  } catch (e: any) {
    mensaje.value = e.message
  } finally {
    csCargando.value = false
  }
}

function descargarCs() {
  const cab = ['fecha', 'jugador', 'accion', 'entidad', 'entidad_id', 'detalle', 'ip', 'request_id']
  const filas = csEntradas.value.map((e) => [
    fechaHora(e.created_at),
    e.actor_nombre || e.actor_id,
    e.accion,
    e.entidad,
    e.entidad_id,
    e.detalle,
    e.ip,
    (e as any).request_id ?? '',
  ])
  const csv = [cab, ...filas]
    .map((f) => f.map((v) => `"${String(v).replace(/"/g, '""')}"`).join(','))
    .join('\n')
  const url = URL.createObjectURL(new Blob(["\ufeff" + csv], { type: 'text/csv;charset=utf-8' }))
  const a = document.createElement('a')
  a.href = url
  a.download = 'cs-timeline.csv'
  a.click()
  URL.revokeObjectURL(url)
  mensaje.value = 'cs-timeline.csv descargado'
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
  tabla_premios: '',
  activa: true,
})
const loteriaEditando = ref<number | null>(null)
const loteriaModal = ref(false)

function nuevaLoteria() {
  loteriaEditando.value = null
  Object.assign(loteriaForm, {
    codigo: '', nombre: '', tipo: 'loteria', descripcion: '',
    costo_numero: 1000, longitud_num: 2, max_numeros: 1, multiplicador: 50, tabla_premios: '', activa: true,
  })
  loteriaModal.value = true
}

function editarLoteria(l: Loteria) {
  loteriaEditando.value = l.id
  Object.assign(loteriaForm, {
    codigo: l.codigo, nombre: l.nombre, tipo: l.tipo, descripcion: l.descripcion,
    costo_numero: l.costo_numero, longitud_num: l.longitud_num,
    max_numeros: l.max_numeros, multiplicador: l.multiplicador, tabla_premios: l.tabla_premios || '', activa: l.activa,
  })
  loteriaModal.value = true
}

async function guardarLoteria() {
  guardando.value = true
  try {
    const payload = { ...loteriaForm }
    // La tabla de premios es JSON ({"1":3,"2":10}); se valida acá y el backend.
    if (payload.tabla_premios) {
      try {
        JSON.parse(payload.tabla_premios)
      } catch {
        error.value = 'La tabla de premios debe ser JSON válido, por ejemplo {"1":3,"2":10}'
        return
      }
    }
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
      <button v-for="t in tabs" :key="t.id" class="tab" :class="{ activa: tab === t.id }" @click="irA(t.id)">
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
            <tr><th>Código</th><th>Nombre</th><th>Tipo</th><th>Costo</th><th>Premio x</th><th>Premios por aciertos</th><th>Estado</th><th></th></tr>
          </thead>
          <tbody>
            <tr v-for="l in loterias" :key="l.id">
              <td class="mono">{{ l.codigo }}</td>
              <td class="fuerte">{{ l.nombre }}</td>
              <td><span class="badge" :class="l.tipo === 'quiniela' ? 'badge-violet' : 'badge-ok'">{{ tipoLabel(l.tipo) }}</span></td>
              <td class="mono">{{ colones(l.costo_numero) }}</td>
              <td class="mono">{{ l.multiplicador }}x</td>
              <td class="mono tabla-premios">{{ l.tabla_premios || '—' }}</td>
              <td><span class="badge" :class="l.activa ? 'badge-ok' : 'badge-bad'">{{ l.activa ? 'Activa' : 'Inactiva' }}</span></td>
              <td><button class="btn btn-ghost btn-sm" @click="editarLoteria(l)">Editar</button></td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-else-if="!cargando" icono="🎰" titulo="Sin loterías" texto="Crea tu primer juego para empezar." />
    </div>

    <div v-show="tab === 'sorteos'">
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
      <div style="display:flex; justify-content:flex-end; margin-bottom:10px">
        <router-link to="/casino" class="btn btn-ghost btn-sm">Abrir lobby del casino</router-link>
      </div>
      <p v-if="mensaje" class="hint" style="color: var(--prim-2)">{{ mensaje }}</p>
      <div v-if="juegos.length" class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>Juego</th><th>Activo</th><th>Mín</th><th>Máx</th><th>RTP %</th><th>RTP al azar %</th>
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
              <td title="Retorno teórico al jugador (%). Se edita desde acá.">
                <input v-model.number="j.rtp" type="number" step="0.1" min="1" max="100" class="input input-sm" />
              </td>
              <td
                class="mono"
                :title="j.rtp_casual > 0
                  ? 'RTP jugando al azar, sin estrategia. Valor declarado en el seed; no se edita desde acá.'
                  : 'Juego de azar puro: no aplica RTP casual.'"
              >{{ j.rtp_casual > 0 ? j.rtp_casual.toFixed(2) : '—' }}</td>
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
      <p v-if="juegos.length" class="hint" style="margin-top:8px">
        «RTP al azar» es el retorno jugando sin estrategia; solo lo declaran los juegos con decisiones
        (dominó, ajedrez, backgammon, tetris). Es el valor declarado en el seed y no se edita desde esta
        pantalla; el RTP «jugando bien» sí.
      </p>
      <EmptyState v-else-if="!cargando" icono="🎲" titulo="Sin juegos" />
    </div>

    <div v-show="tab === 'rifas'">
      <p v-if="mensaje" class="hint" style="color: var(--prim-2)">{{ mensaje }}</p>
      <div class="card" style="margin-bottom: 16px">
        <h3 style="font-size: 16px; margin-bottom: 10px">Nueva rifa</h3>
        <div class="rifa-form">
          <input v-model="rifaForm.titulo" class="input" placeholder="Título" />
          <input v-model="rifaForm.premio" class="input" placeholder="Premio (ej.: Moto Pulsar 200)" />
          <input v-model="rifaForm.descripcion" class="input" placeholder="Descripción" />
          <label>Precio por número
            <input v-model.number="rifaForm.precio_numero" type="number" class="input" min="1" />
          </label>
          <label>Desde
            <input v-model.number="rifaForm.desde_numero" type="number" class="input" />
          </label>
          <label>Hasta
            <input v-model.number="rifaForm.hasta_numero" type="number" class="input" />
          </label>
          <label>Fecha de sorteo
            <input v-model="rifaForm.fecha_sorteo" type="datetime-local" class="input" />
          </label>
          <label>Tipo de sorteo
            <select v-model="rifaForm.tipo_sorteo" class="input">
              <option value="azar">Azar entre vendidos</option>
              <option value="manual">Manual</option>
            </select>
          </label>
          <label>Premio en efectivo (opcional)
            <input v-model.number="rifaForm.premio_monto" type="number" class="input" min="0" />
          </label>
        </div>
        <button class="btn btn-primary" :disabled="guardandoRifa" @click="crearRifa">
          {{ guardandoRifa ? 'Creando…' : 'Crear rifa' }}
        </button>
      </div>
      <div v-if="rifas.length" class="table-wrap">
        <table class="table">
          <thead>
            <tr><th>Título</th><th>Premio</th><th>Precio</th><th>Vendidos</th><th>Recaudado</th><th>Sorteo</th><th>Estado</th><th></th></tr>
          </thead>
          <tbody>
            <tr v-for="r in rifas" :key="r.id">
              <td class="fuerte">{{ r.titulo }}</td>
              <td>{{ r.premio }}</td>
              <td class="mono">{{ colones(r.precio_numero) }}</td>
              <td class="mono">{{ r.vendidos }}/{{ r.total }}</td>
              <td class="mono">{{ colones(r.recaudado) }}</td>
              <td>{{ fechaHora(r.fecha_sorteo) }}</td>
              <td><span class="badge" :class="r.estado === 'abierta' ? 'badge-ok' : r.estado === 'sorteada' ? 'badge-gold' : 'badge-bad'">{{ r.estado }}</span></td>
              <td>
                <button v-if="r.estado === 'abierta'" class="btn btn-ghost btn-sm" @click="sortearRifa(r.id)">Sortear</button>
                <button v-if="r.estado === 'abierta'" class="btn btn-danger btn-sm" @click="cancelarRifa(r.id)">Cancelar</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <EmptyState v-else-if="!cargando" icono="🎁" titulo="Sin rifas" />
    </div>

    <div v-show="tab === 'accesos'">
      <AccesosAdmin />
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

    <div v-show="tab === 'cs'">
      <div class="card">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px">
          <h3 style="font-size: 16px; margin: 0">Alertas para revisar</h3>
          <button class="btn btn-ghost btn-sm" :disabled="csAlertasCargando" @click="cargarAlertas">
            {{ csAlertasCargando ? 'Actualizando…' : 'Actualizar' }}
          </button>
        </div>
        <p class="hint" style="margin-bottom: 12px">
          Las genera el sistema solo: una apuesta que quedó retenida, un reintento que delata un
          corte de conexión o un pago resuelto dos veces. No mueven dinero.
        </p>
        <div v-if="csAlertas.length" class="table-wrap">
          <table class="table">
            <thead>
              <tr><th>Fecha</th><th>Tipo</th><th>Severidad</th><th>Jugador</th><th>Detalle</th><th></th></tr>
            </thead>
            <tbody>
              <tr v-for="a in csAlertas" :key="a.id">
                <td class="mono">{{ fechaHora(a.created_at) }}</td>
                <td class="mono">{{ a.tipo }}</td>
                <td><span class="badge" :class="severidadClase[a.severidad] ?? 'badge-info'">{{ a.severidad }}</span></td>
                <td class="mono">{{ a.user_id }}</td>
                <td class="hint" style="font-size: 12px">{{ a.detalle }}</td>
                <td><button class="btn btn-ghost btn-sm" @click="revisarAlerta(a)">Marcar revisada</button></td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else-if="!csAlertasCargando" class="hint">Sin alertas pendientes.</p>
      </div>

      <div class="card" style="margin-top: 16px">
        <h3 style="font-size: 16px; margin-bottom: 6px">Línea de tiempo del jugador</h3>
        <p class="hint" style="margin-bottom: 12px">
          Busca por jugador o por <span class="mono">request_id</span> (el identificador que la app
          manda en cada petición). Con eso ves la jugada completa: qué hizo, cuándo, desde qué IP
          y con qué resultado, sin adivinar.
        </p>
        <div style="display: flex; gap: 8px; flex-wrap: wrap; align-items: center; margin-bottom: 12px">
          <input v-model="csFiltros.user_id" class="input input-sm" placeholder="ID del jugador" style="width: 140px" />
          <input v-model="csFiltros.request_id" class="input input-sm mono" placeholder="request_id" style="width: 260px" />
          <input v-model="csFiltros.accion" class="input input-sm" placeholder="acción" style="width: 190px" />
          <input v-model="csFiltros.desde" type="date" class="input input-sm" style="width: 150px" />
          <input v-model="csFiltros.hasta" type="date" class="input input-sm" style="width: 150px" />
          <button class="btn btn-primary btn-sm" :disabled="csCargando" @click="cargarCs">
            {{ csCargando ? 'Buscando…' : 'Buscar' }}
          </button>
          <button class="btn btn-ghost btn-sm" :disabled="!csEntradas.length" @click="descargarCs">
            Descargar CSV
          </button>
        </div>

        <div v-if="csJugador" class="hint" style="margin-bottom: 12px">
          <b>{{ csJugador.nombre }}</b> · {{ csJugador.email }} · saldo {{ colones(csJugador.saldo) }} ·
          bono {{ colones(csJugador.bono) }} · {{ csJugador.rol }}
        </div>

        <p v-if="csTotal" class="hint" style="margin-bottom: 8px">{{ csTotal }} acción(es) registrada(s).</p>

        <div v-if="csEntradas.length" class="table-wrap">
          <table class="table">
            <thead>
              <tr><th>Fecha</th><th>Acción</th><th>Entidad</th><th>Detalle</th><th>IP</th><th>request_id</th></tr>
            </thead>
            <tbody>
              <tr v-for="e in csEntradas" :key="e.id">
                <td class="mono">{{ fechaHora(e.created_at) }}</td>
                <td><span class="badge badge-info">{{ e.accion }}</span></td>
                <td class="mono">{{ e.entidad }} {{ e.entidad_id }}</td>
                <td class="hint" style="font-size: 12px">{{ e.detalle }}</td>
                <td class="mono">{{ e.ip }}</td>
                <td class="mono" style="font-size: 11px">{{ (e as any).request_id || '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else-if="!csCargando" class="hint">Sin acciones con esos filtros.</p>
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
        <label class="campo campo-wide">
          <span>Tabla de premios por aciertos (JSON)</span>
          <input v-model="loteriaForm.tabla_premios" class="input mono" placeholder='{"1":3,"2":10}' />
          <small class="ayuda">Multiplicador por cantidad de aciertos. Sin entrada vale el 2x histórico.</small>
        </label>
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

<style scoped>
.tabla-premios {
  font-size: 11px;
  color: var(--muted);
  max-width: 220px;
  word-break: break-all;
}
.ayuda {
  font-size: 11px;
  color: var(--muted);
}
.rifa-form {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}
.rifa-form label {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  color: var(--muted);
}
</style>
