import { createApp } from 'vue'
import App from './App.vue'
import { init } from './storeNav'
import { initEvents } from './api'

import './styles/theme.css'
import './styles/components.css'
import './styles/overlays.css'
import './styles/files.css'
import './styles/panels.css'
import './styles/panels2.css'

initEvents()

const app = createApp(App)
app.mount('#app')

// Bootstrap the backend connection (falls back to demo data in a bare browser).
init()
