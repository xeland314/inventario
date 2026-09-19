# Inventario de Tecnología — Golang + Astro + TSX (Escalable)

Sistema modular **multi-sede** con base relacional **SQLite/PostgreSQL-ready**, backend Go `chi` + `AES-256-GCM` y **frontend Astro + React TSX + Tailwind** servido por Go en un solo puerto `:8080` (sin `npm run dev` en producción). Optimizado para **móvil** (cámara, escáner QR, touch 44px).

## Estructura

```
inventory/
├── backend/               # Go + chi + go-sqlite3 + uuid — SIRVE TODO en :8080
│   ├── main.go            # API + FileServer para ../frontend/dist + fallback SPA
│   ├── go.mod             # go 1.25
│   ├── internal/
│   │   ├── crypto/crypto.go       # EncryptAESGCM / DecryptAESGCM
│   │   ├── db/schema.go           # 6 tablas + índices + migraciones cautelosas
│   │   ├── handlers/              # dispositivos, mantenimientos, tintas, evidencia, movimientos, etiquetas
│   │   └── models/models.go       # Estados, Tipos (38) / TiposComponente (36), Etiqueta, Movimiento
│   ├── encrypted_storage/ # .enc AES-256-GCM (gitignored)
│   └── uploads/pdf/       # PDFs mantenimientos
├── frontend/              # Astro 7 + React 18 TSX + Tailwind 3 (output: static)
│   ├── astro.config.mjs   # output: 'static' — npm run build → dist/ que Go sirve
│   ├── dist/              # generado (index.html, _astro/, dispositivos/, etiquetas/, escaner/)
│   ├── src/
│   │   ├── layouts/Layout.astro          # nav responsive hamburger + theme
│   │   ├── components/
│   │   │   ├── EncryptedGallery.tsx      # galería + captura getUserMedia (móvil 4:3)
│   │   │   ├── Escaner.tsx               # html5-qrcode, BarcodeDetector, vibración
│   │   │   ├── EtiquetasManager.tsx      # pool UUID, QR negro, PDF pdf-lib 1cm
│   │   │   ├── DeviceSearchCombobox.tsx  # búsqueda q (compu/disco/GGM o UUID)
│   │   │   ├── InventarioList.tsx        # filtros + buscador q + badges
│   │   │   ├── InkLevelGauge.tsx         # tinta impresora
│   │   │   └── DispositivoDetail.tsx     # ficha + componentes + movimientos + etiqueta
│   │   └── pages/
│   │       ├── index.astro               # InventarioList client:load
│   │       ├── etiquetas.astro           # EtiquetasManager
│   │       ├── escaner.astro             # Escaner
│   │       └── dispositivos/[id].astro   # placeholder → Go fallback
│   └── package.json       # html5-qrcode, qrcode.react, qrcode, pdf-lib
├── docs/                  # Documentación técnica y usabilidad (8 docs)
└── inventario.db          # SQLite WAL (gitignored)
```

## Esquema SQLite (UUID v4/v7 TEXT)

- **dispositivos**: `id`, `codigo_activo UNIQUE`, `numero_serie`, `tipo` (38 tipos: Desktop, Laptop, Servidor, Tablet, Monitor, Proyector, Impresora, Scanner, Router, Switch, Access Point, Firewall, NAS, UPS, Regulador, Telefono IP, Camara IP, Disco Externo, Ventilador/SSD/HDD/NVMe/RAM/Cable HDMI/Ethernet/USB/SATA/Patch Cord/Fusible…), `marca`, `modelo`, `especificaciones`, `valor_compra`, `proveedor`, `estado` (9: activo, disponible, reserva, en_mantenimiento, en_reparacion, averiado, obsoleto, dado_de_baja, extraviado), `ubicacion`, `sede`, `asignado_a`, `fecha_compra`, `fecha_baja`, `motivo_baja`, `parent_id FK SET NULL` (ventilador en CPU), `tipo_componente` (36), `creado_en`, `actualizado_en`
- **mantenimientos**: `id`, `dispositivo_id FK CASCADE`, `realizado_por`, `descripcion`, `costo`, `fecha_hora`, `url_pdf`, `creado_en`
- **niveles_tinta**: `id`, `dispositivo_id FK`, `color`, `nivel_porcentaje 0..100`, `codigo_tinta`, `ultima_actualizacion`
- **fotos_evidencia**: `id`, `entidad_tipo`, `entidad_id`, `ruta_archivo_cifrado .enc`, `nombre_original`, `mime_type`, `tamano_bytes`, `cargado_en`
- **movimientos**: `id`, `dispositivo_id FK CASCADE`, `origen_parent_id`, `destino_parent_id`, `tipo_movimiento` (asignacion, desasignacion, traslado, canibalizacion, instalacion, retiro), `descripcion`, `fecha_movimiento`
- **etiquetas**: `id PK UUID v4/v7`, `version 4|7`, `estado` (disponible, impreso, asignado, anulado), `dispositivo_id UNIQUE FK SET NULL`, `codigo_barras ET-XXXXXXXX UNIQUE`, `generado_en`, `impreso_en`, `asignado_en`

Índices: `estado`, `ubicacion`, `parent_id`, `tipo_componente`, `etiquetas.estado/dispositivo`, `movimientos.origen/destino`.

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
go test ./... -v      # 19 tests
go run main.go        # Frontend: http://localhost:8080/  API: http://localhost:8080/api/dispositivos
# o binario
go build -o inventario.exe && ./inventario.exe
```

Health: `GET /health` — Env vars: `PORT` (8080), `DB_PATH`, `STORAGE_PATH`, `ENCRYPTION_KEY` (32B), `FRONTEND_DIST`.

Migración cautelosa: `InitDB` hace `ALTER TABLE ADD COLUMN` idempotente y `CREATE TABLE IF NOT EXISTS` sin borrar 10 dispositivos; backup `*.pre-migracion.*.bak`.

## Endpoints clave

- `GET /api/dispositivos?q=compu&estado=activo&sede=Matriz&limit=20` — búsqueda `LIKE` en codigo/marca/modelo/serie/tipo + filtros
- `POST /api/dispositivos` / `PUT /api/dispositivos/{id}` valida `tipo` (38) y `tipo_componente` (36), registra movimiento si cambia `parent_id`
- `GET /api/dispositivos/{id}` incluye `componentes[]` + `etiqueta`, `GET /api/dispositivos/{id}/componentes`
- `POST /api/etiquetas/generar {cantidad:1..500, version:4|7}` → `ids[]` (v7 time-ordered), `GET /api/etiquetas?estado=`, `POST /api/etiquetas/{id}/asociar {dispositivo_id}` (combobox o UUID), `/desasociar`, `/impreso`, `/anular`, `GET /api/etiquetas/scan/{id}` (UUID o ET-XXXX)
- `POST /api/movimientos` / `GET /api/movimientos?dispositivo_id=`
- `POST /api/evidencia/upload` multipart `fotos` → AES-256-GCM `.enc`, `GET /api/evidencia/{id}/raw`, `GET /api/evidencia/galeria?tipo=&id=`

Ver `docs/API.md` para curl completos.

## Flujos cubiertos

- **Estados 9** + `sede` multi-sede, filtros y badges.
- **Tipos ampliados sin tabla** (38 tipos, 36 componentes: cables Poder/HDMI/Ethernet/USB/SATA, Patch Cord, Disco/NVMe, Regulador, UPS, Fusible, Tarjeta Red…).
- **Componentes**: `parent_id` + `tipo_componente` (ej. ventilador en `GGM-COM-014`), historial `movimientos` auto-registro traslado/retiro/instalacion.
- **Etiquetas**: pool UUID v4/v7 imprimible (QR solo negro, `ET-XXXXXXXX`), `4×7` cards cuadradas sin gap, borde negro 0.5pt, margen 1cm, `pdf-lib` + `qrcode` (mismo stack `xeland314.github.io`), tamaño QR ~21mm (mín 15mm, recomendado 20-25mm).
- **Escáner + combobox**: `DeviceSearchCombobox` busca `compu/disco/GGM` o pega UUID; `Escaner` `html5-qrcode` con `facingMode: environment`.
- **Evidencia móvil**: `EncryptedGallery` `getUserMedia` `aspect-[4/3] sm:aspect-video`, captura `canvas.toBlob` → upload cifrado; fallback `capture="environment"`.
- **Tintas** impresora upsert.

## Desarrollo y móvil

- `npm run dev` → `http://localhost:4321` proxy `/api` → `:8080`
- Responsive: nav hamburger `md:hidden`, botones `min-h-[44px] touch-manipulation`, video `max-h-[60vh]`, grids `grid-cols-1 sm:grid-cols-2 lg:grid-cols-3/4`, `@page {margin:1cm}`

Ver `docs/README.md` (8 docs) y `docs/USABILIDAD.md` para guías por rol.
