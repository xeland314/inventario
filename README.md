# Inventario de Tecnología — Golang + Astro (Escalable)

Sistema modular con **base relacional (SQLite/PostgreSQL-ready)**, backend Go con chi + AES-256-GCM y **frontend Astro servido directamente por Go** (un solo puerto :8080, sin `npm run dev` en producción).

## Estructura

```
inventory/
├── backend/               # Go + chi + go-sqlite3 + uuid — SIRVE TODO en :8080
│   ├── main.go            # API + FileServer para ../frontend/dist + fallback SPA
│   ├── go.mod
│   ├── internal/
│   │   ├── crypto/crypto.go       # EncryptAESGCM / DecryptAESGCM
│   │   ├── db/schema.go           # 4 tablas + PRAGMA FK
│   │   ├── handlers/              # dispositivos, mantenimientos, tintas, evidencia
│   │   └── models/models.go
│   ├── encrypted_storage/ # .enc AES-256-GCM (gitignored)
│   └── uploads/pdf/       # PDFs mantenimientos
└── frontend/              # Astro 4 + React + Tailwind (output: static)
    ├── astro.config.mjs   # output: 'static' — npm run build → dist/ que Go sirve
    ├── dist/              # generado, contiene index.html, _astro/, dispositivos/
    ├── src/
    │   ├── layouts/Layout.astro
    │   ├── components/
    │   │   ├── EncryptedGallery.jsx
    │   │   ├── InkLevelGauge.jsx
    │   │   ├── InventarioList.jsx      # fetch client-side a /api
    │   │   └── DispositivoDetail.jsx   # fetch client-side + fallback placeholder
    │   └── pages/
    │       ├── index.astro             # shell → InventarioList client:load
    │       └── dispositivos/[id].astro # getStaticPaths placeholder → Go hace fallback
    └── package.json
```

## Esquema SQLite (UUID v4 TEXT)

- **dispositivos**: `id`, `codigo_activo UNIQUE`, `numero_serie`, `tipo`, `marca`, `modelo`, `especificaciones`, `valor_compra`, `proveedor`, `estado`, `ubicacion`, `asignado_a`, `fecha_compra`, `fecha_baja`, `motivo_baja`, `creado_en`
- **mantenimientos**: `id`, `dispositivo_id FK CASCADE`, `realizado_por`, `descripcion`, `costo`, `fecha_hora`, `url_pdf`, `creado_en`
- **niveles_tinta**: `id`, `dispositivo_id FK CASCADE`, `color`, `nivel_porcentaje CHECK 0..100`, `ultima_actualizacion`, `UNIQUE(dispositivo_id,color)`
- **fotos_evidencia**: `id`, `entidad_tipo`, `entidad_id`, `ruta_archivo_cifrado .enc`, `nombre_original`, `mime_type`, `tamano_bytes`, `cargado_en`

## Inicio unificado (recomendado: Go sirve todo)

```bash
# 1. Compilar frontend estático
cd frontend
npm install
npm run build        # genera frontend/dist/ (index.html, _astro/, dispositivos/placeholder/)

# 2. Levantar Go (sirve API + frontend en :8080)
cd ../backend
cp .env.example .env  # edita ENCRYPTION_KEY= 32 bytes exactos si quieres
go mod tidy
go run main.go        # → Frontend: http://localhost:8080/  API: http://localhost:8080/api/dispositivos
# o binario
go build -o inventario.exe && ./inventario.exe
```

Health: `GET http://localhost:8080/health` — Frontend: `GET http://localhost:8080/` — API y frontend mismo origen, sin CORS.

Env vars: `PORT` (8080), `DB_PATH`, `STORAGE_PATH`, `ENCRYPTION_KEY` (32 bytes), `FRONTEND_DIST` (ruta a dist, default `../frontend/dist`).

## Desarrollo frontend aislado (opcional)

```bash
cd frontend
npm run dev    # http://localhost:4321 (proxea /api → :8080, útil para HMR)
npm run preview # preview estático de dist/
```

## Flujos cubiertos

- `POST /api/dispositivos` → crea equipo (si tipo=Impresora inicializa 4 tintas a 100%)
- `PATCH /api/dispositivos/:id/baja` → `estado=dado_de_baja` + fecha/motivo
- `PUT /api/impresoras/:id/tintas` → upsert niveles
- `POST /api/evidencia/upload` (multipart `fotos`) → cifra AES-256-GCM, guarda `.enc` y metadata con `cargado_en`
- `GET /api/evidencia/:id/raw` → descifra en RAM y stream
- `GET /api/evidencia/galeria?tipo=&id=` → lista para `EncryptedGallery`

Ver READMEs de `backend/` y `frontend/` para detalles de endpoints y componentes.
