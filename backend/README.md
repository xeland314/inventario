# Inventario Backend (Golang + SQLite + AES-256-GCM)

Backend del Sistema de Inventario de Tecnología.

## Stack
- `net/http` + `chi` router
- `go-sqlite3` (CGO)
- `google/uuid` para IDs
- SQLite con `PRAGMA foreign_keys = ON`
- Cifrado AES-256-GCM para fotos de evidencia

## Estructura
```
backend/
├── main.go
├── go.mod
├── internal/
│   ├── crypto/crypto.go       # AES-256-GCM
│   ├── db/schema.go           # 4 tablas + índices
│   ├── handlers/
│   │   ├── dispositivos.go
│   │   ├── mantenimientos.go
│   │   ├── tintas.go
│   │   └── evidencia.go
│   └── models/models.go
├── encrypted_storage/         # .enc cifrados (gitignored)
└── uploads/pdf/               # PDFs de mantenimientos
```

## Variables de entorno
Ver `.env.example`:
- `PORT` (default 8080)
- `DB_PATH` (default ./inventario.db)
- `STORAGE_PATH` (default ./encrypted_storage)
- `ENCRYPTION_KEY` (32 bytes exactos)
- `FRONTEND_DIST` (default `../frontend/dist`; Go sirve ese dir en `/*` con fallback SPA)

## Instalación — Go sirve API + frontend (un solo puerto)

```bash
# 1. Build frontend (genera frontend/dist)
cd ../frontend && npm install && npm run build && cd ../backend

# 2. Run
go mod tidy
go run main.go
# Frontend → http://localhost:8080/
# API      → http://localhost:8080/api/dispositivos
# Health   → http://localhost:8080/health

# o binario
go build -o inventario.exe
./inventario.exe
```

Go detecta `frontend/dist` en `../frontend/dist`, `./frontend/dist`, `frontend/dist` o path en `FRONTEND_DIST`. Si no existe, responde `Frontend no compilado` en `/`.

## Frontend embebido
`main.go:112` — FileServer + SPA fallback: archivos exactos, `index.html` para directorios, `dispositivos/placeholder/index.html` para `/dispositivos/:id`, y `index.html` como fallback general. Las rutas `/api/*`, `/health`, `/uploads/*` tienen prioridad.

## Endpoints
| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/dispositivos?estado=activo&tipo=Laptop` | Listar dispositivos |
| POST | `/api/dispositivos` | Crear dispositivo |
| GET | `/api/dispositivos/{id}` | Obtener detalle + tintas si es impresora |
| PATCH | `/api/dispositivos/{id}/baja` | Dar de baja |
| DELETE | `/api/dispositivos/{id}` | Eliminar |
| GET | `/api/mantenimientos?dispositivo_id=` | Listar mantenimientos |
| POST | `/api/mantenimientos` | Crear (JSON o multipart con PDF) |
| GET | `/api/impresoras/{id}/tintas` | Obtener niveles |
| PUT | `/api/impresoras/{id}/tintas` | Actualizar niveles |
| POST | `/api/evidencia/upload` | Subir fotos (multipart fotos[]) cifradas |
| GET | `/api/evidencia/galeria?tipo=dispositivo&id=` | Listar fotos |
| GET | `/api/evidencia/{id}/raw` | Servir imagen descifrada |
| DELETE | `/api/evidencia/{id}` | Eliminar foto |

## Tablas
- `dispositivos` (id TEXT PK UUID, codigo_activo UNIQUE, estado, fecha_baja, etc.)
- `mantenimientos` (FK dispositivos.id CASCADE)
- `niveles_tinta` (UNIQUE dispositivo_id+color, CHECK 0-100)
- `fotos_evidencia` (ruta_archivo_cifrado .enc, cargado_en)
