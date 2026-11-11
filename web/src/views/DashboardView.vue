<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Connection, RefreshRight, User, Key, Document } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { ApiError } from '../api/client'
import { getHealth, type Health } from '../api/health'
import { listUsers, listRoles, listSessions, listAuthAudit } from '../api/system'
import { useAuthStore } from '../stores/auth'
import type { AuthAudit, Page, Role, Session, User as UserModel } from '../types/api'
const auth = useAuthStore()
const health = ref<Health | null>(null)
const users = ref<Page<UserModel> | null>(null)
const roles = ref<Role[]>([])
const sessions = ref<Page<Session> | null>(null)
const events = ref<Page<AuthAudit> | null>(null)
const loading = ref(false)
const checkedAt = ref('')
const error = ref('')
const can = (permission: string) => auth.can(permission)
async function refresh() {
  loading.value = true
  error.value = ''
  try {
    health.value = await getHealth()
    checkedAt.value = new Date().toLocaleTimeString()
    const tasks: Promise<unknown>[] = []
    if (can('system:user:read'))
      tasks.push(listUsers({ page: 1, page_size: 1 }).then((v) => (users.value = v)))
    if (can('system:role:read')) tasks.push(listRoles().then((v) => (roles.value = v)))
    if (can('system:session:read'))
      tasks.push(listSessions({ page: 1, page_size: 1 }).then((v) => (sessions.value = v)))
    if (can('system:audit:read'))
      tasks.push(listAuthAudit({ page: 1, page_size: 5 }).then((v) => (events.value = v)))
    await Promise.all(tasks)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '加载失败'
    ElMessage.error(error.value)
  } finally {
    loading.value = false
  }
}
onMounted(refresh)
</script>
<template>
  <div class="page-heading">
    <div>
      <div class="eyebrow">WORKSPACE / OVERVIEW</div>
      <h1>总览</h1>
      <p>系统管理概览</p>
    </div>
    <button class="refresh-button" type="button" :disabled="loading" @click="refresh">
      <el-icon><RefreshRight /></el-icon>刷新
    </button>
  </div>
  <section class="stats-grid">
    <div v-if="users" class="stat-card">
      <el-icon><User /></el-icon><span>用户总数</span><strong>{{ users.total }}</strong>
    </div>
    <div v-if="roles.length" class="stat-card">
      <el-icon><Key /></el-icon><span>角色总数</span><strong>{{ roles.length }}</strong>
    </div>
    <div v-if="sessions" class="stat-card">
      <el-icon><Connection /></el-icon><span>活跃会话</span><strong>{{ sessions.total }}</strong>
    </div>
    <div v-if="events" class="stat-card">
      <el-icon><Document /></el-icon><span>最近认证事件</span><strong>{{ events.total }}</strong>
    </div>
  </section>
  <section class="section-block">
    <div class="section-heading">
      <div>
        <h2>服务状态</h2>
        <p>Server 健康检查</p>
      </div>
    </div>
    <div class="service-row">
      <div class="service-icon">
        <el-icon :size="21"><Connection /></el-icon>
      </div>
      <div class="service-name"><strong>API Server</strong><span>管理服务</span></div>
      <div class="service-status">
        <span
          :class="[
            'status-dot',
            loading ? 'pending' : health?.status === 'ok' ? 'online' : 'offline',
          ]"
        />{{ loading ? '检查中' : health?.status === 'ok' ? '运行中' : '不可用' }}
      </div>
      <div class="service-meta">
        <span>版本</span><strong>{{ health?.version || '—' }}</strong>
      </div>
      <div class="service-meta last-checked">
        <span>最近检查</span><strong>{{ checkedAt || '—' }}</strong>
      </div>
    </div>
    <p v-if="error" class="error-message" role="alert">{{ error }}</p>
  </section>
  <section v-if="events?.items.length" class="section-block">
    <div class="section-heading">
      <div>
        <h2>最近认证活动</h2>
        <p>来自安全审计 API 的最新事件</p>
      </div>
    </div>
    <el-table :data="events.items" size="small"
      ><el-table-column prop="created_at" label="时间" /><el-table-column
        prop="event_type"
        label="事件" /><el-table-column prop="outcome" label="结果"
    /></el-table>
  </section>
  <div class="page-footnote"><span>NEXUSOPS</span><span>自托管基础设施管理</span></div>
</template>
