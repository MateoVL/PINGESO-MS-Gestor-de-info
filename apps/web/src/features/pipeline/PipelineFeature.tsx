import { PipelineDemo } from '../../componentes/pipeline-demo/pipeline-demo'
import { EstadoServicio } from '../../componentes/estado-servicio/estado-servicio'

export interface PipelineFeatureProps {
  apiBaseUrl: string
  usuario?: string
  permisos?: string[]
}

export function PipelineFeature({ apiBaseUrl, usuario, permisos }: PipelineFeatureProps) {
  return (
    <div
      className="pipeline-feature"
      style={{ display: 'flex', flexDirection: 'column', gap: '1.75rem' }}
    >
      <PipelineDemo apiBaseUrl={apiBaseUrl} usuario={usuario} permisos={permisos} />
      <EstadoServicio apiBaseUrl={apiBaseUrl} usuario={usuario} permisos={permisos} />
    </div>
  )
}
