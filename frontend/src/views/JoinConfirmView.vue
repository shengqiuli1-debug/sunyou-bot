<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api'
import { COPY, IDENTITY_OPTIONS } from '@/constants/copy'
import type { Identity, Room } from '@/types'

const route = useRoute()
const router = useRouter()
const room = ref<Room | null>(null)
const loading = ref(false)
const joining = ref(false)
const error = ref('')
const identity = ref<Identity>('normal')
const targetStep2 = ref(false)

async function loadRoom() {
  loading.value = true
  try {
    const res = await api.getRoom(String(route.params.id))
    room.value = res.room
  } catch (e: any) {
    error.value = e.message || '加载房间失败'
  } finally {
    loading.value = false
  }
}

async function join() {
  error.value = ''
  if (identity.value === 'target' && !targetStep2.value) {
    targetStep2.value = true
    return
  }

  joining.value = true
  try {
    await api.joinRoom(String(route.params.id), identity.value, identity.value === 'target')
    router.push(`/room/${route.params.id}`)
  } catch (e: any) {
    error.value = e.message || '入场失败'
  } finally {
    joining.value = false
  }
}

onMounted(loadRoom)
onMounted(() => {
  const fromQuery = String(route.query.identity || '').toLowerCase()
  if (fromQuery === 'normal' || fromQuery === 'target' || fromQuery === 'immune') {
    identity.value = fromQuery as Identity
    targetStep2.value = false
  }
})
</script>

<template>
  <main class="page stack">
    <section class="card stack" v-if="room">
      <h2>入场确认</h2>
      <div class="muted">房间分享码：{{ room.shareCode }}</div>
      <div class="muted">房间状态：{{ room.status }} / 结束时间 {{ new Date(room.endAt).toLocaleTimeString() }}</div>

      <h3>选择身份</h3>
      <div class="chips">
        <button
          v-for="item in IDENTITY_OPTIONS"
          :key="item.value"
          class="chip"
          :class="{ active: identity === item.value }"
          @click="identity = item.value as Identity; targetStep2 = false"
        >
          {{ item.label }}
        </button>
      </div>

      <div class="card target-tip" v-if="identity === 'target'">
        <div class="text-warn">target 需要二次确认</div>
        <div class="muted">{{ COPY.targetConfirm }}</div>
        <div class="muted">如果不想被重点关注，可以回到 normal。</div>
      </div>

      <button class="btn" @click="join" :disabled="joining || loading">
        {{ joining ? '入场中...' : identity === 'target' && !targetStep2 ? '继续确认 target 身份' : '确认入场' }}
      </button>

      <div v-if="error" class="text-danger">{{ error }}</div>
    </section>

    <section v-else-if="loading" class="card">加载中...</section>
    <section v-else class="card">{{ error || '房间不存在' }}</section>
  </main>
</template>

<style scoped>
.target-tip {
  margin: 0;
}
</style>
