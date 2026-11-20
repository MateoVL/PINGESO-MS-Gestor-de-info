import type { RegistroEntrada, SolicitudLote, LoteDetalle } from '../../types/openapi'

export interface NormalizedRecord {
  folio: string
  nombre: string
  correo: string
}

export interface RejectedRecord {
  position: number
  folio?: string
  reason: string
}

export interface BatchProcessSummary {
  total: number
  accepted: NormalizedRecord[]
  rejected: RejectedRecord[]
}

// Único lugar con fetch en el microcomponente pipeline-demo
export async function submitBatchToApi(
  apiBaseUrl: string,
  payload: SolicitudLote,
  idempotencyKey?: string,
  authToken?: string
): Promise<LoteDetalle> {
  const baseUrl = apiBaseUrl.replace(/\/$/, '')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  }
  if (idempotencyKey) {
    headers['Idempotency-Key'] = idempotencyKey
  }
  if (authToken) {
    headers['Authorization'] = authToken
  }

  const res = await fetch(`${baseUrl}/api/v1/lotes`, {
    method: 'POST',
    headers,
    body: JSON.stringify(payload),
    signal: AbortSignal.timeout(5000),
  })

  if (!res.ok) {
    throw new Error(`Error HTTP ${res.status} al enviar el lote`)
  }
  return (await res.json()) as LoteDetalle
}

export type { RegistroEntrada }

