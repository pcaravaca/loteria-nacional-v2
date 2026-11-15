-- Datos de prueba para la base de desarrollo PostgreSQL (loteria_dev).
-- Clave de los usuarios demo: demo1234 (hash bcrypt incluido abajo).
--
-- Uso:
--   docker exec -i <contenedor-pg> psql -U dba -d loteria_dev -v ON_ERROR_STOP=1 < demo-data.sql
--   # o desde WSL:
--   ssh peter@100.89.167.67 'docker exec -i fcfeed-pgdev psql -U dba -d loteria_dev -v ON_ERROR_STOP=1' < demo-data.sql
--
-- Es idempotente: cada bloque inserta solo lo que falta (guardas por email,
-- titulo, descripcion o seed+nonce). Se puede correr varias veces.

BEGIN;

-- 1) Usuarios demo ----------------------------------------------------------------
INSERT INTO usuarios (nombre, email, password_hash, rol, telefono, saldo, bono, casino_nonce, created_at)
SELECT v.nombre, v.email, v.password_hash, v.rol, v.telefono, v.saldo, v.bono, v.casino_nonce, v.creado
FROM (VALUES
  ('Maria Ficticia',  'maria@demo.test',  '$2a$10$iIGXv/gK2eutRVBHyrB1d.gFK/rf2LmmMMZmKO.7hRvQCulZYUhTe', 'cliente', '8888-0001', 150000::bigint, 0::bigint, 0::bigint, now() - interval '20 days'),
  ('Pedro Ficticio',  'pedro@demo.test',  '$2a$10$iIGXv/gK2eutRVBHyrB1d.gFK/rf2LmmMMZmKO.7hRvQCulZYUhTe', 'cliente', '8888-0002',  42000::bigint, 0::bigint, 0::bigint, now() - interval '18 days'),
  ('Ana Ficticia',    'ana@demo.test',    '$2a$10$iIGXv/gK2eutRVBHyrB1d.gFK/rf2LmmMMZmKO.7hRvQCulZYUhTe', 'cliente', '8888-0003',   8900::bigint, 0::bigint, 0::bigint, now() - interval '15 days'),
  ('Carlos Ficticio', 'carlos@demo.test', '$2a$10$iIGXv/gK2eutRVBHyrB1d.gFK/rf2LmmMMZmKO.7hRvQCulZYUhTe', 'agente',  '8888-0004',     0::bigint, 0::bigint, 0::bigint, now() - interval '10 days')
) AS v(nombre, email, password_hash, rol, telefono, saldo, bono, casino_nonce, creado)
WHERE NOT EXISTS (SELECT 1 FROM usuarios u WHERE u.email = v.email);

-- 2) Sorteos ----------------------------------------------------------------------
INSERT INTO sorteos (loteria_id, titulo, fecha, estado, numeros_raw, created_at)
SELECT l.id, v.titulo, v.fecha, 'pagado', v.numeros, v.creado
FROM (VALUES
  ('LN', 'Lotería Nacional · Sorteo demo 4821',  now() - interval '3 days', '[7,21,42]'::text, now() - interval '4 days'),
  ('LN', 'Lotería Nacional · Sorteo demo 4822',  now() - interval '1 day',  '[3,9,17]'::text,  now() - interval '2 days'),
  ('QN', 'Quiniela Nacional · Sorteo demo 4821', now() - interval '2 days', '[12,5]'::text,   now() - interval '3 days')
) AS v(codigo_loteria, titulo, fecha, numeros, creado)
JOIN loterias l ON l.codigo = v.codigo_loteria
WHERE NOT EXISTS (SELECT 1 FROM sorteos s WHERE s.titulo = v.titulo);

INSERT INTO sorteos (loteria_id, titulo, fecha, estado, numeros_raw, created_at)
SELECT l.id, v.titulo, v.fecha, 'abierto', NULL, v.creado
FROM (VALUES
  ('LN', 'Lotería Nacional · Sorteo proximo 4823', now() + interval '2 days', now()),
  ('QD', 'Quiniela Deportiva · Jornada proxima',    now() + interval '5 days', now())
) AS v(codigo_loteria, titulo, fecha, creado)
JOIN loterias l ON l.codigo = v.codigo_loteria
WHERE NOT EXISTS (SELECT 1 FROM sorteos s WHERE s.titulo = v.titulo);

-- 3) Apuestas ---------------------------------------------------------------------
INSERT INTO apuestas (user_id, sorteo_id, numeros_raw, monto, estado, aciertos, premio, created_at)
SELECT u.id, s.id, v.numeros, v.monto, v.estado, v.aciertos, v.premio, v.creado
FROM (VALUES
  ('maria@demo.test', 'Lotería Nacional · Sorteo demo 4821', '[7,21,88]'::text,  1000::bigint, 'ganadora', 2::bigint,  5000::bigint, now() - interval '3 days' + interval '2 hours'),
  ('maria@demo.test', 'Lotería Nacional · Sorteo demo 4822', '[3,9,20]'::text,   1000::bigint, 'ganadora', 2::bigint,  5000::bigint, now() - interval '1 day'  + interval '2 hours'),
  ('pedro@demo.test', 'Lotería Nacional · Sorteo demo 4821', '[1,2,3]'::text,     500::bigint, 'perdedora', 0::bigint,     0::bigint, now() - interval '3 days' + interval '3 hours'),
  ('ana@demo.test',   'Quiniela Nacional · Sorteo demo 4821', '[12,5]'::text,   2000::bigint, 'pagada',   2::bigint, 70000::bigint, now() - interval '2 days' + interval '4 hours'),
  ('maria@demo.test', 'Lotería Nacional · Sorteo proximo 4823', '[5,19,33]'::text, 1500::bigint, 'pendiente', 0::bigint, 0::bigint, now() - interval '1 day'),
  ('pedro@demo.test', 'Quiniela Deportiva · Jornada proxima', '[1,1,2,0]'::text, 2000::bigint, 'pendiente', 0::bigint, 0::bigint, now() - interval '12 hours')
) AS v(email, titulo, numeros, monto, estado, aciertos, premio, creado)
JOIN usuarios u ON u.email = v.email
JOIN sorteos s ON s.titulo = v.titulo
WHERE NOT EXISTS (
  SELECT 1 FROM apuestas a
  JOIN usuarios u2 ON u2.id = a.user_id
  JOIN sorteos s2 ON s2.id = a.sorteo_id
  WHERE u2.email = v.email AND s2.titulo = v.titulo
);

-- 4) Movimientos -------------------------------------------------------------------
INSERT INTO movimientos (user_id, tipo, monto, descripcion, created_at)
SELECT u.id, v.tipo, v.monto, v.descripcion, v.creado
FROM (VALUES
  ('maria@demo.test',  'recarga',        200000::bigint, 'Recarga inicial',              now() - interval '20 days'),
  ('maria@demo.test',  'apuesta',         -1000::bigint, 'Apuesta Lotería Nacional',     now() - interval '3 days' + interval '2 hours'),
  ('maria@demo.test',  'premio',           5000::bigint, 'Premio Lotería Nacional',      now() - interval '3 days'),
  ('maria@demo.test',  'casino_apuesta',  -1000::bigint, 'Casino Keno CR',               now() - interval '26 hours'),
  ('maria@demo.test',  'casino_premio',    3000::bigint, 'Casino Keno CR',               now() - interval '26 hours'),
  ('pedro@demo.test',  'recarga',         50000::bigint, 'Recarga inicial',              now() - interval '18 days'),
  ('pedro@demo.test',  'casino_apuesta',   -500::bigint, 'Casino Dados del Sur',         now() - interval '20 hours'),
  ('pedro@demo.test',  'casino_premio',    1000::bigint, 'Casino Dados del Sur',         now() - interval '20 hours'),
  ('ana@demo.test',    'recarga',         10000::bigint, 'Recarga inicial',              now() - interval '15 days'),
  ('ana@demo.test',    'casino_apuesta',   -800::bigint, 'Casino Rueda Fortuna',         now() - interval '8 hours'),
  ('cliente@loteria.test', 'recarga',    10000::bigint, 'Recarga demo PostgreSQL',      now() - interval '1 day')
) AS v(email, tipo, monto, descripcion, creado)
JOIN usuarios u ON u.email = v.email
WHERE NOT EXISTS (
  SELECT 1 FROM movimientos m JOIN usuarios u2 ON u2.id = m.user_id
  WHERE u2.email = v.email AND m.descripcion = v.descripcion
);

-- 5) Rondas del casino (datos para el historial, stats y reportes) ---------------
-- Cada juego genera 6 rondas repartidas en los ultimos 15 dias; el multiplicador
-- depende del indice (algunas perdedoras, alguna vacia) y el premio es coherente.
INSERT INTO casino_rondas (user_id, game_codigo, monto, multiplicador, premio, client_seed, server_seed_hash, nonce, resultado_raw, created_at)
SELECT u.id, v.codigo_juego, v.monto, v.mult, (v.monto * v.mult)::bigint, v.client_seed, v.server_hash, v.nonce, v.resultado, v.creado
FROM (
  SELECT
    i AS idx,
    CASE WHEN i % 2 = 0 THEN 'maria@demo.test' ELSE 'pedro@demo.test' END AS email,
    j.codigo AS codigo_juego,
    (j.minimo * ((i % 4) + 1))::bigint AS monto,
    CASE WHEN i % 4 = 0 THEN 0 ELSE (i % 3) END::bigint AS mult,
    'demo-seed-' || i AS client_seed,
    md5(j.codigo || i) AS server_hash,
    i AS nonce,
    CASE j.codigo
      WHEN 'dados'  THEN jsonb_build_object('dado1', 1 + (i % 6), 'dado2', 1 + ((i * 3) % 6), 'suma', 2 + (i % 6) + ((i * 3) % 6), 'tipo', 'suma')::text
      WHEN 'slots'  THEN jsonb_build_object('rodillos', ARRAY[ARRAY[1 + (i % 6), 1 + ((i + 2) % 6), 1 + ((i + 4) % 6)]], 'combinacion', ARRAY['siete','siete','campana'])::text
      WHEN 'ruleta' THEN jsonb_build_object('numero', (i * 7) % 37, 'color', CASE WHEN (i * 7) % 37 IN (1,3,5,7,9,12,14,16,18,19,21,23,25,27,30,32,34,36) THEN 'rojo' ELSE 'negro' END, 'apuesta', 'color')::text
      WHEN 'raspa'  THEN jsonb_build_object('casillas', ARRAY['cereza','cereza','barra'], 'ganadora', i % 4 <> 0)::text
      ELSE jsonb_build_object('tus_numeros', ARRAY[1 + (i % 10), 2 + (i % 10), 3 + (i % 10), 4 + (i % 10), 5 + (i % 10)], 'bolillas', ARRAY[1, 5, 12, 19, 21, 27, 33, 40, 45, 51, 55, 58, 60, 63, 68], 'aciertos', (i % 4))::text
    END AS resultado,
    now() - ((i * 7) || ' hours')::interval AS creado
  FROM generate_series(1, 6) AS g(i)
  CROSS JOIN (VALUES
    ('dados',  200::bigint),
    ('slots',  500::bigint),
    ('ruleta', 500::bigint),
    ('raspa',  200::bigint),
    ('keno',   500::bigint)
  ) AS j(codigo, minimo)
) AS v
JOIN usuarios u ON u.email = v.email
WHERE NOT EXISTS (
  SELECT 1 FROM casino_rondas r
  WHERE r.user_id = u.id AND r.game_codigo = v.codigo_juego AND r.nonce = v.nonce
);

-- 6) Manos de blackjack terminadas -----------------------------------------------
INSERT INTO casino_blackjack (user_id, monto, mano_raw, crupier_raw, estado, multiplicador, premio, client_seed, server_seed_hash, nonce, created_at)
SELECT u.id, v.monto, v.mano, v.crupier, v.estado, v.mult::numeric, v.premio, v.client_seed, v.server_hash, v.nonce, v.creado
FROM (VALUES
  ('maria@demo.test',  1000::bigint, '[10,20]'::text,  '[5,12]'::text,  'ganada',  2::bigint,  2000::bigint, 'demo-bj-1', md5('bj1'), 1::bigint, now() - interval '22 hours'),
  ('pedro@demo.test',   500::bigint, '[4,6]'::text,    '[9,25]'::text,  'perdida', 0::bigint,     0::bigint, 'demo-bj-2', md5('bj2'), 1::bigint, now() - interval '16 hours'),
  ('ana@demo.test',     800::bigint, '[10,13]'::text,  '[10,7]'::text,  'empate',  1::bigint,   800::bigint, 'demo-bj-3', md5('bj3'), 1::bigint, now() - interval '7 hours'),
  ('cliente@loteria.test', 500::bigint, '[10,20]'::text, '[5,12]'::text, 'pagada', 2::bigint,  1000::bigint, 'demo-bj-4', md5('bj4'), 1::bigint, now() - interval '5 hours')
) AS v(email, monto, mano, crupier, estado, mult, premio, client_seed, server_hash, nonce, creado)
JOIN usuarios u ON u.email = v.email
WHERE NOT EXISTS (
  SELECT 1 FROM casino_blackjack b
  WHERE b.user_id = u.id AND b.client_seed = v.client_seed AND b.nonce = v.nonce
);

COMMIT;

-- Resumen de lo sembrado (si alguna guarda hizo SKIP, los conteos bajan)
SELECT 'usuarios' AS tabla, count(*) FROM usuarios
UNION ALL SELECT 'sorteos', count(*) FROM sorteos
UNION ALL SELECT 'apuestas', count(*) FROM apuestas
UNION ALL SELECT 'movimientos', count(*) FROM movimientos
UNION ALL SELECT 'casino_rondas', count(*) FROM casino_rondas
UNION ALL SELECT 'casino_blackjack', count(*) FROM casino_blackjack;
