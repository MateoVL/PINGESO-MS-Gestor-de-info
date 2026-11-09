SHELL := /bin/bash
ENV_FILE ?= .env

.PHONY: help env dev dev-d prod down down-prod logs logs-prod local-db local-back local-front clean test

# ==============================================================================
# Ayuda y Documentación
# ==============================================================================
help: ## Muestra este menú de ayuda interactivo
	@echo "======================================================================="
	@echo "          MS GESTOR DE INFORMACION - COMANDOS MAKEFILE                 "
	@echo "======================================================================="
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ==============================================================================
# Gestión de Variables de Entorno (.env)
# ==============================================================================
env: ## Crea el archivo .env a partir de .env.example si no existe
	@if [ ! -f $(ENV_FILE) ]; then \
		echo "--> Creando $(ENV_FILE) a partir de .env.example..."; \
		cp .env.example $(ENV_FILE); \
		echo "--> Archivo $(ENV_FILE) generado con valores por defecto funcionales."; \
	else \
		echo "--> El archivo $(ENV_FILE) ya existe. No se sobreescribió."; \
	fi

# ==============================================================================
# Modo 1: Local con Docker Compose (Desarrollo con Hot Reload)
# ==============================================================================
dev: env ## Ejecuta el entorno de desarrollo local con Docker Compose
	@echo "--> Levantando entorno de desarrollo con Docker Compose..."
	docker compose --env-file $(ENV_FILE) -f docker-compose.yml up --build

dev-d: env ## Ejecuta el entorno de desarrollo en segundo plano (detached)
	@echo "--> Levantando entorno de desarrollo en segundo plano..."
	docker compose --env-file $(ENV_FILE) -f docker-compose.yml up -d --build

down: ## Detiene los contenedores del entorno de desarrollo local
	@echo "--> Deteniendo contenedores de desarrollo..."
	docker compose -f docker-compose.yml down

logs: ## Sigue los logs en tiempo real del entorno de desarrollo
	docker compose -f docker-compose.yml logs -f

# ==============================================================================
# Modo 2: Producción con Docker Compose (Multi-stage + Nginx)
# ==============================================================================
check-prod-env:
	@if [ ! -f $(ENV_FILE) ]; then \
		echo -e "\033[31m[ERROR] Para producción se debe contar con un archivo .env configurado con credenciales seguras.\033[0m"; \
		echo -e "Por favor crea tu archivo .env basándote en .env.example antes de desplegar en producción."; \
		exit 1; \
	fi

prod: check-prod-env ## Construye y ejecuta el entorno de producción con Docker Compose
	@echo "--> Levantando entorno de producción con Docker Compose..."
	docker compose --env-file $(ENV_FILE) -f docker-compose.prod.yml up -d --build

down-prod: ## Detiene los contenedores de producción
	@echo "--> Deteniendo contenedores de producción..."
	docker compose --env-file $(ENV_FILE) -f docker-compose.prod.yml down

logs-prod: ## Sigue los logs en tiempo real de producción
	docker compose --env-file $(ENV_FILE) -f docker-compose.prod.yml logs -f

# ==============================================================================
# Modo 3: Ejecución Local Nativa (Host)
# ==============================================================================
local-db: env ## Levanta únicamente MongoDB en Docker para desarrollo nativo
	docker compose --env-file $(ENV_FILE) -f docker-compose.yml up -d --wait mongodb

local-back: env ## Ejecuta el backend Go localmente en el host
	cd backend && go run ./cmd/api

local-front: env ## Ejecuta el frontend Vue localmente en el host
	cd frontend && npm ci && npm run dev

# ==============================================================================
# Mantenimiento y Pruebas
# ==============================================================================
test: ## Ejecuta pruebas de Go y Vue y genera sus reportes de cobertura
	cd backend && go test -coverpkg=./... -coverprofile=coverage.out ./...
	cd frontend && npm ci && npm run test:coverage

clean: ## Limpia contenedores, redes y volúmenes temporales de Docker
	@echo "--> Limpiando contenedores huérfanos y volúmenes de desarrollo..."
	docker compose -f docker-compose.yml down -v --remove-orphans
