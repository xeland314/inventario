# Plan 05 — Toma física de inventario por sede

> Con escáner + etiquetas ya existentes, costo casi 0.

## Objetivo
Abrir sesión “Inventario Icon 2026-09”, escanear QRs en campo y al cerrar comparar:
- **faltantes** → esperados en sede pero no escaneados (candidatos a `extraviado`)
- **sobrantes** → escaneados pero no esperados para esa sede (otra sede o sin sede)
- **correctos** → escaneados que sí estaban esperados

Sin `sede`/`ubicacion` normalizadas (texto libre) el cálculo usa `COALESCE(sede,'Sin sede')` exact match.

## Esquema

```sql
CREATE TABLE IF NOT EXISTS tomas_fisicas (
  id TEXT PRIMARY KEY,
  sede TEXT NOT NULL,
  descripcion TEXT,
  estado TEXT NOT NULL CHECK (estado IN ('abierto','cerrado')) DEFAULT 'abierto',
  creado_por TEXT REFERENCES usuarios(id) ON DELETE SET NULL,
  creado_en TEXT NOT NULL DEFAULT (datetime('now')),
  cerrado_en TEXT,
  cerrado_por TEXT REFERENCES usuarios(id) ON DELETE SET NULL
);
CREATE TABLE IF NOT EXISTS toma_items (
  id TEXT PRIMARY KEY,
  toma_id TEXT NOT NULL REFERENCES tomas_fisicas(id) ON DELETE CASCADE,
  etiqueta_id TEXT REFERENCES etiquetas(id) ON DELETE SET NULL,
  dispositivo_id TEXT REFERENCES dispositivos(id) ON DELETE SET NULL,
  escaneado_en TEXT NOT NULL DEFAULT (datetime('now')),
  escaneado_por TEXT REFERENCES usuarios(id) ON DELETE SET NULL,
  UNIQUE(toma_id, etiqueta_id),
  UNIQUE(toma_id, dispositivo_id)
);
CREATE INDEX IF NOT EXISTS idx_tomas_sede ON tomas_fisicas(sede);
CREATE INDEX IF NOT EXISTS idx_tomas_estado ON tomas_fisicas(estado);
CREATE INDEX IF NOT EXISTS idx_toma_items_toma ON toma_items(toma_id);
```

Esperados = `SELECT id, codigo_activo FROM dispositivos WHERE COALESCE(sede,'Sin sede') = :sede AND estado != 'dado_de_baja'` (no incluye baja).

## API

| Método | Ruta | Rol | Descripción |
|---|---|---|---|
| POST | /api/tomas | tecnico | `{sede, descripcion?}` → 201 `{id}`. `sede` requerida, trim. |
| GET | /api/tomas | lector/tecnico | `?sede=&estado=` lista. |
| GET | /api/tomas/{id} | lector/tecnico | detalle + `esperados_count`, `escaneados_count`, `faltantes`, `sobrantes`, `correctos` calculados al vuelo. Respuesta incluye arrays de dispositivos (id,codigo_activo,estado) para faltantes/sobrantes/correctos limitado a 100 cada uno + totals. |
| POST | /api/tomas/{id}/escanear | lector/tecnico | `{etiqueta_id?, dispositivo_id?, codigo_barras?}` al menos uno. Resuelve `dispositivo_id` vía etiqueta `dispositivo_id` o directa. Inserta en `toma_items` (ignora duplicado). Si toma `cerrado` → 400. Valida etiqueta/dispositivo existe. 201. |
| POST | /api/tomas/{id}/cerrar | tecnico | cambia `abierto→cerrado`, `cerrado_en/por`, audita. Idempotente. |
| DELETE | /api/tomas/{id} | tecnico | solo si `abierto` |
| GET | /api/tomas/{id}/faltantes | lector/tecnico | solo faltantes |
| GET | /api/tomas/{id}/sobrantes | lector/tecnico | solo sobrantes |

Resolver QR: acepta `etiqueta_id` (UUID), `codigo_barras` (`ET-XXXX`), o `dispositivo_id`. Prioridad etiqueta primero. Si etiqueta asignada, usa su `dispositivo_id`; si no asignada, guarda solo `etiqueta_id` y `dispositivo_id=NULL` (no cuenta como correcto, va a sobrantes).

## Lógica comparativa (sin escritura al cerrar)

Para GET `/api/tomas/{id}`:
```
esperados_ids = SELECT id FROM dispositivos WHERE sede = toma.sede AND estado != 'dado_de_baja'
escaneados = SELECT dispositivo_id FROM toma_items WHERE toma_id = ? AND dispositivo_id IS NOT NULL
correctos = escaneados ∩ esperados
faltantes = esperados - escaneados
sobrantes = escaneados - esperados  (más items con etiqueta sin dispositivo van a sobrantes_etiquetas)
```
Sobrantes incluye también `toma_items` con `dispositivo_id IS NULL` (etiqueta sin asignar escaneada).

## Tests

`toma_test.go`:

- `TestTomaCrearListar` — tecnico crea toma Icon, lector lista.
- `TestTomaEscanearYComparar` — crear 3 dispositivos Icon (A,B,C) y 1 Matriz (X), crear toma Icon, escanear A,B (correctos) + X (sobrante, otra sede) → GET toma: faltantes 1 (C), correctos 2, sobrantes 1.
- `TestTomaCodigoBarras` — escanear via `ET-XXXX` y `dispositivo_id` directo.
- `TestTomaCerrarYCandidatosExtraviado` — al cerrar, faltantes sigue disponible para luego pasar a `extraviado` manual (no auto cambia estado).
- `TestTomaRoles` — lector POST /api/tomas 403, lector escanear 201, lector cerrar 403.
- `TestTomaDuplicadoIgnora` — escanear misma etiqueta 2 veces no duplica.
- `TestAuditoriaToma` — crear toma audita `entidad=toma`.
- `TestTomaFiltradaPorSedeSinDadoDeBaja` — dispositivo dado_de_baja no cuenta como esperado.

Con `newTomaRouter`.

## Frontend

No UI completa en este plan (solo API). Dashboard (04) puede mostrar botón “Iniciar toma”.

## Riesgos
- `sede` texto libre → faltantes falsos si “Icon” vs “icon”. Futuro normalizar sedes.
- Escaneo sin etiqueta asignada no aporta a faltantes, queda en sobrantes para detectar etiquetas huérfanas.

## Commit
`feat(toma): inventario físico por sede (faltantes/sobrantes/correctos)`

Archivos: `backend/internal/db/schema.go`, `backend/internal/models/models.go: TomaFisica`, `backend/internal/handlers/toma.go`, `backend/main.go`, `backend/internal/handlers/toma_test.go`, `docs/planes/05-toma-fisica.md`
