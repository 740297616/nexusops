<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'
const auth = useAuthStore()
const router = useRouter()
const username = ref('')
const password = ref('')
const loading = ref(false)
async function submit() {
  if (!username.value || !password.value) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    await auth.signIn(username.value, password.value)
    ElMessage.success('登录成功')
    router.push('/')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '登录失败')
  } finally {
    loading.value = false
  }
}
</script>
<template>
  <div class="login-page">
    <div class="login-card">
      <div class="brand login-brand">
        <span class="brand-mark">N</span
        ><span
          >Nexus<span class="brand-accent">Ops</span><small>INFRASTRUCTURE CONSOLE</small></span
        >
      </div>
      <div class="login-copy">
        <div class="eyebrow">SECURE ADMIN CONSOLE</div>
        <h1>欢迎回来</h1>
        <p>登录以管理您的基础设施</p>
      </div>
      <el-form @submit.prevent="submit"
        ><el-form-item label="用户名"
          ><el-input
            v-model="username"
            autocomplete="username"
            placeholder="输入用户名" /></el-form-item
        ><el-form-item label="密码"
          ><el-input
            v-model="password"
            type="password"
            show-password
            autocomplete="current-password"
            placeholder="输入密码"
            @keyup.enter="submit" /></el-form-item
        ><el-button type="primary" native-type="submit" :loading="loading" class="login-button"
          >登录</el-button
        ></el-form
      >
      <div class="login-note">访问受保护的自托管管理控制台</div>
    </div>
  </div>
</template>
