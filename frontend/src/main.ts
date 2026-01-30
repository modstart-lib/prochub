import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'
import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import { i18n } from './plugins/i18n'
import './style.css'

const app = createApp(App)
app.use(createPinia())
app.use(i18n)
app.use(Antd)
app.mount('#app')
