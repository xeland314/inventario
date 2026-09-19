# Usabilidad — Guías por rol

## Inventario grande multi-sede
1. Filtrar por sede en `/` (InventarioList) → `disponible` para stock, `averiado` para correctivo.
2. Alta en `/dispositivos/nuevo`: elige `sede`, `estado` (obsoleto para previo a baja), si es componente indica `tipo_componente` y `parent_id`.
3. Detalle `/dispositivos/{id}`: edita, ve componentes hijos, historial movimientos, mantenimientos con PDF, etiqueta QR.

## Técnico campo
- **Foto evidencia**: en galería del dispositivo → `📷 Tomar foto` → `Capturar y subir` (cifrado automático). Útil para plug RJ45 de `GGM-COM-025`.
- **Escanear**: `/escaner` o `/etiquetas` → `Iniciar cámara` → apunta QR → si está asociado navega a dispositivo; si no, muestra "Sin asignar" y form asociar. Manual: pega UUID/ET-XXXX.
- **Canibalizar**: en dispositivo componente → cambia `Parent ID` o usa `Historial Movimientos` → `traslado` indica destino.

## Almacén / Etiquetado
1. `/etiquetas` → Generar lote 50 v7 → `Imprimir visibles` (Ctrl+P) → recorta QR + `ET-XXXXXXXX` + UUID.
2. Pega en item (laptop, cargador).
3. En detalle del dispositivo (ej. `GGM-LAP-020`) sección `🏷️ Etiqueta` → pega UUID escaneado → `Asociar`. Luego escanear etiqueta abre ficha.
4. `Marcar impreso` tras imprimir para tracking.

## Administrador
- Estados: `activo` operativo, `en_mantenimiento` preventivo, `en_reparacion` correctivo, `averiado` inoperativo, `obsoleto` previo baja, `extraviado` auditoría.
- Bajas: `Dar de baja` pide motivo → `dado_de_baja` + `fecha_baja`.
- Evidencia: PDFs en `uploads/pdf` y fotos `.enc` en `encrypted_storage/` con `ENCRYPTION_KEY` 32B.

## Impresión
- Grilla `EtiquetasManager.jsx` 4 columnas (3 en print), bordes negros, QR 100px, `ET-XXXX` grande para lectura sin escáner.

## Accesibilidad
- Botones con estados color (verde activo, ámbar mantenimiento, rojo averiado, púrpura extraviado).
- Requiere `https` o `localhost` para cámara; fallback `+ Subir fotos`.
