<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listUsers,
  createUser,
  updateUser,
  setUserStatus,
  resetPassword,
  userRoles,
  setUserRoles,
  listRoles,
} from '../api/system'
import type { User, Page, Role } from '../types/api'
const data = ref<Page<User>>({ items: [], total: 0, page: 1, page_size: 20 })
const roles = ref<Role[]>([])
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
async function create() {
  try {
    const form = await ElMessageBox.prompt('请输入初始密码（至少 12 位）', '创建用户', {
      inputType: 'password',
      inputPattern: /^.{12,}$/,
      inputErrorMessage: '密码至少 12 位',
    })
    const username = await ElMessageBox.prompt('请输入用户名', '创建用户')
    const display = await ElMessageBox.prompt('请输入显示名称', '创建用户')
    await createUser({
      username: username.value,
      display_name: display.value,
      password: form.value,
    })
    ElMessage.success('用户已创建')
    load()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '创建失败')
  }
}
async function toggle(u: User) {
  try {
    const next = u.status === 'active' ? 'disabled' : 'active'
    await ElMessageBox.confirm(
      `确认${next === 'disabled' ? '禁用' : '启用'} ${u.username}？禁用会撤销全部会话。`,
      '确认操作',
    )
    await setUserStatus(u.id, next)
    ElMessage.success('操作成功')
    load()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}
async function edit(u: User) {
  try {
    const display = await ElMessageBox.prompt('修改显示名称', '编辑用户', {
      inputValue: u.display_name,
    })
    await updateUser(u.id, { display_name: display.value })
    ElMessage.success('已更新')
    load()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '更新失败')
  }
}
async function password(u: User) {
  try {
    const result = await ElMessageBox.prompt(`为 ${u.username} 设置新密码`, '重置密码', {
      inputType: 'password',
      inputPattern: /^.{12,}$/,
      inputErrorMessage: '密码至少 12 位',
    })
    await resetPassword(u.id, result.value)
    ElMessage.success('密码已重置，原会话已撤销')
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}
async function assignRoles(u: User) {
  try {
    if (!roles.value.length) roles.value = await listRoles()
    const current = await userRoles(u.id)
    const selected = await ElMessageBox.prompt(
      `当前角色：${current.map((r) => r.code).join(', ') || '无'}。请输入角色 ID，以逗号分隔`,
      '分配角色',
      { inputValue: current.map((r) => r.id).join(',') },
    )
    const ids = selected.value
      .split(',')
      .map((v) => Number(v.trim()))
      .filter(Boolean)
    await setUserRoles(u.id, ids)
    ElMessage.success('角色已更新')
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '角色更新失败')
  }
}
onMounted(load)
</script>
<template>
  <div class="page-heading">
    <div>
      <div class="eyebrow">SYSTEM / USERS</div>
      <h1>用户管理</h1>
      <p>用户状态、密码与角色</p>
    </div>
    <el-button type="primary" @click="create">创建用户</el-button>
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
      ><el-table-column prop="last_login_at" label="最近登录" /><el-table-column
        label="操作"
        width="310"
        ><template #default="{ row }"
          ><el-button link @click="edit(row)">编辑</el-button
          ><el-button link @click="assignRoles(row)">角色</el-button
          ><el-button link @click="password(row)">重置密码</el-button
          ><el-button link @click="toggle(row)">{{
            row.status === 'active' ? '禁用' : '启用'
          }}</el-button></template
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
