# Lotería Nacional CR v2

Plataforma web para jugar Lotería Nacional, quinielas y otros juegos de azar de
Costa Rica. Migración moderna del proyecto legacy `loteria-nacional`
(ASP.NET/WebForms + SQL Server) a una arquitectura Go + Vue.

## Stack

| Capa    | Tecnología                                             |
| ------- | ------------------------------------------------------ |
| Backend | Go 1.22 · Gin · GORM · SQLite                          |
| Auth    | JWT (bcrypt para claves)                               |
| Frontend| Vue 3 · Vite · TypeScript · Pinia · Vue Router · Axios  |
| Estilos | CSS propio (design system oscuro, sin framework UI)    |

## Estructura

```
backend/
  cmd/api/            punto de entrada (GIN, rutas, CORS)
  internal/config/    variables de entorno
  internal/database/  conexión GORM + seed de demostración
  internal/handlers/  auth, juegos, apuestas, reportes, admin
  internal/middleware/ JWT + control de roles
  internal/models/    modelos GORM (User, Loteria, Sorteo, Apuesta, Movimiento)
frontend/
  src/api/            cliente Axios con interceptor de token
  src/components/     AppShell, Modal, NumPad, tarjetas
  src/views/          landing, auth, dashboard, juegos, reportes, admin, agente
  src/stores/         Pinia (auth)
  src/styles/         design system
```

## Requisitos

- Go 1.22+
- Node 18+ y npm

## Backend

```bash
cd backend
cp ../.env.example ../.env      # opcional, hay valores por defecto
go run ./cmd/api
```

Por defecto escucha en `:8080` y usa `./data/loteria.db`. Con `SEED=true`
(por defecto) crea datos de demostración la primera vez.

Credenciales semilla:

| Rol     | Email                  | Clave        |
| ------- | ---------------------- | ------------ |
| admin   | admin@loteria.test     | `admin123`   |
| agente  | agente@loteria.test    | `agente123`  |
| cliente | cliente@loteria.test   | `cliente123` |

## Frontend

```bash
cd frontend
npm install
npm run dev        # http://localhost:5173 (proxy /api -> :8080)
npm run build      # genera dist/
npm run typecheck  # vue-tsc --noEmit
```

## Roles y flujo

- **cliente**: ve sorteos abiertos, juega números, consulta sus apuestas,
  resultados, reportes y movimientos de saldo.
- **agente**: cierra sorteos cargando los números ganadores.
- **admin**: además administra usuarios (ajuste de saldo), loterías y sorteos.

Al cerrar un sorteo el backend evalúa las apuestas pendientes y paga premios:
- acierto parcial → `ganadora`, premio = monto × 2
- acierto total → `pagada`, premio = monto × multiplicador del juego

## API (resumen)

```
POST /api/auth/register | /api/auth/login
GET  /api/auth/me
GET  /api/loterias[/:id]          GET /api/sorteos[/proximos]
GET  /api/dashboard               GET /api/movimientos
GET  /api/resultados              GET /api/reportes/resumen
POST /api/apuestas                GET /api/apuestas/mias
GET  /api/admin/usuarios          POST /api/admin/usuarios/:id/saldo
POST /api/admin/loterias          PUT  /api/admin/loterias/:id
POST /api/admin/sorteos           POST /api/admin/sorteos/:id/cerrar
```

## Notas de entorno (WSL)

- El proyecto está en un mount NTFS (`D:`). Vite lleva `watch.usePolling`
  porque los mtimes del mount no son fiables y puede servir módulos cacheados.
- Los binarios Linux no deben ejecutarse directamente desde el mount NTFS:
  copiarlos a una ruta ext4 (p. ej. `~/`) antes de ejecutarlos.
