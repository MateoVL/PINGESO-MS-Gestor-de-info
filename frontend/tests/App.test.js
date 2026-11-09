// @vitest-environment jsdom
import { createApp, nextTick } from 'vue'
import { afterEach, expect, it, onTestFinished, vi } from 'vitest'
import App from '../src/App.vue'

function healthResponse(connected) {
  return Response.json({
    status: connected ? 'healthy' : 'degraded',
    database: {
      engine: 'MongoDB',
      connected,
      message: connected ? 'Base de datos conectada y operativa' : 'MongoDB no está disponible',
    },
  }, { status: connected ? 200 : 503 })
}

afterEach(() => vi.unstubAllGlobals())

function renderPanel() {
  const panel = document.createElement('div')
  document.body.appendChild(panel)
  const app = createApp(App)
  app.mount(panel)
  onTestFinished(() => {
    app.unmount()
    panel.remove()
  })
  return panel
}

it('consulta la API y muestra la recuperación de MongoDB al revisar el estado', async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(healthResponse(false))
    .mockResolvedValueOnce(healthResponse(true))
  vi.stubGlobal('fetch', fetch)

  const panel = renderPanel()
  await vi.waitFor(() => expect(panel.querySelector('.status-degraded')).not.toBeNull())

  expect(fetch).toHaveBeenCalledWith(expect.stringMatching(/\/api\/health$/), expect.any(Object))
  expect(panel.textContent).toContain('MongoDB no está disponible')

  panel.querySelector('button').click()
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
  const fetch = vi.fn().mockRejectedValue(new Error('Conexión rechazada'))
  vi.stubGlobal('fetch', fetch)

  const demo = renderPanel().querySelector('#prueba-pipeline')
  demo.querySelector('[type="submit"]').click()
  await nextTick()

  expect(demo.querySelector('[role="status"]').textContent)
    .toContain('5 recibidos · 2 listos para importar · 3 rechazados')
  const [accepted, rejected] = demo.querySelectorAll('tbody')
  expect(Array.from(accepted.rows[0].cells, (cell) => cell.textContent))
    .toEqual(['DOC-001', 'Camila Soto', 'camila@example.com'])
  expect(Array.from(rejected.rows, (row) => row.cells[2].textContent))
    .toEqual(['Falta el nombre', 'Correo inválido', 'Folio duplicado'])
  expect(fetch).toHaveBeenCalledTimes(1)
})

it('permite corregir un JSON inválido o restaurar el lote de ejemplo', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(healthResponse(true)))

  const demo = renderPanel().querySelector('#prueba-pipeline')
  const input = demo.querySelector('textarea')
  const submit = demo.querySelector('[type="submit"]')
  const example = input.value

  submit.click()
  await nextTick()
  expect(demo.querySelector('table')).not.toBeNull()

  input.value = '{'
  input.dispatchEvent(new Event('input'))
  submit.click()
  await nextTick()
  expect(demo.querySelector('[role="alert"]').textContent).toContain('sintaxis del JSON')
  expect(demo.querySelector('table')).toBeNull()

  demo.querySelector('[type="button"]').click()
  await nextTick()
  expect(input.value).toBe(example)
  expect(demo.querySelector('[role="alert"]')).toBeNull()
})
