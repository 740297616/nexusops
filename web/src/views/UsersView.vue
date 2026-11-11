<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listUsers, setUserStatus, resetPassword } from '../api/system'
import type { User, Page } from '../types/api'
const data = ref<Page<User>>({ items: [], total: 0, page: 1, page_size: 20 })
const loading = ref(false)
const q = ref('')
const status = ref('')
async function load() {
  loading.value = true
  try {
    data.value = await listUsers({
      page: data.value.page,
      page_size: data.value.page_size,
      q: q.value,
      status: status.value,
    })
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}
async function toggle(u: User) {
  const next = u.status === 'active' ? 'disabled' : 'active'
  try {
    await ElMessageBox.confirm(
      `确认${next === 'disabled' ? '禁用' : '启用'}用户 ${u.username}？禁用会撤销其所有会话。`,
      '确认操作',
    )
    await setUserStatus(u.id, next)
    ElMessage.success('操作成功')
    load()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error instanceof Error ? error.message : '操作失败')
  }
}
async function password(u: User) {
  try {
    const { value } = await ElMessageBox.prompt(
      `为 ${u.username} 设置新密码（至少 12 位）`,
      '重置密码',
      { inputType: 'password', inputPattern: /^.{12,}$/, inputErrorMessage: '密码至少 12 位' },
    )
    if (value) {
      await resetPassword(u.id, value)
      ElMessage.success('密码已重置')
    }
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}
onMounted(load)
</script>
<template>
  <div class="page-heading">
    <div>
      <div class="eyebrow">SYSTEM / USERS</div>
      <h1>用户管理</h1>
      <p>管理用户状态、密码与访问权限</p>
    </div>
    <span class="page-tag">{{ data.total }} 个用户</span>
  </div>
  <section class="section-block">
    <div class="toolbar">
      <el-input
        v-model="q"
        placeholder="搜索用户名或显示名称"
        clearable
        @keyup.enter="load"
      /><el-select v-model="status" placeholder="全部状态" clearable @change="load"
        ><el-option label="启用" value="active" /><el-option
          label="禁用"
          value="disabled" /></el-select
      ><el-button @click="load">查询</el-button>
    </div>
    <el-table v-loading="loading" :data="data.items" stripe
      ><el-table-column prop="username" label="用户名" /><el-table-column
        prop="display_name"
        label="显示名称"
      /><el-table-column prop="status" label="状态"
        ><template #default="{ row }"
          ><el-tag :type="row.status === 'active' ? 'success' : 'info'">{{
            row.status === 'active' ? '启用' : '禁用'
          }}</el-tag></template
        ></el-table-column
      ><el-table-column prop="role" label="兼容角色" /><el-table-column
        prop="last_login_at"
        label="最近登录"
      /><el-table-column label="操作" width="180"
        ><template #default="{ row }"
          ><el-button link @click="toggle(row)">{{
            row.status === 'active' ? '禁用' : '启用'
          }}</el-button
          ><el-button link @click="password(row)">重置密码</el-button></template
        ></el-table-column
      ></el-table
    ><el-pagination
      v-model:current-page="data.page"
      v-model:page-size="data.page_size"
      :total="data.total"
      layout="total, prev, pager, next"
      @current-change="load"
    />
  </section>
</template>
