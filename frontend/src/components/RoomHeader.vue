<script setup lang="ts">
import { computed } from 'vue'
import type { Room } from '@/types'
import { BOT_ROLE_META, normalizeBotRole } from '@/constants/botRoles'

const props = defineProps<{
  room: Room | null
  wsConnected: boolean
}>()

const emit = defineEmits<{
  openMore: []
}>()

const roleText = computed(() => {
  const role = normalizeBotRole(props.room?.botRole)
  return BOT_ROLE_META[role].name
})

const fireText = computed(() => {
  const fire = props.room?.fireLevel
  if (fire === 'low') return '轻嘴'
  if (fire === 'high') return '抽象/发疯'
  return '阴阳'
})
</script>

<template>
  <header class="room-header">
    <div class="title-wrap">
      <div class="title">房间聊天</div>
      <div class="sub">分享码 {{ room?.shareCode || '--' }}</div>
    </div>

    <div class="meta-row">
      <span class="meta-chip">{{ roleText }}</span>
      <span class="meta-chip">{{ fireText }}</span>
      <span class="meta-chip" :class="{ online: wsConnected, offline: !wsConnected }">
        {{ wsConnected ? '实时连接' : '重连中' }}
      </span>
      <button class="more-btn" @click="emit('openMore')" aria-label="更多操作">⋯</button>
    </div>
  </header>
</template>

<style scoped>
.room-header {
  border: 1px solid var(--header-border);
  border-radius: 18px;
  padding: 14px;
  background: var(--header-bg);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.08);
}

.title-wrap {
  margin-bottom: 10px;
}

.title {
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 0.2px;
}

.sub {
  margin-top: 4px;
  color: var(--text-sub);
  font-size: 13px;
}

.meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.meta-chip {
  font-size: 12px;
  border-radius: 999px;
  border: 1px solid var(--header-chip-border);
  padding: 5px 10px;
  color: var(--header-chip-text);
  background: var(--header-chip-bg);
}

.meta-chip.online {
  border-color: var(--header-online-border);
  background: var(--header-online-bg);
  color: var(--header-online-text);
}

.meta-chip.offline {
  border-color: var(--header-offline-border);
  background: var(--header-offline-bg);
  color: var(--header-offline-text);
}

.more-btn {
  margin-left: auto;
  border: 1px solid var(--header-more-border);
  background: var(--header-more-bg);
  color: var(--header-more-text);
  border-radius: 10px;
  width: 34px;
  height: 32px;
  font-size: 20px;
  line-height: 20px;
}
</style>
