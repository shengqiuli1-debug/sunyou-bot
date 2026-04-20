<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { COPY } from '@/constants/copy'
import { api } from '@/api'
import { useThemeStore } from '@/stores/theme'
import ThemePickerSheet from '@/components/ThemePickerSheet.vue'

const router = useRouter()
const themeStore = useThemeStore()
const shareCode = ref('')
const loading = ref(false)
const error = ref('')
const showThemeSheet = ref(false)

async function goByShareCode() {
  error.value = ''
  if (!shareCode.value.trim()) return
  loading.value = true
  try {
    const { room } = await api.getRoomByShare(shareCode.value.trim())
    router.push(`/join/${room.id}`)
  } catch (e: any) {
    error.value = e.message || '房间不存在或已结束'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="page">
    <section class="card stack">
      <h1>{{ COPY.brand.title }}</h1>
      <p class="muted">{{ COPY.brand.slogan }}</p>
      <p class="muted">{{ COPY.safeReminder }}</p>
      <div class="chips">
        <router-link class="btn" to="/create">开一个临时房间</router-link>
        <router-link class="btn ghost" to="/points">我的点数</router-link>
      </div>
    </section>

    <section class="card stack">
      <h3>Bot 角色</h3>
      <div class="muted">{{ COPY.roleIntro.judge }}</div>
      <div class="muted">{{ COPY.roleIntro.npc }}</div>
      <div class="muted">{{ COPY.roleIntro.narrator }}</div>
    </section>

    <section class="card stack">
      <h3>三种入场身份</h3>
      <div class="muted">{{ COPY.identityIntro.normal }}</div>
      <div class="muted">{{ COPY.identityIntro.target }}</div>
      <div class="muted">{{ COPY.identityIntro.immune }}</div>
    </section>

    <section class="card stack">
      <h3>玩法介绍</h3>
      <div v-for="row in COPY.homePlayGuide" :key="row" class="muted">{{ row }}</div>
    </section>

    <section class="card stack">
      <h3>试玩引导</h3>
      <div class="row">
        <input class="field" v-model="shareCode" placeholder="输入房间分享码，例如 A8K3Q2" />
        <button class="btn" @click="goByShareCode" :disabled="loading">进入</button>
      </div>
      <div v-if="error" class="text-danger">{{ error }}</div>
    </section>

    <section class="card stack">
      <h3>主题外观</h3>
      <div class="row">
        <div>
          <div>{{ themeStore.currentTheme.name }}</div>
          <div class="muted">{{ themeStore.currentTheme.tone }}</div>
        </div>
        <button class="btn ghost" @click="showThemeSheet = true">切换主题</button>
      </div>
    </section>

    <ThemePickerSheet :open="showThemeSheet" @close="showThemeSheet = false" />
  </main>
</template>
