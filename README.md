# Inventario de Tecnología — Golang + Astro + TSX (Escalable)

Sistema modular **multi-sede** con base relacional **SQLite/PostgreSQL-ready**, backend Go `chi` + `AES-256-GCM` y **frontend Astro + React TSX + Tailwind** servido por Go en un solo puerto `:8080` (sin `npm run dev` en producción). Optimizado para **móvil** (cámara, escáner QR, touch 44px).

## Estructura

```
inventory/
├── backend/               # Go + chi + go-sqlite3 + uuid — SIRVE TODO en :8080
│   ├── main.go            # API + FileServer para ../frontend/dist + fallback SPA
│   ├── go.mod             # go 1.26
│   ├── internal/
│   │   ├── crypto/crypto.go       # EncryptAESGCM / DecryptAESGCM
│   │   ├── db/schema.go           # 18 tablas + índices + migraciones cautelosas (InitDB idempotente)
│   │   ├── handlers/              # dispositivos, etiquetas, movimientos, mantenimientos, tintas, evidencia,
│   │   │                          #   auditoria, planes, toma, frascos, conjuntos (stock caja), ia (proxy VLM)
│   │   ├── scanner/               # Symbol USB: device.go (udev), evdev_linux.go, hub.go (WS + lookup), barcode.go, catalog.go
│   │   └── models/models.go       # Estados, Tipos (38) / TiposComponente (36), Etiqueta, Movimiento, Conjunto, Conteo
│   ├── encrypted_storage/ # .enc AES-256-GCM (gitignored)
│   └── uploads/pdf/       # PDFs mantenimientos
├── frontend/              # Astro 7 + React 18 TSX + Tailwind 3 (output: static)
│   ├── astro.config.mjs   # output: 'static' — npm run build → dist/ que Go sirve
│   ├── dist/              # generado (index.html, _astro/, dispositivos/, etiquetas/, escaner/, frascos/, conjuntos/)
│   ├── src/
│   │   ├── layouts/Layout.astro          # nav responsive hamburger + theme
│   │   ├── components/
│   │   │   ├── EncryptedGallery.tsx      # galería + captura getUserMedia (móvil 4:3)
│   │   │   ├── Escaner.tsx               # html5-qrcode, BarcodeDetector, vibración (dispositivo o conjunto/caja)
│   │   │   ├── EtiquetasManager.tsx      # pool UUID, QR negro, PDF pdf-lib 1cm, asociar a dispositivo O conjunto
│   │   │   ├── DeviceSearchCombobox.tsx  # búsqueda q (compu/disco/GGM o UUID)
│   │   │   ├── ConjuntoSearchCombobox.tsx# búsqueda conjuntos (tornillos/CJ- o UUID)
│   │   │   ├── ConjuntosManager.tsx      # cards stock por caja + QR + libro movimientos
│   │   │   ├── ContadorIA.tsx            # conteo por imagen: OWL-ViT en navegador (transformers.js) o VLM (Ollama)
│   │   │   ├── FrascosManager.tsx        # frascos de tinta stock (hermana de conjuntos)
│   │   │   ├── InventarioList.tsx        # filtros + buscador q + badges
│   │   │   ├── InkLevelGauge.tsx         # tinta impresora
│   │   │   └── DispositivoDetail.tsx     # ficha + componentes + movimientos + etiqueta
│   │   └── pages/
│   │       ├── index.astro               # InventarioList client:load
│   │       ├── etiquetas.astro           # EtiquetasManager
│   │       ├── escaner.astro             # Escaner
│   │       ├── frascos.astro             # FrascosManager
│   │       ├── conjuntos.astro           # ConjuntosManager + ContadorIA
│   │       └── dispositivos/[id].astro   # placeholder → Go fallback
│   └── package.json       # html5-qrcode, qrcode.react, qrcode, pdf-lib (IA: transformers.js por CDN, sin npm dep)
├── docker/                # Dockerfile + 99-symbol-barcode.rules (udev para el escáner Symbol)
├── docs/                  # Documentación técnica y usabilidad (8 docs)
└── inventario.db          # SQLite WAL (gitignored)
```

## Esquema SQLite (UUID v4/v7 TEXT)

- **dispositivos**: `id`, `codigo_activo UNIQUE`, `numero_serie`, `tipo` (38 tipos: Desktop, Laptop, Servidor, Tablet, Monitor, Proyector, Impresora, Scanner, Router, Switch, Access Point, Firewall, NAS, UPS, Regulador, Telefono IP, Camara IP, Disco Externo, Ventilador/SSD/HDD/NVMe/RAM/Cable HDMI/Ethernet/USB/SATA/Patch Cord/Fusible…), `marca`, `modelo`, `especificaciones`, `valor_compra`, `proveedor`, `estado` (9: activo, disponible, reserva, en_mantenimiento, en_reparacion, averiado, obsoleto, dado_de_baja, extraviado), `ubicacion`, `sede`, `asignado_a`, `fecha_compra`, `fecha_baja`, `motivo_baja`, `parent_id FK SET NULL` (ventilador en CPU), `tipo_componente` (36), `creado_en`, `actualizado_en`
- **mantenimientos**: `id`, `dispositivo_id FK CASCADE`, `realizado_por`, `descripcion`, `costo`, `fecha_hora`, `url_pdf`, `creado_en`
- **niveles_tinta**: `id`, `dispositivo_id FK`, `color`, `nivel_porcentaje 0..100`, `codigo_tinta`, `ultima_actualizacion`
- **fotos_evidencia**: `id`, `entidad_tipo`, `entidad_id`, `ruta_archivo_cifrado .enc`, `nombre_original`, `mime_type`, `tamano_bytes`, `cargado_en`
- **movimientos**: `id`, `dispositivo_id FK CASCADE`, `origen_parent_id`, `destino_parent_id`, `tipo_movimiento` (asignacion, desasignacion, traslado, canibalizacion, instalacion, retiro), `descripcion`, `fecha_movimiento`
- **etiquetas**: `id PK UUID v4/v7`, `version 4|7`, `estado` (disponible, impreso, asignado, anulado), `dispositivo_id UNIQUE FK SET NULL`, `conjunto_id UNIQUE parcial FK SET NULL` (QR en la caja), `codigo_barras ET-XXXXXXXX UNIQUE`, `generado_en`, `impreso_en`, `asignado_en` — una etiqueta va a un dispositivo **o** a un conjunto (caja), nunca a ambos
- **conjuntos**: `id`, `codigo CJ-XXXXXXXX UNIQUE`, `nombre`, `categoria`, `especificaciones`, `unidad` (pieza/gr/ml/m/paquete), `cantidad_actual ≥ 0`, `cantidad_minima`, `ubicacion` (caja/cajón), `sede`, `proveedor`, `costo_unitario`, `creado_en`, `actualizado_en` — stock a granel: 1 fila = caja de piezas idénticas (tornillos, resistores)
- **conjunto_movimientos**: libro transaccional (clon de `frasco_movimientos`): `conjunto_id FK CASCADE`, `tipo` (ingreso/consumo/ajuste/traslado/baja), `cantidad ≠ 0`, `cantidad_previa`, `cantidad_nueva`, `realizado_por`, `nota`, `fecha`
- **conteos**: historial IA/humano por imagen: `conjunto_id FK CASCADE`, `foto_evidencia_id FK SET NULL`, `cantidad_detectada` (IA), `cantidad_confirmada` (humano), `modelo` (`browser:owlvit-base-patch32` | `vlm:<model>` | `manual`), `detecciones_json` (cajas+score para medir precisión), `nota`, `creado_por`, `creado_en`

Índices: `estado`, `ubicacion`, `parent_id`, `tipo_componente`, `etiquetas.estado/dispositivo/conjunto (parcial UNIQUE)`, `movimientos.origen/destino`, `conjuntos.categoria/sede/ubicacion`, `conjunto_movimientos.conjunto/fecha`, `conteos.conjunto/fecha`.

Auditoría: tabla `auditoria` con CHECK de entidades `('dispositivo','movimiento','mantenimiento','etiqueta','reporte','usuario','tinta','evidencia','plan','toma','conjunto')` — migrada por rebuild (patrón plan 05) en BDs existentes.

## Inicio unificado (Go sirve todo)

```bash
# 0. Node (usa 22.19.0 por pdf-lib/qrcode)
nvm use 22.19.0

# 1. Frontend estático
cd frontend
npm install
npm run build        # → frontend/dist/ (index.html, _astro/, etiquetas/, escaner/)

# 2. Backend (usa Strawberry gcc; mingw cuelga)
# En PowerShell quita mingw del PATH para CGO:
$env:PATH = ($env:PATH -split ";" | Where-Object { $_ -ne "C:\msys64\mingw64\bin"}) -join ";"
cd ../backend
cp .env.example .env  # ENCRYPTION_KEY= 32 bytes exactos
go mod tidy
go test ./... -v      # 65 tests (conjuntos + escáner)
go run main.go        # Frontend: http://localhost:8080/  API: http://localhost:8080/api/dispositivos
# o binario
go build -o inventario.exe && ./inventario.exe
```

Health: `GET /health` — Env vars: `PORT` (8080), `DB_PATH`, `STORAGE_PATH`, `ENCRYPTION_KEY` (32B), `FRONTEND_DIST`, `VLM_URL` (ej. `http://localhost:11434`), `VLM_MODEL` (ej. `qwen2.5vl:7b`), `VLM_TYPE` (`ollama`|`openai` para llama.cpp/LM Studio). Sin `VLM_URL` el conteo IA funciona igual vía navegador (transformers.js).

Migración cautelosa: `InitDB` hace `ALTER TABLE ADD COLUMN` idempotente y `CREATE TABLE IF NOT EXISTS` sin borrar 10 dispositivos; backup `*.pre-migracion.*.bak`.

## Endpoints clave

- `GET /api/dispositivos?q=compu&estado=activo&sede=Matriz&limit=20` — búsqueda `LIKE` en codigo/marca/modelo/serie/tipo + filtros
- `POST /api/dispositivos` / `PUT /api/dispositivos/{id}` valida `tipo` (38) y `tipo_componente` (36), registra movimiento si cambia `parent_id`
- `GET /api/dispositivos/{id}` incluye `componentes[]` + `etiqueta`, `GET /api/dispositivos/{id}/componentes`
- `POST /api/etiquetas/generar {cantidad:1..500, version:4|7}` → `ids[]` (v7 time-ordered), `GET /api/etiquetas?estado=`, `POST /api/etiquetas/{id}/asociar {dispositivo_id O conjunto_id}` (exclusivos; combobox o UUID), `/desasociar`, `/impreso`, `/anular`, `GET /api/etiquetas/scan/{id}` (UUID o ET-XXXX) → `{dispositivo}` o `{conjunto}` según dónde esté el QR
- `POST /api/conjuntos {nombre, unidad, cantidad_actual, cantidad_minima, etiqueta_id?}` → `201 {id, codigo CJ-XXXX}`, `POST /api/conjuntos/{id}/movimiento {tipo: ingreso|consumo|ajuste|traslado|baja, cantidad}` (transaccional, previa/nueva en libro), `POST /api/conjuntos/{id}/conteos {cantidad_detectada, cantidad_confirmada, modelo, detecciones?, foto_evidencia_id?, aplicar_ajuste?}` → conteo IA con ajuste atómico
- `GET /api/ia/config` → `{vlm: {disponible, tipo, modelo}, navegador: {disponible}}`; `POST /api/ia/identificar {foto_evidencia_id, prompt?}` → descifra la foto y consulta el VLM local (Ollama `/api/chat` o OpenAI-compatible `/v1/chat/completions`), devuelve JSON `{descripcion, cantidad_estimada, items, confianza}` — `503` si VLM sin configurar
- `POST /api/movimientos` / `GET /api/movimientos?dispositivo_id=`
- `POST /api/evidencia/upload` multipart `fotos` → AES-256-GCM `.enc`, `GET /api/evidencia/{id}/raw`, `GET /api/evidencia/galeria?tipo=&id=`

Ver `docs/API.md` para curl completos.

## Flujos cubiertos

- **Estados 9** + `sede` multi-sede, filtros y badges.
- **Tipos ampliados sin tabla** (38 tipos, 36 componentes: cables Poder/HDMI/Ethernet/USB/SATA, Patch Cord, Disco/NVMe, Regulador, UPS, Fusible, Tarjeta Red…).
- **Componentes**: `parent_id` + `tipo_componente` (ej. ventilador en `GGM-COM-014`), historial `movimientos` auto-registro traslado/retiro/instalacion.
- **Etiquetas**: pool UUID v4/v7 imprimible (QR solo negro, `ET-XXXXXXXX`), `4×7` cards cuadradas sin gap, borde negro 0.5pt, margen 1cm, `pdf-lib` + `qrcode` (mismo stack `xeland314.github.io`), tamaño QR ~21mm (mín 15mm, recomendado 20-25mm).
- **Escáner Symbol USB (05e0:1200) + Teclado Wedge + Cámara**:
  - Detección nativa de hardware Linux `evdev` (`/dev/input/by-id/*Symbol*`, `/dev/input/event*`) en segundo plano sin requerir foco en inputs.
  - Emulación Keyboard Wedge global con análisis de temporización de teclas (<55ms ráfaga) y supresión de envíos accidentales.
  - Efectos de sonido POS sintéticos vía Web Audio API (pitido clásico 1950Hz en lectura, acorde ascendente 1800/2400Hz en coincidencia de inventario).
  - Canales WebSocket en tiempo real (`/ws` y `/api/ws`) con sincronización instantánea y deduplicación (<400ms).
  - Búsqueda universal inteligente: resuelve por UUID de etiqueta, código de barras `ET-XXXX`, código activo de dispositivo, número de serie de hardware, código de caja/conjunto `CJ-XXXX`, frasco de tinta o catálogo demo.
  - Generador dinámico de códigos de barras vectoriales SVG y PNG descargables (EAN-13, Code 128, Code 39, UPC-A, EAN-8, QR Code) en `/api/barcode/{format}/{code}`.
  - Permisos: instalar `docker/99-symbol-barcode.rules` en `/etc/udev/rules.d/` (y `udevadm control --reload && udevadm trigger`) para que el backend lea `/dev/input/event*`; sin permiso queda el modo web wedge. Las rutas `/api/scanner/*` (y alias `/api/scan`, `/api/scans`, `/api/device`, `/api/catalog`) exigen `Authorization: Bearer`; el WebSocket `/ws` se autentica con un primer mensaje `{"type":"auth","token":…}` (5 s de plazo, sin token en la URL). Solo `/api/barcode/*` es público (imágenes generadas, no exponen datos).
  - Hoja imprimible de calibración en `/escaner/imprimible` optimizada para impresión física (`Ctrl + P`).
  - Fallback por cámara móvil (`html5-qrcode`) y búsqueda manual integradas en una interfaz por pestañas con modal zoom para escaneo en pantalla.
- **Evidencia móvil**: `EncryptedGallery` `getUserMedia` `aspect-[4/3] sm:aspect-video`, captura `canvas.toBlob` → upload cifrado; fallback `capture="environment"`.
- **Tintas** impresora upsert.
- **Conjuntos por caja**: stock a granel con QR del pool en la caja; `ContadorIA` analiza la foto **en el navegador** (OWL-ViT `zero-shot-object-detection` vía transformers.js por CDN, WebGPU→WASM) o vía **VLM local** (Ollama/llama.cpp a través del proxy backend — la foto cifrada se descifra solo en el servidor); el humano confirma con stepper, cajas descartables tocándolas, y el conteo guarda detectado vs confirmado (mide precisión IA) con ajuste atómico opcional en el libro.
- **Frascos de tinta**: stock consumible sin QR, mismo patrón de libro que conjuntos.

## Desarrollo y móvil

- `npm run dev` → `http://localhost:4321` proxy `/api` → `:8080`
- Responsive: nav hamburger `md:hidden`, botones `min-h-[44px] touch-manipulation`, video `max-h-[60vh]`, grids `grid-cols-1 sm:grid-cols-2 lg:grid-cols-3/4`, `@page {margin:1cm}`

Ver `docs/README.md` (8 docs) y `docs/USABILIDAD.md` para guías por rol.
