<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/api'
import { useRoomStore } from '@/stores/room'
import type { RoomReport } from '@/types'

const route = useRoute()
const roomStore = useRoomStore()
const report = ref<RoomReport | null>(roomStore.report)
const loading = ref(false)
const error = ref('')

async function loadReport() {
  if (report.value) return
  loading.value = true
  try {
    const data = await api.roomReport(String(route.params.id))
    report.value = data.report
  } catch (e: any) {
    error.value = e.message || '加载战报失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadReport)
</script>

<template>
  <main class="page stack">
    <section class="card stack" v-if="report">
      <h2>本局战报</h2>
      <div class="card inner-card">{{ report.hardmouthLabel }}</div>
      <div class="card inner-card">{{ report.bestAssistLabel }}</div>
      <div class="card inner-card">
        今日最安静时刻：冷场 {{ report.quietMomentSecs }} 秒
        <span class="muted" v-if="report.quietMomentAt">（{{ new Date(report.quietMomentAt).toLocaleTimeString() }}）</span>
      </div>
      <div class="card inner-card">{{ report.savefaceLabel }}</div>
      <div class="card inner-card quote-card">
        <div class="muted">Bot 今日金句</div>
        <h3 class="quote-text">{{ report.botQuote }}</h3>
      </div>
      <router-link class="btn" to="/">返回首页</router-link>
    </section>

    <section class="card" v-else-if="loading">战报生成中...</section>
    <section class="card text-danger" v-else>{{ error || '暂无战报' }}</section>
  </main>
</template>

<style scoped>
.inner-card {
  margin: 0;
}

.quote-card {
  border-color: var(--accent);
}

.quote-text {
  margin: 8px 0 0;
}
</style>
