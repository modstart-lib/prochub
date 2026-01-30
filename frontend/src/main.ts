import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'
import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import { i18n } from './plugins/i18n'
import { trackError, trackOpen } from './services/analytics'
import './style.css'

const app = createApp(App)
app.use(createPinia())
app.use(i18n)
app.use(Antd)

// Track app open event
trackOpen()

// Global error handler for Vue
app.config.errorHandler = (err, _instance, info) => {
  const errorMessage = err instanceof Error ? err.message : String(err)
  trackError(`Vue Error: ${errorMessage} (${info})`)
  console.error('Vue Error:', err, info)
}

// Global unhandled promise rejection handler
window.addEventListener('unhandledrejection', (event) => {
  const errorMessage = event.reason instanceof Error ? event.reason.message : String(event.reason)
  trackError(`Unhandled Promise Rejection: ${errorMessage}`)
})

// Global error handler for uncaught errors
window.addEventListener('error', (event) => {
  trackError(`Uncaught Error: ${event.message}`)
})

app.mount('#app')
