<script setup lang="ts">
import { onMounted } from 'vue'
import { useMarcaStore } from '@/stores/marca'

const marca = useMarcaStore()
onMounted(() => marca.cargar())

// RTP declarado en backend/internal/database/seed.go (SeedCasino). Es el mismo
// valor que muestra cada tarjeta del casino; se mide por simulacion sobre el
// algoritmo de sorteo, no por auditoria externa.
const rtpJuegos = [
  { nombre: 'Dados del Sur', codigo: 'dados', rtp: '88.0' },
  { nombre: 'Fortuna CR', codigo: 'slots', rtp: '94.7' },
  { nombre: 'Rueda Fortuna', codigo: 'ruleta', rtp: '94.4' },
  { nombre: 'Raspa y Gana', codigo: 'raspa', rtp: '92.1' },
  { nombre: 'Keno CR', codigo: 'keno', rtp: '92.5' },
  { nombre: '21 Blackjack', codigo: 'blackjack', rtp: '99.4' },
  { nombre: 'Cascada Fortuna', codigo: 'cascada', rtp: '96.5' },
  { nombre: 'Mayor o Menor', codigo: 'hilo', rtp: '92.3' },
  { nombre: 'Bajo o Sobre', codigo: 'alto', rtp: '95.0' },
  { nombre: 'Fortuna Rápida', codigo: 'fortuna', rtp: '95.0' },
  { nombre: 'Cara o Sello', codigo: 'moneda', rtp: '99.0' },
  { nombre: 'Número de la Suerte', codigo: 'suertudo', rtp: '75.0' },
  { nombre: 'Parley de 2', codigo: 'parley', rtp: '91.0' },
  { nombre: 'Baccarat', codigo: 'baccarat', rtp: '98.7' },
  { nombre: 'Bingo 75', codigo: 'bingo', rtp: '88.3' },
  { nombre: 'Gemas', codigo: 'gemas', rtp: '95.0' },
]
</script>

<template>
  <div class="transp-page">
    <nav class="nav-publica">
      <div class="brand">
        <img v-if="marca.icono" class="brand-logo" :src="marca.icono" alt="" />
        <div v-else class="brand-mark">LN</div>
        <div>
          <div class="brand-name">{{ marca.nombre }}</div>
          <div class="brand-sub">Costa Rica</div>
        </div>
      </div>
      <div class="acciones">
        <router-link to="/casino" class="btn btn-ghost btn-sm">Ir al casino</router-link>
        <router-link to="/login" class="btn btn-ghost btn-sm">Ingresar</router-link>
        <router-link to="/registro" class="btn btn-primary btn-sm">Crear cuenta</router-link>
      </div>
    </nav>

    <main class="wrap cert-wrap">
      <!-- ================= Certificado ================= -->
      <section class="certificado" aria-labelledby="titulo-cert">
        <div class="cert-marco">
          <header class="cert-head">
            <div class="sello" role="img" aria-label="Escudo de transparencia con check">
              <svg viewBox="0 0 64 72" width="64" height="72" aria-hidden="true">
                <defs>
                  <linearGradient id="sello-g" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0" stop-color="#f0d98a" />
                    <stop offset="1" stop-color="#c9a227" />
                  </linearGradient>
                </defs>
                <path
                  d="M32 3 58 12v22c0 16-11 27-26 33C17 61 6 50 6 34V12z"
                  fill="rgba(201,162,39,.10)"
                  stroke="url(#sello-g)"
                  stroke-width="2.5"
                  stroke-linejoin="round"
                />
                <path
                  d="M22 37l8 8 15-17"
                  fill="none"
                  stroke="url(#sello-g)"
                  stroke-width="5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
                <circle cx="32" cy="12" r="2.4" fill="#f0d98a" />
              </svg>
            </div>
            <div class="cert-titulos">
              <div class="eyebrow">Transparencia · Provably Fair</div>
              <h1 id="titulo-cert">Certificado de transparencia</h1>
              <p class="cert-sub">
                Cómo se sortea el mini casino, con qué datos y cómo podés reproducirlo vos mismo.
              </p>
              <div class="cert-badges">
                <span class="badge badge-ok">Sorteo comprometido antes de jugar</span>
                <span class="badge badge-gold">Verificacion reproducible</span>
              </div>
            </div>
          </header>

          <div class="cert-alcance">
            <div class="alcance">
              <h2>Qué cubre</h2>
              <ul>
                <li>El resultado queda comprometido por un hash <b>antes</b> de que apostes.</li>
                <li>La semilla del servidor se revela al terminar la ronda.</li>
                <li>Cada jugada tiene su botón <b>Verificar</b> con los datos para reproducirla.</li>
              </ul>
            </div>
            <div class="alcance no">
              <h2>Qué no cubre</h2>
              <ul>
                <li>No es certificación de laboratorio ni una licencia de juego.</li>
                <li>No hay auditoría externa: es el mecanismo, no un sello de un tercero.</li>
                <li>El RTP está declarado por juego y medido por simulación, no auditado.</li>
              </ul>
            </div>
          </div>
        </div>
      </section>

      <!-- ================= 4 pasos ================= -->
      <section class="pasos" aria-labelledby="titulo-pasos">
        <h2 id="titulo-pasos">Cómo funciona, en 4 pasos</h2>
        <p class="seccion-nota">
          El sorteo no se puede alterar después: la casa se compromete primero, y la prueba queda a la vista.
        </p>

        <ol class="pasos-grid">
          <li class="paso">
            <div class="paso-arte" role="img" aria-label="Sobre cerrado con un sello: la semilla de la casa">
              <svg viewBox="0 0 120 104" width="120" height="104" aria-hidden="true">
                <rect x="18" y="26" width="84" height="58" rx="6" fill="rgba(16,185,129,.08)" stroke="#34d399" stroke-width="2" />
                <path d="M18 32l42 26 42-26" fill="none" stroke="#34d399" stroke-width="2" stroke-linejoin="round" />
                <circle cx="60" cy="58" r="14" fill="rgba(201,162,39,.16)" stroke="#f5b942" stroke-width="2.5" />
                <path d="M54 52h12M54 64h12M57 49v18M63 49v18" stroke="#f5b942" stroke-width="2" stroke-linecap="round" />
              </svg>
            </div>
            <h3>1. La casa compromete su semilla</h3>
            <p>
              Antes de que juegues, el servidor genera una semilla secreta y te muestra solo su huella
              (SHA-256). Ese hash queda visible en el lobby y en cada jugada.
            </p>
          </li>

          <li class="paso">
            <div class="paso-arte" role="img" aria-label="Etiqueta con la semilla del jugador">
              <svg viewBox="0 0 120 104" width="120" height="104" aria-hidden="true">
                <rect x="14" y="24" width="92" height="56" rx="8" fill="rgba(245,185,66,.08)" stroke="#f5b942" stroke-width="2" />
                <circle cx="26" cy="52" r="6" fill="none" stroke="#f5b942" stroke-width="2" />
                <path d="M34 52h60" stroke="#f5b942" stroke-width="2" stroke-linecap="round" />
                <path d="M26 34h52M26 68h38" stroke="rgba(233,239,246,.5)" stroke-width="2" stroke-linecap="round" />
                <circle cx="92" cy="30" r="9" fill="none" stroke="#34d399" stroke-width="2" />
                <path d="M92 39v16M92 47h6M92 52h4" stroke="#34d399" stroke-width="2" stroke-linecap="round" />
              </svg>
            </div>
            <h3>2. Vos ponés tu semilla</h3>
            <p>
              Tu semilla de cliente (por defecto <span class="mono">cli-usuario-nonce</span>) se combina con la de la
              casa. Si rotás la semilla del servidor, el hash cambia y queda registrado.
            </p>
          </li>

          <li class="paso">
            <div class="paso-arte" role="img" aria-label="Mezclador con la formula HMAC-SHA256">
              <svg viewBox="0 0 120 104" width="120" height="104" aria-hidden="true">
                <path d="M40 16h40l-6 32H46z" fill="rgba(16,185,129,.08)" stroke="#34d399" stroke-width="2" stroke-linejoin="round" />
                <path d="M47 26h26" stroke="rgba(233,239,246,.45)" stroke-width="2" stroke-linecap="round" />
                <path d="M52 48l16 8M68 48l-16 8" stroke="#34d399" stroke-width="2" stroke-linecap="round" />
                <path d="M44 50h32l-7 24H51z" fill="rgba(201,162,39,.14)" stroke="#f5b942" stroke-width="2" stroke-linejoin="round" />
                <circle cx="54" cy="60" r="2.4" fill="#f5b942" />
                <circle cx="62" cy="60" r="2.4" fill="#f5b942" />
                <circle cx="34" cy="14" r="4" fill="#34d399" />
                <circle cx="86" cy="14" r="4" fill="#f5b942" />
                <path d="M34 22v8l8 6M86 22v8l-8 6" fill="none" stroke="rgba(233,239,246,.55)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
                <text x="60" y="96" text-anchor="middle" fill="#e9eff6" font-size="11" font-family="ui-monospace, monospace">HMAC</text>
              </svg>
            </div>
            <h3>3. Se mezclan y salen los números</h3>
            <p>
              <span class="mono">HMAC-SHA256(server_seed, "client_seed:nonce:i")</span> produce los números, rodillos o
              bolillas. Cada posición usa un mensaje distinto (<span class="mono">i</span>), así nada se repite.
            </p>
          </li>

          <li class="paso">
            <div class="paso-arte" role="img" aria-label="Lupa sobre el hash revelado">
              <svg viewBox="0 0 120 104" width="120" height="104" aria-hidden="true">
                <rect x="18" y="20" width="72" height="54" rx="8" fill="rgba(16,185,129,.08)" stroke="#34d399" stroke-width="2" />
                <path d="M26 34h56M26 44h40M26 54h48" stroke="rgba(233,239,246,.45)" stroke-width="2" stroke-linecap="round" />
                <circle cx="78" cy="62" r="18" fill="rgba(245,185,66,.12)" stroke="#f5b942" stroke-width="3" />
                <path d="M91 75l14 14" stroke="#f5b942" stroke-width="5" stroke-linecap="round" />
                <path d="M70 62l6 6 11-12" fill="none" stroke="#f5b942" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </div>
            <h3>4. Se destapa y vos comparás</h3>
            <p>
              Al terminar, la semilla del servidor se revela. Repetí la mezcla con los mismos datos y el
              resultado tiene que ser idéntico. Si el hash no coincide, algo se alteró.
            </p>
          </li>
        </ol>
      </section>

      <!-- ================= Verificar ================= -->
      <section class="verificador" aria-labelledby="titulo-verif">
        <h2 id="titulo-verif">Verificá por vos mismo</h2>
        <p class="seccion-nota">
          Este es un ejemplo real con <b>HMAC-SHA256</b> de verdad. Son los mismos campos que devuelve cada ronda.
        </p>

        <div class="verif-grid">
          <div class="verif-datos">
            <h3>Datos de la ronda de ejemplo</h3>
            <dl>
              <div>
                <dt>Semilla del servidor (revelada)</dt>
                <dd class="mono break">4f3a9c1e7b2d8056a1c4e9f0b7d32865a9c0e1f4b6d728394a5b6c7d8e9f0a12</dd>
              </div>
              <div>
                <dt>Hash comprometido antes de jugar</dt>
                <dd class="mono break">7089615d2d5f837c61362c88426e2f2a43978465b688b73eca84da2004bb89ab</dd>
              </div>
              <div>
                <dt>Tu semilla de cliente</dt>
                <dd class="mono">cli-7-0</dd>
              </div>
              <div>
                <dt>Nonce de la ronda</dt>
                <dd class="mono">12</dd>
              </div>
              <div>
                <dt>Mensaje (client_seed:nonce:i)</dt>
                <dd class="mono">cli-7-0:12:0</dd>
              </div>
              <div>
                <dt>Ronda de ejemplo</dt>
                <dd class="mono">#4582 · monto ₡1.000</dd>
              </div>
              <div>
                <dt>Juego de ejemplo</dt>
                <dd>Rueda Fortuna (ruleta, 37 casillas)</dd>
              </div>
            </dl>
          </div>

          <div class="verif-hmac">
            <h3>Resultado HMAC-SHA256</h3>
            <p class="formula mono">HMAC-SHA256(server_seed, "cli-7-0:12:0")</p>
            <code class="hmac mono break">992588092ccca90d7f5c9401a70032296e69ce5eb5a00eed62990ae21e80825d</code>
            <p class="derivado">
              Primeros 4 bytes (big-endian): <b class="mono">2569373705</b><br />
              Ruleta: 2569373705 mod 37 = <b class="gold">21</b> · Slots: mod 6 = <b class="gold">5</b>
            </p>
            <pre class="comando"><code>python3 -c "import hmac,hashlib;print(hmac.new(
 '4f3a9c1e7b2d8056a1c4e9f0b7d32865a9c0e1f4b6d728394a5b6c7d8e9f0a12'.encode(),
 b'cli-7-0:12:0', hashlib.sha256).hexdigest())"</code></pre>
            <p class="nota">
              El botón <b>Verificar</b> de cada jugada del casino hace exactamente este cálculo con tus datos reales
              (endpoint <span class="mono">GET /api/casino/rounds/:id/verify</span>, con tu sesión) y te dice si el hash
              coincide.
            </p>
            <router-link to="/casino" class="btn btn-gold btn-sm">Probarlo en el casino</router-link>
          </div>
        </div>
      </section>

      <!-- ================= RTP ================= -->
      <section class="rtp" aria-labelledby="titulo-rtp">
        <h2 id="titulo-rtp">RTP declarado por juego</h2>
        <p class="seccion-nota">
          Retorno al jugador (RTP) declarado en el seed del casino y medido por <b>simulación sobre el mismo
          algoritmo de sorteo</b>. No es una auditoría externa.
        </p>
        <div class="table-wrap">
          <table class="table">
            <thead>
              <tr><th>Juego</th><th>Código</th><th>RTP declarado</th></tr>
            </thead>
            <tbody>
              <tr v-for="j in rtpJuegos" :key="j.codigo">
                <td class="fuerte">{{ j.nombre }}</td>
                <td class="mono">{{ j.codigo }}</td>
                <td class="mono">{{ j.rtp }}%</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p class="nota">
          Las máquinas AST (tragamonedas y video poker) muestran su RTP por banda (90–98%) en cada tarjeta.
        </p>
      </section>

      <!-- ================= Limitaciones ================= -->
      <section class="limitaciones" aria-labelledby="titulo-lim">
        <h2 id="titulo-lim">Lo que este certificado no dice</h2>
        <ul>
          <li>No certifica el software ni a la empresa: no es un sello de laboratorio ni una licencia de juego.</li>
          <li>No garantiza que el RTP de una sesión corta coincida con el promedio; el RTP es una media de largo plazo.</li>
          <li>Que el hash coincida prueba que el sorteo no se alteró después, no que siempre se pueda ganar.</li>
          <li>Aplica al mini casino. Los sorteos de lotería y quinielas los firma la institución, no este mecanismo.</li>
        </ul>
      </section>

      <!-- ================= Juego responsable ================= -->
      <section class="jr" aria-labelledby="titulo-jr">
        <h2 id="titulo-jr">Juego responsable</h2>
        <p>
          Solo mayores de 18 años. Podés poner límites de depósito y gasto, y solicitar autoexclusión desde tu perfil.
          Si el juego deja de divertir, pedí ayuda.
        </p>
        <router-link to="/perfil" class="btn btn-ghost btn-sm">Límites y autoexclusión en tu perfil</router-link>
      </section>
    </main>

    <footer class="footer">
      {{ marca.nombre }} · Costa Rica — Certificado de transparencia del mini casino. Juega con responsabilidad · +18.
    </footer>
  </div>
</template>

<style scoped>
.cert-wrap { max-width: 980px; }

/* ---------- certificado ---------- */
.certificado {
  position: relative;
  margin-top: 26px;
  border-radius: 22px;
  background:
    radial-gradient(700px 320px at 50% -10%, rgba(201, 162, 39, .14), transparent 70%),
    linear-gradient(180deg, var(--surface), var(--bg-2));
  border: 1px solid rgba(245, 185, 66, .35);
  box-shadow: var(--shadow);
  overflow: hidden;
}
.certificado::before {
  content: '';
  position: absolute;
  inset: 10px;
  border: 1px dashed rgba(245, 185, 66, .35);
  border-radius: 15px;
  pointer-events: none;
}
.cert-marco { position: relative; padding: 34px 34px 28px; }

.cert-head {
  display: flex;
  gap: 22px;
  align-items: flex-start;
  padding-bottom: 22px;
  border-bottom: 1px solid var(--border-soft);
}
.sello { flex: 0 0 auto; filter: drop-shadow(0 6px 16px rgba(201, 162, 39, .25)); }
.cert-titulos { min-width: 0; }
.cert-titulos h1 { font-size: clamp(26px, 4vw, 36px); margin-top: 4px; }
.cert-sub { color: var(--muted); margin: 8px 0 0; max-width: 620px; }
.cert-badges { display: flex; gap: 8px; flex-wrap: wrap; margin-top: 14px; }

.cert-alcance {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  margin-top: 20px;
}
.alcance {
  border: 1px solid var(--border-soft);
  border-left: 3px solid var(--prim);
  border-radius: 12px;
  padding: 14px 16px;
  background: rgba(16, 185, 129, .05);
}
.alcance.no { border-left-color: var(--gold); background: rgba(245, 185, 66, .05); }
.alcance h2 { font-size: 14px; text-transform: uppercase; letter-spacing: .1em; margin-bottom: 8px; }
.alcance.no h2 { color: var(--gold); }
.alcance ul { margin: 0; padding-left: 18px; color: var(--muted); font-size: 13.5px; display: grid; gap: 6px; }
.alcance b { color: var(--text); }

/* ---------- secciones ---------- */
section { margin-top: 44px; }
section h2 { font-size: clamp(20px, 3vw, 26px); }
.seccion-nota { color: var(--muted); margin: 8px 0 18px; max-width: 720px; }

/* ---------- pasos ---------- */
.pasos-grid {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  counter-reset: paso;
}
.paso {
  background: linear-gradient(180deg, var(--surface), var(--bg-2));
  border: 1px solid var(--border-soft);
  border-radius: 14px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.paso-arte {
  height: 104px;
  display: grid;
  place-items: center;
  border-radius: 11px;
  background: radial-gradient(circle at 50% 35%, rgba(16, 185, 129, .10), transparent 70%);
  border: 1px solid var(--border-soft);
  margin-bottom: 4px;
}
.paso h3 { font-size: 15.5px; }
.paso p { margin: 0; color: var(--muted); font-size: 13px; }

/* ---------- verificador ---------- */
.verificador {
  border: 1px solid rgba(245, 185, 66, .28);
  border-radius: 18px;
  background:
    radial-gradient(600px 260px at 100% 0%, rgba(16, 185, 129, .08), transparent 70%),
    linear-gradient(180deg, var(--surface), var(--bg-2));
  padding: 24px;
}
.verif-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 20px;
  margin-top: 6px;
}
.verif-datos, .verif-hmac { min-width: 0; }
.verif-grid h3 { font-size: 15px; margin-bottom: 10px; }
.verif-datos dl { margin: 0; display: grid; gap: 10px; }
.verif-datos dl > div {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-soft);
}
.verif-datos dt { font-size: 11px; text-transform: uppercase; letter-spacing: .08em; color: var(--faint); }
.verif-datos dd { margin: 0; color: var(--text); font-size: 13.5px; }
.formula {
  font-size: 12.5px;
  color: var(--gold);
  background: var(--bg-2);
  border: 1px solid var(--border-soft);
  border-radius: 9px;
  padding: 8px 10px;
  overflow-x: auto;
}
.hmac {
  display: block;
  font-size: 12.5px;
  color: var(--prim-2);
  background: var(--bg-2);
  border: 1px solid var(--border-soft);
  border-radius: 9px;
  padding: 10px 12px;
}
.derivado { color: var(--muted); font-size: 13px; margin: 12px 0 0; }
.gold { color: var(--gold); }
.comando {
  margin: 14px 0 0;
  background: #05080d;
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  padding: 10px 12px;
  overflow-x: auto;
}
.comando code {
  font-family: ui-monospace, monospace;
  font-size: 12px;
  color: #cfe3da;
  white-space: pre-wrap;
  word-break: normal;
  overflow-wrap: anywhere;
}
.nota { color: var(--muted); font-size: 12.5px; margin: 12px 0 0; }
.verif-hmac .btn { margin-top: 16px; }

/* ---------- rtp / limitaciones ---------- */
.rtp .table-wrap { border-radius: 12px; }
.rtp .nota { margin-top: 10px; }
.limitaciones ul {
  margin: 0;
  padding-left: 20px;
  color: var(--muted);
  font-size: 14px;
  display: grid;
  gap: 8px;
  max-width: 780px;
}

/* ---------- juego responsable ---------- */
.jr {
  border: 1px solid var(--border-soft);
  border-radius: 14px;
  background: rgba(239, 68, 68, .05);
  padding: 18px 20px;
  display: grid;
  gap: 8px;
}
.jr p { margin: 0; color: var(--muted); font-size: 13.5px; max-width: 720px; }
.jr .btn { justify-self: start; }

.footer { margin-top: 46px; }

/* ---------- utilidades ---------- */
.mono { font-family: ui-monospace, monospace; font-weight: 500; }
.break { word-break: break-all; overflow-wrap: anywhere; }
.acciones .btn { white-space: nowrap; }

a:focus-visible, button:focus-visible {
  outline: 2px solid var(--gold);
  outline-offset: 2px;
}

/* ---------- responsive ---------- */
@media (max-width: 980px) {
  .pasos-grid { grid-template-columns: repeat(2, 1fr); }
  .verif-grid { grid-template-columns: minmax(0, 1fr); }
}

@media (max-width: 760px) {
  .cert-marco { padding: 22px 18px; }
  .cert-head { flex-direction: column; gap: 12px; }
  .cert-alcance { grid-template-columns: 1fr; }
  .pasos-grid { grid-template-columns: 1fr; }
  .verificador { padding: 18px; }
  .nav-publica { flex-wrap: wrap; gap: 10px; }
  .nav-publica .acciones { margin-left: 0; width: 100%; }
}
</style>
