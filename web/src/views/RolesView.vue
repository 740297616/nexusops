<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listRoles,
  listPermissions,
  rolePermissions,
  setRolePermissions,
  createRole,
  updateRole,
  setRoleStatus,
  deleteRole,
} from '../api/system'
import type { Role, Permission } from '../types/api'
const roles = ref<Role[]>([])
const permissions = ref<Permission[]>([])
const selected = ref<Role | null>(null)
const checked = ref<number[]>([])
async function load() {
  try {
    roles.value = await listRoles()
    permissions.value = await listPermissions()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  }
}
async function choose(r: Role) {
  selected.value = r
  checked.value = (await rolePermissions(r.id)).map((p) => p.id)
}
async function create() {
  try {
    const c = await ElMessageBox.prompt('角色标识（小写字母、数字、下划线）', '创建角色')
    const n = await ElMessageBox.prompt('角色名称', '创建角色')
    await createRole({ code: c.value, name: n.value, description: '' })
    ElMessage.success('角色已创建')
    load()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '创建失败')
  }
}
async function edit() {
  if (!selected.value) return
  try {
    const n = await ElMessageBox.prompt('角色名称', '编辑角色', { inputValue: selected.value.name })
    await updateRole(selected.value.id, { name: n.value, description: selected.value.description })
    ElMessage.success('已更新')
    load()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '更新失败')
  }
}
async function toggle() {
  if (!selected.value) return
  try {
    const next = selected.value.status === 'active' ? 'disabled' : 'active'
    await ElMessageBox.confirm(`确认${next === 'disabled' ? '禁用' : '启用'}角色？`, '确认操作')
    await setRoleStatus(selected.value.id, next)
    ElMessage.success('已更新')
    load()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}
async function remove() {
  if (!selected.value) return
  try {
    await ElMessageBox.confirm('删除角色前请确认没有用户依赖它。', '危险操作')
    await deleteRole(selected.value.id)
    ElMessage.success('已删除')
    selected.value = null
    load()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}
async function save() {
  if (!selected.value) return
  try {
    await setRolePermissions(selected.value.id, checked.value)
    ElMessage.success('权限已更新')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}
onMounted(load)
</script>
<template>
  <div class="page-heading">
    <div>
      <div class="eyebrow">SYSTEM / RBAC</div>
      <h1>角色与权限</h1>
      <p>权限修改会立即影响后续请求</p>
    </div>
    <el-button type="primary" @click="create">创建角色</el-button>
  </div>
  <div class="split-grid">
    <section class="section-block">
      <div class="section-heading">
        <div>
          <h2>角色</h2>
          <p>{{ roles.length }} 个角色</p>
        </div>
      </div>
      <el-table :data="roles" @row-click="choose"
        ><el-table-column prop="code" label="标识" /><el-table-column
          prop="name"
          label="名称"
        /><el-table-column label="状态"
          ><template #default="{ row }"
            ><el-tag>{{ row.status }}</el-tag
            ><el-tag v-if="row.is_system" type="warning" class="ml">系统</el-tag></template
          ></el-table-column
        ></el-table
      >
    </section>
    <section class="section-block">
      <div v-if="selected" class="section-heading">
        <div>
          <h2>{{ selected.name }}</h2>
          <p>{{ selected.code }}</p>
        </div>
        <div>
          <el-button v-if="!selected.is_system" @click="edit">编辑</el-button
          ><el-button v-if="!selected.is_system" @click="toggle">{{
            selected.status === 'active' ? '禁用' : '启用'
          }}</el-button
          ><el-button v-if="!selected.is_system" type="danger" @click="remove">删除</el-button
          ><el-button type="primary" :disabled="selected.is_system" @click="save"
            >保存权限</el-button
          >
        </div>
      </div>
      <div v-if="selected" class="permission-list">
        <el-checkbox-group v-model="checked"
          ><el-checkbox v-for="p in permissions" :key="p.id" :value="p.id" :label="p.code"
        /></el-checkbox-group>
      </div>
      <div v-else class="empty-state">
        <strong>选择一个角色</strong><span>查看或编辑其权限</span>
      </div>
    </section>
  </div>
</template>
