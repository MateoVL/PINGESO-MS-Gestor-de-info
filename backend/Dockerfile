# Etapa 1: Compilación
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Compilar binario estático optimizado
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -o /app/server ./cmd/api

# Etapa 2: Imagen mínima de producción
FROM alpine:3.23

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl

# Usuario no root para seguridad
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

COPY --from=builder /app/server /app/server

EXPOSE 8080

CMD ["/app/server"]
