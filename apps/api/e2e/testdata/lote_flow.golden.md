# Flujo E2E Golden: Ingesta y Consulta de Lote

## Escenario

1. `GET /api/health` responde `200 OK` con `status: "healthy"` y `database.engine: "PostgreSQL"`.
2. `POST /api/v1/lotes` con `Idempotency-Key: e2e-golden-1` y 3 registros (2 válidos y 1 con correo inválido) responde `201 Created` con `recibidos: 3`, `aceptados: 2`, `rechazados: 1`.
3. `GET /api/v1/registros` devuelve los 2 registros normalizados (`DOC-E2E-1` y `DOC-E2E-2`).

