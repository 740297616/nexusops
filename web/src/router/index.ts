import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import PlaceholderView from '../views/PlaceholderView.vue'

export const navigation = [
  { path: '/', name: 'Dashboard', label: '总览', icon: 'Grid' },
  { path: '/hosts', name: 'Hosts', label: '主机', icon: 'Monitor' },
  { path: '/monitoring', name: 'Monitoring', label: '监控', icon: 'DataLine' },
  { path: '/docker', name: 'Docker', label: '容器', icon: 'Box' },
  { path: '/alerts', name: 'Alerts', label: '告警', icon: 'Bell' },
  { path: '/settings', name: 'Settings', label: '设置', icon: 'Setting' },
] as const

export default createRouter({
  history: createWebHistory(),
  routes: navigation.map((item) => ({
    path: item.path,
    name: item.name,
    component: item.path === '/' ? DashboardView : PlaceholderView,
    meta: { label: item.label },
  })),
})
