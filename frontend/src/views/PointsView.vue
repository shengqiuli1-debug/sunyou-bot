<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/api'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const ledger = ref<Array<Record<string, unknown>>>([])
const records = ref<Array<Record<string, unknown>>>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    await userStore.refreshMe()
    const [l, r] = await Promise.all([api.pointLedger(), api.roomRecords()])
    ledger.value = l.items
    records.value = r.items
  } finally {
    loading.value = false
  }
}

async function recharge(amount: number) {
  await api.recharge(amount)
  await load()
}

onMounted(load)
</script>

<template>
  <main class="page stack">
    <section class="card stack">
      <h2>我的点数</h2>
      <div class="row">
        <span class="muted">当前余额</span>
        <h1 class="balance">{{ userStore.user?.points ?? 0 }}</h1>
      </div>
      <div class="chips">
        <button class="btn" @click="recharge(10)">模拟充值 +10</button>
        <button class="btn" @click="recharge(30)">模拟充值 +30</button>
      </div>
      <div class="muted">说明：当前为模拟充值，后续可无缝升级真实支付。</div>
    </section>

    <section class="card stack">
      <h3>消费记录</h3>
      <div v-if="loading" class="muted">加载中...</div>
      <div v-for="item in ledger" :key="String(item.id)" class="card inner-card">
        <div class="row">
          <b>{{ item.reason }}</b>
          <span>{{ Number(item.changeAmount) > 0 ? '+' : '' }}{{ item.changeAmount }}</span>
        </div>
        <div class="muted">余额：{{ item.balanceAfter }} / {{ item.createdAt }}</div>
      </div>
    </section>

    <section class="card stack">
      <h3>房间记录</h3>
      <div v-for="item in records" :key="String(item.roomId)" class="card inner-card">
        <div class="row">
          <b>{{ item.shareCode }}</b>
          <span>{{ item.status }}</span>
        </div>
        <div class="muted">
          {{ item.botRole }} / {{ item.fireLevel }} / {{ item.durationMinutes }} 分钟
        </div>
      </div>
    </section>
  </main>
</template>

<style scoped>
.balance {
  margin: 0;
}

.inner-card {
  margin: 0;
}
</style>
