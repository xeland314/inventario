# 04 — Etiquetas UUID y escáner

## Problema
- `codigo_activo` humano no sirve para pegatina escaneable; no hay pool pre-impreso.
- Necesidad: generar UUIDs sin dispositivo, imprimir QR, pegar, luego asociar y escanear inventario.

## Solución
### Modelo `backend/internal/models/models.go:44`
`Etiqueta{id, version(4|7), estado('disponible','impreso','asignado','anulado'), dispositivo_id UNIQUE, codigo_barras UNIQUE, generado_en, impreso_en, asignado_en}`

### Schema `backend/internal/db/schema.go:86`
`etiquetas(id PK, version CHECK(4,7), estado CHECK, dispositivo_id UNIQUE FK, codigo_barras UNIQUE)` + índices `idx_etiquetas_estado/dispositivo`.

### Generación UUIDv7 `backend/internal/handlers/etiquetas.go:23`
- `newUUIDv7()`: 48-bit timestamp ms big-endian + version `0111` + variant `10` + 74-bit random `crypto/rand`; formato `8-4-4-4-12`. Sin dependencia externa, ordenable por tiempo (mejor índice para lotes grandes vs v4 random).
- `generarCodigoBarras()`: `ET-` + últimos 8 hex (random, evita colisión por timestamp).
- `POST /api/etiquetas/generar {cantidad:1..500, version:7|4}` en transacción con retry en colisión UNIQUE; `time.Sleep(ms)` cada 10 para unicidad ms.

### Endpoints `backend/main.go:100`
- `POST /api/etiquetas/generar`
- `GET /api/etiquetas?estado=&version=&limit=&offset=`
- `GET /api/etiquetas/{id}` + `GET /api/etiquetas/scan/{id}` (acepta `id` o `codigo_barras` ET-XXXX) → `{etiqueta, dispositivo}`
- `POST /api/etiquetas/{id}/asociar {dispositivo_id}` valida `disponible/impreso` y unicidad dispositivo
- `POST /api/etiquetas/{id}/desasociar` → `disponible`
- `POST /api/etiquetas/{id}/impreso` / `anular` / `DELETE`
- `GET /api/dispositivos/{id}` ahora incluye `etiqueta` (`handlers/dispositivos.go:144` join).

### Frontend
- `frontend/src/components/Escaner.jsx:1` usa `html5-qrcode` ( `Html5Qrcode` con `facingMode: environment`, `fps:10, qrbox:250`) import dinámico, `onScan` + `onDeviceFound` fetch `/api/etiquetas/scan/{uuid}`, vibra 120ms.
- `frontend/src/components/EtiquetasManager.jsx:1` genera lote (selector v7/v4), filtros `disponible/impreso/asignado/anulado`, `Escaner` integrado, form asociar manual (`etiquetaId + dispositivoId`), grid imprimible con `QRCodeSVG` (`qrcode.react`), `codigo_barras`, estado, botones `Marcar impreso`/`Desasociar`/`Anular`, `window.print()` con `@media print`.
- `frontend/src/pages/etiquetas.astro` y `escaner.astro` + nav `Layout.astro:19`.
- `frontend/src/components/DispositivoDetail.jsx:423` sección `🏷️ Etiqueta UUID`: muestra QR (`api.qrserver.com` fallback), `ET-XXXX`, o form asociar con input mono y link a escáner.

### Usabilidad
1. Generar 50 v7 → Imprimir → Pegar en laptops `GGM-LAP-020`, desktops.
2. En campo, abrir `/escaner` o sección etiqueta en detalle → `Iniciar cámara` → apuntar QR → auto-asocia o navega a `/dispositivos/{id}`.
3. Inventario: escanear etiqueta sin asignar muestra "Sin asignar. Asóciela."

### Tests `backend/internal/handlers/inventario_test.go:554`
`TestEtiquetasGenerarV7YV4`, `GenerarInvalido`, `AsociarYDesasociar` (verifica `etiqueta` en `GET /api/dispositivos`), `AsociarConflictos` (409 si ya asignada o dispositivo con etiqueta), `ImpresoYAnulado`, `MigracionPreserva`.

### Migración
`InitDB` crea `etiquetas` si no existe; preserva 10 dispositivos.
