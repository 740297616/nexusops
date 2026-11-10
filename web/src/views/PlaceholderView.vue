<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Files } from '@element-plus/icons-vue'
const route = useRoute()
const title = computed(() => String(route.meta.label))
</script>

<template>
  <div class="page-heading">
    <div>
      <div class="eyebrow">WORKSPACE / {{ route.name }}</div>
      <h1>{{ title }}</h1>
      <p>基础设施管理</p>
    </div>
  </div>
  <section class="section-block">
    <div class="empty-state placeholder">
      <div class="empty-icon">
        <el-icon :size="24"><Files /></el-icon>
      </div>
      <strong>{{ title }}功能尚未实现</strong><span>此页面目前没有可显示的数据</span>
    </div>
  </section>
</template>
