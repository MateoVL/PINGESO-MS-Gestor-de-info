# Pipeline de Información · Grupo 2 · PINGESO

Base del microservicio **PT2 (Gestor de Información)** alineada con las reglas de arquitectura, contrato API-first y versiones exactas de `AGENTS.md`. Su objetivo en la Línea de Productos de Software es recibir datos masivos del **PT5, Extractor de Datos OCR**, limpiarlos, validarlos y almacenarlos en PostgreSQL para que el **PT1, Motor de Reportes**, pueda consumirlos.

## Índice

- [Stack y estructura del repositorio (`AGENTS.md`)](#stack-y-estructura-del-repositorio-agentsmd)
  - [Estructura de carpetas](#estructura-de-carpetas)
- [Guía para Desarrolladores](#guía-para-desarrolladores)
  - [1. Tecnologías utilizadas y su rol](#1-tecnologías-utilizadas-y-su-rol)
  - [2. ¿Qué debes descargar e instalar?](#2-qué-debes-descargar-e-instalar)
  - [3. ¿Cómo trabajar en el repositorio?](#3-cómo-trabajar-en-el-repositorio)
  - [4. SonarQube Cloud: Qué hace, cómo funciona y cómo ingresar](#4-sonarqube-cloud-qué-hace-cómo-funciona-y-cómo-ingresar)
- [Comandos principales (`Makefile`)](#comandos-principales-makefile)
- [Escáneres obligatorios: SonarQube y Trivy (`AGENTS.md` sección 7)](#escáneres-obligatorios-sonarqube-y-trivy-agentsmd-sección-7)
  - [Registro de actualizaciones de seguridad del stack (`AGENTS.md` secciones 3, 6 y 7)](#registro-de-actualizaciones-de-seguridad-del-stack-agentsmd-secciones-3-6-y-7)

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

## Guía para Desarrolladores

### 1. Tecnologías utilizadas y su rol

#### Backend (`apps/api/`)

- **Go (`1.26.11`)**: Lenguaje principal del servicio y del servidor estático (`spa-server`) del frontend en producción.
- **Huma (`v2.34.1`)**: Framework HTTP y generador del contrato **OpenAPI 3.1** directamente desde structs de Go (enfoque _API-first_).
- **pgx (`v5.10.0`)**: Driver y pool de conexiones de alto rendimiento para PostgreSQL.
- **sqlc (`v1.29.0`)**: Compilador que transforma consultas SQL (`db/queries/*.sql`) en código Go fuertemente tipado (`internal/repository/`).
- **Goose (`v3.24.1`)**: Herramienta para gestionar migraciones incrementales del esquema de PostgreSQL (`db/migrations/`).
- **oapi-codegen (`v2.8.2`)**: Generador de clientes HTTP en Go a partir de contratos OpenAPI de otros microservicios (`contracts/externos/`).
- **golang-jwt (`v5.3.1`) + keyfunc (`v3.9.1`)**: Validación de tokens JWT y lectura/caché de llaves públicas (JWKS) del servicio de identidad.
- **Testcontainers (`v0.43.0`)**: Levanta instancias reales de PostgreSQL en contenedores efímeros durante las pruebas de integración y E2E.

#### Frontend (`apps/web/`)

- **React (`19.2.8`) + TypeScript (`6.0.3`) + Vite (`8.1.6`)**: Base para construir los microcomponentes autocontenidos (`@pingeso/ms-gestor-de-info-ui`) y el entorno de desarrollo local.
- **Tailwind CSS (`4.3.3`) + Base UI (`1.6.0`)**: Sistema de estilos utilitarios y primitivas de interfaz accesibles.
- **openapi-typescript (`7.13.0`) + openapi-fetch (`0.15.2`)**: Generan tipos TypeScript desde `contracts/openapi.yaml` y proveen un cliente HTTP tipado sin desincronización con el backend.
- **TanStack Query (`5.101.4`) + React Hook Form (`7.83.0`)**: Manejo de estado asíncrono/caché de peticiones y gestión de formularios.
- **MSW (`2.15.0`) + Prism (`5.16.0`)**: Simulación (mocking) de APIs para desarrollar y probar microcomponentes en paralelo antes de tener la lógica final del backend.
- **Vitest (`4.1.10`) + Playwright (`1.62.0`)**: Ejecución de pruebas unitarias, de componentes y pruebas End-to-End (E2E).

#### Base de Datos, Calidad, Seguridad y Contratos

- **PostgreSQL (`17.11`)**: Base de datos relacional exclusiva del microservicio.
- **Lefthook (`2.1.10`)**: Gestor de Git Hooks (`pre-commit` y `pre-push`) que ejecuta validaciones automáticas antes de permitir un commit o un push.
- **Oxlint (`1.75.0`) + Prettier (`3.9.6`)**: Linter de alta velocidad y formateador de código para TypeScript/React. Bloquea malas prácticas como usar `fetch` fuera de `*.api.ts` o usar `localStorage`.
- **Gitleaks (`v8.30.1`)**: Escáner de secretos (contraseñas, API keys, tokens JWT, llaves privadas). Corre tanto en tu máquina antes de cada commit (`gitleaks protect --staged`) como en GitHub Actions sobre todo el historial para impedir que una credencial llegue al repositorio.
- **govulncheck (`v1.9.0`)**: Analizador oficial de seguridad de Go. A diferencia de un escáner común, analiza el grafo de llamadas del código para detectar si la librería estándar de Go o algún paquete de `go.mod` tiene vulnerabilidades conocidas en funciones que nuestro código realmente invoca.
- **Trivy (`v0.76.1`)**: Escáner integral de seguridad que actúa en dos etapas (`trivy fs` y `trivy image`):
  1. **En el repositorio (`fs`)**: Revisa `go.sum` y `pnpm-lock.yaml` buscando CVEs (`HIGH` o `CRITICAL` con parche disponible), detecta errores de configuración en `Dockerfile` y `docker-compose.yaml`, y busca secretos expuestos.
  2. **En las imágenes construidas (`image`)**: Inspecciona las capas finales de `api` y `web` para asegurar que ni el sistema base (`distroless`/`alpine`) ni los binarios compilados contengan vulnerabilidades o secretos embebidos.
- **Hadolint (`v2.15.1`)**: Linter de `Dockerfile` que valida buenas prácticas de construcción de imágenes (fijación de imágenes por digest SHA-256, uso correcto de instrucciones `RUN`/`COPY`, usuario no root, etc.). Falla ante cualquier regla de nivel `error`.
- **Dockle (`v0.4.15`)**: Auditor de seguridad para imágenes de contenedores ya construidas (benchmarks CIS). Verifica que la imagen final no corra como `root`, no tenga archivos sensibles, ni exponga credenciales o configuraciones inseguras en sus metadatos.
- **oasdiff (`v1.35.0`)**: Comparador de contratos OpenAPI. En cada PR compara tu `contracts/openapi.yaml` contra el de la rama `main` y bloquea el pipeline si detecta un cambio incompatible (_breaking change_, como eliminar un campo o cambiar un tipo de dato).
- **SonarQube Cloud**: Plataforma de inspección continua de calidad y seguridad sobre el código propio (detallada en la sección 4).

---

### 2. ¿Qué debes descargar e instalar?

Para trabajar en el repositorio debes instalar las siguientes herramientas respetando las versiones exactas fijadas en `AGENTS.md`:

1. **Git**: Para el control de versiones.
2. **Docker, Docker Compose v2 y Docker Buildx**:
   - **Importante sobre `docker-buildx`**: Debes tener instalado el plugin **Buildx** (`docker buildx version`). En muchas distribuciones de Linux viene en un paquete separado (ej. `docker-buildx-plugin` o `docker-buildx`). Es obligatorio porque nuestros Dockerfiles usan características avanzadas de BuildKit (`# syntax=docker/dockerfile:1.7`, cachés con `RUN --mount=type=cache` y variables multi-arquitectura `--platform=$BUILDPLATFORM`), además de usarse para consultar digests multi-arquitectura (`docker buildx imagetools inspect`).
3. **GNU Make**: Para utilizar los comandos del `Makefile` (incluido en Linux/macOS; en Windows usar WSL2).
4. **Go `1.26.11`**: Requerido para compilar el backend, correr pruebas y ejecutar herramientas como `sqlc`, `goose`, `govulncheck` y `oasdiff`.
5. **Node.js `22.22.0`**: Recomendado instalarlo mediante `nvm` o `fnm` (el repositorio incluye `.nvmrc`, por lo que basta correr `nvm use`).
6. **pnpm `10.13.1`**: Gestor de paquetes del monorepo. Puedes activarlo con Node ejecutando:
   ```bash
   corepack enable && corepack prepare pnpm@10.13.1 --activate
   ```
7. **Gitleaks `v8.30.1`**: **Obligatorio en local**, ya que el hook `pre-commit` de Lefthook lo ejecuta automáticamente en cada commit. Si no lo tienes instalado en tu `PATH`, el commit fallará.
8. **Trivy `v0.76.1` y SonarScanner CLI** _(opcionales pero recomendados)_: Para ejecutar en tu máquina las mismas auditorías de vulnerabilidades y calidad que correrán en GitHub Actions.

---

### 3. ¿Cómo trabajar en el repositorio?

#### Paso 1: Configuración inicial (solo la primera vez)

```bash
# 1. Instalar dependencias de Node (respeta el lockfile exacto)
pnpm install --frozen-lockfile

# 2. Activar los Git Hooks locales (.npmrc tiene ignore-scripts=true por seguridad)
pnpm exec lefthook install

# 3. Descargar y verificar módulos de Go
cd apps/api && go mod download && cd ../..

# 4. Generar el archivo .env local con valores funcionales de desarrollo
make env
```

#### Paso 2: Levantar el entorno de desarrollo

Tienes dos modos de trabajo según lo que estés desarrollando:

- **Opción A — Todo en Docker Compose (recomendado para probar integración completa):**
  ```bash
  make dev        # Construye y levanta postgres, migraciones, backend (:5121) y frontend (:5120)
  make down       # Detiene los contenedores
  ```
- **Opción B — Ejecución nativa en tu máquina (más rápida para iterar código):**
  ```bash
  make local-db      # 1. Levanta únicamente PostgreSQL en Docker (:5432)
  make migrate-up    # 2. Aplica las migraciones en la base local con goose
  make local-back    # 3. En una terminal: corre la API Go en el host
  make local-front   # 4. En otra terminal: corre el servidor Vite con Hot-Reload
  ```

**Endpoints locales para verificar que todo levantó bien:**

- **Panel web** (React 19, `spa-server` en contenedor `:8080`): [http://127.0.0.1:5120](http://127.0.0.1:5120)
- **Healthcheck Web**: [http://127.0.0.1:5120/healthz](http://127.0.0.1:5120/healthz)
- **Healthcheck API (vivo)**: [http://127.0.0.1:5121/healthz](http://127.0.0.1:5121/healthz)
- **Readiness API (PostgreSQL)**: [http://127.0.0.1:5121/readyz](http://127.0.0.1:5121/readyz)
- **Diagnóstico de salud**: [http://127.0.0.1:5121/api/health](http://127.0.0.1:5121/api/health)
- **Documentación interactiva OpenAPI 3.1 (`API_ENV=dev`)**: [http://127.0.0.1:5121/api/docs](http://127.0.0.1:5121/api/docs)
- **Mailpit UI (perfil `dev`, `docker compose --profile dev up`)**: [http://127.0.0.1:8025](http://127.0.0.1:8025)

#### Paso 3: Flujo de desarrollo diario

1. **Respetar Clean Architecture en el Backend (`Handler → Service → Store`)**:
   - `internal/api/`: Solo traduce HTTP y define structs del contrato Huma. **Nunca** llama directo al `store`.
   - `internal/service/`: Reglas de negocio puras (sin detalles HTTP ni SQL).
   - `internal/store/` + `internal/repository/`: Acceso a PostgreSQL.
2. **Contrato API-first (cuando modifiques o agregues un endpoint)**:
   - Edita los structs/handlers en `apps/api/internal/api/`.
   - Ejecuta `make codegen` (o `pnpm codegen`) para actualizar `contracts/openapi.yaml` y regenerar los tipos TypeScript en `apps/web/src/types/openapi.d.ts`. **Ambos archivos deben incluirse en el commit**.
3. **Base de datos y consultas SQL**:
   - Para cambios de esquema, crea una migración nueva en `apps/api/db/migrations/` y aplícala con `make migrate-up`.
   - Si agregas o modificas queries en `apps/api/db/queries/*.sql`, ejecuta `make sqlc` para regenerar `apps/api/internal/repository/` (nunca edites `internal/repository/` a mano).
4. **Microcomponentes en el Frontend**:
   - Cada componente vive en `apps/web/src/componentes/<kebab-case>/` dividido en tres archivos: `<nombre>.tsx` (vista sin lógica), `<nombre>.api.ts` (único archivo donde se permite usar `fetch`/cliente API) y `use-<nombre>.ts` (estado y hooks).
   - Está prohibido usar `localStorage` o `sessionStorage`, e importar código entre microcomponentes distintos.

#### Paso 4: Verificación antes de hacer Commit / Pull Request

Antes de subir tus cambios y abrir un PR, ejecuta:

```bash
# Ejecuta linter, formato, chequeo de tipos, pruebas con cobertura, build y go vet
pnpm verify

# Verifica que el contrato OpenAPI y los tipos TS estén sincronizados sin cambios pendientes
pnpm codegen && git diff --exit-code
```

> **Nota:** Nunca subas versiones de dependencias a mano dentro de un PR de funcionalidad; el repositorio exige fijación exacta (`save-exact=true`) verificada en CI.

---

### 4. SonarQube Cloud: Qué hace, cómo funciona y cómo ingresar

#### ¿Qué hace SonarQube Cloud?

Mientras que herramientas como Trivy o `govulncheck` vigilan las dependencias externas y contenedores, **SonarQube Cloud** audita el **código propio** escrito por el equipo en `apps/api` (Go) y `apps/web/src` (TypeScript/React). Detecta:

- **Bugs y Vulnerabilidades** en la lógica de negocio.
- **Code Smells** (problemas de mantenibilidad y complejidad cognitiva).
- **Security Hotspots** (fragmentos sensibles de seguridad que requieren revisión humana).
- **Porcentaje de código duplicado** y **Cobertura de pruebas (Coverage)** sobre las líneas nuevas o modificadas.

#### ¿Cómo funciona en nuestro pipeline?

1. **Generación de cobertura en CI**: En cada push o Pull Request, el job `calidad` de GitHub Actions ejecuta:
   - `pnpm test:coverage` en el frontend, generando el reporte LCOV en `apps/web/coverage/lcov.info`.
   - `pnpm test:e2e:api` en el backend, que compila el binario de la API instrumentado con `go build -cover`, ejecuta las pruebas unitarias y E2E reales, y exporta el perfil de cobertura a `coverage.out`.
2. **Escaneo (`sonarqube-scan-action`)**: Lee la configuración de `sonar-project.properties` en la raíz del repositorio (donde se excluyen archivos autogenerados como `internal/repository/**` o `src/types/**`, mocks y entrypoints) y envía el código junto con ambos reportes de cobertura a SonarQube Cloud.
3. **Bloqueo por Quality Gate (`sonarqube-quality-gate-action`)**: El pipeline espera el veredicto de SonarQube Cloud. Para que el PR pueda integrarse a `main`, el **código nuevo** debe cumplir el _Quality Gate_:
   - **0** Bugs y **0** Vulnerabilidades nuevas.
   - **100%** de Security Hotspots revisados.
   - **< 3%** de líneas duplicadas.
   - **≥ 60%** de cobertura de pruebas en código nuevo.

#### ¿Cómo ingresar a SonarQube Cloud?

1. Entra a [https://sonarcloud.io](https://sonarcloud.io) e inicia sesión con tu cuenta de **GitHub**.
2. Accede directamente al panel del proyecto mediante este enlace:
   - **Dashboard del proyecto**: [https://sonarcloud.io/project/overview?id=MateoVL_PINGESO-MS-Gestor-de-info](https://sonarcloud.io/project/overview?id=MateoVL_PINGESO-MS-Gestor-de-info)
   - **Organización (`sonar.organization`)**: `mateovl`
   - **Project Key (`sonar.projectKey`)**: `MateoVL_PINGESO-MS-Gestor-de-info`
3. Dentro del panel, en la pestaña **Pull Requests** o **Branches**, puedes revisar el detalle exacto de por qué falló un Quality Gate (qué líneas nuevas no tienen tests, qué _Code Smell_ apareció o qué _Security Hotspot_ falta revisar).
4. Si necesitas permisos de administración o revisión de _Security Hotspots_, solicítalos al administrador de la organización en GitHub/SonarQube Cloud.

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
- **Alcance de cobertura en SonarQube Cloud (`sonar.exclusions`)**: Dado que el nuevo motor de alcance de SonarQube Cloud analiza todo el repositorio por defecto e ignora `sonar.sources`, se excluyen explícitamente en `sonar-project.properties` los archivos generados (`apps/api/internal/repository/**`, `apps/web/src/types/**`), entrypoints de proceso (`apps/api/cmd/**`, `apps/web/server.go`, `apps/web/src/main.tsx`), mocks de desarrollo (`apps/web/src/mocks/**`) y archivos de prueba para que el Quality Gate mida únicamente la cobertura sobre las capas de negocio y microcomponentes.
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
| **`golang.org/x/crypto`**             | `v0.51.0`                                | `v0.55.0`                                 | `apps/api/go.mod`, `apps/api/go.sum`, `AGENTS.md`                                        | Remediación de 10 CVEs de severidad `HIGH` en `golang.org/x/crypto/ssh` detectados por `trivy fs` sobre `apps/api/go.mod` (`CVE-2026-39828`, `CVE-2026-39829`, `CVE-2026-39830`, `CVE-2026-39831`, `CVE-2026-39832`, `CVE-2026-39835`, `CVE-2026-42508`, `CVE-2026-46595`, `CVE-2026-46597`, `CVE-2026-56854`), requeridos transitivamente por `testcontainers-go`.                                                                                                                                                                                                                                                                                                        |
| **`github.com/klauspost/compress`**   | `v1.18.5`                                | `v1.18.7`                                 | `apps/api/go.mod`, `apps/api/go.sum`                                                     | Remediación de `CVE-2026-63209` (severidad `HIGH`: DoS por desbordamiento entero en procesamiento de diccionarios), detectado por `trivy fs` en `apps/api/go.mod` (dependencia transitiva de `testcontainers-go`).                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| **`github.com/moby/go-archive`**      | `v0.2.0`                                 | `v0.3.0`                                  | `apps/api/go.mod`, `apps/api/go.sum`                                                     | Remediación de `CVE-2026-17106` (severidad `HIGH`: escritura arbitraria de archivos vía enlaces simbólicos al extraer tar), detectado por `trivy fs` en `apps/api/go.mod` (dependencia transitiva de `testcontainers-go`).                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| **`golang.org/x/crypto`**             | `v0.51.0`                                | `v0.55.0`                                 | `apps/api/go.mod`, `apps/api/go.sum`, `AGENTS.md`                                        | Remediación de 10 CVEs de severidad `HIGH` en `golang.org/x/crypto/ssh` detectados por `trivy fs` sobre `apps/api/go.mod` (`CVE-2026-39828`, `CVE-2026-39829`, `CVE-2026-39830`, `CVE-2026-39831`, `CVE-2026-39832`, `CVE-2026-39835`, `CVE-2026-42508`, `CVE-2026-46595`, `CVE-2026-46597`, `CVE-2026-56854`), requeridos transitivamente por `testcontainers-go`.                                                                                                                                                                                                                                                                                                        |
| **`github.com/klauspost/compress`**   | `v1.18.5`                                | `v1.18.7`                                 | `apps/api/go.mod`, `apps/api/go.sum`                                                     | Remediación de `CVE-2026-63209` (severidad `HIGH`: DoS por desbordamiento entero en procesamiento de diccionarios), detectado por `trivy fs` en `apps/api/go.mod` (dependencia transitiva de `testcontainers-go`).                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| **`github.com/moby/go-archive`**      | `v0.2.0`                                 | `v0.3.0`                                  | `apps/api/go.mod`, `apps/api/go.sum`                                                     | Remediación de `CVE-2026-17106` (severidad `HIGH`: escritura arbitraria de archivos vía enlaces simbólicos al extraer tar), detectado por `trivy fs` en `apps/api/go.mod` (dependencia transitiva de `testcontainers-go`).                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| **`vite`**                            | `8.1.5`                                  | `8.1.6`                                   | `pnpm-workspace.yaml` (`catalog`), `pnpm-lock.yaml`, `AGENTS.md`                         | Remediación de `GHSA-587h-7wmj-h5pm` (severidad `HIGH`: bypass de `fs.existsSync()` en `launch-editor`), detectado por `pnpm audit --audit-level high`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| **`lodash`** _(transitiva)_           | `<=4.17.23`                              | `4.18.0`                                  | `package.json` (`pnpm.overrides`), `pnpm-lock.yaml`                                      | Remediación de `GHSA-r5fr-rjxr-66jc` (severidad `HIGH`: inyección de código vía `_.template`), introducida transitivamente por `@stoplight/prism-cli > ... > postman-collection`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| **`@faker-js/faker`** _(transitiva)_  | `<=10.4.0`                               | `10.5.0`                                  | `package.json` (`pnpm.overrides`), `pnpm-lock.yaml`                                      | Remediación de `GHSA-qxc2-j82w-r537` (severidad `HIGH`: ejecución arbitraria de código en `helpers.fake`), introducida transitivamente por `@stoplight/prism-cli > ... > postman-collection`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
