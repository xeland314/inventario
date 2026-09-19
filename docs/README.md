# Inventario TI — Documentación de Avances

> Golang + Astro + SQLite + AES-256-GCM — Sistema modular para inventario grande multi-sede

Este directorio documenta los **avances técnicos y de usabilidad** implementados entre 2026-09-10 y 2026-09-18. Cada documento detalla problema, solución, archivos clave, endpoints y guía de uso.

## Índice

1. [Estados expandidos y sede](./01-estados-sede.md) — 9 estados, sede, filtros, validación, migración cautelosa
2. [Trazabilidad componentes y movimientos](./02-componentes-movimientos.md) — padre/hijo (ventilador en CPU), historial canibalización
3. [Evidencia con cámara en tiempo real](./03-evidencia-camara.md) — captura `getUserMedia` → cifrado servidor
4. [Etiquetas UUID y escáner](./04-etiquetas-escaner.md) — pool v4/v7 imprimible, QR, asociación y escaneo
5. [API completa](./API.md) — todos los endpoints con ejemplos
6. [Usabilidad](./USABILIDAD.md) — flujos para usuario final por rol
7. [Migración](./MIGRACION.md) — estrategia sin pérdida de datos en BD grande
8. [Tests](./TESTS.md) — cobertura y cómo ejecutar

## Stack relevante

- Backend `backend/main.go:27` sirve API + frontend `frontend/dist` en `:8080`
- DB `backend/internal/db/schema.go:8` 6 tablas + índices
- Frontend `frontend/src/layouts/Layout.astro:17` nav central

## Cómo probar rápido

```bash
cd frontend && npm install && npm run build
cd ../backend && go test ./... -v
go run main.go # http://localhost:8080  http://localhost:8080/etiquetas  /escaner
```

> Ver `USABILIDAD.md` para flujos paso a paso y `API.md` para curl.
