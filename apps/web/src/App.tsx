import { PipelineFeature } from './features/pipeline/PipelineFeature'

export interface AppProps {
  apiBaseUrl?: string
}

export function App({ apiBaseUrl = '' }: AppProps = {}) {
  const cleanBaseUrl = apiBaseUrl.replace(/\/$/, '')

  return (
    <div className="layout-container">
      <header className="header">
        <div className="header-content">
          <div className="brand">
            <div className="brand-icon">
              <svg
                width="24"
                height="24"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
                <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
                <line x1="12" y1="22.08" x2="12" y2="12" />
              </svg>
            </div>
            <div>
              <div className="badge-mini">PINGESO · Grupo 2 · Hito 0</div>
              <h1 className="service-title">MS Gestor de Información</h1>
            </div>
          </div>

          <div className="header-actions">
            <div className="env-tag">
              <span className="pulse-dot" />
              <span>Vite + React 19</span>
            </div>
          </div>
        </div>
      </header>

      <main className="main-content">
        <PipelineFeature
          apiBaseUrl={cleanBaseUrl}
          usuario="desarrollador-local"
          permisos={['lotes:operar', 'lotes:leer', 'registros:leer']}
        />
      </main>

      <footer
        className="footer"
        style={{
          padding: '1.5rem',
          textAlign: 'center',
          fontSize: '0.8rem',
          color: 'var(--text-dim)',
        }}
      >
        <p>Proyecto de Ingeniería de Software (PINGESO) • Base de Microservicio Gestor de Info</p>
      </footer>
    </div>
  )
}

export default App
