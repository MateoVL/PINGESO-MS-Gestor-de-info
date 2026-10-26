<template>
  <div class="layout-container">
    <!-- Header -->
    <header class="header">
      <div class="header-content">
        <div class="brand">
          <div class="brand-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
              <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
              <line x1="12" y1="22.08" x2="12" y2="12"/>
            </svg>
          </div>
          <div>
            <div class="badge-mini">PINGESO • LPS</div>
            <h1 class="service-title">MS Gestor de Información</h1>
          </div>
        </div>

        <div class="header-actions">
          <div class="env-tag">
            <span class="pulse-dot"></span>
            <span>Vite + Vue 3</span>
          </div>
          <button class="btn-refresh" @click="fetchHealth" :disabled="loading">
            <svg class="spin-icon" :class="{ spinning: loading }" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.19"/>
            </svg>
            {{ loading ? 'Verificando...' : 'Revisar Estado' }}
          </button>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="main-content">
      <!-- Status Banner -->
      <section class="banner" :class="overallStatusClass">
        <div class="banner-icon">
          <span v-if="health && health.status === 'healthy'">✓</span>
          <span v-else-if="loading">⏳</span>
          <span v-else>!</span>
        </div>
        <div class="banner-body">
          <h2>{{ bannerTitle }}</h2>
          <p>{{ bannerSubtitle }}</p>
        </div>
        <div class="banner-meta" v-if="latency !== null">
          <span class="mono-badge">{{ latency }} ms</span>
        </div>
      </section>

      <!-- System Health Grid -->
      <section class="grid-cards">
        <!-- Backend Card -->
        <div class="card">
          <div class="card-header">
            <div class="card-icon icon-go">Go</div>
            <div>
              <h3>Backend API</h3>
              <p class="card-subtitle">Go 1.23 • Gin HTTP Server</p>
            </div>
            <span class="status-pill" :class="backendConnected ? 'pill-ok' : 'pill-err'">
              {{ backendConnected ? 'ONLINE' : 'OFFLINE' }}
            </span>
          </div>
          <div class="card-body">
            <div class="metric-row">
              <span class="metric-label">Endpoint</span>
              <span class="metric-val mono">{{ apiBaseUrl }}/health</span>
            </div>
            <div class="metric-row">
              <span class="metric-label">Uptime</span>
              <span class="metric-val mono">{{ health?.uptime || 'N/A' }}</span>
            </div>
            <div class="metric-row">
              <span class="metric-label">Entorno</span>
              <span class="metric-val mono">{{ health?.environment || 'detectando...' }}</span>
            </div>
          </div>
        </div>

        <!-- Database Card -->
        <div class="card">
          <div class="card-header">
            <div class="card-icon icon-db">PG</div>
            <div>
              <h3>Base de Datos</h3>
              <p class="card-subtitle">PostgreSQL 16</p>
            </div>
            <span class="status-pill" :class="dbConnected ? 'pill-ok' : 'pill-err'">
              {{ dbConnected ? 'CONECTADA' : 'DESCONECTADA' }}
            </span>
          </div>
          <div class="card-body">
            <div class="metric-row">
              <span class="metric-label">Motor</span>
              <span class="metric-val">{{ health?.database?.engine || 'PostgreSQL' }}</span>
            </div>
            <div class="metric-row">
              <span class="metric-label">Estado</span>
              <span class="metric-val">{{ health?.database?.message || (backendConnected ? 'Sin respuesta' : 'Inaccesible') }}</span>
            </div>
            <div class="metric-row">
              <span class="metric-label">Init Script</span>
              <span class="metric-val mono">scripts/init-db.sql</span>
            </div>
          </div>
        </div>

        <!-- Frontend Card -->
        <div class="card">
          <div class="card-header">
            <div class="card-icon icon-vue">Vue</div>
            <div>
              <h3>Frontend</h3>
              <p class="card-subtitle">Vue 3 • Vite Single Page App</p>
            </div>
            <span class="status-pill pill-ok">ACTIVO</span>
          </div>
          <div class="card-body">
            <div class="metric-row">
              <span class="metric-label">Puerto</span>
              <span class="metric-val mono">:3000</span>
            </div>
            <div class="metric-row">
              <span class="metric-label">VITE_API_BASE_URL</span>
              <span class="metric-val mono">{{ apiBaseUrl }}</span>
            </div>
            <div class="metric-row">
              <span class="metric-label">Hot Reload</span>
              <span class="metric-val">Habilitado</span>
            </div>
          </div>
        </div>
      </section>

      <!-- Architecture and Product Line Position -->
      <section class="section-box">
        <div class="section-title">
          <h3>Arquitectura de Línea de Productos de Software (LPS)</h3>
          <p>Ubicación y responsabilidades del Microservicio Gestor de Información</p>
        </div>

        <div class="arch-flow">
          <div class="arch-node prev-node">
            <span class="node-tag">Dependencia Upstream</span>
            <h4>MS Digitalización</h4>
            <p>Captura de documentos, OCR y digitalización</p>
          </div>

          <div class="arch-arrow">➔</div>

          <div class="arch-node current-node">
            <span class="node-tag current-tag">Este Microservicio</span>
            <h4>MS Gestor de Información</h4>
            <ul>
              <li>📥 Carga de información (Ingesta)</li>
              <li>⚙️ Procesamiento y validación</li>
              <li>🗂️ Ordenamiento de datos para reporte</li>
            </ul>
          </div>

          <div class="arch-arrow">➔</div>

          <div class="arch-node next-node">
            <span class="node-tag">Dependencia Downstream</span>
            <h4>MS Reportes</h4>
            <p>Generación de reportes finales (PDF, Excel)</p>
          </div>
        </div>
      </section>

      <!-- Quick Commands Guide -->
      <section class="section-box commands-box">
        <div class="section-title">
          <h3>Comandos del Makefile</h3>
          <p>Operaciones estandarizadas para desarrollo local y despliegue con Docker Compose</p>
        </div>

        <div class="commands-grid">
          <div class="cmd-item">
            <div class="cmd-code"><code>make dev</code></div>
            <div class="cmd-desc">Inicia el entorno de desarrollo local con Docker Compose (Hot reload en Go y Vue).</div>
          </div>
          <div class="cmd-item">
            <div class="cmd-code"><code>make prod</code></div>
            <div class="cmd-desc">Construye y levanta imágenes de producción optimizadas (multi-stage + Nginx).</div>
          </div>
          <div class="cmd-item">
            <div class="cmd-code"><code>make down</code></div>
            <div class="cmd-desc">Detiene todos los contenedores y redes asociadas.</div>
          </div>
          <div class="cmd-item">
            <div class="cmd-code"><code>make env</code></div>
            <div class="cmd-desc">Crea el archivo <code>.env</code> a partir de <code>.env.example</code> si no existe.</div>
          </div>
        </div>
      </section>
    </main>

    <!-- Footer -->
    <footer class="footer">
      <p>Proyecto de Ingeniería de Software (PINGESO) • Base de Microservicio Gestor de Info</p>
    </footer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'
const loading = ref(false)
const health = ref(null)
const error = ref(null)
const latency = ref(null)

const backendConnected = computed(() => !!health.value && !error.value)
const dbConnected = computed(() => !!health.value?.database?.connected)

const overallStatusClass = computed(() => {
  if (loading.value) return 'status-loading'
  if (backendConnected.value && dbConnected.value) return 'status-healthy'
  if (backendConnected.value && !dbConnected.value) return 'status-degraded'
  return 'status-error'
})

const bannerTitle = computed(() => {
  if (loading.value) return 'Verificando servicios...'
  if (backendConnected.value && dbConnected.value) return 'Todos los sistemas operativos'
  if (backendConnected.value && !dbConnected.value) return 'Backend operativo, Base de Datos no conectada'
  return 'No se pudo contactar el Backend'
})

const bannerSubtitle = computed(() => {
  if (loading.value) return 'Consultando endpoint /api/health...'
  if (backendConnected.value && dbConnected.value) {
    return 'Go Gin API y PostgreSQL 16 están comunicándose correctamente.'
  }
  if (backendConnected.value && !dbConnected.value) {
    return `Error en BD: ${health.value?.database?.message || 'Revisar variables DB_*'}`
  }
  return error.value || 'Verifica que el contenedor de backend o el servidor Go esté corriendo en ' + apiBaseUrl
})

async function fetchHealth() {
  loading.value = true
  error.value = null
  const start = performance.now()
  try {
    const res = await fetch(`${apiBaseUrl}/api/health`, {
      headers: { 'Accept': 'application/json' }
    })
    latency.value = Math.round(performance.now() - start)
    const data = await res.json()
    health.value = data
  } catch (err) {
    latency.value = null
    health.value = null
    error.value = err.message || 'Error de conexión'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchHealth()
})
</script>

<style scoped>
.layout-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.header {
  border-bottom: 1px solid var(--border-subtle);
  background: rgba(10, 14, 23, 0.8);
  backdrop-filter: blur(12px);
  position: sticky;
  top: 0;
  z-index: 100;
}

.header-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 1rem 1.5rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.brand {
  display: flex;
  align-items: center;
  gap: 0.85rem;
}

.brand-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--primary), #8b5cf6);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  box-shadow: 0 4px 14px var(--primary-glow);
}

.badge-mini {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--primary);
  font-weight: 700;
}

.service-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.02em;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.env-tag {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.35rem 0.75rem;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-subtle);
  border-radius: 999px;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.pulse-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--success);
  box-shadow: 0 0 8px var(--success);
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(0.85); }
}

.btn-refresh {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.45rem 1rem;
  background: var(--primary);
  color: white;
  font-weight: 600;
  font-size: 0.85rem;
  border-radius: 8px;
  transition: all 0.2s ease;
  box-shadow: 0 4px 12px var(--primary-glow);
}

.btn-refresh:hover:not(:disabled) {
  background: #4f46e5;
  transform: translateY(-1px);
}

.btn-refresh:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  100% { transform: rotate(360deg); }
}

.main-content {
  max-width: 1200px;
  width: 100%;
  margin: 0 auto;
  padding: 2rem 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.75rem;
  flex: 1;
}

/* Banner */
.banner {
  display: flex;
  align-items: center;
  gap: 1.25rem;
  padding: 1.25rem 1.5rem;
  border-radius: 12px;
  border: 1px solid transparent;
  transition: all 0.3s ease;
}

.status-healthy {
  background: rgba(16, 185, 129, 0.1);
  border-color: rgba(16, 185, 129, 0.3);
}

.status-healthy .banner-icon {
  background: var(--success);
  color: #064e3b;
}

.status-degraded {
  background: rgba(245, 158, 11, 0.1);
  border-color: rgba(245, 158, 11, 0.3);
}

.status-degraded .banner-icon {
  background: var(--warning);
  color: #78350f;
}

.status-error {
  background: rgba(239, 68, 68, 0.1);
  border-color: rgba(239, 68, 68, 0.3);
}

.status-error .banner-icon {
  background: var(--error);
  color: white;
}

.status-loading {
  background: rgba(99, 102, 241, 0.1);
  border-color: rgba(99, 102, 241, 0.3);
}

.banner-icon {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  font-size: 1.2rem;
  flex-shrink: 0;
}

.banner-body {
  flex: 1;
}

.banner-body h2 {
  font-size: 1.1rem;
  font-weight: 700;
  margin-bottom: 0.2rem;
}

.banner-body p {
  font-size: 0.9rem;
  color: var(--text-muted);
}

.mono-badge {
  font-family: var(--font-mono);
  font-size: 0.8rem;
  background: rgba(0, 0, 0, 0.3);
  padding: 0.25rem 0.6rem;
  border-radius: 6px;
  color: var(--text-muted);
}

/* Grid */
.grid-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 1.25rem;
}

.card {
  background: var(--bg-card);
  backdrop-filter: blur(8px);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  transition: all 0.2s ease;
}

.card:hover {
  background: var(--bg-card-hover);
  border-color: var(--border-highlight);
  transform: translateY(-2px);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.card-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  font-size: 0.85rem;
}

.icon-go {
  background: #00add8;
  color: white;
}

.icon-db {
  background: #336791;
  color: white;
}

.icon-vue {
  background: #42b883;
  color: #0f2e21;
}

.card-header h3 {
  font-size: 1rem;
  font-weight: 700;
}

.card-subtitle {
  font-size: 0.75rem;
  color: var(--text-dim);
}

.status-pill {
  margin-left: auto;
  font-size: 0.7rem;
  font-weight: 700;
  padding: 0.25rem 0.55rem;
  border-radius: 999px;
  letter-spacing: 0.05em;
}

.pill-ok {
  background: rgba(16, 185, 129, 0.15);
  color: var(--success);
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.pill-err {
  background: rgba(239, 68, 68, 0.15);
  color: var(--error);
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}

.metric-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.85rem;
  padding-bottom: 0.4rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}

.metric-label {
  color: var(--text-muted);
}

.metric-val {
  color: var(--text-main);
  font-weight: 500;
}

.mono {
  font-family: var(--font-mono);
  font-size: 0.8rem;
}

/* Architecture Section */
.section-box {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.section-title h3 {
  font-size: 1.1rem;
  font-weight: 700;
}

.section-title p {
  font-size: 0.85rem;
  color: var(--text-muted);
}

.arch-flow {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.arch-node {
  flex: 1;
  min-width: 240px;
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid var(--border-subtle);
  border-radius: 10px;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.current-node {
  border-color: var(--border-highlight);
  background: rgba(99, 102, 241, 0.08);
  box-shadow: 0 0 20px rgba(99, 102, 241, 0.1);
}

.node-tag {
  font-size: 0.7rem;
  text-transform: uppercase;
  font-weight: 700;
  letter-spacing: 0.05em;
  color: var(--text-dim);
}

.current-tag {
  color: var(--primary);
}

.arch-node h4 {
  font-size: 0.95rem;
  font-weight: 700;
}

.arch-node p, .arch-node ul {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.arch-node ul {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-top: 0.25rem;
}

.arch-arrow {
  color: var(--text-dim);
  font-size: 1.4rem;
  font-weight: bold;
}

/* Commands */
.commands-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 1rem;
}

.cmd-item {
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.cmd-code code {
  font-family: var(--font-mono);
  background: rgba(99, 102, 241, 0.15);
  color: #a5b4fc;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  font-size: 0.85rem;
  font-weight: 600;
}

.cmd-desc {
  font-size: 0.8rem;
  color: var(--text-muted);
  line-height: 1.4;
}

/* Footer */
.footer {
  border-top: 1px solid var(--border-subtle);
  padding: 1.5rem;
  text-align: center;
  font-size: 0.8rem;
  color: var(--text-dim);
  margin-top: auto;
}

@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
    align-items: flex-start;
  }
  .arch-arrow {
    display: none;
  }
}
</style>
