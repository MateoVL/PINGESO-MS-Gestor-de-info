# Pipeline de Información · Grupo 2 · PINGESO

Base del microservicio **PT2 (Gestor de Información)** alineada con las reglas de arquitectura, contrato API-first y versiones exactas de `AGENTS.md`. Su objetivo en la Línea de Productos de Software es recibir datos masivos del **PT5, Extractor de Datos OCR**, limpiarlos, validarlos y almacenarlos en PostgreSQL para que el **PT1, Motor de Reportes**, pueda consumirlos.

## Stack y estructura del repositorio (`AGENTS.md`)

| Componente | Tecnología y Versión Exacta | Ubicación |
|---|---|---|
| **Backend** | Go `1.26.5` (`go 1.26.0` + `toolchain go1.26.5`), Huma `v2.34.1`, pgx `v5.10.0`, sqlc `v1.29.0`, goose `v3.24.1` | `apps/api/` |
| **Frontend** | Node `22.22.0`, pnpm `10.13.1`, React `19.2.8`, TypeScript `6.0.3`, Vite `8.1.5`, Tailwind `4.3.3` | `apps/web/` |
| **Base de datos** | PostgreSQL `17.11` | `docker-compose.yaml` |
| **Contrato API-first** | OpenAPI 3.1.0 generado desde structs de Huma y commiteado | `contracts/openapi.yaml` |
| **Modos de falla** | Documentación de resiliencia, idempotencia y timeouts | `docs/modos-de-falla.md` |

### Estructura de carpetas

```text
.
├── apps/
│   ├── api/
│   │   ├── cmd/api/            # main: arranque, config, señales
│   │   ├── cmd/openapi/        # exportador del contrato OpenAPI 3.1 desde Huma
│   │   ├── internal/api/       # handlers Huma + structs del contrato + middlewares
│   │   ├── internal/service/   # reglas de negocio, sin HTTP ni SQL + cliente HTTP 5s
│   │   ├── internal/store/     # acceso a datos (PostgreSQL / memoria API-first)
│   │   ├── internal/repository/# generado por sqlc, no editar
│   │   ├── db/migrations/      # migraciones goose
│   │   ├── db/queries/         # consultas SQL para sqlc
│   │   ├── e2e/                # pruebas e2e + testdata/*.golden.md
│   │   ├── Dockerfile          # golang:1.26.5-alpine3.23 -> distroless static-debian13:nonroot
│   │   └── Dockerfile.migrate  # goose v3.24.1 sobre alpine:3.23.7
│   └── web/
│       ├── src/componentes/    # microcomponentes React (<nombre>.tsx, <nombre>.api.ts, use-<nombre>.ts)
│       └── src/features/       # composición de vistas de desarrollo local
├── contracts/openapi.yaml      # contrato OpenAPI 3.1 generado y commiteado
├── docs/modos-de-falla.md
├── docker-compose.yaml
├── .env.example
├── lefthook.yml
└── .github/workflows/ci.yml
```

## Levantar el entorno

Desde la raíz, con Docker y Docker Compose v2 en ejecución:

```bash
docker compose up --build
```

- **Panel web** (React 19, `spa-server` en contenedor `:8080`): [http://127.0.0.1:5120](http://127.0.0.1:5120)
- **Healthcheck Web**: [http://127.0.0.1:5120/healthz](http://127.0.0.1:5120/healthz)
- **Healthcheck API (vivo)**: [http://127.0.0.1:5121/healthz](http://127.0.0.1:5121/healthz)
- **Readiness API (PostgreSQL)**: [http://127.0.0.1:5121/readyz](http://127.0.0.1:5121/readyz)
- **Diagnóstico de salud**: [http://127.0.0.1:5121/api/health](http://127.0.0.1:5121/api/health)
- **Documentación interactiva OpenAPI 3.1 (`API_ENV=dev`)**: [http://127.0.0.1:5121/api/docs](http://127.0.0.1:5121/api/docs)
- **Mailpit UI (perfil `dev`, `docker compose --profile dev up`)**: [http://127.0.0.1:8025](http://127.0.0.1:8025)

## Comandos principales (`Makefile`)

- `make dev`: Levanta PostgreSQL `17.11`, migraciones (`goose`), API Go (`apps/api`) y Frontend React (`apps/web`).
- `make prod`: Levanta la variante de producción con imágenes `distroless/static-debian13:nonroot` fijadas por digest SHA-256.
- `make codegen`: Regenera `contracts/openapi.yaml` desde los structs de Huma y los tipos TypeScript con `openapi-typescript`.
- `make sqlc`: Ejecuta `sqlc@v1.29.0` sobre `apps/api/db/queries`.
- `make test`: Ejecuta pruebas y cobertura en `apps/api` y `apps/web`.
