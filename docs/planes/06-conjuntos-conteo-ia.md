# Plan 06 — Conjuntos por caja + Contador IA local

> Tornillos, resistores, tuercas: se etiqueta **la caja**, no cada pieza. Patrón `frascos` (stock + libro) generalizado + QR del pool existente + conteo asistido por IA.

## Objetivo

Dos modos de stock ya existen en el sistema:

| Modo | Entidad | Lógica | QR |
|---|---|---|---|
| Activo único | `dispositivos` | 1 fila = 1 objeto con identidad (serie, código activo) | 1 por unidad (1:1) |
| Cantidad contable | `frascos` | 1 fila = stock de ítems idénticos con cantidad + libro | sin QR |

Un conjunto es del **segundo modo**: 1 fila = un tipo de ítem idéntico dentro de una caja/organizador, con `cantidad_actual`, mínimo de alerta y libro de movimientos — más **una etiqueta QR del pool** pegada en la caja física. Escanear la caja → ficha con qué hay, cuánto queda y si está bajo mínimo.

No se añade `cantidad` a `dispositivos`: contaminaría la lógica de activo único (serie/código activo sin sentido, `toma_física` cuenta unidades, `movimientos` es de jerarquía `parent_id`). La decisión de diseño ya tomada con frascos aplica igual.

## Esquema

```sql
CREATE TABLE IF NOT EXISTS conjuntos (
  id TEXT PRIMARY KEY,
  codigo TEXT UNIQUE NOT NULL,             -- CJ-XXXXXXXX (auto si vacío)
  nombre TEXT NOT NULL,                    -- "Tornillos M3x10 acero inox"
  categoria TEXT,                         -- tornillos, resistores, tuercas... (texto libre)
  especificaciones TEXT,
  unidad TEXT NOT NULL DEFAULT 'pieza' CHECK (unidad IN ('pieza','gr','ml','m','paquete')),
  cantidad_actual INTEGER NOT NULL DEFAULT 0 CHECK (cantidad_actual >= 0),
  cantidad_minima INTEGER NOT NULL DEFAULT 0 CHECK (cantidad_minima >= 0),
  ubicacion TEXT,                          -- "Caja organizadora A, cajón 3"
  sede TEXT,
  proveedor TEXT,
  costo_unitario REAL NOT NULL DEFAULT 0,
  creado_en TEXT NOT NULL DEFAULT (datetime('now')),
  actualizado_en TEXT
);
CREATE TABLE IF NOT EXISTS conjunto_movimientos (  -- clon de frasco_movimientos
  id TEXT PRIMARY KEY,
  conjunto_id TEXT NOT NULL REFERENCES conjuntos(id) ON DELETE CASCADE,
  tipo TEXT NOT NULL CHECK (tipo IN ('ingreso','consumo','ajuste','traslado','baja')),
  cantidad INTEGER NOT NULL CHECK (cantidad != 0),
  cantidad_previa INTEGER NOT NULL,
  cantidad_nueva INTEGER NOT NULL,
  realizado_por TEXT,
  nota TEXT,
  fecha TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS conteos (        -- historial IA/humano por imagen
  id TEXT PRIMARY KEY,
  conjunto_id TEXT NOT NULL REFERENCES conjuntos(id) ON DELETE CASCADE,
  foto_evidencia_id TEXT REFERENCES fotos_evidencia(id) ON DELETE SET NULL,
  cantidad_detectada INTEGER NOT NULL DEFAULT 0,
  cantidad_confirmada INTEGER NOT NULL CHECK (cantidad_confirmada >= 0),
  modelo TEXT NOT NULL DEFAULT 'manual',    -- browser:owlvit-base-patch32 | ollama:qwen2.5vl | manual
  detecciones_json TEXT,                    -- cajas [{box,score,label}] para stats
  nota TEXT,
  creado_por TEXT,
  creado_en TEXT NOT NULL DEFAULT (datetime('now'))
);
-- etiquetas del pool reutilizadas para cajas:
ALTER TABLE etiquetas ADD COLUMN conjunto_id TEXT REFERENCES conjuntos(id) ON DELETE SET NULL;
CREATE UNIQUE INDEX idx_etiquetas_conjunto ON etiquetas(conjunto_id) WHERE conjunto_id IS NOT NULL;
```

Migración cautelosa (estilo `InitDB` actual): `CREATE TABLE IF NOT EXISTS` + `ALTER TABLE ... ADD COLUMN` idempotente + índice parcial UNIQUE (SQLite no permite `ADD COLUMN` con UNIQUE inline). Rebuild de `auditoria` (patrón existente del plan 05) para añadir entidad `'conjunto'` al CHECK. `fotos_evidencia` es polimórfica: fotos del contenido con `entidad_tipo='conjunto'` sin cambios.

## API

| Método | Ruta | Rol | Descripción |
|---|---|---|---|
| POST | /api/conjuntos | tecnico | `{nombre*, categoria?, especificaciones?, unidad?, cantidad_actual?, cantidad_minima?, ubicacion?, sede?, proveedor?, costo_unitario?, codigo?, etiqueta_id?}` → 201. Código auto `CJ-XXXXXXXX`. Etiqueta opcional atómica (pool disponible/impreso). Ingreso "stock inicial" en libro si cantidad > 0. |
| GET | /api/conjuntos | lector | `?q=&categoria=&sede=&bajo=1`. LEFT JOIN etiquetas → `etiqueta_id`, `etiqueta_codigo` (ET-XXXX). Estado derivado: `agotado` (0) / `bajo` (≤ mín) / `disponible`. |
| GET/PUT/PATCH/DELETE | /api/conjuntos/{id} | lector / tecnico | Detalle + `movimientos[]` + `conteos[]` + `etiqueta`. PUT no permite `cantidad_actual` (solo vía libro). |
| POST | /api/conjuntos/{id}/movimiento | tecnico | `{tipo: ingreso|consumo|ajuste|traslado|baja, cantidad, realizado_por?, nota?}` transaccional con `cantidad_previa/nueva` (clon frascos). |
| GET | /api/conjuntos/{id}/movimientos | lector | libro (100 últimos). |
| POST | /api/conjuntos/{id}/conteos | tecnico | `{cantidad_detectada?, cantidad_confirmada*, modelo?, detecciones?, foto_evidencia_id?, nota?, aplicar_ajuste?}` → si `aplicar_ajuste` y delta ≠ 0, crea `movimiento tipo=ajuste` atómico. |
| GET | /api/conjuntos/{id}/conteos | lector | historial (permite medir precisión IA: detectada vs confirmada). |
| POST | /api/etiquetas/{id}/asociar | tecnico | ahora acepta `{dispositivo_id}` **o** `{conjunto_id}` (exclusivos). Valida caja sin etiqueta previa (409). |
| GET | /api/etiquetas/scan/{id} | lector | si `conjunto_id` seteado → retorna `conjunto` (el `Escaner.tsx` redirige a `/conjuntos`). |
| GET | /api/ia/config | lector | `{vlm: {disponible, tipo, modelo}, navegador: {disponible}}`. |
| POST | /api/ia/identificar | lector | `{foto_evidencia_id, prompt?}` → proxy al VLM local (ver IA local). |

Auditoría: entidad `conjunto`, acciones existentes (`crear/actualizar/eliminar/asociar`); conteo → accion `crear` campo `conteo`.

## IA local — dos vías, mismo destino (`conteos`)

**Vía 1 — Navegador (principal, cero backend):** `ContadorIA.tsx` captura con `getUserMedia` (patrón `EncryptedGallery`) o sube imagen → **transformers.js** (ONNX Runtime Web, WebGPU con fallback WASM) con pipeline `zero-shot-object-detection` y **OWL-ViT** (`Xenova/owlvit-base-patch32`): búsqueda por texto ("screw", "resistor") sin entrenamiento. La lib se carga por CDN con `/* @vite-ignore */` (evita empaquetar onnxruntime-web); el modelo (~100MB cuantizado) se cachea en el navegador tras la primera vez. Umbral de confianza ajustable (default 0.15). Overlay con cajas sobre la foto: tocar para descartar falsos positivos. Conteo final **siempre confirmado por humano** (stepper +/−).

**Vía 2 — Proxy VLM Ollama/llama.cpp (mejor identificación):** env `VLM_URL` (ej. `http://localhost:11434`), `VLM_MODEL` (ej. `qwen2.5vl:7b`), `VLM_TYPE` (`ollama`|`openai`). El backend descifra la foto (`fotos_evidencia` está cifrada en disco), la envía al VLM con prompt de JSON estricto `{descripcion, cantidad_estimada, items:[{nombre,cantidad}], confianza}` y devuelve el resultado al mismo flujo de confirmación. Proxy en backend (no directo del navegador) por CORS de Ollama, cifrado de la foto y URL/modelo en `.env`. Si no está configurado → `/api/ia/config` dice `disponible: false` y la UI oculta la opción.

**Expectativa honesta:** <100 objetos separados funciona bien; montones densos de cientos son difíciles para cualquier modelo. Por eso: IA propone → humano confirma → `conteos` guarda detectada vs confirmada para medir precisión real. Fase futura si OWL-ViT no da con las piezas: fine-tune de YOLO pequeño con fotos propias.

## Frontend

- `conjuntos.astro` + `ConjuntosManager.tsx` — clon de patrón `FrascosManager`: cards con stock/estado, badge bajo-stock, modal de movimiento, historial, `EncryptedGallery entidadTipo="conjunto"`, asociación de etiqueta (pegar UUID o ET-XXXX), botón **Contar (IA)**.
- `ContadorIA.tsx` — cámara/subida + detectar (navegador) + identificar (VLM si hay) + confirmar + guardar (foto cifrada + conteo + ajuste opcional).
- Nav "Conjuntos" en `Layout.astro` (desktop + móvil) · `Escaner.tsx` muestra conjuntos al escanear la caja.
- Sin página de detalle aparte: todo en cards (patrón frascos, no requiere SPA fallback).

## Tests (`conjuntos_test.go`, con `newConjuntosRouter`)

- `TestConjuntoCrearListarBajo` — crear tornillos M3 (500, mín 50): `?bajo=1` vacío; consumo 480 → aparece.
- `TestConjuntoMovimientoAritmetica` — previa/nueva correctas, stock insuficiente → 400, `cantidad_actual` no editable por PUT.
- `TestConjuntoEtiquetaAsociarScan` — generar etiqueta, asociar a conjunto, `scan` retorna `conjunto`; etiqueta ya asignada → 409; caja con etiqueta previa → 409; desasociar libera.
- `TestConjuntoConteosAjuste` — conteo con `aplicar_ajuste` (confirmada 480 vs actual 500) → ajuste −20 en libro + historial visible.
- `TestConjuntoRoles` — lector GET 200, POST 403.
- `TestConjuntoEliminar` — cascada de libro/conteos, etiqueta liberada a `disponible`.

## Riesgos

- OWL-ViT con objetos densos/pequeños: precision limitada → confirmación humana obligatoria y registro de delta para estadísticas.
- Primera carga de la lib/modelo por CDN requiere internet una vez; luego cacheado. Futuro offline total: servir modelos desde `/models/` del backend + `env.localModelPath`.
- `categoria`/`unidad` texto libre con CHECK de unidad acotado (pieza/gr/ml/m/paquete).
- El rebuild de `auditoria` sigue el patrón ya probado del plan 05 (backup implícito WAL + INSERT SELECT).

## Commit

`feat(conjuntos): stock por caja con QR + conteo IA local (navegador/VLM)`

Archivos: `backend/internal/db/schema.go`, `backend/internal/models/models.go`, `backend/internal/handlers/conjuntos.go` (nuevo), `backend/internal/handlers/ia.go` (nuevo), `backend/internal/handlers/etiquetas.go`, `backend/internal/handlers/dispositivos.go`, `backend/main.go`, `backend/internal/handlers/conjuntos_test.go`, `frontend/src/pages/conjuntos.astro`, `frontend/src/components/ConjuntosManager.tsx`, `frontend/src/components/ContadorIA.tsx`, `frontend/src/components/Escaner.tsx`, `frontend/src/layouts/Layout.astro`, `README.md`, `docs/API.md`.
