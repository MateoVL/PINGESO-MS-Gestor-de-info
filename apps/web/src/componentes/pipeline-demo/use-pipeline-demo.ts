import { useState } from 'react'
import exampleBatch from '../../mocks/lote-ocr.json'
import { evaluateBatchRecords, type BatchProcessSummary } from './pipeline-demo.api'

const exampleText = JSON.stringify(exampleBatch, null, 2)

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

    setResult(evaluateBatchRecords(records))
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
