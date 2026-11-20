# Etapa 1: Build de React con Node 22.22.0-alpine3.23 y pnpm 10.13.1 (fijados según sección 3 de AGENTS.md)
FROM node:22.22.0-alpine3.23@sha256:e4bf2a82ad0a4037d28035ae71529873c069b13eb0455466ae0bc13363826e34 AS web-builder

WORKDIR /workspace

RUN corepack enable && corepack prepare pnpm@10.13.1 --activate

COPY .npmrc package.json pnpm-workspace.yaml pnpm-lock.yaml* ./
COPY apps/web/package.json ./apps/web/package.json

RUN pnpm install --frozen-lockfile

COPY apps/web ./apps/web

ARG VITE_API_BASE_URL
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

RUN pnpm --filter @pingeso/ms-gestor-de-info-ui build

# Etapa 2: Compilar servidor estático ultraligero para distroless
FROM golang:1.26.5-alpine3.23@sha256:622e56dbc11a8cfe87cafa2331e9a201877271cbff918af53d3be315f3da88cc AS server-builder

WORKDIR /srv
COPY apps/web/server.go ./server.go
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /srv/web-server ./server.go

# Etapa 3: Runtime distroless static-debian13:nonroot fijado por digest según sección 3 de AGENTS.md
FROM gcr.io/distroless/static-debian13:nonroot@sha256:f7f8f729987ad0fdf6b05eeeae94b26e6a0f613bdf46feea7fc40f7bd72953e6

WORKDIR /app

COPY --from=server-builder /srv/web-server /app/web-server
COPY --from=web-builder /workspace/apps/web/dist /app/dist

USER nonroot:nonroot

EXPOSE 3000

ENTRYPOINT ["/app/web-server"]

