# Inventario Frontend (Astro + React — servido por Go)

Frontend del Sistema de Inventario TI. **Build estático (`output: 'static'`) que Go sirve en :8080**; no requiere Node en producción.

## Stack
- Astro 4 con `output: 'static'` (ver `astro.config.mjs:8`)
- React islands (`client:load`)
- Tailwind CSS

## Componentes
- `EncryptedGallery.jsx` — galería que consume `/api/evidencia/galeria` y muestra imágenes desde `/api/evidencia/{id}/raw` (descifrado en RAM por Go)
- `InkLevelGauge.jsx` — vial visual de tintas con colores y slider editable
- `InventarioList.jsx` — listado con filtros, fetch client-side a `/api/dispositivos` (mismo origen :8080)
- `DispositivoDetail.jsx` — ficha + tintas + galería + mantenimientos + baja, fetch client-side, maneja `placeholder` fallback de Go

## Páginas
- `/` — shell → `InventarioList` client:load
- `/dispositivos/nuevo` — alta (form client-side POST a `/api/dispositivos`)
- `/dispositivos/[id]` — `getStaticPaths` con `placeholder`; Go hace fallback a `placeholder/index.html` para cualquier id y el componente lee el id real de `window.location`

## Desarrollo (HMR aislado)
```bash
npm install
npm run dev    # http://localhost:4321 (proxea /api → :8080)
```

El dev server proxea `/api` → `http://localhost:8080` (ver `astro.config.mjs`).

## Build — servido por Go (producción)
```bash
npm run build          # genera dist/ (static) con _astro/, index.html, dispositivos/placeholder/
# luego Go sirve dist/ en :8080 — no necesitas astro preview
npm run preview        # solo para preview estático local sin Go
```

## Integración con backend Go (un solo puerto)
```bash
npm run build
cd ../backend
go run main.go # :8080 sirve API y frontend en el mismo origen
# Frontend → http://localhost:8080/
# API      → http://localhost:8080/api/dispositivos
```
