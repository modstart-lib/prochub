import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'
import { createPinia } from 'pinia'
import { createApp, nextTick } from 'vue'
import App from './App.vue'
import { i18n } from './plugins/i18n'
import { trackError, trackOpen } from './services/analytics'
import { installRendererDiagnostics, logMounted, logRenderer } from './services/diagnostics'
import './style.css'

const app = createApp(App)
app.use(createPinia())
app.use(i18n)
app.use(Antd)

// Diagnostics are registered before mount on purpose so startup / mount errors
// are captured instead of only after the app resolved.
installRendererDiagnostics()

// Track app open event
trackOpen()

// Global error handler for Vue render / lifecycle errors.
app.config.errorHandler = (err, _instance, info) => {
  const message = err instanceof Error ? err.message : String(err)
  const stack = err instanceof Error ? err.stack : undefined
  logRenderer('error', 'vue.errorHandler', { message, stack, info })
  console.error('Vue Error:', err, info)
  trackError(`Vue Error: ${message} (${info})`)
}

app.mount('#app')
nextTick(() => logMounted())
