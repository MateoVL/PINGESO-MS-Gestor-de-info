import type { components } from '../../types/openapi'

export type HealthResponse = components['schemas']['HealthResponse']

export interface HealthCheckResult {
  data: HealthResponse
  latencyMs: number
}

// Único lugar con fetch en el microcomponente estado-servicio
export async function fetchServiceHealth(apiBaseUrl: string): Promise<HealthCheckResult> {
  const baseUrl = apiBaseUrl.replace(/\/$/, '')
  const start = performance.now()
  const res = await fetch(`${baseUrl}/api/health`, {
    headers: { Accept: 'application/json' },
    signal: AbortSignal.timeout(5000),
  })

  if (res.status !== 200 && res.status !== 503) {
    throw new Error(`La API respondió con HTTP ${res.status}.`)
  }

  const data = (await res.json()) as HealthResponse
  const connected = res.status === 200
  if (
    data?.status !== (connected ? 'healthy' : 'degraded') ||
    data.database?.connected !== connected
  ) {
    throw new Error('La API devolvió un diagnóstico inválido.')
  }

  return {
    data,
    latencyMs: Math.round(performance.now() - start),
  }
}
