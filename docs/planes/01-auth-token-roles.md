# Plan 01 — Auth token-based + roles técnico / lector

> Un plan a la vez. Este archivo es el único en progreso hasta commit.

## Objetivo
Cerrar el acceso anónimo. Dos roles solo:
- **tecnico**: CRUD total (dispositivos, movimientos, etiquetas, mantenimientos, tintas, evidencia, planes/auditoría/dashboard cuando existan).
- **lector**: solo lectura (`GET`) + puede **reportar** y **cancelar su propio reporte** vía QR (`POST /api/reportes`, `POST /api/reportes/{id}/cancelar`). No borra, no edita inventario.

Auth sin JWT: token opaco aleatorio en tabla `sesiones`. Anular sesión = `DELETE FROM sesiones WHERE token = ?`.

## Esquema

```sql
CREATE TABLE IF NOT EXISTS usuarios (
  id TEXT PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  rol TEXT NOT NULL CHECK (rol IN ('tecnico','lector')),
  creado_en TEXT NOT NULL DEFAULT (datetime('now')),
  activo INTEGER NOT NULL DEFAULT 1
);
CREATE TABLE IF NOT EXISTS sesiones (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
  token TEXT UNIQUE NOT NULL,
  creado_en TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_sesiones_token ON sesiones(token);
CREATE INDEX IF NOT EXISTS idx_sesiones_user ON sesiones(user_id);

-- reportes mínimos para que lector tenga acción útil desde QR
CREATE TABLE IF NOT EXISTS reportes (
  id TEXT PRIMARY KEY,
  dispositivo_id TEXT REFERENCES dispositivos(id) ON DELETE SET NULL,
  etiqueta_id TEXT REFERENCES etiquetas(id) ON DELETE SET NULL,
  creado_por TEXT NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
  descripcion TEXT NOT NULL,
  estado TEXT NOT NULL CHECK (estado IN ('abierto','cancelado','resuelto')) DEFAULT 'abierto',
  creado_en TEXT NOT NULL DEFAULT (datetime('now')),
  cancelado_en TEXT
);
CREATE INDEX IF NOT EXISTS idx_reportes_dispositivo ON reportes(dispositivo_id);
CREATE INDEX IF NOT EXISTS idx_reportes_creador ON reportes(creado_por);
```

Seed: si `SELECT count(*) FROM usuarios = 0` → crear `admin / Admin123!` rol tecnico (bcrypt cost 10). Log en stdout.

Migración: `InitDB` crea tablas si no existen; no rompe BD con 10 dispositivos existentes. `ALTER TABLE` no aplica (tablas nuevas).

## Endpoints

| Método | Ruta | Auth | Rol | Descripción |
|---|---|---|---|---|
| POST | /api/auth/register | opcional* | tecnico (o bootstrap) | `{username,password,rol}` → 201 `{id}`. Si hay 0 usuarios, permite sin token (bootstrap). Si hay ≥1, exige tecnico. Valida username 3-32 alfanum, password ≥8, rol. Hash bcrypt. |
| POST | /api/auth/login | público | — | `{username,password}` → 200 `{token, usuario:{id,username,rol}}`. Token 32 bytes hex (crypto/rand). Inserta en sesiones. |
| POST | /api/auth/logout | sí | cualquiera | `Authorization: Bearer <token>` → borra fila sesiones → 200. Anular = logout. |
| GET | /api/auth/me | sí | cualquiera | devuelve usuario del token. |
| GET | /api/usuarios | sí | tecnico | lista usuarios sin hash. |
| DELETE | /api/usuarios/{id} | sí | tecnico | no auto-borrado. |
| POST | /api/reportes | sí | lector, tecnico | lector/tecnico: `{dispositivo_id?, etiqueta_id?, descripcion}` dispositivo/etiqueta opcional pero uno requerido. Resuelve dispositivo vía etiqueta si solo etiqueta. Estado abierto. |
| POST | /api/reportes/{id}/cancelar | sí | lector dueño o tecnico | solo creador o tecnico. `abierto → cancelado`. |
| POST | /api/reportes/{id}/resolver | sí | tecnico | `abierto → resuelto`. |
| GET | /api/reportes | sí | cualquiera | filtro `?estado=abierto&dispositivo_id=` |

*Compat: registrar via curl sin token solo para primer admin.

## Middleware

`AuthMiddleware(db)`:
- lee `Authorization: Bearer <token>` o `X-Auth-Token`.
- si falta o `SELECT user FROM sesiones JOIN usuarios WHERE token=? AND activo=1` no encuentra → `401 {"error":"no autenticado"}`.
- inyecta `userID, username, rol` en `context.WithValue(r.Context(), ctxKeyUser, u)`.
- helpers: `RequireRole(roles...)` → 403 si rol no coincide.

Protección de rutas en `backend/main.go:81`:
- públicas: `POST /api/auth/login`, `POST /api/auth/register` (con lógica bootstrap), `GET /health`.
- grupo autenticado `r.Group(func(r chi.Router){ r.Use(AuthMiddleware); ... })` envuelve todo `/api/*` restante.
- dentro, sub-grupo `RequireRole("tecnico")` para POST/PUT/PATCH/DELETE dispositivos, movimientos, etiquetas generar/asociar, mantenimientos, tintas, evidencia upload/delete, usuarios, reportes resolver. Lectores pasan solo GET + POST reportes + POST cancelar propio. Implementar check fino por handler si hace falta (reportes handler valida dueño).

CORS ya expone `Authorization`.

## Frontend

- `frontend/src/pages/login.astro` — form username/password → `fetch /api/auth/login` → guarda `localStorage token + usuario` → redirect `/`.
- `frontend/src/lib/auth.ts` — helpers `getToken(), getUser(), logout(), authFetch()`.
- `frontend/src/layouts/Layout.astro:19` — muestra usuario/rol y botón logout (→ POST /api/auth/logout + clear localStorage).
- Guard: si no hay token y ruta != /login → redirect. Lector ve badge "solo lectura" y deshabilita botones crear/borrar; botón "Reportar problema" siempre activo (si hay QR escaneado).
- No JWT en localStorage más que token opaco; no guardar password.

## Tests `backend/internal/handlers/inventario_test.go` (append)

- `TestAuthRegisterLoginLogout` — register tecnico, login ok, login bad password 401, me 200 con token, logout 200, me 401 tras logout.
- `TestAuthRoles` — register lector, login lector → GET /api/dispositivos 200, POST /api/dispositivos 403, POST /api/reportes 201 con etiqueta, cancelar propio 200, cancelar ajeno 403.
- `TestTecnicoPuedeTodo` — tecnico CRUD full.
- `TestBootstrapSoloPrimerUsuario` — sin usuarios register sin token ok, segundo register sin token 401.
- Helpers: `loginAs(t, router, username, password) string` retorna token; `authReq(method, path, token, body)`.

Todos `go test ./... -count=1 -v` con Strawberry gcc como antes. `InitDB` en `:memory:` debe incluir usuarios/sesiones/reportes.

## Seguridad / Riesgos

- Token aleatorio 32 bytes hex (64 chars) via `crypto/rand`, único, no expiración; revocar = DELETE. Futuro: añadir columna `expira_en` si hace falta.
- Bcrypt cost 10 (~70ms).
- No JWT → no validación firma, no expiración implícita; estado en DB.
- Rate limit no incluido (futuro).
- HTTPS: mismo riesgo operativo que cámara (`getUserMedia`) — documentar que prod necesita Caddy/mkcert.

## Verificación manual

```bash
go test ./internal/handlers -run TestAuth -count=1 -v
go build -o inventario.exe .
npx astro build (o npm run build) — login.astro compila
curl -X POST localhost:8080/api/auth/register -d '{"username":"admin","password":"Admin123!","rol":"tecnico"}'
curl -X POST localhost:8080/api/auth/login -d '{"username":"admin","password":"Admin123!"}'
curl -H "Authorization: Bearer <token>" localhost:8080/api/dispositivos
```

## Commit

Mensaje: `feat(auth): token-based tecnico/lector + reportes QR (lector reporta/cancela)`

Archivos tocados esperados: `backend/internal/models/models.go`, `backend/internal/db/schema.go`, `backend/internal/handlers/auth.go`, `backend/internal/handlers/reportes.go`, `backend/internal/middleware/auth.go` (o dentro handlers), `backend/main.go`, `backend/go.mod`, `frontend/src/pages/login.astro`, `frontend/src/lib/auth.ts`, `frontend/src/layouts/Layout.astro`, `backend/internal/handlers/inventario_test.go`, `docs/planes/01-auth-token-roles.md`.
