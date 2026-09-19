# Migración — Sin pérdida en inventario grande

## Principio
- `ALTER TABLE ADD COLUMN` idempotente ignora error si existe.
- `CREATE TABLE IF NOT EXISTS` para nuevas tablas.
- `PRAGMA foreign_keys=ON` y `WAL` (`backend/main.go:39`).

## Implementación `backend/internal/db/schema.go:87`
```go
var tableExists int; db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name='dispositivos'`).Scan(&tableExists)
if tableExists==0 { Exec(Schema) } else { // BD antigua
  Exec(CREATE TABLE movimientos, etiquetas, fotos_evidencia...) // sin recrear dispositivos
}
migrations := []string{
  `ALTER TABLE dispositivos ADD COLUMN parent_id TEXT REFERENCES ...`,
  `ALTER TABLE dispositivos ADD COLUMN tipo_componente TEXT`,
  `ALTER TABLE dispositivos ADD COLUMN sede TEXT`,
  `ALTER TABLE dispositivos ADD COLUMN actualizado_en TEXT`,
}
for _,m := range migrations { db.Exec(m) } // ignora error
// índices nuevos después de columnas
```

## Historial
- 2026-09-18: añade `parent_id, tipo_componente, sede, actualizado_en`, tabla `movimientos`, luego `etiquetas (id, version, estado, dispositivo_id UNIQUE, codigo_barras UNIQUE)`.
- `codigo_barras` usa últimos 8 hex para evitar colisión v7 por timestamp.

## Verificación real
```bash
cp inventario.db inventario.db.pre-migracion.20260918.bak
go run migrar_etiquetas.go # InitDB
sqlite3 inventario.db "SELECT COUNT(*) FROM dispositivos" # 10 intactos
sqlite3 inventario.db "PRAGMA table_info(dispositivos)" # 20 cols
sqlite3 inventario.db "SELECT COUNT(*) FROM sqlite_master WHERE name='etiquetas'" # 1
```

## Rollback
Restaurar `*.bak`. `DROP TABLE etiquetas` no afecta `dispositivos` por FK `SET NULL`.

## Tests migración
`TestMigracionNoPierdeDatos` y `TestEtiquetasMigracionPreserva` simulan BD antigua (solo `dispositivos` sin columnas) y ejecutan `InitDB`.
