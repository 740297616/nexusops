<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listAuthAudit } from '../api/system'
import type { AuthAudit, Page } from '../types/api'
const data = ref<Page<AuthAudit>>({ items: [], total: 0, page: 1, page_size: 20 })
async function load() {
  try {
    data.value = await listAuthAudit({ page: data.value.page, page_size: data.value.page_size })
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  }
}
onMounted(load)
</script>
<template>
  <div class="page-heading">
    <div>
      <div class="eyebrow">SECURITY / AUDIT</div>
      <h1>安全审计</h1>
      <p>认证和系统安全事件</p>
    </div>
  </div>
  <section class="section-block">
    <el-table :data="data.items" stripe
      ><el-table-column prop="created_at" label="时间" /><el-table-column
        prop="event_type"
        label="事件" /><el-table-column prop="outcome" label="结果" /><el-table-column
        prop="user_id"
        label="用户 ID" /><el-table-column prop="source_ip" label="来源 IP" /><el-table-column
        prop="user_agent"
        label="User-Agent"
        show-overflow-tooltip /></el-table
    ><el-pagination
      v-model:current-page="data.page"
      :page-size="data.page_size"
      :total="data.total"
      layout="total, prev, pager, next"
      @current-change="load"
    />
  </section>
</template>
