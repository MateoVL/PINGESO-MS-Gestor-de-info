## 1. Introducción
**Regla general.** Lo que este documento marca como obligatorio lo verifica el pipeline. Un PR que no pasa el pipeline no se integra, sin excepciones manuales.


## 2. Arquitectura y contrato API

Cada microservicio tiene su propio repositorio, su propia base de datos, su propio pipeline y un contrato OpenAPI publicado. Ningún servicio accede a los datos de otro si no es por su API.

### Reglas de arquitectura

- **Dueño único por dato.** Cada dato lo escribe un solo servicio. Los demás lo piden por API.
- **Dependencias en un solo sentido.** No hay ciclos entre servicios. Si dos servicios se necesitan mutuamente, el límite entre ellos está mal trazado.
- **API-first.** Un servicio publica su contrato antes que su lógica, para que quien depende de él trabaje en paralelo con mocks.
- **Clean Architecture.** Handler → Service → Store dentro de cada servicio (sección 3).
- **Microcomponentes.** Cada servicio entrega su frontend como microcomponentes, que después compone cada grupo cliente (sección 3).

### Comunicación entre servicios

- **REST síncrono con JSON**, siempre contra el contrato publicado del otro servicio. Mensajería asíncrona solo si el documento de componentes de la sección la define.
- Un servicio llama a otro por su URL interna (`<SERVICIO>_URL`), reenviando el JWT del usuario y el `X-Request-Id`.
- **Una operación que cambia datos de otro servicio se confirma solo si ese servicio la confirma.** La llamada lleva `Idempotency-Key`, para que un reintento no la aplique dos veces. La reversa es una operación inversa con su propia clave.
- Timeouts explícitos en todo cliente HTTP (5 s por defecto). Un servicio caído devuelve un error claro, no cuelga la request.

### Contrato API-first

El contrato se publica antes que la lógica. Con Huma el contrato nace de los structs Go, así que API-first significa: primero structs y handlers que devuelven datos simulados, después la lógica real.

1. Definir structs de request y response en `internal/api/` con tags `json`, validaciones y ejemplos.
2. `pnpm codegen` genera `contracts/openapi.yaml` (OpenAPI 3.1) y los tipos TypeScript. Ambos se commitean.
3. El servicio consumidor copia el `openapi.yaml` del proveedor a `contracts/externos/<servicio>.yaml` y genera desde ahí su cliente (openapi-typescript en el front, oapi-codegen en Go).
4. Mientras el proveedor no tenga lógica, el consumidor trabaja contra `prism mock contracts/externos/<servicio>.yaml` o contra handlers MSW.
5. Un cambio incompatible (quitar un campo, cambiar un tipo, volver obligatorio un parámetro) lo bloquea oasdiff en CI. Si es necesario, se publica `/api/v2/...` y se avisa a los consumidores antes de retirar `v1`.

En `API_ENV=dev` cada servicio sirve su documentación interactiva en `/api/docs`.


## 3. Stack tecnológico y estructura de repositorio

Todos los servicios parten de la plantilla base: mismo stack, mismas versiones, misma estructura. Cambiar una pieza del stack requiere aprobación del jefe de proyecto y queda registrado aquí.

El frontend es React y la base de datos es PostgreSQL en todas las secciones.

Las versiones de abajo son exactas, sin rangos. Subir una versión es un PR propio, que pasa por el pipeline completo y actualiza esta tabla.

### Backend (Go)

| Componente | Versión | Dónde se fija | Uso |
| --- | --- | --- | --- |
| Go (toolchain) | 1.26.5 | `go.mod`: `go 1.26.0` + `toolchain go1.26.5` y la imagen de build | Todos |
| `github.com/danielgtaylor/huma/v2` | v2.34.1 | `go.mod` | HTTP + OpenAPI 3.1 |
| `github.com/jackc/pgx/v5` | v5.10.0 | `go.mod` | Driver PostgreSQL |
| `github.com/sqlc-dev/sqlc` | v1.29.0 | `Makefile` (`go run ...@v1.29.0`) | Generación de queries |
| `github.com/pressly/goose/v3` | v3.24.1 | `go.mod`, `Makefile`, `Dockerfile.migrate` | Migraciones |
| `github.com/riverqueue/river` (+ `riverpgxv5`, `rivertype`) | v0.40.0 | `go.mod` | Jobs, solo si el servicio los usa |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | `go.mod` | Emisión (servicio de identidad) y validación de JWT |
| `github.com/MicahParks/keyfunc/v3` | v3.9.1 | `go.mod` | Lectura y caché del JWKS |
| `golang.org/x/crypto` | v0.54.0 | `go.mod` | Hash de contraseñas, solo servicio de identidad |
| `github.com/oapi-codegen/oapi-codegen/v2` | v2.8.2 | `Makefile` (`go run ...@v2.8.2`) | Cliente Go de servicios externos |
| `github.com/testcontainers/testcontainers-go/modules/postgres` | v0.43.0 | `go.mod` | PostgreSQL real en pruebas |
| `github.com/wneessen/go-mail` | v0.8.1 | `go.mod` | SMTP, solo si el servicio envía correo |

### Frontend

| Componente | Versión | Dónde se fija |
| --- | --- | --- |
| Node.js | 22.22.0 | `.nvmrc` y `engines` de `package.json` y la imagen de build |
| pnpm | 10.13.1 | `packageManager` de `package.json` |
| react / react-dom | 19.2.8 | catalog de `pnpm-workspace.yaml` |
| typescript | 6.0.3 | catalog |
| vite | 8.1.5 | catalog |
| @vitejs/plugin-react | 6.0.4 | `apps/web/package.json` |
| tailwindcss / @tailwindcss/vite | 4.3.3 | `apps/web/package.json` |
| @base-ui/react (shadcn, estilo `base-nova`) | 1.6.0 | `apps/web/package.json`, `components.json` |
| react-router | 8.3.0 | `apps/web/package.json` |
| @tanstack/react-query | 5.101.4 | `apps/web/package.json` |
| react-hook-form | 7.83.0 | `apps/web/package.json` |
| openapi-fetch | 0.15.2 | `apps/web/package.json` |
| openapi-typescript | 7.13.0 | `apps/web/package.json` |
| msw | 2.15.0 | `apps/web/package.json` |
| vitest | 4.1.10 | `apps/web/package.json` |
| @playwright/test | 1.62.0 | `apps/web/package.json` |
| oxlint | 1.75.0 | `package.json` raíz |
| prettier / prettier-plugin-tailwindcss | 3.9.6 / 0.8.1 | `package.json` raíz |
| lefthook | 2.1.10 | `package.json` raíz |

### Imágenes base y servicios de Compose

| Imagen | Tag | Digest |
| --- | --- | --- |
| golang (build) | `1.26.5-alpine3.23` | `sha256:622e56dbc11a8cfe87cafa2331e9a201877271cbff918af53d3be315f3da88cc` |
| node (build web) | `22.22.0-alpine3.23` | `sha256:e4bf2a82ad0a4037d28035ae71529873c069b13eb0455466ae0bc13363826e34` |
| distroless (runtime api y web) | `static-debian13:nonroot` | `sha256:f7f8f729987ad0fdf6b05eeeae94b26e6a0f613bdf46feea7fc40f7bd72953e6` |
| alpine (runtime migrate) | `3.23.7` | Se fija en el Hito 0 |
| postgres | `17.11` | Se fija en el Hito 0 |
| axllent/mailpit (solo perfil dev) | `v1.31.3` | Se fija en el Hito 0 |

Para obtener un digest: `docker buildx imagetools inspect postgres:17.11 --format '{{json .Manifest.Digest}}'`. Se usa el digest del índice multi-arquitectura, no el de una plataforma.

### Reglas de fijación

- `.npmrc` con `save-exact=true`: `pnpm add` escribe `19.2.8`, no `^19.2.8`. El catalog también va sin `^`.
- `pnpm install --frozen-lockfile` en CI y en Docker, y `go mod verify` en CI. Si el lockfile no coincide, falla.
- Ninguna herramienta se invoca con `@latest` ni con un tag flotante como `postgres:17` o `alpine:3.23`.
- Las GitHub Actions se fijan por SHA de commit, con la versión en un comentario (sección 6).
- Dependabot o Renovate abre los PR de actualización. Nadie sube versiones a mano dentro de un PR de funcionalidad.

### Estructura de cada repo

```
<servicio>/
├── apps/
│   ├── api/
│   │   ├── cmd/api/            # main: arranque, config, señales
│   │   ├── internal/api/       # handlers Huma + structs del contrato
│   │   ├── internal/service/   # reglas de negocio, sin HTTP ni SQL
│   │   ├── internal/store/     # acceso a PostgreSQL
│   │   ├── internal/repository/# generado por sqlc, no editar
│   │   ├── db/migrations/      # goose
│   │   ├── db/queries/         # SQL para sqlc
│   │   ├── e2e/                # e2e + testdata/*.golden.md
│   │   ├── Dockerfile
│   │   └── Dockerfile.migrate
│   └── web/
│       └── src/features/<feature>/
├── contracts/openapi.yaml      # generado y commiteado
├── docs/modos-de-falla.md
├── docker-compose.yaml
├── .env.example
├── lefthook.yml
└── .github/workflows/ci.yml
```

### Capas del backend

Handler → Service → Store. El handler solo traduce HTTP, el service no conoce HTTP ni SQL y el store no tiene reglas de negocio. Un handler que llama al store directamente se rechaza en revisión.

### Frontend: microcomponentes

Cada servicio entrega su frontend como microcomponentes: componentes React autocontenidos, cada uno con su propio cliente de API tipado. La base común queda formada por las APIs y estos microcomponentes, y cada grupo cliente los compone después en su propio producto.

- Un microcomponente no importa a otro ni depende de un layout, una ruta o un estado global.
- Recibe por props la URL base de su API y lo que necesite del contexto (usuario, permisos).
- Estructura: `src/componentes/<kebab-case>/` con `<nombre>.tsx` (vista, sin lógica de negocio), `<nombre>.api.ts` (único lugar con `fetch`) y `use-<nombre>.ts` (estado).
- Se publica como paquete npm versionado (`@<org>/<servicio>-ui`) y cada versión declara qué versión del contrato API consume.
- El repo incluye una app de desarrollo local que monta los microcomponentes con mocks MSW. Esa app no se publica.


