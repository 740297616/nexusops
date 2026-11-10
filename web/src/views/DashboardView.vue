<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Connection, RefreshRight } from '@element-plus/icons-vue'
import { ApiError } from '../api/client'
import { getHealth, type Health } from '../api/health'

const health = ref<Health | null>(null)
const loading = ref(false)
const error = ref('')
const checkedAt = ref('')

async function refresh() {
  loading.value = true
  error.value = ''
  health.value = null
  try {
    health.value = await getHealth()
    checkedAt.value = new Date().toLocaleTimeString()
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '健康检查失败'
    checkedAt.value = new Date().toLocaleTimeString()
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
      <p>基础设施运行状态</p>
    </div>
    <span class="page-tag">系统概览</span>
  </div>
  <section class="section-block">
    <div class="section-heading">
      <div>
        <h2>服务状态</h2>
        <p>管理服务连接</p>
      </div>
      <button class="refresh-button" type="button" :disabled="loading" @click="refresh">
        <el-icon><RefreshRight /></el-icon> 刷新
      </button>
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
    <p v-if="error" class="error-message" role="alert">
      {{ error }}。请检查 Server 是否运行及代理配置。
    </p>
  </section>
  <section class="section-block inventory">
    <div class="section-heading">
      <div>
        <h2>基础设施</h2>
        <p>资源概览</p>
      </div>
    </div>
    <div class="empty-state">
      <div class="empty-icon">
        <el-icon :size="24"><Connection /></el-icon>
      </div>
      <strong>暂无资源信息</strong><span>主机和资源管理功能尚未实现</span>
    </div>
  </section>
  <div class="page-footnote"><span>NEXUSOPS</span><span>自托管基础设施管理</span></div>
</template>
