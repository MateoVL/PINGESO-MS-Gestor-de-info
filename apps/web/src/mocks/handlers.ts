import { http, HttpResponse } from 'msw'
import type { components } from '../types/openapi'

type HealthResponse = components['schemas']['HealthResponse']
type LoteDetalle = components['schemas']['LoteDetalle']

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
      capabilities: [
        'Carga de información',
        'Limpieza y validación de datos',
        'Inserción masiva de datos',
      ],
    },
  }
}

export const handlers = [
  http.get('*/api/health', () => {
    return HttpResponse.json(createMockHealthResponse(true), { status: 200 })
  }),
  http.post('*/api/v1/lotes', async () => {
    const detail: LoteDetalle = {
      id: '550e8400-e29b-41d4-a716-446655440000',
      origen: 'Extractor OCR',
      referencia_externa: 'lote-msw-001',
      recibidos: 5,
      aceptados: 2,
      rechazados: 3,
      creado_en: '2026-11-20T12:00:00Z',
      rechazos: [],
    }
    return HttpResponse.json(detail, { status: 201 })
  }),
]
