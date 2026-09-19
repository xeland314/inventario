# Plan 04 — Dashboard / Estadísticas

> Alimentado por audit log (02) + planes (03) + inventario base.

## Objetivo
Responder en una sola llamada:
- conteo por estado/sede/tipo, valor total por sede
- costo mantenimiento por mes/tipo/dispositivo, top costosos
- antigüedad promedio, candidatos obsolescencia
- tiempo medio en `en_reparacion` (MTTR) y frecuencia fallas por marca/modelo (usa auditoria)
- tintas bajo umbral, planes vencidos/próximos

## API

| Método | Ruta | Rol | Descripción |
|---|---|---|---|
| GET | /api/dashboard | lector/tecnico | `?sede=&tinta_umbral=20` (default 20). Respuesta JSON único para frontend. Sin paginación. |
| GET | /api/estadisticas | lector/tecnico | alias de `/api/dashboard` |

Respuesta ejemplo:
```json
{
  "conteo_estado": {"activo":12,"averiado":2},
  "conteo_sede": {"Matriz":10,"Icon":4},
  "conteo_tipo": {"Laptop":5,"UPS":2},
  "valor_total": 45200.5,
  "valor_por_sede": {"Matriz": 32000},
  "costo_por_mes": [{"mes":"2026-01","costo":120},{"mes":"2026-02","costo":300}],
  "costo_por_tipo": {"Laptop":400},
  "costo_por_dispositivo": [{"dispositivo_id":"...","codigo_activo":"GGM-LAP-01","costo":500}],
  "top_costosos": [...5],
  "antiguedad_promedio_dias": 420,
  "candidatos_obsolescencia": [{"id":"...","codigo_activo":"...","dias":1825}],
  "tiempo_medio_en_reparacion_horas": 72.5,
  "frecuencia_por_marca": [{"marca":"Dell","fallas":5}],
  "frecuencia_por_modelo": [{"modelo":"XPS","fallas":3}],
  "tintas_bajo_umbral": [{"dispositivo_id":"...","color":"negro","nivel":15}],
  "planes_vencidos": 2, "planes_proximos_30": 5
}
```

## Lógica SQL (Go calcula, sin cron)

- `conteo_estado/sede/tipo` — `SELECT estado, count(*) GROUP BY estado` etc.
- `valor_total` — `SELECT SUM(valor_compra)`; por sede similar.
- `costo_por_mes` — `SELECT substr(fecha_hora,1,7) as mes, SUM(costo) FROM mantenimientos GROUP BY mes ORDER BY mes`
- `costo_por_tipo` — join `mantenimientos JOIN dispositivos ON dispositivo_id=id GROUP BY tipo`
- `costo_por_dispositivo` — `GROUP BY dispositivo_id` order `SUM(costo) DESC`, incluir `codigo_activo`
- `antiguedad_promedio` — `AVG(julianday('now') - julianday(fecha_compra))`
- `candidatos_obsolescencia` — `WHERE estado='obsoleto' OR julianday('now')-julianday(fecha_compra) > 1825` (5 años)
- `tiempo_medio_en_reparacion` — vía `auditoria` : para cada `entidad=dispositivo campo=estado valor_nuevo='en_reparacion'` buscar siguiente `valor_anterior='en_reparacion'` del mismo `entidad_id` orden `fecha`, diff horas ` (julianday(fin)-julianday(inicio))*24`, promedio. Si no hay cierre, ignora.
- `frecuencia_fallas` — `SELECT marca, count(*) FROM auditoria JOIN dispositivos ON entidad_id=id WHERE entidad='dispositivo' AND campo='estado' AND valor_nuevo IN ('averiado','en_reparacion') GROUP BY marca`
- `tintas_bajo_umbral` — `SELECT dispositivo_id, color, nivel FROM niveles_tinta WHERE nivel <= ?` (param umbral)
- `planes_vencidos/proximos` — reutiliza lógica `planes_mantenimiento` `proxima_fecha < hoy` / `hoy..hoy+30`

Sin tablas nuevas.

## Tests

Archivo `dashboard_test.go`:

- `TestDashboardResumen` — crear 2 sede Matriz/Icon con valor 100/200, 1 averiado, 1 activo → verifica conteos y valor_total 300.
- `TestDashboardCostoPorMes` — crear mantenimientos con fecha_hora `2026-01-15` costo 100 y `2026-02-10` costo 200 → `costo_por_mes` 2 entries.
- `TestDashboardTiempoMedioReparacion` — crear dispositivo, cambiar estado `activo→en_reparacion` (audita), esperar 1s, cambiar `en_reparacion→activo` → MTTR >0 horas.
- `TestDashboardFrecuenciaFallas` — 2 fallas Dell vs 1 HP → verifica orden.
- `TestDashboardTintas` — crear Impresora con tintas 10/100 → con umbral 20 solo 10 aparece.
- `TestDashboardPlanes` — crear plan vencido y proximo → verifica counts.
- `TestDashboardRoles` — lector GET 200, tecnico GET 200.

Todos con `newDashboardRouter` (auth + dashboard). `go test -run TestDashboard -count=1 -v`

## Frontend

No UI completa en este plan (solo API). 05 Toma física consumirá dashboard para mostrar faltantes. Documentar `curl`.

## Riesgos
- MTTR requiere audit log; si no hay cierre, no cuenta.
- Fechas SQLite `julianday` depende de `YYYY-MM-DD`.
- Tintas solo para Impresora.

## Commit
`feat(dashboard): estadisticas agregadas (estado/sede/tipo, valor, costo, MTTR, fallas, tintas, planes)`

Archivos: `backend/internal/handlers/dashboard.go`, `backend/main.go`, `backend/internal/handlers/dashboard_test.go`, `docs/planes/04-dashboard-estadisticas.md`
