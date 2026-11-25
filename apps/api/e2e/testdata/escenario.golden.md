# Flujo de carga y consulta de lotes

## Estado del servicio

GET /healthz → 200
GET /readyz → 200

## Autorización

Sin token → 401 (application/problem+json)
Token sin lotes:operar → 403 (application/problem+json)

## Lote con registros válidos e inválidos

POST /api/v1/lotes → 201
Recibidos: 5, aceptados: 2, rechazados: 3
- Rechazo posición 3 (folio "DOC-003"): falta_nombre
- Rechazo posición 4 (folio ""): falta_folio, correo_invalido
- Rechazo posición 5 (folio "DOC-001"): folio_duplicado
Registros guardados:
- DOC-001 | Camila Soto | camila@example.com
- DOC-002 | Diego Pérez | diego@example.com

## Reintentos

Misma clave y contenido → 200, mismo lote: true
Misma clave con otro contenido → 409 (application/problem+json)

## Cinco lotes simultáneos con el mismo folio

POST → 201, aceptados: 1, rechazados: 1
POST → 201, aceptados: 1, rechazados: 1
POST → 201, aceptados: 1, rechazados: 1
POST → 201, aceptados: 1, rechazados: 1
POST → 201, aceptados: 2, rechazados: 0
Registros con folio DOC-900: 1
Motivo de los rechazos: folio_ya_registrado

## Auditoría

- crear_lote por operador-ocr: 6
Eventos sin request_id: 0

## Errores

Lote sin registros → 400 (application/problem+json)
CSV sin columna correo → 400 (application/problem+json)
Content-Type no soportado → 415 (application/problem+json)
Lote inexistente → 404 (application/problem+json)
Id que no es UUID → 400 (application/problem+json)
