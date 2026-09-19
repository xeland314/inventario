# Usabilidad — Guías por rol

## Inventario grande multi-sede
1. Buscar en `/` con buscador `q` (compu, disco, GGM-..., UUID) + filtros `estado`/`sede` → `disponible` para stock, `averiado` para correctivo. Resultados `q` incluyen `tipo` ampliado (38 tipos: Desktop, Cable HDMI, Regulador…).
2. Alta en `/dispositivos/nuevo`: elige `sede`, `estado` (obsoleto para previo a baja), `tipo` (20) / `tipo_componente` (36: Cable Poder/Ethernet, NVMe, Fusible…), si es componente indica `parent_id`.
3. Detalle `/dispositivos/{id}`: edita, ve componentes hijos, historial movimientos, mantenimientos con PDF, etiqueta QR.

## Técnico campo
- **Foto evidencia**: en galería del dispositivo → `📷 Tomar foto` → `Capturar y subir` (cifrado automático). Útil para plug RJ45 de `GGM-COM-025`.
- **Escanear**: `/escaner` o `/etiquetas` → `Iniciar cámara` → apunta QR → si está asociado navega a dispositivo; si no, muestra "Sin asignar" y form asociar. Manual: pega UUID/ET-XXXX.
- **Canibalizar**: en dispositivo componente → cambia `Parent ID` o usa `Historial Movimientos` → `traslado` indica destino.

## Almacén / Etiquetado
1. `/etiquetas` → Generar lote 50 v7 → `PDF 1cm` (pdf-lib, `qrcode` como `xeland314.github.io`, A4 margen 1cm, 4×7 cards cuadradas sin gap, borde negro 0.5pt, QR solo negro 21mm, `ET-XXXXXXXX` + `uuid` sin estado) o `Print` navegador → recorta por borde.
2. Pega en item (laptop, cargador, cable).
3. Asociar: en `/etiquetas` usa `DeviceSearchCombobox` — escribe `compu`, `disco`, `GGM-COM` y elige de tabla (muestra `codigo_activo · tipo | marca modelo | id·estado`) o pega UUID directo; o en detalle del dispositivo (ej. `GGM-LAP-020`) sección `🏷️ Etiqueta` → pega/scannea UUID → `Asociar`. Luego escanear etiqueta abre ficha.
4. `Marcar impreso` tras imprimir para tracking.

## Administrador
- Estados: `activo` operativo, `en_mantenimiento` preventivo, `en_reparacion` correctivo, `averiado` inoperativo, `obsoleto` previo baja, `extraviado` auditoría.
- Bajas: `Dar de baja` pide motivo → `dado_de_baja` + `fecha_baja`.
- Evidencia: PDFs en `uploads/pdf` y fotos `.enc` en `encrypted_storage/` con `ENCRYPTION_KEY` 32B.

## Impresión
- `EtiquetasManager.tsx` `QRCodeSVG` `fgColor="#000000"` pantalla, `print:block` cuadrada `border-black 0.5pt gap-0`, `@page {margin:1cm}`; PDF `pdf-lib` 4×7 (28/hoja) sin espacios para corte. QR mínimo 15mm (M), recomendado 20-25mm a 300 DPI.

## Accesibilidad
- Botones con estados color (verde activo, ámbar mantenimiento, rojo averiado, púrpura extraviado).
- Requiere `https` o `localhost` para cámara; fallback `+ Subir fotos`.
