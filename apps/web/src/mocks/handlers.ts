import type { HealthResponse } from '../types/openapi'

export function createMockHealthResponse(connected = true): HealthResponse {
  return {
    status: connected ? 'healthy' : 'degraded',
    service_name: 'MS Gestor de Información',
    environment: 'development',
    uptime: '1m30s',
    timestamp: '2026-11-20T12:00:00Z',
    database: {
      engine: 'PostgreSQL',
      connected,
      message: connected ? 'Base de datos conectada y operativa' : 'PostgreSQL no está disponible',
    },
    role: 'Gestor de Información',
    line_of_work: {
      depends_on: ['Extractor de Datos OCR'],
      consumers: ['Motor de Reportes'],
      capabilities: ['Carga de información', 'Limpieza y validación de datos', 'Inserción masiva de datos'],
    },
  }
}

