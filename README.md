# Pipeline de Información · Grupo 2 · PINGESO

Base del microservicio **PT2 (Gestor de Información)** alineada con las reglas de arquitectura, contrato API-first y versiones exactas de `AGENTS.md`. Su objetivo en la Línea de Productos de Software es recibir datos masivos del **PT5, Extractor de Datos OCR**, limpiarlos, validarlos y almacenarlos en PostgreSQL para que el **PT1, Motor de Reportes**, pueda consumirlos.

## Stack y estructura del repositorio (`AGENTS.md`)

| Componente             | Tecnología y Versión Exacta                                                                                        | Ubicación                |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------ | ------------------------ |
| **Backend**            | Go `1.26.11` (`go 1.26.0` + `toolchain go1.26.11`), Huma `v2.34.1`, pgx `v5.10.0`, sqlc `v1.29.0`, goose `v3.24.1` | `apps/api/`              |
| **Frontend**           | Node `22.22.0`, pnpm `10.13.1`, React `19.2.8`, TypeScript `6.0.3`, Vite `8.1.6`, Tailwind `4.3.3`                 | `apps/web/`              |
| **Base de datos**      | PostgreSQL `17.11`                                                                                                 | `docker-compose.yaml`    |
| **Contrato API-first** | OpenAPI 3.1.0 generado desde structs de Huma y commiteado                                                          | `contracts/openapi.yaml` |
| **Modos de falla**     | Documentación de resiliencia, idempotencia y timeouts                                                              | `docs/modos-de-falla.md` |

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
│   │   ├── Dockerfile          # golang:1.26.11-alpine3.23 -> distroless static-debian13:nonroot
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

## Escáneres obligatorios: SonarQube y Trivy (`AGENTS.md` sección 7)

- **Pipeline de CI (GitHub Actions)**: [https://github.com/MateoVL/PINGESO-MS-Gestor-de-info/actions/workflows/ci.yml](https://github.com/MateoVL/PINGESO-MS-Gestor-de-info/actions/workflows/ci.yml)
- **Organización SonarQube Cloud (`sonar.organization`)**: `pingeso-gestor-info` (`sonar-project.properties`)
- **Clave del proyecto SonarQube (`sonar.projectKey`)**: `MateoVL_PINGESO-MS-Gestor-de-info` (`sonar-project.properties`)
- **Requisito en SonarQube Cloud**: En **Administration → Analysis Method**, mantener **Automatic Analysis** desactivado para permitir el análisis desde GitHub Actions (`CI-based Analysis`) con lectura de cobertura (`coverage.out` y `apps/web/coverage/lcov.info`).
- **Ejecución local de SonarQube**:
  ```bash
  pnpm test:coverage && pnpm test:e2e:api
  sonar-scanner -Dsonar.host.url="$SONAR_HOST_URL" -Dsonar.token="$SONAR_TOKEN"
  ```
- **Ejecución local de Trivy (`v0.76.1`)**:
  ```bash
  trivy fs --scanners vuln,misconfig,secret --severity HIGH,CRITICAL --ignore-unfixed --exit-code 1 .
  docker build -t api:local -f apps/api/Dockerfile .
  trivy image --scanners vuln,secret --severity HIGH,CRITICAL --ignore-unfixed --exit-code 1 api:local
  docker build -t web:local -f apps/web/Dockerfile .
  trivy image --scanners vuln,secret --severity HIGH,CRITICAL --ignore-unfixed --exit-code 1 web:local
  ```

### Registro de actualizaciones de seguridad del stack (`AGENTS.md` secciones 3, 6 y 7)

#### ¿Por qué se actualizan estas versiones respecto a la plantilla inicial?

El repositorio aplica fijación estricta de versiones (`save-exact=true` en npm, catálogo sin rangos en `pnpm-workspace.yaml`, `toolchain` exacto en `go.mod` e imágenes Docker por digest SHA-256). Al mismo tiempo, el pipeline de CI (`.github/workflows/ci.yml`) consulta bases de datos de vulnerabilidades en tiempo real en cada ejecución:

- **`govulncheck` (`v1.9.0`)**: Falla ante cualquier vulnerabilidad conocida en la librería estándar del toolchain de Go activo o en símbolos alcanzables de módulos en `go.mod`.
- **`pnpm audit --audit-level high`**: Falla si existe al menos una vulnerabilidad de severidad `HIGH` o `CRITICAL` en dependencias directas o transitivas de `pnpm-lock.yaml` (las vulnerabilidades `low` y `moderate` no bloquean el pipeline).
- **`trivy fs` y `trivy image` (`v0.76.1`)**: Fallan ante cualquier CVE `HIGH` o `CRITICAL` con corrección disponible (`--ignore-unfixed`) tanto en lockfiles (`go.sum`, `pnpm-lock.yaml`) como en los binarios compilados dentro de las imágenes Docker.

Siguiendo la directriz de `AGENTS.md` sección 3 (_"Subir una versión es un PR propio, que pasa por el pipeline completo y actualiza esta tabla"_) y sección 7.5 (_"Si hay un hallazgo con corrección disponible, se actualiza la dependencia o la imagen base y se reconstruye. No se baja la severidad ni se ignora el CVE"_), se aplicaron las siguientes actualizaciones de seguridad:

| Componente / Paquete                  | Versión anterior                         | Versión actualizada                       | Dónde se fija                                                                            | Justificación técnica y CVEs remediados                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| ------------------------------------- | ---------------------------------------- | ----------------------------------------- | ---------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Go (toolchain)**                    | `1.26.5`                                 | `1.26.11`                                 | `apps/api/go.mod`, `AGENTS.md`                                                           | `actions/setup-go` instala la versión declarada en `apps/api/go.mod`. Go `1.26.5` presentaba **17 vulnerabilidades en la librería estándar** (`net/http`, `crypto/x509`, `crypto/tls`, `net/textproto`, `net/url`, `encoding/xml`, `encoding/asn1`): `GO-2026-6859`, `GO-2026-6809`, `GO-2026-6617`, `GO-2026-6613`, `GO-2026-6612`, `GO-2026-6611`, `GO-2026-6610`, `GO-2026-6608`, `GO-2026-6607`, `GO-2026-6605`, `GO-2026-6603`, `GO-2026-6218`, `GO-2026-6090`, `GO-2026-6089`, `GO-2026-6088`, `GO-2026-5972` y `GO-2026-5026`. Como `govulncheck` no permite ignorar vulnerabilidades de la stdlib, se actualizó al parche `1.26.11` donde quedan todas corregidas. |
| **`golang` (imagen Docker de build)** | `1.26.5-alpine3.23` (`sha256:622e56db…`) | `1.26.11-alpine3.23` (`sha256:23f19893…`) | `apps/api/Dockerfile`, `apps/api/Dockerfile.migrate`, `apps/web/Dockerfile`, `AGENTS.md` | Alineación de la etapa de compilación multi-arquitectura con Go `1.26.11` para que los binarios `/app/servicio`, `/usr/local/bin/goose` y `/app/spa-server` no embeban la stdlib vulnerable al ser escaneados por `trivy image`.                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| **`golang.org/x/text`**               | `v0.37.0`                                | `v0.42.0`                                 | `apps/api/go.mod`, `apps/api/go.sum`                                                     | Remediación de `GO-2026-6629` (fijo en `v0.41.0`) y `GO-2026-5970` (fijo en `v0.39.0`) en `x/text/secure/precis` y `x/text/unicode/norm`, alcanzables transitivamente desde `github.com/jackc/pgx/v5/pgxpool`.                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| **`vite`**                            | `8.1.5`                                  | `8.1.6`                                   | `pnpm-workspace.yaml` (`catalog`), `pnpm-lock.yaml`, `AGENTS.md`                         | Remediación de `GHSA-587h-7wmj-h5pm` (severidad `HIGH`: bypass de `fs.existsSync()` en `launch-editor`), detectado por `pnpm audit --audit-level high`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| **`lodash`** _(transitiva)_           | `<=4.17.23`                              | `4.18.0`                                  | `package.json` (`pnpm.overrides`), `pnpm-lock.yaml`                                      | Remediación de `GHSA-r5fr-rjxr-66jc` (severidad `HIGH`: inyección de código vía `_.template`), introducida transitivamente por `@stoplight/prism-cli > ... > postman-collection`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| **`@faker-js/faker`** _(transitiva)_  | `<=10.4.0`                               | `10.5.0`                                  | `package.json` (`pnpm.overrides`), `pnpm-lock.yaml`                                      | Remediación de `GHSA-qxc2-j82w-r537` (severidad `HIGH`: ejecución arbitraria de código en `helpers.fake`), introducida transitivamente por `@stoplight/prism-cli > ... > postman-collection`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
