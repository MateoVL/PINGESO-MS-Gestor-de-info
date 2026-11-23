import type { components } from '../../types/openapi'

export type RegistroEntrada = components['schemas']['RegistroEntrada']
export type SolicitudLote = components['schemas']['SolicitudLote']
export type LoteDetalle = components['schemas']['LoteDetalle']

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

function isValidEmail(email: string): boolean {
  if (!email || /\s/.test(email)) {
    return false
  }
  const atIndex = email.indexOf('@')
  if (atIndex <= 0 || atIndex !== email.lastIndexOf('@')) {
    return false
  }
  const domain = email.slice(atIndex + 1)
  const dotIndex = domain.indexOf('.')
  return dotIndex > 0 && !domain.startsWith('.') && !domain.endsWith('.')
}

function textValue(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

export function evaluateBatchRecords(records: unknown[]): BatchProcessSummary {
  const accepted: NormalizedRecord[] = []
  const rejected: RejectedRecord[] = []
  const seen = new Set<string>()

  for (const [index, raw] of records.entries()) {
    const position = index + 1
    if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
      rejected.push({ position, reason: 'Se esperaba un objeto JSON.' })
      continue
    }

    const obj = raw as Record<string, unknown>
    const folio = textValue(obj.folio).toUpperCase()
    const nombre = textValue(obj.nombre).replace(/\s+/g, ' ')
    const correo = textValue(obj.correo).toLowerCase()
    const issues: string[] = []

    if (!folio) issues.push('Falta el folio')
    if (!nombre) issues.push('Falta el nombre')
    if (!isValidEmail(correo)) issues.push('Correo inválido')
    if (seen.has(folio)) issues.push('Folio duplicado')

    if (issues.length > 0) {
      rejected.push({ position, folio, reason: issues.join(' · ') })
      continue
    }

    seen.add(folio)
    accepted.push({ folio, nombre, correo })
  }

  return { total: records.length, accepted, rejected }
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
    credentials: 'include',
    body: JSON.stringify(payload),
    signal: AbortSignal.timeout(5000),
  })

  if (!res.ok) {
    throw new Error(`Error HTTP ${res.status} al enviar el lote`)
  }
  return (await res.json()) as LoteDetalle
}
