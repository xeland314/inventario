# Plan 03 — Mantenimiento planificado + ICS

> Sin cron. Cálculo al consultar + feed .ics para Google Calendar.

## Objetivo
Hoy `mantenimientos` solo es historial. Falta **qué va a pasar**: planes periódicos
- por dispositivo (ej. `GGM-SRV-01` pasta térmica cada 365d)
- por tipo (todos `UPS` cada 180d, `Impresora` cada 90d)
Al registrar un mantenimiento real se recalcula `proxima_fecha = fecha_real + cada_dias` (cuenta desde real, no desde planificado → si hay retraso no queda vencido inmediato).

## Esquema

```sql
CREATE TABLE IF NOT EXISTS planes_mantenimiento (
  id TEXT PRIMARY KEY,
  dispositivo_id TEXT REFERENCES dispositivos(id) ON DELETE CASCADE,
  tipo TEXT, -- si dispositivo_id NULL, aplica a todos de este tipo (valida EsTipoValido si no NULL)
  cada_dias INTEGER NOT NULL CHECK (cada_dias > 0 AND cada_dias <= 3650),
  proxima_fecha TEXT NOT NULL, -- YYYY-MM-DD
  descripcion TEXT NOT NULL,
  activo INTEGER NOT NULL DEFAULT 1,
  creado_en TEXT NOT NULL DEFAULT (datetime('now')),
  actualizado_en TEXT,
  creado_por TEXT REFERENCES usuarios(id) ON DELETE SET NULL,
  CHECK (dispositivo_id IS NOT NULL OR tipo IS NOT NULL OR descripcion != '')
);
CREATE INDEX IF NOT EXISTS idx_planes_dispositivo ON planes_mantenimiento(dispositivo_id);
CREATE INDEX IF NOT EXISTS idx_planes_tipo ON planes_mantenimiento(tipo);
CREATE INDEX IF NOT EXISTS idx_planes_proxima ON planes_mantenimiento(proxima_fecha);
CREATE INDEX IF NOT EXISTS idx_planes_activo ON planes_mantenimiento(activo);
```

`tipo` valida `EsTipoValido` si no vacío (permite `UPS` etc). `cada_dias` 1..3650.

## API

| Método | Ruta | Rol | Descripción |
|---|---|---|---|
| GET | /api/planes | lector/tecnico | `?dispositivo_id=&tipo=&activo=&q=` lista. |
| POST | /api/planes | tecnico | `{dispositivo_id?, tipo?, cada_dias, proxima_fecha (YYYY-MM-DD), descripcion, activo?}` → 201. Valida al menos uno de dispositivo_id/tipo/descripcion, valida dispositivo existe, tipo válido, cada_dias. |
| GET | /api/planes/{id} | lector/tecnico | detalle + `dispositivo` join si aplica |
| PUT/PATCH | /api/planes/{id} | tecnico | actualiza parcial |
| DELETE | /api/planes/{id} | tecnico | borra |
| GET | /api/planes/proximos | lector/tecnico | `?dias=30&incluye_vencidos=true` calcula en Go: `hoy=YYYY-MM-DD`, `vencidos: proxima_fecha < hoy`, `proximos: hoy <= proxima_fecha <= hoy+dias`. Sin cron. |
| GET | /api/planes/vencidos | lector/tecnico | alias `proximos` con `dias=0` + vencidos |
| GET | /api/planes/calendario.ics | lector/tecnico | feed ICS `text/calendar` suscribible: `BEGIN:VCALENDAR`, `VEVENT` por cada plan activo con `DTSTART:proxima_fecha`, `RRULE:FREQ=DAILY` no — solo evento único + `DURATION`, `SUMMARY`, `DESCRIPTION` con `cada_dias`. Usa `cada_dias` solo informativo; repetición se maneja al recalcular. |
| POST | /api/mantenimientos | tecnico | existente, ahora además: tras insertar historial, busca planes activos `WHERE (dispositivo_id=? OR tipo=?) AND activo=1` y actualiza `proxima_fecha = fecha_real + cada_dias` y `actualizado_en`. Audita `actualizar` campo `proxima_fecha`. |

## Lógica
- Al crear mantenimiento: `fecha_real = COALESCE(req.fecha_hora, today)` parse `YYYY-MM-DD` o `RFC3339`; para cada plan matching, `nueva = fecha_real + cada_dias`; `UPDATE planes_mantenimiento SET proxima_fecha = nueva`.
- Si `proxima_fecha` ya vencida y se registra mantenimiento tardío, nueva queda en futuro → no queda vencido doble.
- `GET proximos` no escribe DB, solo filtra.

## Tests

Archivo `planes_test.go` (o parte de `auditoria_test.go` renombrado):
- `TestPlanesCrearPorTipoYDispositivo` — crear plan tipo UPS 180d, plan dispositivo específico, GET list filtra.
- `TestPlanesValidacion` — cada_dias 0 → 400, tipo inválido →400, dispositivo inexistente →404.
- `TestPlanesProximosYVencidos` — crear plan vencido (ayer) y proximo (hoy+5), GET /api/planes/proximos?dias=30 debe incluir ambos; vencidos solo ayer.
- `TestMantenimientoRecalculaProxima` — crear plan con proxima `2026-01-01` cada 90d, luego POST mantenimiento `2026-01-10` → plan proxima debe ser `2026-04-10` (real+90), no `2026-04-01`.
- `TestICSFeed` — GET calendario.ics contiene `BEGIN:VCALENDAR` y `SUMMARY`.
- `TestPlanesRoles` — lector GET 200, POST 403.
- `TestAuditoriaPlan` — crear plan genera auditoria `entidad=plan accion=crear`.

Usar `newPlanesRouter` con `AuthHandler.Middleware` + auditoria.

## Frontend

Mínimo en este plan: API listo. Futuro 04 Dashboard consumirá `/api/planes/proximos`. No bloquea.

## Riesgos
- `proxima_fecha` como TEXT `YYYY-MM-DD` lexical compara con `date('now')` en Go, no SQLite `date()`.
- `cada_dias` hasta 3650 (10 años).
- ICS no recurrente automático; cliente recalcula al registrar.

## Commit
`feat(planes): mantenimiento planificado + recalculo + proximos/vencidos + ICS`

Archivos: `backend/internal/db/schema.go`, `backend/internal/models/models.go:PlanMantenimiento`, `backend/internal/handlers/planes.go`, `backend/internal/handlers/mantenimientos.go` (hook recalc), `backend/main.go`, `backend/internal/handlers/planes_test.go`, `docs/planes/03-planes-mantenimiento.md`
