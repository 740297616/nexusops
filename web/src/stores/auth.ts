import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as api from '../api/auth'
import { configureAuth, setAccessToken, setRefreshToken } from '../api/client'
import type { Profile } from '../types/api'
export const useAuthStore = defineStore('auth', () => {
  const profile = ref<Profile | null>(null)
  const ready = ref(false)
  const authenticated = computed(() => !!profile.value)
  const permissions = computed(() => new Set(profile.value?.permissions || []))
  function clear() {
    profile.value = null
    setAccessToken(null)
    setRefreshToken(null)
  }
  async function initialize() {
    configureAuth(clear)
    ready.value = true
  }
  async function load() {
    try {
      if (!profile.value) profile.value = await api.me()
    } catch {
      clear()
    } finally {
      ready.value = true
    }
  }
  async function signIn(username: string, password: string) {
    const tokens = await api.login(username, password)
    setAccessToken(tokens.access_token)
    setRefreshToken(tokens.refresh_token)
    profile.value = await api.me()
  }
  async function signOut() {
    try {
      if (authenticated.value) await api.logout()
    } finally {
      clear()
    }
  }
  function can(permission: string) {
    return permissions.value.has(permission)
  }
  return { profile, ready, authenticated, permissions, initialize, load, signIn, signOut, can }
})
