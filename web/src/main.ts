import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { ElIcon, ElTooltip } from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import './styles.css'
import App from './App.vue'
import router from './router'

createApp(App).use(createPinia()).use(router).use(ElIcon).use(ElTooltip).mount('#app')
