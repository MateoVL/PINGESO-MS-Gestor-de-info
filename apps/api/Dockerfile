# Etapa 1: Compilación (imagen fijada por tag exacto y digest según sección 3 de AGENTS.md)
FROM golang:1.26.5-alpine3.23@sha256:622e56dbc11a8cfe87cafa2331e9a201877271cbff918af53d3be315f3da88cc AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -o /app/server ./cmd/api

# Etapa 2: Runtime distroless nonroot fijado por digest según sección 3 de AGENTS.md
FROM gcr.io/distroless/static-debian13:nonroot@sha256:f7f8f729987ad0fdf6b05eeeae94b26e6a0f613bdf46feea7fc40f7bd72953e6

WORKDIR /app

COPY --from=builder /app/server /app/server

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/server"]

