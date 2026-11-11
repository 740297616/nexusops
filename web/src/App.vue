<script setup lang="ts">
import AppLayout from './layouts/AppLayout.vue'
import { useThemeStore } from './stores/theme'
import { useRoute } from 'vue-router'
import { useAuthStore } from './stores/auth'

useThemeStore().initialize()
useAuthStore().initialize()
const route = useRoute()
</script>

<template>
  <RouterView v-if="route.meta.public" />
  <AppLayout v-else />
</template>
