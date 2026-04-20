import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api'
import type { User } from '@/types'

const TOKEN_KEY = 'sunyou_token'
const USER_KEY = 'sunyou_user'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem(TOKEN_KEY) || '')
  const user = ref<User | null>(JSON.parse(localStorage.getItem(USER_KEY) || 'null'))
  const loading = ref(false)

  async function ensureGuest() {
    if (token.value && user.value) return
    loading.value = true
    try {
      const nickname = `游客${Math.floor(Math.random() * 900 + 100)}`
      const data = await api.createGuest(nickname)
      token.value = data.token
      user.value = data.user
      localStorage.setItem(TOKEN_KEY, data.token)
      localStorage.setItem(USER_KEY, JSON.stringify(data.user))
    } finally {
      loading.value = false
    }
  }

  async function refreshMe() {
    if (!token.value) return
    const data = await api.me()
    user.value = data.user
    localStorage.setItem(USER_KEY, JSON.stringify(data.user))
  }

  async function refreshPoints() {
    if (!token.value || !user.value) return
    const data = await api.points()
    user.value.points = data.points
    localStorage.setItem(USER_KEY, JSON.stringify(user.value))
  }

  function setUser(next: User) {
    user.value = next
    localStorage.setItem(USER_KEY, JSON.stringify(next))
  }

  return { token, user, loading, ensureGuest, refreshMe, refreshPoints, setUser }
})
