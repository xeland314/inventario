# Tests

## Backend `backend/internal/handlers/inventario_test.go`
Helper `setupTestDB` `:memory:?_fk=1` + `InitDB` + `newRouter` con `DispositivoHandler, MovimientoHandler, EtiquetaHandler`.

19 tests (`go test ./... -v` 1.2s):
- Estados: `TestEstadosValidos` (8 estados), `TestEstadoInvalido` (400), `TestListarFiltroEstado`, `TestActualizarEstadoInvalido`
- Componentes: `TestCrearComponenteConParent` (verifica `GET /componentes` y `etiqueta`), `TestCrearComponenteParentInexistente` (400), `TestSelfReferenceCiclo` (ciclo A→B→A 400), `TestTrasladoGeneraMovimiento` (instalacion+traslado), `TestRetiroGeneraMovimiento`
- Movimientos POST: `TestCrearMovimientoTraslado` (actualiza parent), `TestCrearMovimientoInvalido` (400)
- Filtros: `TestFiltroSedeYComponente` (`sede=Icon`, `solo_componentes`)
- Migración: `TestMigracionNoPierdeDatos`, `TestEtiquetasMigracionPreserva`
- Etiquetas: `TestEtiquetasGenerarV7YV4` (5+5, lista 10), `TestEtiquetasGenerarInvalido` (cantidad 0, version 9), `TestEtiquetaAsociarYDesasociar` (GET dispositivo incluye etiqueta, scan, desasociar → nil), `TestEtiquetaAsociarConflictos` (409 ya asignada / dispositivo con etiqueta), `TestEtiquetaImpresoYAnulado` (disponible→impreso→anulado)

## Ejecutar
```bash
cd backend
# usar gcc Strawberry (mingw cuelga)
$env:PATH = ($env:PATH -split ";" | Where { $_ -ne "C:\msys64\mingw64\bin"}) -join ";"
go test ./... -count=1 -v
go build -o inventario.exe .
```

## Frontend
`npm run build` verifica Astro + React (5 páginas: `/`, `/etiquetas`, `/escaner`, `/dispositivos/nuevo`, `/dispositivos/placeholder`). Escáner y cámara probados manualmente: `html5-qrcode` requiere `https`/localhost.

## Cobertura pendiente
- E2E Playwright para flujo completo generar→imprimir→asociar→escanear
- Test de `ET-XXXX` duplicado v7 (ahora usa últimos 8 hex, retry UNIQUE)
