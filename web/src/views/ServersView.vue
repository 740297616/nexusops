<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createAgentRegistrationToken, createServer, deleteServer, getServer, listServerGroups, listServerTags, listServers, serverGroups, serverTags, setServerGroups, setServerTags, setServerStatus } from '../api/system'
import type { Page, ServerAsset, ServerGroup, ServerTag } from '../types/api'
const data = ref<Page<ServerAsset>>({ items: [], total: 0, page: 1, page_size: 20 })
const loading = ref(false)
const q = ref('')
const status = ref('')
const selectedToken = ref('')
const tokenDialog = ref(false)
const selectedServer = ref<ServerAsset | null>(null)
const detailDialog = ref(false)
const selectedGroupIDs = ref<number[]>([])
const selectedTagIDs = ref<number[]>([])
const groups = ref<ServerGroup[]>([])
const tags = ref<ServerTag[]>([])
const groupID = ref('')
const tagID = ref('')
async function load() {
  loading.value = true
  try { data.value = await listServers({ page: data.value.page, page_size: data.value.page_size, q: q.value, status: status.value, group_id: groupID.value, tag_id: tagID.value }) }
  catch (e) { ElMessage.error(e instanceof Error ? e.message : '加载失败') }
  finally { loading.value = false }
}
async function create() {
  try {
    const name = await ElMessageBox.prompt('服务器名称', '新增服务器', { inputPattern: /\S+/, inputErrorMessage: '名称不能为空' })
    await createServer({ name: name.value, description: '' }); ElMessage.success('服务器已创建'); await load()
  } catch (e) { if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '创建失败') }
}
async function toggle(row: ServerAsset) {
  try { const next = row.status === 'disabled' ? 'pending' : 'disabled'; await ElMessageBox.confirm(`确认${next === 'disabled' ? '禁用' : '启用'} ${row.name}？`, '确认操作'); await setServerStatus(row.id, next); ElMessage.success('状态已更新'); await load() }
  catch (e) { if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '操作失败') }
}
async function remove(row: ServerAsset) {
  try { await ElMessageBox.confirm(`确认删除 ${row.name}？该操作会撤销关联 Agent。`, '危险操作', { type: 'warning' }); await deleteServer(row.id); ElMessage.success('服务器已删除'); await load() }
  catch (e) { if (e !== 'cancel') ElMessage.error(e instanceof Error ? e.message : '删除失败') }
}
async function token(row: ServerAsset) {
  try { const result = await createAgentRegistrationToken(row.id); selectedServer.value = row; selectedToken.value = result.token; tokenDialog.value = true }
  catch (e) { ElMessage.error(e instanceof Error ? e.message : '生成 Token 失败') }
}
async function details(row: ServerAsset) {
  try { selectedServer.value = await getServer(row.id); selectedGroupIDs.value=(await serverGroups(row.id)).map((v)=>v.id); selectedTagIDs.value=(await serverTags(row.id)).map((v)=>v.id); detailDialog.value = true }
  catch (e) { ElMessage.error(e instanceof Error ? e.message : '加载详情失败') }
}
async function saveTaxonomy() {
  if (!selectedServer.value) return
  try { await setServerGroups(selectedServer.value.id, selectedGroupIDs.value); await setServerTags(selectedServer.value.id, selectedTagIDs.value); ElMessage.success('分组和标签已更新') } catch (e) { ElMessage.error(e instanceof Error ? e.message : '更新失败') }
}
function formatBytes(value: number) { if (!value) return '-'; const units = ['B', 'KB', 'MB', 'GB', 'TB']; let n = value; let i = 0; while (n >= 1024 && i < units.length - 1) { n /= 1024; i++ }; return `${n.toFixed(i ? 1 : 0)} ${units[i]}` }
onMounted(async () => { try { groups.value = (await listServerGroups()).items; tags.value = await listServerTags() } catch { /* server may not expose management permissions */ } await load() })
</script>
<template>
  <div class="page-heading"><div><div class="eyebrow">INFRASTRUCTURE / SERVERS</div><h1>服务器资产</h1><p>管理服务器资产与 Agent 接入状态</p></div><el-button type="primary" @click="create">新增服务器</el-button></div>
  <section class="section-block"><div class="toolbar"><el-input v-model="q" placeholder="搜索名称或主机名" clearable @keyup.enter="load" /><el-select v-model="status" placeholder="全部状态" clearable @change="load"><el-option label="待接入" value="pending" /><el-option label="在线" value="online" /><el-option label="离线" value="offline" /><el-option label="已禁用" value="disabled" /></el-select><el-select v-model="groupID" placeholder="全部分组" clearable @change="load"><el-option v-for="g in groups" :key="g.id" :label="g.name" :value="String(g.id)" /></el-select><el-select v-model="tagID" placeholder="全部标签" clearable @change="load"><el-option v-for="t in tags" :key="t.id" :label="t.name" :value="String(t.id)" /></el-select><el-button @click="load">查询</el-button></div>
    <el-table v-loading="loading" :data="data.items" stripe><el-table-column label="名称"><template #default="{ row }"><el-button link @click="details(row)">{{ row.name }}</el-button></template></el-table-column><el-table-column prop="hostname" label="主机名" /><el-table-column prop="primary_ip" label="IP" /><el-table-column prop="os" label="系统" /><el-table-column label="资源"><template #default="{ row }">{{ row.cpu_cores || '-' }} 核 / {{ formatBytes(row.memory_total) }}</template></el-table-column><el-table-column label="状态"><template #default="{ row }"><el-tag :type="row.status === 'online' ? 'success' : row.status === 'disabled' ? 'info' : 'warning'">{{ row.status }}</el-tag></template></el-table-column><el-table-column prop="last_heartbeat_at" label="最近心跳" /><el-table-column label="操作" width="280"><template #default="{ row }"><el-button link @click="token(row)">注册 Token</el-button><el-button link @click="toggle(row)">{{ row.status === 'disabled' ? '启用' : '禁用' }}</el-button><el-button link type="danger" @click="remove(row)">删除</el-button></template></el-table-column></el-table>
    <el-pagination v-model:current-page="data.page" :page-size="data.page_size" :total="data.total" layout="total, prev, pager, next" @current-change="load" />
  </section>
  <el-dialog v-model="tokenDialog" title="一次性 Agent 注册 Token" width="560px"><p>仅在本次显示。Token 过期时间由 Server 配置决定，使用后立即失效。</p><el-input :model-value="selectedToken" readonly /><p v-if="selectedServer">在目标 Linux 主机设置 <code>NEXUSOPS_AGENT_REGISTRATION_TOKEN</code> 后启动 Agent。</p></el-dialog>
  <el-dialog v-model="detailDialog" title="服务器详情" width="640px"><el-descriptions v-if="selectedServer" :column="2" border><el-descriptions-item label="名称">{{ selectedServer.name }}</el-descriptions-item><el-descriptions-item label="主机名">{{ selectedServer.hostname || '-' }}</el-descriptions-item><el-descriptions-item label="系统">{{ selectedServer.os }} {{ selectedServer.os_version }}</el-descriptions-item><el-descriptions-item label="架构">{{ selectedServer.architecture || '-' }}</el-descriptions-item><el-descriptions-item label="内核">{{ selectedServer.kernel_version || '-' }}</el-descriptions-item><el-descriptions-item label="CPU">{{ selectedServer.cpu_model || '-' }} / {{ selectedServer.cpu_cores }} 核</el-descriptions-item><el-descriptions-item label="内存">{{ formatBytes(selectedServer.memory_total) }}</el-descriptions-item><el-descriptions-item label="磁盘">{{ formatBytes(selectedServer.disk_total) }}</el-descriptions-item><el-descriptions-item label="Agent">{{ selectedServer.agent_version || '-' }}</el-descriptions-item><el-descriptions-item label="最近上报">{{ selectedServer.last_report_at || '-' }}</el-descriptions-item></el-descriptions><el-divider/><el-select v-model="selectedGroupIDs" multiple placeholder="选择分组"><el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" /></el-select><el-select v-model="selectedTagIDs" multiple placeholder="选择标签"><el-option v-for="t in tags" :key="t.id" :label="t.name" :value="t.id" /></el-select><el-button type="primary" @click="saveTaxonomy">保存分组和标签</el-button></el-dialog>
</template>
