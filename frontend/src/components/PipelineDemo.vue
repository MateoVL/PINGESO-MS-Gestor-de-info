<template>
  <section class="pipeline-demo" id="prueba-pipeline" aria-labelledby="titulo-prueba">
    <header>
      <span class="demo-label">PT2 · Datos simulados</span>
      <h2 id="titulo-prueba">Prueba el pipeline de información</h2>
      <p>Procesa un lote simulado del extractor OCR y revisa qué registros quedarían listos para importar.</p>
    </header>

    <form class="batch-form" @submit.prevent="runBatch">
      <label for="lote-ocr">Lote de entrada en JSON</label>
      <p id="formato-lote">Cada registro usa los campos <code>folio</code>, <code>nombre</code> y <code>correo</code>.</p>
      <textarea
        id="lote-ocr"
        v-model="input"
        rows="10"
        spellcheck="false"
        aria-describedby="formato-lote"
        :aria-invalid="Boolean(error)"
        @input="resetResult"
      ></textarea>
      <div class="demo-actions">
        <button class="run-button" type="submit">Procesar lote</button>
        <button class="reset-button" type="button" @click="loadExample">Restaurar ejemplo</button>
      </div>
      <p v-if="error" class="batch-error" role="alert">{{ error }}</p>
    </form>

    <p v-if="result" class="batch-summary" role="status">
      {{ result.total }} recibidos · {{ result.accepted.length }} listos para importar · {{ result.rejected.length }} rechazados
    </p>

    <div v-if="result?.accepted.length" class="table-scroll">
      <table>
        <caption>Registros normalizados</caption>
        <thead>
          <tr><th scope="col">Folio</th><th scope="col">Nombre</th><th scope="col">Correo</th></tr>
        </thead>
        <tbody>
          <tr v-for="record in result.accepted" :key="record.folio">
            <td>{{ record.folio }}</td><td>{{ record.nombre }}</td><td>{{ record.correo }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="result?.rejected.length" class="table-scroll">
      <table>
        <caption>Registros rechazados</caption>
        <thead>
          <tr><th scope="col">Posición</th><th scope="col">Folio</th><th scope="col">Motivo</th></tr>
        </thead>
        <tbody>
          <tr v-for="record in result.rejected" :key="record.position">
            <td>{{ record.position }}</td><td>{{ record.folio || 'Sin folio' }}</td><td>{{ record.reason }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup>
import { ref } from 'vue'
import exampleBatch from '../mocks/lote-ocr.json'

const example = JSON.stringify(exampleBatch, null, 2)
const input = ref(example)
const result = ref(null)
const error = ref('')
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

function resetResult() {
  result.value = null
  error.value = ''
}

function loadExample() {
  input.value = example
  resetResult()
}

function textValue(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function runBatch() {
  resetResult()

  let records
  try {
    records = JSON.parse(input.value)
  } catch {
    error.value = 'Revisa la sintaxis del JSON e inténtalo de nuevo.'
    return
  }
  if (!Array.isArray(records) || records.length === 0) {
    error.value = 'Ingresa un arreglo JSON con al menos un registro.'
    return
  }

  const accepted = []
  const rejected = []
  const seen = new Set()

  for (const [index, raw] of records.entries()) {
    const position = index + 1
    if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
      rejected.push({ position, reason: 'Se esperaba un objeto JSON.' })
      continue
    }

    const folio = textValue(raw.folio).toUpperCase()
    const nombre = textValue(raw.nombre).replace(/\s+/g, ' ')
    const correo = textValue(raw.correo).toLowerCase()
    const issues = []

    if (!folio) issues.push('Falta el folio')
    if (!nombre) issues.push('Falta el nombre')
    if (!emailPattern.test(correo)) issues.push('Correo inválido')
    if (seen.has(folio)) issues.push('Folio duplicado')

    if (issues.length) {
      rejected.push({ position, folio, reason: issues.join(' · ') })
      continue
    }

    seen.add(folio)
    accepted.push({ folio, nombre, correo })
  }

  result.value = { total: records.length, accepted, rejected }
}
</script>

<style scoped>
.pipeline-demo {
  display: grid;
  gap: 1rem;
  padding: clamp(1rem, 3vw, 1.5rem);
  border: 1px solid var(--border-highlight);
  border-radius: 12px;
  background: var(--bg-card);
}

.demo-label {
  color: #a5b4fc;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.pipeline-demo h2 {
  margin-block: 0.25rem;
  font-size: clamp(1.2rem, 3vw, 1.55rem);
}

.pipeline-demo p {
  color: var(--text-muted);
  font-size: 0.9rem;
}

.batch-form {
  display: grid;
  gap: 0.5rem;
  min-width: 0;
}

.batch-form label {
  font-weight: 700;
}

.batch-form textarea {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid var(--border-highlight);
  border-radius: 8px;
  background: var(--bg-primary);
  color: var(--text-main);
  font: 0.85rem/1.5 var(--font-mono);
  resize: vertical;
}

.demo-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.demo-actions button {
  padding: 0.5rem 0.85rem;
  border-radius: 8px;
  color: var(--text-main);
  font-weight: 600;
}

.run-button {
  background: #4f46e5;
}

.reset-button {
  border: 1px solid var(--border-highlight);
  background: var(--bg-secondary);
}

.pipeline-demo .batch-error {
  color: #fca5a5;
}

.pipeline-demo .batch-summary {
  font-weight: 700;
  color: var(--text-main);
}

.table-scroll {
  min-width: 0;
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85rem;
}

caption {
  padding-bottom: 0.5rem;
  text-align: left;
  font-weight: 700;
}

th,
td {
  padding: 0.45rem 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
  text-align: left;
  overflow-wrap: anywhere;
}

th {
  background: var(--bg-secondary);
  color: var(--text-muted);
}
</style>
