# syntax=docker/dockerfile:1.7
FROM --platform=$BUILDPLATFORM node:22.22.0-alpine3.23@sha256:e4bf2a82ad0a4037d28035ae71529873c069b13eb0455466ae0bc13363826e34 AS build-web
WORKDIR /src
RUN corepack enable && corepack prepare pnpm@10.13.1 --activate
COPY .npmrc package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY apps/web/package.json ./apps/web/package.json
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile --ignore-scripts
COPY apps/web/ ./apps/web/
RUN pnpm --filter @pingeso/ms-gestor-de-info-ui build

FROM --platform=$BUILDPLATFORM golang:1.26.11-alpine3.23@sha256:23f198931d4bd83b4f5a613bd75656257ae8bf832dcbc2db80e1050b5219ca2a AS build-server
WORKDIR /src
COPY apps/web/server.go ./server.go
ARG TARGETOS TARGETARCH VERSION=dev
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -buildvcs=false -trimpath \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o /out/spa-server ./server.go

FROM gcr.io/distroless/static-debian13:nonroot@sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5
WORKDIR /app
COPY --from=build-server --chown=nonroot:nonroot /out/spa-server /app/spa-server
COPY --from=build-web --chown=nonroot:nonroot /src/apps/web/dist /srv
USER nonroot:nonroot
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD ["/app/spa-server", "-healthcheck"]
ENTRYPOINT ["/app/spa-server"]
