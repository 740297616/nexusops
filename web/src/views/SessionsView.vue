<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listSessions } from '../api/system'
import type { Page, Session } from '../types/api'
const data = ref<Page<Session>>({ items: [], total: 0, page: 1, page_size: 20 })
async function load() {
  try {
    data.value = await listSessions({ page: data.value.page, page_size: data.value.page_size })
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  }
}
onMounted(load)
</script>
<template>
  <div class="page-heading">
    <div>
      <div class="eyebrow">SECURITY / SESSIONS</div>
      <h1>会话管理</h1>
      <p>查看当前有效的管理员会话</p>
    </div>
  </div>
  <section class="section-block">
    <el-table :data="data.items" stripe
      ><el-table-column prop="id" label="会话 ID" /><el-table-column
        prop="created_at"
        label="创建时间"
      /><el-table-column prop="expires_at" label="过期时间" /><el-table-column label="状态"
        ><template #default="{ row }"
          ><el-tag v-if="row.current" type="success">当前会话</el-tag
          ><span v-else>有效</span></template
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
