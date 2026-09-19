# 01 — Estados expandidos y sede

## Problema
- Solo 3 estados en BD (`activo`, `en_mantenimiento`, `dado_de_baja`) sin `CHECK`; frontend mostraba `reserva` sin existir.
- Sin distinción entre `en_mantenimiento` preventivo vs `averiado` correctivo ni `obsoleto`/`extraviado`.
- Sin campo `sede` para inventario grande multi-sede (GGM Matriz, Icon, Vicsan).

## Solución
### Modelo `backend/internal/models/models.go:3`
```go
var EstadosValidos = {"activo","disponible","reserva","en_mantenimiento","en_reparacion","averiado","obsoleto","dado_de_baja","extraviado"}
```
`EsEstadoValido()` usado en `handlers/dispositivos.go:44` (Listar) y `:235` (Crear) / `:434` (Actualizar).

### Schema `backend/internal/db/schema.go:21`
```sql
estado TEXT CHECK (estado IN (...9...))
sede TEXT, actualizado_en TEXT, parent_id TEXT, tipo_componente TEXT
```
Migración cautelosa: `InitDB` detecta tabla existente y hace `ALTER TABLE ADD COLUMN` idempotente. Preserva 10 dispositivos.

### API
- `POST /api/dispositivos` valida estado; por defecto `activo`; `GET /api/dispositivos?estado=averiado&sede=Icon&ubicacion=...&solo_componentes=true`

### Frontend
- `frontend/src/components/InventarioList.jsx:51` 9 botones filtro + badges color
- `frontend/src/pages/dispositivos/nuevo.astro:74` select 8 estados + inputs `sede`, `tipo_componente`, `parent_id`
- `frontend/src/components/DispositivoDetail.jsx:368` edición con sede/estado ampliado

## Usabilidad
- Filtro rápido por sede para conteos por local.
- `disponible`/`reserva` para stock sin asignar; `en_reparacion` vs `averiado` clarifica SLA.
- `actualizado_en` audita cambios.

## Tests `backend/internal/handlers/inventario_test.go:60`
`TestEstadosValidos`, `TestEstadoInvalido`, `TestFiltroSedeYComponente`, `TestMigracionNoPierdeDatos` (simula BD antigua sin columnas).

## Migración real
Backup `inventario.db.pre-migracion.20260918.bak`, `go run migrar_etiquetas.go` verifica 10 filas intactas.
