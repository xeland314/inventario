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
- `GET /api/etiquetas?estado=disponible&limit=&offset=` → `{etiquetas}` (cada etiqueta incluye `conjunto_id`)
- `GET /api/etiquetas/{id}` → `{etiqueta, dispositivo, conjunto}`
- `POST /api/etiquetas/{id}/asociar {dispositivo_id | conjunto_id}` → `200` (**exclusivos**: la etiqueta va a un activo único o a una caja de conjunto, nunca a ambos; `409` si el destino ya tiene etiqueta)
- `POST /api/etiquetas/{id}/desasociar` → `200` libera a `disponible` (limpia ambos)
- `POST /api/etiquetas/{id}/impreso` → `disponible→impreso`
- `POST /api/etiquetas/{id}/anular` → `anulado` (no si asignado)
- `DELETE /api/etiquetas/{id}` → solo `disponible/anulado`
- `GET /api/etiquetas/scan/{id}` / `GET /api/escaner/{id}` alias → busca por `id` o `codigo_barras` (ET-XXXX) → si el QR está en una caja devuelve `{conjunto: {id, codigo, nombre, unidad, cantidad_actual, cantidad_minima, estado, ubicacion}}`

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
- `POST /api/evidencia/upload` `multipart fotos, entidad_tipo, entidad_id` → cifra AES-256-GCM, guarda `.enc` (`entidad_tipo` también `frasco` y `conjunto`)
- `GET /api/evidencia/galeria?tipo=dispositivo&id=` → `{fotos}`
- `GET /api/evidencia/{id}/raw` → descifra y stream `mime_type`
- `DELETE /api/evidencia/{id}`

## Conjuntos (stock por caja con QR)
- `GET /api/conjuntos?q=&categoria=&sede=&ubicacion=&bajo=1` → `{conjuntos:[{...conjunto, estado: disponible|bajo|agotado, etiqueta_id, etiqueta_codigo}]}` — `bajo=1` filtra `cantidad_actual <= cantidad_minima`
- `POST /api/conjuntos {nombre*, categoria?, especificaciones?, unidad? (pieza|gr|ml|m|paquete, default pieza), cantidad_actual?, cantidad_minima?, ubicacion?, sede?, proveedor?, costo_unitario?, codigo? (auto CJ-XXXXXXXX), etiqueta_id? (UUID o ET-XXXX, atómico)}` → `201 {id, codigo}`
- `GET /api/conjuntos/{id}` (id o CJ-XXXX) → conjunto + `movimientos[]` + `conteos[]`
- `PUT` / `PATCH /api/conjuntos/{id}` → parcial; **no permite `cantidad_actual`** (solo vía libro/movimientos); `DELETE` libera la etiqueta de la caja
- `POST /api/conjuntos/{id}/movimiento {tipo: ingreso|consumo|ajuste|traslado|baja, cantidad (ingreso +, consumo/baja −, ajuste ±), realizado_por?, nota?}` → transaccional: `UPDATE` + entrada en libro (previa/nueva), `400` si stock insuficiente
- `GET /api/conjuntos/{id}/movimientos` → `{movimientos}` (100 últimos)
- `POST /api/conjuntos/{id}/conteos {cantidad_detectada?, cantidad_confirmada*, modelo? ('browser:owlvit-base-patch32'|'vlm:<model>'|'manual'), detecciones?: [{box:[xmin,ymin,xmax,ymax], score, label}], foto_evidencia_id?, nota?, aplicar_ajuste?}` → `201 {id, delta, ajuste_aplicado}`: si `aplicar_ajuste` y `confirmada ≠ stock`, en una transacción crea el movimiento `tipo=ajuste` con `delta = confirmada − actual`
- `GET /api/conjuntos/{id}/conteos` → `{conteos}` (histórico detectada vs confirmada para medir precisión IA)

Rol: GET lector+tecnico; escritura solo tecnico (auditoría entidad `conjunto`).

## IA local (conteo de imágenes)
- `GET /api/ia/config` → `{navegador: {disponible: true, modelo: 'owlvit-base-patch32'}, vlm: {disponible, tipo (ollama|openai), modelo}}` — el frontend muestra/oculta la vía VLM según esto
- `POST /api/ia/identificar {foto_evidencia_id*, prompt?}` → devuelve `{vlm: true, tipo, modelo, resultado: {descripcion, cantidad_estimada, items: [{nombre, cantidad}], confianza}, texto}` — el backend descifra la foto (la clave y el archivo nunca salen del servidor) y consulta el VLM local: Ollama (`VLM_TYPE=ollama`, `POST {VLM_URL}/api/chat` con base64 y `format:json`) o OpenAI-compatible (llama.cpp `llama-server`/LM Studio, `VLM_TYPE=openai`, `POST {VLM_URL}/v1/chat/completions` con `image_url` data-URI) — `503 {VLM no configurado: define VLM_URL y VLM_MODEL}` si no hay env
- La vía navegador (principal) no usa backend: `ContadorIA.tsx` carga transformers.js desde CDN y corre OWL-ViT zero-shot (`Xenova/owlvit-base-patch32`, ~100MB, cacheado en el navegador tras la primera vez, WebGPU con fallback WASM); prompts en inglés mapeados de es→en; umbral ajustable (default 0.15); contingencia manual siempre disponible

> Para tomas físicas, frascos, planes de mantenimiento, dashboard y auditoría ver `docs/planes/01..05-*.md` y `docs/02-componentes-movimientos.md`.

## Escáner Symbol USB (05e0:1200) & Códigos de Barras
> Auth: todo requiere `Authorization: Bearer <token>` salvo `/api/barcode/*`. El WebSocket se autentica con el primer mensaje `{"type":"auth","token":"..."}` (5 s; si falla recibe `{type:"error"}` y se cierra). No se acepta token en la URL.
- `GET /api/scanner/device` (alias `/api/device`) → información técnica de hardware del lector USB (`connected`, `vendor_id`, `product_id`, `product_name`, `device_node`, `is_readable`, `evdev_active`, `web_wedge_active`)
- `POST /api/scanner/scan` (alias `/api/scan`) `{code, duration_ms?, source?}` → procesa escaneo con deduplicación (<400ms), enriquece con coincidencia en base de datos y difunde evento a clientes WebSocket
- `GET /api/scanner/scans` (alias `/api/scans`) → historial de escaneos de la sesión en memoria
- `DELETE /api/scanner/scans` (alias `/api/scans`) → limpia el historial de escaneos de la sesión y notifica a los clientes
- `GET /api/scanner/catalog` (alias `/api/catalog`) → catálogo de productos demo para pruebas y calibración
- `GET /api/scanner/lookup/{code}` → búsqueda universal en inventario por cualquier código (UUID de etiqueta, `ET-XXXX`, `codigo_activo`, `numero_serie`, `CJ-XXXX`, frasco de tinta o producto demo)
- `GET /api/barcode/{format}/{code}` (soporta `.png` y `.svg`, formatos `ean13`, `code128`, `code39`, `upca`, `ean8`, `qrcode`) → genera imagen dinámica nítida o vector SVG
- `WS /ws` (y `/api/ws`) → conexión WebSocket en tiempo real para eventos de lectura óptica (`scan_event`, `history_cleared`, `init`)

## Códigos
`400` estado/tipo/unidad inválido, ciclo, stock insuficiente, `404` no encontrado, `409` etiqueta ya asignada/destino con etiqueta, `503` VLM sin configurar.
