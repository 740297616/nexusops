<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Bell, Box, DataLine, Grid, Monitor, Moon, Setting, Sunny } from '@element-plus/icons-vue'
import { navigation } from '../router'
import { useThemeStore } from '../stores/theme'

const icons = { Grid, Monitor, DataLine, Box, Bell, Setting }
const route = useRoute()
const theme = useThemeStore()
const pageTitle = computed(() => String(route.meta.label || '总览'))
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <span class="brand-mark">N</span
        ><span>Nexus<span class="brand-accent">Ops</span><small>INFRASTRUCTURE</small></span>
      </div>
      <div class="nav-heading">工作区</div>
      <nav aria-label="主导航">
        <RouterLink
          v-for="item in navigation"
          :key="item.path"
          :to="item.path"
          :aria-label="item.label"
          :title="item.label"
          class="nav-item"
          :class="{ active: route.path === item.path }"
        >
          <el-icon :size="18"><component :is="icons[item.icon]" /></el-icon
          ><span>{{ item.label }}</span
          ><span v-if="route.path === item.path" class="active-dot" />
        </RouterLink>
      </nav>
      <div class="sidebar-footer">
        <span class="footer-dot" /> 自托管工作区 <span class="footer-version">PHASE 1.1</span>
      </div>
    </aside>
    <div class="main-shell">
      <header class="topbar">
        <div class="breadcrumb">
          <span>工作区</span><span class="separator">/</span><strong>{{ pageTitle }}</strong>
        </div>
        <div class="top-actions">
          <span class="environment-label">NEXUSOPS CONSOLE</span
          ><el-tooltip
            :content="theme.dark ? '切换到浅色主题' : '切换到深色主题'"
            placement="bottom"
            ><button
              class="icon-button"
              type="button"
              :aria-label="theme.dark ? '切换到浅色主题' : '切换到深色主题'"
              @click="theme.toggle"
            >
              <el-icon :size="18"><Sunny v-if="theme.dark" /><Moon v-else /></el-icon></button
          ></el-tooltip>
        </div>
      </header>
      <main class="content"><RouterView /></main>
    </div>
  </div>
</template>
