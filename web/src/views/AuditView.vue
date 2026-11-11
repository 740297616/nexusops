<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listAuthAudit, listRoleAudit } from '../api/system'
import type { AuthAudit, Page, RoleAudit } from '../types/api'
const tab = ref('auth')
const auth = ref<Page<AuthAudit>>({ items: [], total: 0, page: 1, page_size: 20 })
const roles = ref<Page<RoleAudit>>({ items: [], total: 0, page: 1, page_size: 20 })
const event = ref('')
const outcome = ref('')
const from = ref('')
const to = ref('')
async function load() {
  try {
    if (tab.value === 'auth')
      auth.value = await listAuthAudit({
        page: auth.value.page,
        page_size: auth.value.page_size,
        event_type: event.value,
        outcome: outcome.value,
        from: from.value,
        to: to.value,
      })
    else
      roles.value = await listRoleAudit({
        page: roles.value.page,
        page_size: roles.value.page_size,
      })
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
      <p>只读查看认证与角色变更事件</p>
    </div>
  </div>
  <section class="section-block">
    <el-tabs v-model="tab" @tab-change="load"
      ><el-tab-pane label="认证事件" name="auth"
        ><div class="toolbar">
          <el-input v-model="event" placeholder="事件类型" clearable /><el-select
            v-model="outcome"
            placeholder="结果"
            clearable
            ><el-option label="成功" value="success" /><el-option
              label="失败"
              value="failure" /><el-option label="拒绝" value="denied" /></el-select
          ><el-date-picker
            v-model="from"
            type="datetime"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
            placeholder="开始时间"
          /><el-date-picker
            v-model="to"
            type="datetime"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
            placeholder="结束时间"
          /><el-button @click="load">筛选</el-button>
        </div>
        <el-table :data="auth.items" stripe
          ><el-table-column prop="created_at" label="时间" /><el-table-column
            prop="event_type"
            label="事件" /><el-table-column prop="outcome" label="结果" /><el-table-column
            prop="user_id"
            label="用户 ID" /><el-table-column prop="source_ip" label="来源 IP" /></el-table
        ><el-pagination
          v-model:current-page="auth.page"
          :page-size="auth.page_size"
          :total="auth.total"
          layout="total, prev, pager, next"
          @current-change="load" /></el-tab-pane
      ><el-tab-pane label="角色变更" name="roles"
        ><el-table :data="roles.items" stripe
          ><el-table-column prop="created_at" label="时间" /><el-table-column
            prop="action"
            label="操作" /><el-table-column prop="actor_user_id" label="操作者" /><el-table-column
            prop="target_user_id"
            label="目标用户" /></el-table
        ><el-pagination
          v-model:current-page="roles.page"
          :page-size="roles.page_size"
          :total="roles.total"
          layout="total, prev, pager, next"
          @current-change="load" /></el-tab-pane
    ></el-tabs>
  </section>
</template>
