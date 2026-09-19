# API — Referencia completa

Base `http://localhost:8080/api` (mismo origen frontend). Salud `GET /health`.

## Dispositivos
- `GET /api/dispositivos?q=&estado=&tipo=&ubicacion=&sede=&parent_id=&tipo_componente=&solo_componentes=&solo_padres=&limit=` → `{dispositivos:[]}` `handlers/dispositivos.go:31` `q` busca `LIKE` en `codigo_activo, numero_serie, marca, modelo, tipo, id, especificaciones` (combobox `compu/disco/GGM` o UUID)
- `POST /api/dispositivos {codigo_activo, numero_serie, tipo (38), marca, modelo, estado, ubicacion, sede, parent_id, tipo_componente (36)}` → `201 {id}` valida `estado`/`tipo`/`tipo_componente`/parent ciclo
- `GET /api/dispositivos/{id}` → dispositivo + `tintas` si Impresora + `componentes[]` + `etiqueta`
- `PUT /api/dispositivos/{id}` / `PATCH` → actualiza parcial, registra movimiento si cambia `parent_id`
- `PATCH /api/dispositivos/{id}/baja {motivo_baja}` → `dado_de_baja`
- `GET /api/dispositivos/{id}/componentes` → `{componentes, parent_id}`
- `DELETE /api/dispositivos/{id}`

Ejemplo generar con etiqueta luego:
```bash
curl -X POST /api/etiquetas/generar -H "Content-Type: application/json" -d '{"cantidad":1,"version":7}'
curl -X POST /api/etiquetas/{uuid}/asociar -d '{"dispositivo_id":"<id>"}'
curl /api/etiquetas/scan/{uuid}
```

## Etiquetas
- `POST /api/etiquetas/generar {cantidad:1..500, version:4|7}` → `201 {generadas, ids}`
- `GET /api/etiquetas?estado=disponible&limit=&offset=` → `{etiquetas}`
- `GET /api/etiquetas/{id}` → `{etiqueta, dispositivo}`
- `POST /api/etiquetas/{id}/asociar {dispositivo_id}` → `200`
- `POST /api/etiquetas/{id}/desasociar` → `200`
- `POST /api/etiquetas/{id}/impreso` → `disponible→impreso`
- `POST /api/etiquetas/{id}/anular` → `anulado` (no si asignado)
- `DELETE /api/etiquetas/{id}` → solo `disponible/anulado`
- `GET /api/etiquetas/scan/{id}` / `GET /api/escaner/{id}` alias → busca por `id` o `codigo_barras` (ET-XXXX)

## Movimientos (trazabilidad)
- `GET /api/movimientos?dispositivo_id=&parent_id=` → `{movimientos}`
- `POST /api/movimientos {dispositivo_id, destino_parent_id, tipo_movimiento, descripcion}` → actualiza `parent_id` en transacción
- `GET /api/movimientos/{id}` / `DELETE`

## Mantenimientos
- `GET /api/mantenimientos?dispositivo_id=`
- `POST /api/mantenimientos` JSON o `multipart/form-data` con `pdf`
- `PUT /api/mantenimientos/{id}` / `DELETE`

## Tintas
- `GET /api/dispositivos/{id}/tintas` / `PUT`

## Evidencia cifrada
- `POST /api/evidencia/upload` `multipart fotos, entidad_tipo, entidad_id` → cifra AES-256-GCM, guarda `.enc`
- `GET /api/evidencia/galeria?tipo=dispositivo&id=` → `{fotos}`
- `GET /api/evidencia/{id}/raw` → descifra y stream `mime_type`
- `DELETE /api/evidencia/{id}`

## Códigos
`400` estado/tipo inválido, ciclo, `404` no encontrado, `409` etiqueta ya asignada/dispositivo con etiqueta.
