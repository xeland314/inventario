# 02 — Trazabilidad componentes y movimientos

## Problema
- `especificaciones` texto libre (`Hikvision SSD + Samsung HDD` en `modelo`) no permitía saber que ventilador/SSD pertenece a CPU X ni canibalizar.
- Sin historial de traslados.

## Solución
### Schema `backend/internal/db/schema.go:27` y `68`
`dispositivos.parent_id TEXT REFERENCES dispositivos(id) ON DELETE SET NULL` + `tipo_componente` + índice `idx_dispositivos_parent`. `movimientos(id, dispositivo_id FK, origen_parent_id, destino_parent_id, tipo_movimiento CHECK(6 valores), descripcion, fecha_movimiento)`.

Tipos válidos `backend/internal/models/models.go:21`: `asignacion, desasignacion, traslado, canibalizacion, instalacion, retiro`.

### Backend `backend/internal/handlers/dispositivos.go:144` y `backend/internal/handlers/movimientos.go:1`
- `validarParent()` evita `self-reference` y ciclo (recorre 20 ancestros).
- `Obtener` carga `componentes[]` hijos + `ListarComponentes`.
- `Actualizar` con `parent_id` genera movimiento automático `instalacion/retiro/traslado` con `uuid.NewString()`.
- `MovimientoHandler.Crear` en transacción actualiza `parent_id` y registra movimiento.

### Endpoints
- `GET /api/dispositivos/{id}/componentes`
- `GET /api/movimientos?dispositivo_id=&parent_id=`
- `POST /api/movimientos {dispositivo_id, destino_parent_id, tipo_movimiento}`

### Frontend `frontend/src/components/DispositivoDetail.jsx:423`
- Sección `🧩 Componentes` lista hijos, indica padre, link.
- Sección `📦 Historial Movimientos` lista con origen→destino, form manual.

### Usabilidad
- Ejemplo: SSD de `GGM-COM-014` falla → se extrae (`retiro`), se instala en `DESKTOP-35RN9VH` (`instalacion`), queda trazado. Batería `N18GG65` como componente `Bateria` con health 6.3%.
- Permite auditoría canibalización y reporte por `sede`.

### Tests
`TestCrearComponenteConParent`, `TestSelfReferenceCiclo`, `TestTrasladoGeneraMovimiento`, `TestRetiroGeneraMovimiento`.

### Migración
`ALTER TABLE` añade columnas; `CREATE TABLE movimientos` sin borrar datos.
