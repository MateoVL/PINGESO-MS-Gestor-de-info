// Generated from contracts/openapi.yaml (OpenAPI 3.1.0)

export interface DatabaseStatus {
  connected: boolean
  engine: string
  message: string
}

export interface LineOfWorkInfo {
  depends_on: string[]
  consumers: string[]
  capabilities: string[]
}

export interface HealthResponse {
  status: 'healthy' | 'degraded'
  service_name: string
  environment: string
  uptime: string
  timestamp: string
  database: DatabaseStatus
  role: string
  line_of_work: LineOfWorkInfo
}

export interface RegistroEntrada {
  folio: string
  nombre: string
  correo: string
}

export interface SolicitudLote {
  origen: string
  referencia_externa: string
  registros: RegistroEntrada[]
}

export interface Rechazo {
  posicion: number
  folio: string
  motivos: string[]
}

export interface LoteDetalle {
  id: string
  origen: string
  referencia_externa: string
  recibidos: number
  aceptados: number
  rechazados: number
  creado_en: string
  rechazos: Rechazo[]
}

export interface paths {
  '/api/health': {
    get: {
      responses: {
        200: { content: { 'application/json': HealthResponse } }
        503: { content: { 'application/json': HealthResponse } }
      }
    }
  }
  '/api/v1/lotes': {
    post: {
      requestBody: {
        content: {
          'application/json': SolicitudLote
        }
      }
      responses: {
        200: { content: { 'application/json': LoteDetalle } }
        201: { content: { 'application/json': LoteDetalle } }
      }
    }
  }
}

