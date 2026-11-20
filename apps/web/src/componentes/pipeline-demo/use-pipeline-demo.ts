import { useState } from 'react'
import exampleBatch from '../../mocks/lote-ocr.json'
import type { BatchProcessSummary, NormalizedRecord, RejectedRecord } from './pipeline-demo.api'

const exampleText = JSON.stringify(exampleBatch, null, 2)
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

function textValue(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

export function usePipelineDemo() {
  const [input, setInput] = useState<string>(exampleText)
  const [result, setResult] = useState<BatchProcessSummary | null>(null)
  const [error, setError] = useState<string>('')

  function resetResult() {
    setResult(null)
    setError('')
  }

  function handleInputChange(nextValue: string) {
    setInput(nextValue)
    resetResult()
  }

  function loadExample() {
    setInput(exampleText)
    resetResult()
  }

  function runBatch() {
    resetResult()

    let records: unknown
    try {
      records = JSON.parse(input)
    } catch {
      setError('Revisa la sintaxis del JSON e inténtalo de nuevo.')
      return
    }
    if (!Array.isArray(records) || records.length === 0) {
      setError('Ingresa un arreglo JSON con al menos un registro.')
      return
    }

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
      if (!emailPattern.test(correo)) issues.push('Correo inválido')
      if (seen.has(folio)) issues.push('Folio duplicado')

      if (issues.length > 0) {
        rejected.push({ position, folio, reason: issues.join(' · ') })
        continue
      }

      seen.add(folio)
      accepted.push({ folio, nombre, correo })
    }

    setResult({ total: records.length, accepted, rejected })
  }

  return {
    input,
    result,
    error,
    handleInputChange,
    loadExample,
    runBatch,
  }
}

