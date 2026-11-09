# Pipeline de Información · Grupo 2 · PINGESO

Base del microservicio **PT2** para el Hito 0 de Entorno y Seguridad. Su objetivo en la Línea de Productos de Software es recibir datos masivos del **PT5, Extractor de Datos OCR**, limpiarlos, validarlos y almacenarlos en MongoDB para que el **PT1, Motor de Reportes**, pueda consumirlos.

En el Hito 0 se entrega el entorno de Go, Vue y MongoDB, junto con el pipeline de calidad y seguridad. Para el Hito 1 quedan la publicación del contrato OpenAPI 3.1.0 y los endpoints con datos de prueba CSV o JSON. La lógica real de ETL y su persistencia corresponden al Hito 2.

## Levantar el entorno

Desde la raíz, con Docker y Docker Compose v2 en ejecución:

```bash
docker compose up --build
```

Abre el [panel del Grupo 2](http://localhost:3000). El panel consulta la [API de diagnóstico](http://localhost:8080/api/health), que hace un ping a MongoDB y muestra el resultado. Para detener los servicios, ejecuta `docker compose down`.

| Componente | Tecnología | Ubicación |
|---|---|---|
| Backend | Go y Gin | `backend/` |
| Frontend | Vue 3 con Composition API y Vite | `frontend/` |
| Base de datos | MongoDB 8.0 | `scripts/init-mongo.js` crea el usuario de aplicación |

El entorno de desarrollo tiene valores locales por defecto. Para personalizarlos, copia `.env.example` a `.env` (`Copy-Item .env.example .env` en PowerShell) y edita el archivo. MongoDB guarda sus datos en un volumen propio. El usuario de aplicación tiene rol `readWrite` solo sobre `MONGO_DATABASE`, y el script de creación se ejecuta cuando se inicializa un volumen nuevo. Si cambias credenciales en un volumen existente, actualiza el usuario en MongoDB antes de reiniciar la aplicación.

Para ejecutar Go y Vue en el equipo, copia primero `.env.example` a `.env`. Levanta MongoDB con `make local-db` y abre dos terminales para ejecutar `make local-back` y `make local-front`. Se requiere Go 1.26 o superior y Node.js 24 actualizado. Si cambias el puerto de MongoDB, ajusta también `MONGO_URI` para el backend nativo.

La variante de producción se construye con `docker compose -f docker-compose.prod.yml up -d --build`. Usa credenciales propias en `.env`. MongoDB queda en la red interna y Nginx reenvía `/api/` al backend desde el mismo origen del frontend.

### Probar el pipeline con datos simulados

Desde `frontend/`, ejecuta `npm ci` y `npm run dev`. En el panel, pulsa **Procesar lote** para ver los registros normalizados y el motivo de cada rechazo. Puedes editar el JSON o recuperar los datos con **Restaurar ejemplo**.

El lote de ejemplo simula una respuesta OCR con folios, nombres y correos. La prueba se ejecuta en el navegador y sus resultados son temporales.

## Rutas de diagnóstico

| Ruta | Resultado |
|---|---|
| `GET /health` y `GET /api/health` | HTTP 200 si MongoDB responde o 503 con estado `degraded` si está indisponible |
| `GET /api/info` | Identificación del PT2 y rutas previstas para una fase posterior |

## Calidad y seguridad del Hito 0

El workflow `hito-0.yml` se ejecuta en pull requests hacia `main`, pushes a `main` y ejecuciones manuales. En GitHub, los resultados de Trivy y la comprobación de integración se ven en la pestaña **Actions**.

- **Pruebas y cobertura.** Go genera `backend/coverage.out` y Vue genera `frontend/coverage/lcov.info`. El workflow también compila Vue y verifica que el frontend pueda consultar el backend y MongoDB por la ruta de Nginx.
- **Trivy.** Escanea dependencias del repositorio e imágenes de backend y frontend. En el Hito 0 publica los hallazgos en el log para establecer una línea base. Los umbrales de aprobación corresponden al Hito 2.
- **SonarQube.** Analiza Go y Vue en `main` e importa ambos informes de cobertura. El archivo `sonar-project.properties` concentra su configuración.

### Conectar SonarQube en GitHub

1. En una instancia de SonarQube Server accesible desde el runner de GitHub, crea el proyecto con clave `pingeso-grupo-2-pipeline-info` y genera un token de análisis.
2. En **Settings → Secrets and variables → Actions**, crea la variable de repositorio `SONAR_HOST_URL` con la URL del servidor y el secreto `SONAR_TOKEN` con el token.
3. Ejecuta **Hito 0 · Calidad y seguridad** desde **Actions** en la rama `main` y revisa el resultado del análisis y la cobertura en SonarQube.

Con Docker, puedes levantar una instancia local de SonarQube Community Build para la demostración:

```bash
docker run -d --name sonar-pingeso -p 127.0.0.1:9000:9000 sonarqube:community
```

Crea el proyecto en el [panel local de SonarQube](http://localhost:9000). Genera los reportes con `make test` y ejecuta `sonar-scanner` desde la raíz con `SONAR_HOST_URL` y `SONAR_TOKEN` definidos en el entorno de tu terminal. Usa un escáner con aprovisionamiento automático de Java o configura `JAVA_HOME` con JDK 21 o superior. El runner alojado por GitHub necesita una URL de SonarQube a la que pueda conectarse.

### Pruebas sin Makefile

```text
cd backend
go test -coverprofile=coverage.out ./...
cd ../frontend
npm ci
npm run test:coverage
npm run build
```
