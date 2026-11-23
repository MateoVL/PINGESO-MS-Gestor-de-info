SHELL := /bin/bash
ENV_FILE ?= .env

.PHONY: help env dev dev-d prod down down-prod logs logs-prod local-db local-back local-front sqlc generate migrate-up migrate-down oapi-codegen codegen clean test

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
env: ## Crea el archivo .env a partir de .env.example con valores funcionales para desarrollo si no existe
	@if [ ! -f $(ENV_FILE) ]; then \
		echo "--> Creando $(ENV_FILE) con valores funcionales para desarrollo local..."; \
		sed \
			-e 's|API_PORT=replace-with-.*|API_PORT=8080|' \
			-e 's|API_ENV=replace-with-.*|API_ENV=dev|' \
			-e 's|DATABASE_URL=replace-with-.*|DATABASE_URL=postgres://gestor_user:dev-gestor-password@postgres:5432/gestor_info_db?sslmode=disable|' \
			-e 's|AUTH_JWKS_URL=replace-with-.*|AUTH_JWKS_URL=http://localhost:5101/api/v1/auth/jwks|' \
			-e 's|FRONTEND_URL=replace-with-.*|FRONTEND_URL=http://localhost:5120|' \
			-e 's|TZ=replace-with-.*|TZ=America/Santiago|' \
			-e 's|DIGITALIZACION_URL=replace-with-.*|DIGITALIZACION_URL=http://localhost:5151|' \
			-e 's|REPORTES_URL=replace-with-.*|REPORTES_URL=http://localhost:5111|' \
			-e 's|IDENTIDAD_URL=replace-with-.*|IDENTIDAD_URL=http://localhost:5101|' \
			-e 's|APP_NAME=replace-with-.*|APP_NAME="MS Gestor de Información"|' \
			-e 's|APP_ENV=replace-with-.*|APP_ENV=development|' \
			-e 's|LOG_LEVEL=replace-with-.*|LOG_LEVEL=info|' \
			-e 's|CORS_ALLOWED_ORIGINS=replace-with-.*|CORS_ALLOWED_ORIGINS=http://localhost:5120,http://127.0.0.1:5120,http://localhost:3000,http://127.0.0.1:3000|' \
			-e 's|POSTGRES_DB=replace-with-.*|POSTGRES_DB=gestor_info_db|' \
			-e 's|POSTGRES_USER=replace-with-.*|POSTGRES_USER=gestor_user|' \
			-e 's|POSTGRES_PASSWORD=replace-with-.*|POSTGRES_PASSWORD=dev-gestor-password|' \
			-e 's|POSTGRES_PORT=replace-with-.*|POSTGRES_PORT=5432|' \
			-e 's|FRONTEND_PORT=replace-with-.*|FRONTEND_PORT=5120|' \
			-e 's|BACKEND_PORT=replace-with-.*|BACKEND_PORT=5121|' \
			-e 's|BACKEND_URL=replace-with-.*|BACKEND_URL=http://backend:8080|' \
			-e 's|SERVICE_ROLE=replace-with-.*|SERVICE_ROLE="Gestor de Información"|' \
			-e 's|SERVICE_DEPENDS_ON=replace-with-.*|SERVICE_DEPENDS_ON="Extractor de Datos OCR"|' \
			-e 's|SERVICE_CONSUMERS=replace-with-.*|SERVICE_CONSUMERS="Motor de Reportes"|' \
			-e 's|SERVICE_CAPABILITIES=replace-with-.*|SERVICE_CAPABILITIES="Carga de información,Limpieza y validación de datos,Inserción masiva de datos"|' \
			.env.example > $(ENV_FILE); \
		echo "--> Archivo $(ENV_FILE) generado con valores por defecto funcionales."; \
	elif grep -q "replace-with-" $(ENV_FILE); then \
		echo "--> Detectados placeholders 'replace-with-' en $(ENV_FILE). Regenerando con valores funcionales para desarrollo..."; \
		sed \
			-e 's|API_PORT=replace-with-.*|API_PORT=8080|' \
			-e 's|API_ENV=replace-with-.*|API_ENV=dev|' \
			-e 's|DATABASE_URL=replace-with-.*|DATABASE_URL=postgres://gestor_user:dev-gestor-password@postgres:5432/gestor_info_db?sslmode=disable|' \
			-e 's|AUTH_JWKS_URL=replace-with-.*|AUTH_JWKS_URL=http://localhost:5101/api/v1/auth/jwks|' \
			-e 's|FRONTEND_URL=replace-with-.*|FRONTEND_URL=http://localhost:5120|' \
			-e 's|TZ=replace-with-.*|TZ=America/Santiago|' \
			-e 's|DIGITALIZACION_URL=replace-with-.*|DIGITALIZACION_URL=http://localhost:5151|' \
			-e 's|REPORTES_URL=replace-with-.*|REPORTES_URL=http://localhost:5111|' \
			-e 's|IDENTIDAD_URL=replace-with-.*|IDENTIDAD_URL=http://localhost:5101|' \
			-e 's|APP_NAME=replace-with-.*|APP_NAME="MS Gestor de Información"|' \
			-e 's|APP_ENV=replace-with-.*|APP_ENV=development|' \
			-e 's|LOG_LEVEL=replace-with-.*|LOG_LEVEL=info|' \
			-e 's|CORS_ALLOWED_ORIGINS=replace-with-.*|CORS_ALLOWED_ORIGINS=http://localhost:5120,http://127.0.0.1:5120,http://localhost:3000,http://127.0.0.1:3000|' \
			-e 's|POSTGRES_DB=replace-with-.*|POSTGRES_DB=gestor_info_db|' \
			-e 's|POSTGRES_USER=replace-with-.*|POSTGRES_USER=gestor_user|' \
			-e 's|POSTGRES_PASSWORD=replace-with-.*|POSTGRES_PASSWORD=dev-gestor-password|' \
			-e 's|POSTGRES_PORT=replace-with-.*|POSTGRES_PORT=5432|' \
			-e 's|FRONTEND_PORT=replace-with-.*|FRONTEND_PORT=5120|' \
			-e 's|BACKEND_PORT=replace-with-.*|BACKEND_PORT=5121|' \
			-e 's|BACKEND_URL=replace-with-.*|BACKEND_URL=http://backend:8080|' \
			-e 's|SERVICE_ROLE=replace-with-.*|SERVICE_ROLE="Gestor de Información"|' \
			-e 's|SERVICE_DEPENDS_ON=replace-with-.*|SERVICE_DEPENDS_ON="Extractor de Datos OCR"|' \
			-e 's|SERVICE_CONSUMERS=replace-with-.*|SERVICE_CONSUMERS="Motor de Reportes"|' \
			-e 's|SERVICE_CAPABILITIES=replace-with-.*|SERVICE_CAPABILITIES="Carga de información,Limpieza y validación de datos,Inserción masiva de datos"|' \
			.env.example > $(ENV_FILE); \
		echo "--> Archivo $(ENV_FILE) actualizado con valores por defecto funcionales."; \
	else \
		echo "--> El archivo $(ENV_FILE) ya existe. No se sobreescribió."; \
	fi

# ==============================================================================
# Generación de código y Migraciones (versiones exactas fijadas según AGENTS.md)
# ==============================================================================
sqlc: ## Genera el código de consultas SQL con sqlc v1.29.0
	cd apps/api && go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0 generate

generate: sqlc ## Regenera internal/repository/ con sqlc v1.29.0 (AGENTS.md sección 5)

migrate-up: env ## Ejecuta las migraciones de PostgreSQL con goose v3.24.1
	cd apps/api && go run github.com/pressly/goose/v3/cmd/goose@v3.24.1 -dir db/migrations postgres "$${POSTGRES_URI:-postgres://127.0.0.1:5432/gestor_info_db?sslmode=disable}" up

migrate-down: env ## Revierte la última migración con goose v3.24.1
	cd apps/api && go run github.com/pressly/goose/v3/cmd/goose@v3.24.1 -dir db/migrations postgres "$${POSTGRES_URI:-postgres://127.0.0.1:5432/gestor_info_db?sslmode=disable}" down

oapi-codegen: ## Genera clientes Go de servicios externos desde contracts/externos/*.yaml con oapi-codegen v2.8.2
	@for spec in contracts/externos/*.yaml; do \
		[ -e "$$spec" ] || continue; \
		name=$$(basename "$$spec" .yaml); \
		cd apps/api && go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.2 -package "$$name" -generate types,client "../../$$spec" > "internal/service/$${name}_client.gen.go"; \
	done

codegen: ## Genera contracts/openapi.yaml (OpenAPI 3.1) y tipos TypeScript con pnpm codegen
	pnpm codegen

# ==============================================================================
# Modo 1: Local con Docker Compose (Desarrollo)
# ==============================================================================
dev: env ## Ejecuta el entorno de desarrollo local con Docker Compose
	@echo "--> Levantando entorno de desarrollo con Docker Compose..."
	docker compose --env-file $(ENV_FILE) -f docker-compose.yaml up --build

dev-d: env ## Ejecuta el entorno de desarrollo en segundo plano (detached)
	@echo "--> Levantando entorno de desarrollo en segundo plano..."
	docker compose --env-file $(ENV_FILE) -f docker-compose.yaml up -d --build

down: ## Detiene los contenedores del entorno de desarrollo local
	@echo "--> Deteniendo contenedores de desarrollo..."
	docker compose -f docker-compose.yaml down

logs: ## Sigue los logs en tiempo real del entorno de desarrollo
	docker compose -f docker-compose.yaml logs -f

# ==============================================================================
# Modo 2: Producción con Docker Compose (Multi-stage + Distroless)
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
	POSTGRES_USER=$${POSTGRES_USER:-down} POSTGRES_PASSWORD=$${POSTGRES_PASSWORD:-down} docker compose $$( [ -f $(ENV_FILE) ] && echo "--env-file $(ENV_FILE)" ) -f docker-compose.prod.yml down

logs-prod: ## Sigue los logs en tiempo real de producción
	docker compose --env-file $(ENV_FILE) -f docker-compose.prod.yml logs -f

# ==============================================================================
# Modo 3: Ejecución Local Nativa (Host)
# ==============================================================================
local-db: env ## Levanta únicamente PostgreSQL en Docker para desarrollo nativo
	docker compose --env-file $(ENV_FILE) -f docker-compose.yaml up -d --wait postgres

local-back: env ## Ejecuta el backend Go localmente en el host
	cd apps/api && go run ./cmd/api

local-front: env ## Ejecuta el frontend React localmente en el host con pnpm
	pnpm install --frozen-lockfile && pnpm dev

# ==============================================================================
# Mantenimiento y Pruebas
# ==============================================================================
test: ## Ejecuta pruebas de Go y React y genera sus reportes de cobertura
	cd apps/api && go test -coverpkg=./... -coverprofile=coverage.out ./...
	pnpm install --frozen-lockfile && pnpm test:coverage

clean: ## Limpia contenedores, redes, volúmenes e imágenes locales del proyecto
	@echo "--> Limpiando contenedores, redes, volúmenes e imágenes locales..."
	docker compose --profile dev -f docker-compose.yaml down -v --rmi local --remove-orphans
	POSTGRES_USER=$${POSTGRES_USER:-clean} POSTGRES_PASSWORD=$${POSTGRES_PASSWORD:-clean} docker compose -f docker-compose.prod.yml down -v --rmi local --remove-orphans
