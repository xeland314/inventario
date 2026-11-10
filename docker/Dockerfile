FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/main.go ./
COPY backend/internal/ ./internal/
RUN CGO_ENABLED=1 go build -o /inventario .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /data/uploads /data/encrypted_storage /app
WORKDIR /app
COPY --from=build /inventario /app/inventario
COPY frontend/dist/ /app/frontend/dist/
RUN ln -s /data/uploads /app/uploads
ENV PORT=8080 \
    DB_PATH=/data/inventario.db \
    STORAGE_PATH=/data/encrypted_storage \
    FRONTEND_DIST=/app/frontend/dist \
    ENCRYPTION_KEY=0123456789abcdef0123456789abcdef
VOLUME /data
EXPOSE 8080
CMD ["/app/inventario"]
