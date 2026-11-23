import { useEstadoServicio } from './use-estado-servicio'

export interface EstadoServicioProps {
  apiBaseUrl: string
  usuario?: string
  permisos?: string[]
}

// Microcomponente autocontenido: vista pura sin lógica de negocio ni dependencia de otros microcomponentes
export function EstadoServicio({ apiBaseUrl }: EstadoServicioProps) {
  const {
    loading,
    health,
    latency,
    backendConnected,
    dbConnected,
    overallStatusClass,
    bannerTitle,
    bannerSubtitle,
    checkHealth,
  } = useEstadoServicio(apiBaseUrl)

  const frontendPort =
    typeof window !== 'undefined'
      ? window.location.port || (window.location.protocol === 'https:' ? '443' : '80')
      : '3000'

  return (
    <div className="estado-servicio-widget">
      <div className="header-actions" style={{ marginBottom: '1rem', justifyContent: 'flex-end' }}>
        <button
          className="btn-refresh"
          type="button"
          onClick={() => void checkHealth()}
          disabled={loading}
        >
          {loading ? 'Verificando...' : 'Revisar Estado'}
        </button>
      </div>

      <section className={`banner ${overallStatusClass}`} aria-live="polite">
        <div className="banner-icon">
          {loading ? (
            <span>⏳</span>
          ) : health?.status === 'healthy' ? (
            <span>✓</span>
          ) : (
            <span>!</span>
          )}
        </div>
        <div className="banner-body">
          <h2>{bannerTitle}</h2>
          <p>{bannerSubtitle}</p>
        </div>
        {latency !== null ? (
          <div className="banner-meta">
            <span className="mono-badge">{latency} ms</span>
          </div>
        ) : null}
      </section>

      <section className="grid-cards" style={{ marginTop: '1.25rem' }}>
        <div className="card">
          <div className="card-header">
            <div className="card-icon icon-go">Go</div>
            <div>
              <h3>Backend API</h3>
              <p className="card-subtitle">Go 1.26.11 • Huma v2 OpenAPI 3.1</p>
            </div>
            <span className={`status-pill ${backendConnected ? 'pill-ok' : 'pill-err'}`}>
              {backendConnected ? 'ONLINE' : 'OFFLINE'}
            </span>
          </div>
          <div className="card-body">
            <div className="metric-row">
              <span className="metric-label">Endpoint</span>
              <span className="metric-val mono">{apiBaseUrl}/api/health</span>
            </div>
            <div className="metric-row">
              <span className="metric-label">Uptime</span>
              <span className="metric-val mono">{health?.uptime || 'N/A'}</span>
            </div>
            <div className="metric-row">
              <span className="metric-label">Entorno</span>
              <span className="metric-val mono">{health?.environment || 'detectando...'}</span>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <div className="card-icon icon-db">DB</div>
            <div>
              <h3>Base de Datos</h3>
              <p className="card-subtitle">PostgreSQL 17.11</p>
            </div>
            <span className={`status-pill ${dbConnected ? 'pill-ok' : 'pill-err'}`}>
              {dbConnected ? 'CONECTADA' : 'DESCONECTADA'}
            </span>
          </div>
          <div className="card-body">
            <div className="metric-row">
              <span className="metric-label">Motor</span>
              <span className="metric-val">{health?.database?.engine || 'PostgreSQL'}</span>
            </div>
            <div className="metric-row">
              <span className="metric-label">Estado</span>
              <span className="metric-val">
                {health?.database?.message || (backendConnected ? 'Sin respuesta' : 'Inaccesible')}
              </span>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <div className="card-icon icon-react">UI</div>
            <div>
              <h3>Microcomponentes</h3>
              <p className="card-subtitle">React 19.2.8 • Vite 8.1.5</p>
            </div>
            <span className="status-pill pill-ok">ACTIVO</span>
          </div>
          <div className="card-body">
            <div className="metric-row">
              <span className="metric-label">Puerto</span>
              <span className="metric-val mono">{frontendPort}</span>
            </div>
            <div className="metric-row">
              <span className="metric-label">Origen de la API</span>
              <span className="metric-val mono">{apiBaseUrl || 'Mismo origen'}</span>
            </div>
            <div className="metric-row">
              <span className="metric-label">Paquete de trabajo</span>
              <span className="metric-val">PT2</span>
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}
