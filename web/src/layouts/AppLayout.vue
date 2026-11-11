<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Bell,
  Box,
  Connection,
  DataLine,
  Document,
  Grid,
  Key,
  Monitor,
  Moon,
  Setting,
  Sunny,
  User,
} from '@element-plus/icons-vue'
import { navigation } from '../router'
import { useThemeStore } from '../stores/theme'
import { useAuthStore } from '../stores/auth'
const icons = { Grid, User, Key, Connection, Document, Setting, Monitor, DataLine, Box, Bell }
const route = useRoute()
const router = useRouter()
const theme = useThemeStore()
const auth = useAuthStore()
const title = computed(() => String(route.meta.label || '总览'))
const items = computed(() =>
  navigation.filter((item) => !('permission' in item) || auth.can(item.permission)),
)
async function logout() {
  await auth.signOut()
  router.push('/login')
}
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
          v-for="item in items"
          :key="item.path"
          :to="item.path"
          :aria-label="item.label"
          :title="item.label"
          class="nav-item"
          :class="{ active: route.path === item.path }"
          ><el-icon :size="18"><component :is="icons[item.icon]" /></el-icon
          ><span>{{ item.label }}</span
          ><span v-if="route.path === item.path" class="active-dot"
        /></RouterLink>
      </nav>
      <div class="sidebar-footer"><span class="footer-dot" /> 自托管工作区</div>
    </aside>
    <div class="main-shell">
      <header class="topbar">
        <div class="breadcrumb">
          <span>工作区</span><span class="separator">/</span><strong>{{ title }}</strong>
        </div>
        <div class="top-actions">
          <button
            class="icon-button"
            type="button"
            :aria-label="theme.dark ? '切换到浅色主题' : '切换到深色主题'"
            @click="theme.toggle"
          >
            <el-icon :size="18"><Sunny v-if="theme.dark" /><Moon v-else /></el-icon></button
          ><el-dropdown trigger="click"
            ><button class="account-button">
              <span class="account-avatar">{{
                auth.profile?.display_name?.slice(0, 1) || 'U'
              }}</span
              ><span>{{ auth.profile?.display_name }}</span></button
            ><template #dropdown
              ><el-dropdown-menu
                ><el-dropdown-item disabled>{{ auth.profile?.username }}</el-dropdown-item
                ><el-dropdown-item @click="logout">退出登录</el-dropdown-item></el-dropdown-menu
              ></template
            ></el-dropdown
          >
        </div>
      </header>
      <main class="content"><RouterView /></main>
    </div>
  </div>
</template>
