<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listRoles, listPermissions, rolePermissions, setRolePermissions } from '../api/system'
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
async function choose(role: Role) {
  selected.value = role
  checked.value = (await rolePermissions(role.id)).map((p) => p.id)
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
        <el-button type="primary" :disabled="selected.is_system" @click="save">保存权限</el-button>
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
