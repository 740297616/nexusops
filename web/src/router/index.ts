import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import LoginView from '../views/LoginView.vue'
import DashboardView from '../views/DashboardView.vue'
import UsersView from '../views/UsersView.vue'
import RolesView from '../views/RolesView.vue'
import AuditView from '../views/AuditView.vue'
import SessionsView from '../views/SessionsView.vue'
import ConfigView from '../views/ConfigView.vue'
import PlaceholderView from '../views/PlaceholderView.vue'
import ServersView from '../views/ServersView.vue'
export const navigation = [
  { path: '/', name: 'Dashboard', label: '总览', icon: 'Grid' },
  {
    path: '/users',
    name: 'Users',
    label: '用户管理',
    icon: 'User',
    permission: 'system:user:read',
  },
  {
    path: '/roles',
    name: 'Roles',
    label: '角色与权限',
    icon: 'Key',
    permission: 'system:role:read',
  },
  {
    path: '/sessions',
    name: 'Sessions',
    label: '会话管理',
    icon: 'Connection',
    permission: 'system:session:read',
  },
  {
    path: '/audit',
    name: 'Audit',
    label: '安全审计',
    icon: 'Document',
    permission: 'system:audit:read',
  },
  {
    path: '/settings',
    name: 'Settings',
    label: '系统配置',
    icon: 'Setting',
    permission: 'system:config:read',
  },
  { path: '/hosts', name: 'Hosts', label: '主机', icon: 'Monitor', permission: 'infrastructure:server:read' },
  { path: '/monitoring', name: 'Monitoring', label: '监控', icon: 'DataLine' },
  { path: '/docker', name: 'Docker', label: '容器', icon: 'Box' },
  { path: '/alerts', name: 'Alerts', label: '告警', icon: 'Bell' },
] as const
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'Login', component: LoginView, meta: { public: true } },
    { path: '/', name: 'Dashboard', component: DashboardView, meta: { label: '总览' } },
    {
      path: '/users',
      name: 'Users',
      component: UsersView,
      meta: { label: '用户管理', permission: 'system:user:read' },
    },
    {
      path: '/roles',
      name: 'Roles',
      component: RolesView,
      meta: { label: '角色与权限', permission: 'system:role:read' },
    },
    {
      path: '/sessions',
      name: 'Sessions',
      component: SessionsView,
      meta: { label: '会话管理', permission: 'system:session:read' },
    },
    {
      path: '/audit',
      name: 'Audit',
      component: AuditView,
      meta: { label: '安全审计', permission: 'system:audit:read' },
    },
    {
      path: '/settings',
      name: 'Settings',
      component: ConfigView,
      meta: { label: '系统配置', permission: 'system:config:read' },
    },
    {
      path: '/hosts',
      name: 'Hosts',
      component: ServersView,
      meta: { label: '主机', permission: 'infrastructure:server:read' },
    },
    {
      path: '/monitoring',
      name: 'Monitoring',
      component: PlaceholderView,
      meta: { label: '监控' },
    },
    { path: '/docker', name: 'Docker', component: PlaceholderView, meta: { label: '容器' } },
    { path: '/alerts', name: 'Alerts', component: PlaceholderView, meta: { label: '告警' } },
  ],
})
router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.ready) await auth.load()
  if (to.meta.public && auth.authenticated) return '/'
  if (!to.meta.public && !auth.authenticated) return '/login'
  if (typeof to.meta.permission === 'string' && !auth.can(to.meta.permission)) return '/'
  return true
})
export default router
