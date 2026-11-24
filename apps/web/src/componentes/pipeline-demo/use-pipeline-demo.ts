import { useState } from 'react'
import exampleBatch from '../../mocks/lote-ocr.json'
import {
  evaluateBatchRecords,
  submitBatchToApi,
  type BatchProcessSummary,
  type RegistroEntrada,
} from './pipeline-demo.api'

const exampleText = JSON.stringify(exampleBatch, null, 2)

export function usePipelineDemo(apiBaseUrl = '') {
  const [input, setInput] = useState<string>(exampleText)
  const [result, setResult] = useState<BatchProcessSummary | null>(null)
  const [error, setError] = useState<string>('')
  const [syncedBatchId, setSyncedBatchId] = useState<string | null>(null)

  function resetResult() {
    setResult(null)
    setError('')
    setSyncedBatchId(null)
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

    const summary = evaluateBatchRecords(records)
    setResult(summary)

    const entries: RegistroEntrada[] = summary.accepted.map((item) => ({
      folio: item.folio,
      nombre: item.nombre,
      correo: item.correo,
    }))
    if (entries.length > 0) {
      void submitBatchToApi(apiBaseUrl, {
        origen: 'Extractor OCR',
        referencia_externa: 'ui-demo',
        registros: entries,
      })
        .then((detail) => {
          setSyncedBatchId(detail.id)
        })
        .catch(() => {
          setSyncedBatchId(null)
        })
    }
  }

  return {
    input,
    result,
    error,
    syncedBatchId,
    handleInputChange,
    loadExample,
    runBatch,
  }
}
