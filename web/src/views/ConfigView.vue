<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getConfig, setConfig } from '../api/system'
import type { ConfigItem } from '../types/api'
const items = ref<ConfigItem[]>([])
async function load() {
  try {
    items.value = await getConfig()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  }
}
async function save(item: ConfigItem) {
  try {
    await setConfig({ [item.key]: item.value })
    ElMessage.success('配置已保存')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}
onMounted(load)
</script>
<template>
  <div class="page-heading">
    <div>
      <div class="eyebrow">SYSTEM / CONFIGURATION</div>
      <h1>系统配置</h1>
      <p>仅显示服务端允许在线管理的非敏感配置</p>
    </div>
  </div>
  <section class="section-block">
    <el-table :data="items" stripe
      ><el-table-column prop="key" label="配置项" /><el-table-column label="值"
        ><template #default="{ row }"
          ><el-input v-model="row.value" :disabled="!row.editable" /></template></el-table-column
      ><el-table-column label="操作" width="120"
        ><template #default="{ row }"
          ><el-button link :disabled="!row.editable" @click="save(row)">保存</el-button></template
        ></el-table-column
      ></el-table
    >
  </section>
</template>
