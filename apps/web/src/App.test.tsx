// @vitest-environment jsdom
import { createElement, act } from 'react'
import { createRoot } from 'react-dom/client'
import { afterEach, expect, it, onTestFinished, vi } from 'vitest'
import App from './App'

function healthResponse(connected: boolean) {
  return Response.json(
    {
      status: connected ? 'healthy' : 'degraded',
      database: {
        engine: 'PostgreSQL',
        connected,
        message: connected
          ? 'Base de datos conectada y operativa'
          : 'PostgreSQL no está disponible',
      },
    },
    { status: connected ? 200 : 503 }
  )
}

afterEach(() => vi.unstubAllGlobals())

function renderPanel() {
  const panel = document.createElement('div')
  document.body.appendChild(panel)
  const root = createRoot(panel)
  act(() => {
    root.render(createElement(App))
  })
  onTestFinished(() => {
    act(() => {
      root.unmount()
    })
    panel.remove()
  })
  return panel
}

it('consulta la API y muestra la recuperación de PostgreSQL al revisar el estado', async () => {
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(healthResponse(false))
    .mockResolvedValueOnce(healthResponse(true))
  vi.stubGlobal('fetch', fetchMock)

  const panel = renderPanel()
  await vi.waitFor(() => expect(panel.querySelector('.status-degraded')).not.toBeNull())

  expect(fetchMock).toHaveBeenCalledWith(
    expect.stringMatching(/\/api\/health$/),
    expect.any(Object)
  )
  expect(panel.textContent).toContain('PostgreSQL no está disponible')

  const refreshBtn = panel.querySelector('.btn-refresh') as HTMLButtonElement
  act(() => {
    refreshBtn.click()
  })
  await vi.waitFor(() => expect(panel.querySelector('.status-healthy')).not.toBeNull())
  expect(panel.textContent).toContain('Todos los sistemas operativos')
})

it('muestra que la API está inaccesible cuando falla la conexión', async () => {
  vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('Conexión rechazada')))

  const panel = renderPanel()
  await vi.waitFor(() => expect(panel.textContent).toContain('Conexión rechazada'))

  expect(panel.querySelector('.status-error')).not.toBeNull()
  expect(panel.textContent).toContain('No se pudo contactar el Backend')
})

it('procesa el lote OCR simulado aunque la API esté inaccesible', async () => {
  const fetchMock = vi.fn().mockRejectedValue(new Error('Conexión rechazada'))
  vi.stubGlobal('fetch', fetchMock)

  const panel = renderPanel()
  const demo = panel.querySelector('#prueba-pipeline') as HTMLElement
  const submitBtn = demo.querySelector('[type="submit"]') as HTMLButtonElement
  act(() => {
    submitBtn.click()
  })

  expect(demo.querySelector('[role="status"]')?.textContent).toContain(
    '5 recibidos · 2 listos para importar · 3 rechazados'
  )
  const [accepted, rejected] = demo.querySelectorAll('tbody')
  expect(Array.from(accepted.rows[0].cells, (cell) => cell.textContent)).toEqual([
    'DOC-001',
    'Camila Soto',
    'camila@example.com',
  ])
  expect(Array.from(rejected.rows, (row) => row.cells[2].textContent)).toEqual([
    'Falta el nombre',
    'Correo inválido',
    'Folio duplicado',
  ])
  expect(fetchMock).toHaveBeenCalledTimes(1)
})

it('permite corregir un JSON inválido o restaurar el lote de ejemplo', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(healthResponse(true)))

  const panel = renderPanel()
  const demo = panel.querySelector('#prueba-pipeline') as HTMLElement
  const input = demo.querySelector('textarea') as HTMLTextAreaElement
  const submit = demo.querySelector('[type="submit"]') as HTMLButtonElement
  const reset = demo.querySelector('[type="button"]') as HTMLButtonElement
  const example = input.value

  act(() => {
    submit.click()
  })
  expect(demo.querySelector('table')).not.toBeNull()

  act(() => {
    const setter = Object.getOwnPropertyDescriptor(
      window.HTMLTextAreaElement.prototype,
      'value'
    )?.set
    setter?.call(input, '{')
    input.dispatchEvent(new Event('input', { bubbles: true }))
  })
  act(() => {
    submit.click()
  })
  expect(demo.querySelector('[role="alert"]')?.textContent).toContain('sintaxis del JSON')
  expect(demo.querySelector('table')).toBeNull()

  act(() => {
    reset.click()
  })
  expect(input.value).toBe(example)
  expect(demo.querySelector('[role="alert"]')).toBeNull()
})
