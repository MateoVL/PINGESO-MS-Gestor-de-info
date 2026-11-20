import { useState, useEffect, useCallback } from 'react'
import type { HealthResponse } from '../../types/openapi'
import { fetchServiceHealth } from './estado-servicio.api'

export function useEstadoServicio(apiBaseUrl: string) {
  const [loading, setLoading] = useState<boolean>(false)
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [latency, setLatency] = useState<number | null>(null)

  const checkHealth = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await fetchServiceHealth(apiBaseUrl)
      setHealth(result.data)
      setLatency(result.latencyMs)
    } catch (err) {
      setLatency(null)
      setHealth(null)
      setError(err instanceof Error ? err.message : 'Error de conexión')
    } finally {
      setLoading(false)
    }
  }, [apiBaseUrl])

  useEffect(() => {
    void checkHealth()
  }, [checkHealth])

  const backendConnected = Boolean(health && !error)
  const dbConnected = Boolean(health?.database?.connected)

  let overallStatusClass = 'status-error'
  if (loading) overallStatusClass = 'status-loading'
  else if (backendConnected && dbConnected) overallStatusClass = 'status-healthy'
  else if (backendConnected && !dbConnected) overallStatusClass = 'status-degraded'

  let bannerTitle = 'No se pudo contactar el Backend'
  if (loading) bannerTitle = 'Verificando servicios...'
  else if (backendConnected && dbConnected) bannerTitle = 'Todos los sistemas operativos'
  else if (backendConnected && !dbConnected) bannerTitle = 'Backend operativo, Base de Datos no conectada'

  let bannerSubtitle = error || 'Consulta el estado de la API para ver el diagnóstico.'
  if (loading) bannerSubtitle = 'Consultando endpoint /api/health...'
  else if (backendConnected && dbConnected) bannerSubtitle = 'La API de Go está conectada a PostgreSQL.'
  else if (backendConnected && !dbConnected && health) bannerSubtitle = health.database.message

  return {
    loading,
    health,
    error,
    latency,
    backendConnected,
    dbConnected,
    overallStatusClass,
    bannerTitle,
    bannerSubtitle,
    checkHealth,
  }
}

