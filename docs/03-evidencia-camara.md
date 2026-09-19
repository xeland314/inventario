# 03 — Evidencia con cámara en tiempo real

## Problema
- Solo `input file` para subir fotos; en campo (aula híbrida, matriz) es más rápido fotografiar daño de plug RJ45 o SSD.

## Solución
### Frontend `frontend/src/components/EncryptedGallery.jsx:43`
- `navigator.mediaDevices.getUserMedia({video:{facingMode, ideal 1280x720}})` requiere `https` o `localhost` (ok en `:8080`).
- Manejo errores: `NotAllowedError` (permiso), `NotFoundError`, `NotReadableError` (ocupada), `SecurityError`.
- `videoRef` + `canvasRef` + `streamRef`; `toggleFacing()` cambia `environment↔user` (espeja canvas si frontal).
- `handleCapture()`: `canvas.toBlob('image/jpeg',0.85)` → `File(captura_${tipo}_${Date.now()}.jpg)` → `FormData fotos` → `POST /api/evidencia/upload`.
- UI: botón `📷 Tomar foto` / `✕ Cerrar`, panel video negro 16:9 max 360px, overlay `Cifrando...`, botones `Capturar y subir`/`Girar`.

### Backend `backend/internal/handlers/evidencia.go:24`
Sin cambios: `Upload` recibe `multipart fotos`, `crypto.EncryptAESGCM` con `ENCRYPTION_KEY` 32B, guarda `.enc` en `encrypted_storage/`, metadata en `fotos_evidencia`. `ServirRaw` descifra en RAM.

### Usabilidad
- En detalle de `GGM-COM-025` (Ethernet), técnico abre galería → `Tomar foto` → captura plug dañado → se cifra y aparece en galería sin salir de página.
- Fallback `capture="environment"` en input file para móviles sin `getUserMedia`.

### Requisitos
- Navegador con `mediaDevices`; localhost permitido. Vibrar `navigator.vibrate(120)` al capturar si disponible.

### Tests
Manual: abrir `/dispositivos/{id}`, galería, permitir cámara, capturar, verificar `GET /api/evidencia/galeria?tipo=dispositivo&id=` lista foto.
