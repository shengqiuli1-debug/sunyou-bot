<script setup lang="ts">
import type { BotRole } from '@/types'
import { BOT_ROLE_META } from '@/constants/botRoles'

interface BotProfile {
  role: BotRole
  name: string
  bio: string
  identity: string
  duty: string
}

const props = defineProps<{
  open: boolean
  profile: BotProfile | null
}>()

const emit = defineEmits<{ close: [] }>()

function avatarText(role?: BotRole) {
  if (!role) return BOT_ROLE_META.npc.avatarText
  return BOT_ROLE_META[role].avatarText
}

function roleDesc(role?: BotRole) {
  if (!role) return BOT_ROLE_META.npc.styleDesc
  return BOT_ROLE_META[role].styleDesc
}
</script>

<template>
  <Teleport to="body">
    <div v-if="props.open" class="wrap">
      <button class="mask" @click="emit('close')"></button>

      <section class="panel">
        <header class="head">
          <div class="avatar" :class="props.profile ? `bot-${props.profile.role}` : 'bot-npc'">
            {{ avatarText(props.profile?.role) }}
          </div>
          <div class="meta">
            <h3>{{ props.profile?.name || '房间角色' }}</h3>
            <p>{{ props.profile?.bio || '房间特殊成员' }}</p>
          </div>
        </header>

        <div class="line">
          <span class="k">身份</span>
          <span class="v">{{ props.profile?.identity || '房间角色 / 特殊成员' }}</span>
        </div>
        <div class="line">
          <span class="k">职责</span>
          <span class="v">{{ props.profile?.duty || '-' }}</span>
        </div>
        <div class="line">
          <span class="k">风格</span>
          <span class="v">{{ roleDesc(props.profile?.role) }}</span>
        </div>

        <button class="close" @click="emit('close')">知道了</button>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
.wrap {
  position: fixed;
  inset: 0;
  z-index: 90;
}

.mask {
  position: absolute;
  inset: 0;
  border: 0;
  background: rgba(4, 8, 20, 0.48);
}

.panel {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: min(92vw, 360px);
  border-radius: 16px;
  border: 1px solid var(--sheet-border);
  background: var(--sheet-bg);
  box-shadow: var(--card-shadow);
  padding: 14px;
}

.head {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;
}

.avatar {
  width: 42px;
  height: 42px;
  border-radius: 50%;
  border: 1px solid var(--input-border);
  display: grid;
  place-items: center;
  font-weight: 700;
}

.avatar.bot-judge {
  background: linear-gradient(140deg, rgba(214, 176, 255, 0.22), rgba(141, 191, 255, 0.22));
  border-color: rgba(145, 150, 255, 0.5);
}

.avatar.bot-npc {
  background: linear-gradient(140deg, rgba(129, 228, 185, 0.25), rgba(120, 181, 255, 0.22));
  border-color: rgba(113, 189, 154, 0.52);
}

.avatar.bot-narrator {
  background: linear-gradient(140deg, rgba(168, 181, 255, 0.2), rgba(190, 138, 255, 0.2));
  border-color: rgba(149, 135, 231, 0.52);
}

.meta h3 {
  margin: 0;
  font-size: 17px;
}

.meta p {
  margin: 4px 0 0;
  color: var(--text-sub);
  font-size: 13px;
}

.line {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 8px;
}

.k {
  min-width: 34px;
  font-size: 12px;
  color: var(--text-sub);
}

.v {
  font-size: 13px;
  line-height: 1.45;
}

.close {
  margin-top: 14px;
  width: 100%;
  border: 0;
  border-radius: 10px;
  padding: 9px 0;
  background: var(--accent);
  color: var(--accent-contrast);
  font-weight: 700;
}
</style>
