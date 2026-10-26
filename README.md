# MS Gestor de Información - PINGESO

Microservicio base de **Gestor de Información**, desarrollado en el marco del ramo **Proyecto de Ingeniería de Software (PINGESO)** para una arquitectura de **Línea de Productos de Software (LPS)**.

---

## 📌 Contexto en la Línea de Productos de Software (LPS)

En el ecosistema de la línea de productos, este microservicio es el encargado de recibir información no estructurada/estructurada, procesarla, ordenarla y dejarla disponible para los servicios consumidores:

| Microservicio | Qué contiene | Dependencias |
|---|---|---|
| **MS Reportes** | Generación de reportes finales (PDF, Excel) | Depende de: Plantillas de reporte, **Gestor de Info** |
| **MS Gestor de Info (Este repo)** | **Carga de información, procesamiento, orden** | **Depende de: Digitalización** |
| **MS Plantillas de reporte** | Guardar, crear y versionar plantillas | Depende de: Usuarios y Roles |
| **MS Formularios** | Motor CRUD y recolección dinámica de respuestas | Depende de: Usuarios y Roles |
| **MS Digitalización** | Captura de documentos, OCR | Depende de: Gestor Documental |
| **MS Usuarios y Roles** | Perfiles (MAD), Autenticación, Roles | Independiente (Servicio Base) |
| **MS Gestor Documental** | Almacenamiento de archivos, firmas | Depende de: Usuarios y Roles |
| **MS Auditoría** | Trazabilidad de eventos, logs centrales | Recibe eventos de todos los MS |
| **MS Agenda** | Gestión de reservas, bloqueos y calendarios | Depende de: Notificaciones |
| **MS Notificaciones** | Gateway centralizado de correos y alertas | Depende de: Usuarios y Roles |

---

## 🛠️ Stack Tecnológico

- **Backend:** Go 1.23 con framework Gin HTTP, CORS configurado, pool de conexiones y apagado elegante (*graceful shutdown*).
- **Frontend:** Vue 3 + Vite, diseño moderno responsive con monitor de salud y métricas de conectividad.
- **Base de Datos:** PostgreSQL 16 con script de inicialización (`scripts/init-db.sql`) y pruebas de salud automáticas (*healthcheck*).
- **Orquestación:** Docker & Docker Compose (ambientes separados para desarrollo con *hot reload* y producción multi-stage).
- **Automatización:** `Makefile` con comandos estandarizados.

---

## 📁 Estructura del Repositorio

```text
PINGESO-MS-Gestor-de-info/
├── backend/
│   ├── cmd/api/main.go          # Punto de entrada de la API en Go
│   ├── internal/
│   │   ├── config/config.go     # Carga de variables de entorno (.env)
│   │   ├── database/database.go # Conexión resiliente a PostgreSQL
│   │   └── handlers/health.go   # Handlers HTTP de salud y estado
│   ├── go.mod / go.sum          # Dependencias de Go
│   ├── Dockerfile.dev           # Contenedor de desarrollo Go
│   └── Dockerfile               # Contenedor multi-stage optimizado para producción
├── frontend/
│   ├── src/
│   │   ├── App.vue              # Vista principal con diagnósticos de conexión
│   │   ├── main.js              # Inicialización de Vue 3
│   │   └── assets/main.css      # Sistema de diseño y estilos CSS
│   ├── index.html               # Plantilla HTML con tipografía moderna
│   ├── package.json             # Dependencias de Vue y Vite
│   ├── vite.config.js           # Configuración de Vite con polling para Docker
│   ├── nginx.conf               # Servidor web Nginx para producción
│   ├── Dockerfile.dev           # Contenedor de desarrollo Node + Vite
│   └── Dockerfile               # Contenedor multi-stage de producción (Nginx Alpine)
├── scripts/
│   └── init-db.sql              # Script SQL inicial de PostgreSQL
├── .env.example                 # Plantilla de variables de entorno documentada y funcional
├── .gitignore                   # Exclusión de credenciales (.env), binarios y node_modules
├── Makefile                     # Automatización de tareas (local, docker, prod)
├── docker-compose.yml           # Orquestación de desarrollo local (hot reload)
├── docker-compose.prod.yml      # Orquestación de producción (multi-stage + Nginx)
└── README.md
```

---

## 🔐 Manejo de Variables de Entorno (.env)

> [!IMPORTANT]
> **Nunca incluyas credenciales sensibles ni archivos `.env` en los commits.** El archivo `.gitignore` ya excluye `.env`.

- **`.env.example`**: Archivo de ejemplo completamente funcional para levantar el proyecto de forma inmediata tanto en local nativo como con Docker Compose.
- **`.env`**: Archivo de variables reales para tu entorno local o de producción. Para crearlo a partir del ejemplo:
  ```bash
  make env
  # o manualmente:
  cp .env.example .env
  ```
- **Producción:** En producción, cada administrador o desarrollador debe crear su propio `.env` con contraseñas seguras, certificados SSL y URLs reales de los demás microservicios.

---

## 🚀 Guía de Uso Rápido con `Makefile`

Para ver todos los comandos disponibles en consola:
```bash
make help
```

### 1. Ejecutar en Local con Docker Compose (Desarrollo)

Levanta PostgreSQL, Backend (Go) y Frontend (Vue) con recarga en vivo (*hot-reloading*):

```bash
# En primer plano (muestra todos los logs combinados):
make dev

# O en segundo plano (detached):
make dev-d
```

- **Frontend:** [http://localhost:3000](http://localhost:3000)
- **Backend API:** [http://localhost:8080](http://localhost:8080)
- **API Healthcheck:** [http://localhost:8080/api/health](http://localhost:8080/api/health)
- **PostgreSQL:** `localhost:5432`

Para ver los logs en segundo plano o detener los servicios:
```bash
make logs   # Ver logs en vivo
make down   # Detener contenedores
```

---

### 2. Ejecutar como Producción con Docker Compose

Utiliza las imágenes multi-stage optimizadas (binario estático de Go y Frontend servido en Nginx Alpine):

```bash
# Requiere que exista un archivo .env configurado
make prod
```

Para detener o ver logs del entorno de producción:
```bash
make logs-prod
make down-prod
```

---

### 3. Ejecución Local Nativa (En el Host)

Si deseas ejecutar los componentes directamente en tu máquina:

```bash
# 1. Levantar solo la base de datos PostgreSQL en Docker
make local-db

# 2. En una terminal, ejecutar el Backend Go (conectándose a localhost:5432)
make local-back

# 3. En otra terminal, ejecutar el Frontend Vue
make local-front
```

---

### 4. Pruebas y Limpieza

```bash
# Ejecutar pruebas unitarias de Go
make test

# Limpiar contenedores, volúmenes de desarrollo y redes
make clean
```

---

## 📡 Endpoints Base Disponibles

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/health` | Diagnóstico general del servicio y estado de conexión a la BD |
| `GET` | `/api/health` | Diagnóstico de salud con detalles de LPS y dependencias |
| `GET` | `/api/info` | Endpoint base placeholder para futuras funcionalidades de ingesta y orden |