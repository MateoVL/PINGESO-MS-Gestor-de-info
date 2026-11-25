# Modos de Falla · MS Gestor de Información (PT2)

Este documento describe los modos de falla previstos para el microservicio **Gestor de Información (PT2)**, su impacto en la Línea de Productos de Software (LPS) y los mecanismos de mitigación implementados según las reglas de arquitectura de `AGENTS.md`.

## 1. Indisponibilidad o caída de PostgreSQL

- **Síntoma**: Timeouts al consultar o insertar lotes/registros; el ping de diagnóstico supera los 2 segundos.
- **Detección**: `GET /readyz`, `GET /health` y `GET /api/health` responden `HTTP 503 Service Unavailable` con `status: "degraded"` y `database.connected: false` sin exponer trazas ni credenciales internas, mientras que `GET /healthz` mantiene `HTTP 200 OK` indicando que el proceso sigue vivo.
- **Mitigación**:
  - Pool de conexiones `pgxpool` con reconexión automática.
  - Si un lote falla durante su persistencia, la operación aborta de forma atómica y el emisor (`PT5 Extractor OCR`) puede reintentar de manera segura utilizando la misma cabecera `Idempotency-Key`.

## 2. Reintentos de red y lotes duplicados (Idempotencia)

- **Síntoma**: El servicio upstream (`PT5 Extractor OCR`) reenvía un mismo lote por corte de red o timeout HTTP (> 5 s).
- **Detección**: Cabecera `Idempotency-Key` presente en `POST /api/v1/lotes`.
- **Mitigación**:
  - Se calcula la huella SHA-256 (`HuellaContenido`) del payload (`origen`, `referencia_externa` y `registros`).
  - Si la clave ya existe con la **misma huella**, se devuelve `HTTP 200 OK` con el resultado original sin duplicar registros.
  - Si la clave ya existe con **distinta huella**, se rechaza con `HTTP 409 Conflict` (`code: "conflict"`).

## 3. Folios duplicados intra-lote o inter-lote (Concurrencia)

- **Síntoma**: Un lote incluye el mismo folio más de una vez, o dos lotes concurrentes intentan registrar el mismo folio simultáneamente.
- **Detección**: Validación en capa `service` (intra-lote) y restricción `UNIQUE` en `store` / PostgreSQL (inter-lote).
- **Mitigación**:
  - Los registros con folio repetido dentro del mismo lote se separan automáticamente como rechazados con motivo `folio_duplicado`.
  - Si el `store` detecta colisión con un folio ya persistido (`ErrFoliosDuplicados`), el servicio reclasifica únicamente los folios en conflicto al listado de rechazos con motivo `folio_ya_registrado` y guarda el resto de los registros válidos del lote (hasta 5 reintentos ante contención concurrente).
- **Verificación**: el e2e `TestE2EBinarioPostgres` (`apps/api/e2e/binario_test.go`) envía cinco lotes simultáneos con el folio `DOC-900` al binario real contra PostgreSQL real, y el reporte (`e2e/testdata/escenario.golden.md`) exige que quede guardado una sola vez. Al quitar a propósito la restricción `UNIQUE` y la verificación previa del `store`, el e2e falla: los cinco lotes guardan `DOC-900`.

## 4. Datos sucios o malformados desde OCR

- **Síntoma**: Registros con espacios sobrantes, mayúsculas/minúsculas inconsistentes, folios vacíos, nombres vacíos o correos inválidos; archivos CSV sin columnas requeridas o payloads > 5 MB.
- **Detección**: Validación estructural en `internal/api` (Huma / lector CSV con límite `MaxBytesReader` de 5 MB) y normalización en `internal/service`.
- **Mitigación**:
  - Payloads > 5 MB se cortan con `HTTP 413 Payload Too Large`.
  - `Content-Type` distinto de `application/json` o `text/csv` responde `HTTP 415 Unsupported Media Type`.
  - Registros individuales defectuosos no abortan el lote completo: se registran en `rechazos` indicando su `posicion` (1-indexed), `folio` normalizado y lista de `motivos` (`falta_folio`, `falta_nombre`, `correo_invalido`).

## 5. Caída o lentitud de servicios externos (`<SERVICIO>_URL`)

- **Síntoma**: Un servicio externo no responde dentro del SLA.
- **Mitigación**:
  - Todo cliente HTTP hacia servicios externos (por ejemplo `DIGITALIZACION_URL`) fija un timeout explícito de **5 segundos** por defecto, propaga el JWT y el encabezado `X-Request-ID`, y ante falla devuelve un error controlado sin colgar la request entrante.

## 6. Operación sin registro de quién la hizo (Auditoría)

- **Síntoma**: Se crea un lote y no queda constancia de quién lo envió ni cuándo.
- **Mitigación**:
  - Cada lote nuevo registra un evento en la tabla `auditoria` (migración `00002_auditoria.sql`) con `actor` (el `sub` del JWT), `accion` (`crear_lote`), `entidad`, `entidad_id`, `request_id`, `ocurrido_en` y un `detalle` con los totales del lote.
  - El evento se guarda en la misma transacción que el lote: si una de las dos escrituras falla, no se guarda ninguna.
  - Un reintento idempotente no cambia datos, así que no genera un evento nuevo.
- **Verificación**: `internal/service/auditoria_test.go` y el e2e, que consulta la tabla `auditoria` directamente en PostgreSQL.
