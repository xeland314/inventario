# Plan 02 — Historial de cambios (audit log)

> Base para dashboard y estadísticas. Sin esto no hay MTTR, frecuencia por marca, ni trazabilidad.

## Objetivo
Registrar **quién cambió qué y cuándo** para todo inventario. Hoy solo `actualizado_en`. Con audit habilitas:
- Tiempo medio en `en_reparacion` / `averiado`
- Frecuencia de fallas por marca/modelo/tipo/sede
- Valor de quién modificó, disputas, auditoría externa
- Dashboard y toma física alimentados

## Esquema

```sql
CREATE TABLE IF NOT EXISTS auditoria (
  id TEXT PRIMARY KEY,
  entidad TEXT NOT NULL CHECK (entidad IN ('dispositivo','movimiento','mantenimiento','etiqueta','reporte','usuario','tinta','evidencia')),
  entidad_id TEXT NOT NULL,
  accion TEXT NOT NULL CHECK (accion IN ('crear','actualizar','eliminar','baja','asociar','desasociar','impreso','anulado','traslado','reportar','cancelar','resolver')),
  campo TEXT,                 -- NULL = acción global (crear/eliminar), valor = campo modificado
  valor_anterior TEXT,
  valor_nuevo TEXT,
  usuario_id TEXT REFERENCES usuarios(id) ON DELETE SET NULL,
  username TEXT,              -- snapshot por si usuario se borra
  fecha TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_auditoria_entidad ON auditoria(entidad, entidad_id);
CREATE INDEX IF NOT EXISTS idx_auditoria_fecha ON auditoria(fecha);
CREATE INDEX IF NOT EXISTS idx_auditoria_usuario ON auditoria(usuario_id);
CREATE INDEX IF NOT EXISTS idx_auditoria_entidad_fecha ON auditoria(entidad, fecha);
```

No `FOREIGN KEY` estricta a `dispositivos(id)` porque auditoría debe sobrevivir a `DELETE`.

## API

| Método | Ruta | Rol | Descripción |
|---|---|---|---|
| GET | /api/auditoria | tecnico, lector* | lectura para ambos (lector solo lectura). Query: `?entidad=dispositivo&entidad_id=xxx&accion=actualizar&campo=estado&usuario_id=&limit=100&offset=0&desde=2026-01-01&hasta=2026-12-31` → `{auditoria:[], total}` orden `fecha DESC`. |
| GET | /api/auditoria/{id} | tecnico, lector | detalle |
| GET | /api/dispositivos/{id}/historial | tecnico, lector | alias filtrado por `entidad=dispositivo && entidad_id=id` |

*Sin escribir desde API; escritura solo vía hooks internos.

## Hooks (dónde registrar)

`backend/internal/handlers/auditoria.go:Helper` `func RegistrarAuditoria(db *sql.DB, r *http.Request, entidad, entidadID, accion, campo, anterior, nuevo string)` extrae `usuario_id/username` vía `GetAuthUser(r)` (si no hay user en test, registra `sistema`).

Integración mínima para este plan:
- `DispositivoHandler.Crear` → `crear` con `campo=NULL` + logs de `codigo_activo`, `estado`, `sede`
- `DispositivoHandler.Actualizar` → por cada campo enviado y distinto al `current`, `actualizar` con `campo`, `anterior`, `nuevo` (ej. `estado: activo→averiado`, `sede: Matriz→Icon`, `parent_id`)
- `DarDeBaja` → `baja` campo `estado` + `motivo_baja`
- `Eliminar` → `eliminar`
- `MovimientoHandler.Crear` → `traslado`/`instalacion` etc en entidad `movimiento` y `dispositivo`
- `EtiquetaHandler.Asociar/Desasociar/Impreso/Anular/Generar` → acciones correspondientes entidad `etiqueta`
- `ReporteHandler.Crear/Cancelar/Resolver` → `reportar/cancelar/resolver`
- `MantenimientoHandler.Crear/Actualizar/Eliminar` → entidad `mantenimiento`
- `TintaHandler.Actualizar` → `actualizar` campo `nivel`

Si `r.Context` no tiene user (tests sin auth), registrar `usuario_id=NULL, username='sistema'` para no romper.

Transacción: auditoría se hace después de `Exec` exitoso, en misma función (no necesita transacción compartida por ahora; si falla auditoría loguea warning pero no revierte operación).

## Frontend

No UI completa en este plan (solo API). Para verificar: `curl -H "Authorization: Bearer <tok>" /api/auditoria?entidad=dispositivo&entidad_id=xxx`

Futuro 04 Dashboard usará `/api/auditoria` para MTTR y frecuencia. Este plan no bloquea frontend.

## Tests `backend/internal/handlers/auditoria_test.go` (o append a auth_test.go)

Helper `newAuditRouter = newAuthRouter + auditoria routes`

- `TestAuditoriaCrearDispositivo` — tecnico crea dispositivo → GET /api/auditoria?entidad=dispositivo&entidad_id=id debe contener `accion=crear`
- `TestAuditoriaActualizarEstado` — PUT estado `activo→averiado` → auditoría `campo=estado, anterior=activo, nuevo=averiado, username=admin` y `fecha` ISO8601
- `TestAuditoriaNoLogSinCambio` — PUT sin cambiar no genera duplicado
- `TestAuditoriaFiltros` — filtrar por `accion` y `usuario_id`
- `TestAuditoriaLectorSoloLectura` — lector GET 200, POST no existe 404/405, auditoría de lector también se registra (reporte)
- `TestAuditoriaReportes` — lector crea reporte → auditoría `entidad=reporte accion=reportar`

Usar `setupTestDB` + `newAuthRouter` extendido con `AuditoriaHandler`. Todos `go test ./internal/handlers -run TestAuditoria -count=1 -v` OK.

## Riesgos / Notas

- Texto `valor_anterior/nuevo` se guarda como string (para `valor_compra` se guarda `fmt.Sprint`). No JSON.
- No borrar auditoría en cascada; retención indefinida.
- Performance: índices cubren filtros habituales; `limit` default 100, max 500.
- Retrocompat: tablas existentes no se migran (solo inserciones nuevas).

## Verificación manual

```bash
go test ./internal/handlers -run TestAuditoria -count=1 -v
curl -X POST /api/dispositivos -H "Authorization: Bearer <tok>" -d '{...}'
curl -H "Authorization: Bearer <tok>" "/api/auditoria?entidad=dispositivo&entidad_id=<id>"
```

## Commit

`feat(audit): historial de cambios (auditoria) con usuario, campo y fecha`

Archivos: `backend/internal/models/models.go` (opcional), `backend/internal/db/schema.go`, `backend/internal/handlers/auditoria.go`, `backend/internal/handlers/dispositivos.go`, `...etiquetas.go, reportes.go, movimientos.go, mantenimientos.go, tintas.go`, `backend/main.go`, `backend/internal/handlers/auditoria_test.go`, `docs/planes/02-audit-log.md`
