import type { FormEvent } from 'react'
import { usePipelineDemo } from './use-pipeline-demo'

export interface PipelineDemoProps {
  apiBaseUrl: string
  usuario?: string
  permisos?: string[]
}

// Microcomponente autocontenido: vista pura sin lógica de negocio ni dependencia de otros microcomponentes
export function PipelineDemo({ apiBaseUrl, usuario, permisos }: PipelineDemoProps) {
  const { input, result, error, handleInputChange, loadExample, runBatch } =
    usePipelineDemo(apiBaseUrl)
  const canOperate =
    !permisos ||
    permisos.length === 0 ||
    permisos.includes('lotes:operar') ||
    permisos.includes('lotes:administrar') ||
    permisos.includes('lotes:crear')

  function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!canOperate) return
    runBatch()
  }

  return (
    <section
      className="pipeline-demo"
      id="prueba-pipeline"
      aria-labelledby="titulo-prueba"
      data-api-base-url={apiBaseUrl}
      data-usuario={usuario ?? 'anonimo'}
    >
      <header>
        <span className="demo-label">PT2 · Datos simulados</span>
        <h2 id="titulo-prueba">Prueba el pipeline de información</h2>
        <p>
          Procesa un lote simulado del extractor OCR y revisa qué registros quedarían listos para
          importar.
        </p>
      </header>

      <form className="batch-form" onSubmit={onSubmit}>
        <label htmlFor="lote-ocr">Lote de entrada en JSON</label>
        <p id="formato-lote">
          Cada registro usa los campos <code>folio</code>, <code>nombre</code> y <code>correo</code>
          .
        </p>
        <textarea
          id="lote-ocr"
          value={input}
          rows={10}
          spellCheck={false}
          aria-describedby="formato-lote"
          aria-invalid={Boolean(error)}
          onChange={(e) => handleInputChange(e.target.value)}
        />
        <div className="demo-actions">
          <button className="run-button" type="submit">
            Procesar lote
          </button>
          <button className="reset-button" type="button" onClick={loadExample}>
            Restaurar ejemplo
          </button>
        </div>
        {error ? (
          <p className="batch-error" role="alert">
            {error}
          </p>
        ) : null}
      </form>

      {result ? (
        <p className="batch-summary" role="status">
          {result.total} recibidos · {result.accepted.length} listos para importar ·{' '}
          {result.rejected.length} rechazados
        </p>
      ) : null}

      {result && result.accepted.length > 0 ? (
        <div className="table-scroll">
          <table>
            <caption>Registros normalizados</caption>
            <thead>
              <tr>
                <th scope="col">Folio</th>
                <th scope="col">Nombre</th>
                <th scope="col">Correo</th>
              </tr>
            </thead>
            <tbody>
              {result.accepted.map((record) => (
                <tr key={record.folio}>
                  <td>{record.folio}</td>
                  <td>{record.nombre}</td>
                  <td>{record.correo}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {result && result.rejected.length > 0 ? (
        <div className="table-scroll">
          <table>
            <caption>Registros rechazados</caption>
            <thead>
              <tr>
                <th scope="col">Posición</th>
                <th scope="col">Folio</th>
                <th scope="col">Motivo</th>
              </tr>
            </thead>
            <tbody>
              {result.rejected.map((record) => (
                <tr key={record.position}>
                  <td>{record.position}</td>
                  <td>{record.folio || 'Sin folio'}</td>
                  <td>{record.reason}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  )
}
