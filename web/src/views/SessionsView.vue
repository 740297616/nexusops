<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listSessions, revokeUserSession, revokeUserSessions } from '../api/system'
import type { Page, Session } from '../types/api'
const data = ref<Page<Session>>({ items: [], total: 0, page: 1, page_size: 20 })
const userId = ref('')
async function load() {
  try {
    data.value = await listSessions({ page: data.value.page, page_size: data.value.page_size })
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  }
}
async function revoke(session: Session) {
  try {
    const target = userId.value
    if (!target) {
      ElMessage.warning('请输入用户 ID 后撤销跨用户会话')
      return
    }
    await ElMessageBox.confirm(`确认撤销用户 ${target} 的会话 ${session.id}？`, '确认操作')
    await revokeUserSession(Number(target), session.id)
    ElMessage.success('会话已撤销')
    load()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '撤销失败')
  }
}
async function revokeAll() {
  if (!userId.value) return
  try {
    await ElMessageBox.confirm(`确认撤销用户 ${userId.value} 的全部会话？`, '确认操作')
    await revokeUserSessions(Number(userId.value))
    ElMessage.success('会话已撤销')
    load()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '撤销失败')
  }
}
onMounted(load)
</script>
<template>
  <div class="page-heading">
    <div>
      <div class="eyebrow">SECURITY / SESSIONS</div>
      <h1>会话管理</h1>
      <p>管理员可查询和撤销系统会话</p>
    </div>
  </div>
  <section class="section-block">
    <div class="toolbar">
      <el-input v-model="userId" placeholder="目标用户 ID" /><el-button @click="revokeAll"
        >撤销该用户全部会话</el-button
      >
    </div>
    <el-table :data="data.items" stripe
      ><el-table-column prop="id" label="会话 ID" /><el-table-column
        prop="created_at"
        label="创建时间"
      /><el-table-column prop="expires_at" label="过期时间" /><el-table-column label="操作"
        ><template #default="{ row }"
          ><el-button link @click="revoke(row)">撤销</el-button></template
        ></el-table-column
      ></el-table
    ><el-pagination
      v-model:current-page="data.page"
      :page-size="data.page_size"
      :total="data.total"
      layout="total, prev, pager, next"
      @current-change="load"
    />
  </section>
</template>
