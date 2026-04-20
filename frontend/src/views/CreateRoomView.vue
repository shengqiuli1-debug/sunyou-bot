<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api'
import { ApiError } from '@/api/http'
import { DURATION_OPTIONS, FIRE_OPTIONS, ROLE_OPTIONS } from '@/constants/copy'
import { useUserStore } from '@/stores/user'
import type { BotRole, FireLevel } from '@/types'

const router = useRouter()
const userStore = useUserStore()
const durationMinutes = ref(5)
const botRole = ref<BotRole>('judge')
const fireLevel = ref<FireLevel>('medium')
const generateReport = ref(true)
const loading = ref(false)
const error = ref('')
const recharging = ref(false)

const cost = computed(() => {
  const base: Record<number, number> = { 5: 2, 15: 5, 30: 8 }
  let v = base[durationMinutes.value] ?? 2
  if (fireLevel.value === 'high') v += 2
  if (fireLevel.value === 'low') v -= 1
  return Math.max(1, v)
})

const points = computed(() => userStore.user?.points ?? 0)
const freeTrialRooms = computed(() => userStore.user?.freeTrialRooms ?? 0)
const insufficientPoints = computed(() => freeTrialRooms.value <= 0 && points.value < cost.value)

onMounted(async () => {
  try {
    await userStore.refreshMe()
  } catch {
    // keep stale local user; room creation still goes through backend validation.
  }
})

async function submit() {
  error.value = ''
  if (insufficientPoints.value) {
    error.value = `点数不足：当前 ${points.value} 点，需要 ${cost.value} 点。先充值再开房。`
    return
  }
  loading.value = true
  try {
    const { room } = await api.createRoom({
      durationMinutes: durationMinutes.value,
      botRole: botRole.value,
      fireLevel: fireLevel.value,
      generateReport: generateReport.value
    })
    await userStore.refreshMe().catch(() => {})
    router.push(`/join/${room.id}?from=create`)
  } catch (e: any) {
    if (e instanceof ApiError && e.status === 402) {
      const required = e.data?.requiredCost ?? cost.value
      const current = e.data?.currentBalance ?? points.value
      error.value = `点数不足：当前 ${current} 点，需要 ${required} 点。`
      await userStore.refreshMe().catch(() => {})
    } else {
      error.value = e.message || '创建失败'
    }
  } finally {
    loading.value = false
  }
}

async function quickRecharge() {
  error.value = ''
  recharging.value = true
  try {
    await api.recharge(10)
    await userStore.refreshMe()
  } catch (e: any) {
    error.value = e.message || '充值失败'
  } finally {
    recharging.value = false
  }
}
</script>

<template>
  <main class="page stack">
    <section class="card stack">
      <h2>创建房间</h2>
      <label>房间时长</label>
      <div class="chips">
        <button
          v-for="item in DURATION_OPTIONS"
          :key="item.value"
          class="chip"
          :class="{ active: durationMinutes === item.value }"
          @click="durationMinutes = item.value"
        >
          {{ item.label }}
        </button>
      </div>

      <label>Bot 角色</label>
      <div class="chips">
        <button
          v-for="item in ROLE_OPTIONS"
          :key="item.value"
          class="chip"
          :class="{ active: botRole === item.value }"
          @click="botRole = item.value as BotRole"
        >
          {{ item.label }}
        </button>
      </div>

      <label>火力等级</label>
      <div class="chips">
        <button
          v-for="item in FIRE_OPTIONS"
          :key="item.value"
          class="chip"
          :class="{ active: fireLevel === item.value }"
          @click="fireLevel = item.value as FireLevel"
        >
          {{ item.label }}
        </button>
      </div>

      <label class="row">
        <span>是否生成战报</span>
        <input type="checkbox" v-model="generateReport" />
      </label>

      <div class="card inner-card">
        <div class="row"><span>预计消耗</span><strong>{{ cost }} 点</strong></div>
        <div class="row"><span>当前点数</span><strong>{{ points }} 点</strong></div>
        <div class="row" v-if="freeTrialRooms > 0"><span>免费开房</span><strong>剩余 {{ freeTrialRooms }} 次</strong></div>
        <div class="muted">新用户默认有 1 次免费试玩开房机会</div>
      </div>

      <button class="btn" @click="submit" :disabled="loading">{{ loading ? '创建中...' : '创建并进入入场确认' }}</button>
      <button v-if="insufficientPoints" class="btn btn-secondary" @click="quickRecharge" :disabled="recharging">
        {{ recharging ? '充值中...' : '点数不足，先模拟充值 10 点' }}
      </button>
      <div v-if="error" class="text-danger">{{ error }}</div>
    </section>
  </main>
</template>

<style scoped>
.inner-card {
  margin: 0;
}

.btn-secondary {
  background: transparent;
  border: 1px solid var(--btn-ghost-border, var(--border-default, rgba(255, 255, 255, 0.2)));
  color: var(--text-primary);
}
</style>
