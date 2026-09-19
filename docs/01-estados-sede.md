# 01 — Estados, tipos ampliados y sede

## Problema
- Solo 3 estados en BD (`activo`, `en_mantenimiento`, `dado_de_baja`) sin `CHECK`; frontend mostraba `reserva` sin existir.
- Sin distinción entre `en_mantenimiento` preventivo vs `averiado` correctivo ni `obsoleto`/`extraviado`.
- Lista `tipo` reducida (7: Laptop, Monitor, Impresora...) y `tipo_componente` 13, insuficiente para cables, discos, reguladores.
- Sin campo `sede` para inventario grande multi-sede (GGM Matriz, Icon, Vicsan).

## Solución
### Modelo `backend/internal/models/models.go:3`
```go
var EstadosValidos = {"activo","disponible","reserva","en_mantenimiento","en_reparacion","averiado","obsoleto","dado_de_baja","extraviado"}
var TiposValidos = {Desktop, Laptop, Servidor, Tablet, Monitor, Proyector, Impresora, Scanner, Router, Switch, Access Point, Firewall, NAS, UPS, Regulador, Telefono IP, Camara IP, Disco Externo, Ventilador, SSD, HDD, NVMe, Cable Poder/HDMI/VGA/DisplayPort/Ethernet/USB/SATA, Patch Cord, Fusible... 38 total}
var TiposComponenteValidos = {Ventilador, SSD, HDD, NVMe, RAM, Bateria, Fuente, Placa, CPU, GPU, Cable HDMI/Ethernet..., Regulador, UPS, Tarjeta Red... 36 total}
```
`EsEstadoValido()`/`EsTipoValido()`/`EsTipoComponenteValido()` en `handlers/dispositivos.go:44` (Listar) y `:242`/`419` (Crear/Actualizar). Sin tabla DB para preservar integridad (ampliación solo código, 0 riesgo FK).

### Schema `backend/internal/db/schema.go:21`
```sql
estado TEXT CHECK (estado IN (...9...))
sede TEXT, actualizado_en TEXT, parent_id TEXT, tipo_componente TEXT
```
Migración cautelosa: `InitDB` detecta tabla existente y hace `ALTER TABLE ADD COLUMN` idempotente. Preserva 10 dispositivos.

### API
- `POST /api/dispositivos` valida `estado` y `tipo` (38) + `tipo_componente` (36); por defecto `activo`; `GET /api/dispositivos?estado=averiado&sede=Icon&q=compu&solo_componentes=true`

### Frontend
- `frontend/src/components/InventarioList.tsx:51` 9 botones filtro + badges + buscador `q` (compu/disco/GGM o UUID)
- `frontend/src/pages/dispositivos/nuevo.astro:23` select 20 tipos + 36 componentes + `sede`, `parent_id`
- `frontend/src/components/DispositivoDetail.tsx:321` edición con sede/estado/tipos ampliados

## Usabilidad
- Filtro rápido por sede para conteos por local.
- `disponible`/`reserva` para stock sin asignar; `en_reparacion` vs `averiado` clarifica SLA.
- `actualizado_en` audita cambios.

## Tests `backend/internal/handlers/inventario_test.go:60`
`TestEstadosValidos`, `TestEstadoInvalido`, `TestFiltroSedeYComponente`, `TestMigracionNoPierdeDatos` (simula BD antigua sin columnas).

## Migración real
Backup `inventario.db.pre-migracion.20260918.bak`, `go run migrar_etiquetas.go` verifica 10 filas intactas.
