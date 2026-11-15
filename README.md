# Lotería Nacional CR v2

Plataforma web para jugar Lotería Nacional, quinielas y otros juegos de azar de
Costa Rica. Migración moderna del proyecto legacy `loteria-nacional`
(ASP.NET/WebForms + SQL Server) a una arquitectura Go + Vue.

## Stack

| Capa    | Tecnología                                             |
| ------- | ------------------------------------------------------ |
| Backend | Go 1.22 · Gin · GORM · SQLite o PostgreSQL             |
| Auth    | JWT (bcrypt para claves)                               |
| Frontend| Vue 3 · Vite · TypeScript · Pinia · Vue Router · Axios  |
| Estilos | CSS propio (design system oscuro, sin framework UI)    |

## Estructura

```
backend/
  cmd/api/            punto de entrada (GIN, rutas, CORS)
  internal/config/    variables de entorno
  internal/database/  conexión GORM + seed de demostración
  internal/handlers/  auth, juegos, apuestas, reportes, admin, casino
  internal/casino/    RNG provably fair y lógica de los 6 juegos
  internal/middleware/ JWT + control de roles
  internal/models/    modelos GORM (User, Loteria, Sorteo, Apuesta, Movimiento)
frontend/
  src/api/            cliente Axios con interceptor de token
  src/components/     AppShell, Modal, NumPad, tarjetas
  src/views/          landing, auth, dashboard, juegos, reportes, admin, agente, casino
  src/stores/         Pinia (auth)
  src/styles/         design system
```

## Requisitos

- Go 1.22+
- Node 18+ y npm

## Base de datos

Por defecto usa un archivo SQLite (`./data/loteria.db`). Si se define
`DATABASE_URL` (o `DB_HOST`), el backend usa **PostgreSQL** y crea el esquema con
`AutoMigrate` en cada arranque.

```bash
# PostgreSQL en el servidor de desarrollo
export DATABASE_URL="postgres://loteria:CLAVE@100.89.167.67:5499/loteria_dev?sslmode=disable"
# o por variables sueltas: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_TZ
```

La base de desarrollo real vive en `100.89.167.67:5499` (contenedor `fcfeed-pgdev`,
base `loteria_dev`, rol `loteria`). Ver `Plans/Loteria-DB-Postgres.md` en el Vault.

### Datos de prueba

`backend/scripts/demo-data.sql` puebla la base con usuarios demo (clave
`demo1234`), sorteos abiertos y cerrados, apuestas, movimientos, rondas de los 5
juegos rapidos y manos de blackjack. Es idempotente:

```bash
ssh peter@100.89.167.67 "docker exec -i fcfeed-pgdev psql -U dba -d loteria_dev" < backend/scripts/demo-data.sql
```

## Backend

```bash
cd backend
cp ../.env.example ../.env      # opcional, hay valores por defecto
go run ./cmd/api
```

Por defecto escucha en `:8080`. Con `SEED=true` (por defecto) crea datos de
demostración la primera vez.

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

## Mini casino (implementado)

Además de loterías y quinielas, la app incluye un mini casino con el mismo saldo y la
misma sesión. Ver el plan completo en el Vault: `Plans/Mini-Casino-Plan.md`. Resumen:

- **Stack:** componentes Vue con DOM/CSS (slots, blackjack) y canvas 2D (ruleta,
  dados, raspa, keno). Cero dependencias nuevas; SFX sintetizados con Web Audio.
- **Backend:** lógica de resultado y pago en Go (`internal/casino/`), con esquema
  *provably fair* (`server_seed` + `client_seed` + `nonce` → HMAC-SHA256) y modelos
  `CasinoGame` / `CasinoRound`.
- **Juegos:** slots, ruleta, dados, blackjack, raspa y gana, keno.
- **Operación:** topes diarios por juego y usuario, techo de premio por jugada
  (`max_premio`; acota la apuesta máxima sin tocar el multiplicador que audita
  el jugador) y panel de admin en `/admin` (activar/pausar, límites, jugadas y
  beneficio). El blackjack no usa más valores fijos: toma mínimo/máximo del
  juego configurado.
- **Modelo:** venta B2B de la plataforma; el cliente opera con sus propios permisos
  (ver `Plans/Regulacion-Juegos-CR.md`). Pendiente: si el blackjack necesita
  seguro y rendirse.

Rutas del casino:

```
GET  /api/casino/games            POST /api/casino/play
GET  /api/casino/seed             POST /api/casino/seed/rotate
POST /api/casino/blackjack        POST /api/casino/blackjack/:id/hit|stand|double
POST /api/casino/blackjack/:id/surrender | /insurance
GET  /api/casino/rounds           GET  /api/casino/rounds/:id/verify
GET  /api/admin/casino/stats      PATCH /api/admin/casino/games/:codigo
```

El blackjack incluye rendicion (late surrender: recupera la mitad de la apuesta,
solo con la mano inicial) y seguro (con As visible del crupier: cuesta la mitad
y paga 2:1 si el crupier tiene blackjack natural).

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
