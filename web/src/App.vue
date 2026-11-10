<script setup lang="ts">
import AppLayout from './layouts/AppLayout.vue'
import { useThemeStore } from './stores/theme'

useThemeStore().initialize()
</script>

<template>
  <AppLayout />
</template>
