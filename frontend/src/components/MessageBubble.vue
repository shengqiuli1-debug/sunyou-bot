<script setup lang="ts">
import { computed, ref } from 'vue'
import type { BotRole, ChatMessage } from '@/types'
import { BOT_ROLE_META, resolveBotRole } from '@/constants/botRoles'

interface BotProfilePayload {
  role: BotRole
  name: string
  bio: string
  identity: string
  duty: string
}

const props = defineProps<{ msg: ChatMessage; selfUserId?: string }>()

const emit = defineEmits<{
  quote: [msg: ChatMessage]
  openProfile: [payload: BotProfilePayload]
}>()

const isSelf = computed(() => !!props.selfUserId && props.msg.userId === props.selfUserId)
const isBot = computed(() => props.msg.senderType === 'bot' || !!props.msg.isBotMessage)
const isSystem = computed(() => props.msg.senderType === 'system')
const canQuote = computed(() => !isSystem.value)

const touchTimer = ref<number | null>(null)

function doQuote() {
  if (!canQuote.value) return
  emit('quote', props.msg)
}

function onTouchStart() {
  if (!canQuote.value) return
  touchTimer.value = window.setTimeout(() => {
    doQuote()
  }, 420)
}

function onTouchEnd() {
  if (touchTimer.value) {
    clearTimeout(touchTimer.value)
    touchTimer.value = null
  }
}

const botRole = computed<BotRole>(() => resolveBotRole(props.msg.botRole, props.msg.nickname))
const botProfile = computed(() => BOT_ROLE_META[botRole.value])

const displayName = computed(() => {
  const raw = String(props.msg.nickname || '').trim()
  if (!isBot.value) return raw || '成员'
  return botProfile.value.name
})

const avatarText = computed(() => {
  if (isBot.value) return botProfile.value.avatarText
  const n = displayName.value
  return Array.from(n)[0] || '友'
})

function openProfile() {
  if (!isBot.value) return
  emit('openProfile', {
    role: botRole.value,
    name: displayName.value,
    bio: botProfile.value.bio,
    identity: botProfile.value.identity,
    duty: botProfile.value.duty
  })
}
</script>

<template>
  <div class="msg-row" :class="{ self: isSelf, bot: isBot, system: isSystem }">
    <div class="system-pill" v-if="isSystem">
      {{ msg.content }}
    </div>

    <template v-else>
      <button
        v-if="!isSelf"
        class="avatar"
        :class="[isBot ? `bot-${botRole}` : 'user']"
        :aria-label="`打开 ${displayName} 资料`"
        @click="openProfile"
      >
        {{ avatarText }}
      </button>

      <div class="bubble-wrap">
        <div class="meta">
          <div class="name-wrap">
            <button v-if="isBot" class="name-btn" @click="openProfile">{{ displayName }}</button>
            <b v-else>{{ displayName }}</b>
            <span v-if="isBot" class="hint">NPC</span>
          </div>
          <span class="time">{{ new Date(msg.createdAt).toLocaleTimeString() }}</span>
        </div>

        <div
          class="bubble"
          @touchstart="onTouchStart"
          @touchend="onTouchEnd"
          @touchcancel="onTouchEnd"
        >
          <div class="reply-box" v-if="msg.replyToMessageId && msg.replyToSenderName">
            <div class="reply-name">回复 {{ msg.replyToSenderName }}</div>
            <div class="reply-preview">{{ msg.replyToPreview || '' }}</div>
          </div>

          <div class="content">{{ msg.content }}</div>
        </div>

        <button class="quote-btn" v-if="canQuote" @click="doQuote">引用回复</button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.msg-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 10px;
}

.msg-row.self {
  justify-content: flex-end;
}

.msg-row.system {
  justify-content: center;
}

.system-pill {
  max-width: 86%;
  font-size: 12px;
  border-radius: 999px;
  padding: 6px 10px;
  color: var(--bubble-system-text);
  border: 1px solid var(--bubble-system-border);
  background: var(--bubble-system-bg);
}

.avatar {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  border: 1px solid var(--input-border);
  background: var(--bg-float);
  color: var(--text-main);
  font-size: 13px;
  font-weight: 700;
  flex: 0 0 auto;
  box-shadow: var(--card-shadow);
}

.avatar.user {
  background: var(--bg-float);
  border-color: var(--input-border);
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

.bubble-wrap {
  max-width: 88%;
}

.msg-row.self .bubble-wrap {
  align-items: flex-end;
}

.meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
  font-size: 12px;
  color: var(--bubble-meta);
}

.name-wrap {
  display: flex;
  align-items: center;
  gap: 6px;
}

.name-btn {
  border: 0;
  background: transparent;
  color: inherit;
  padding: 0;
  font-size: 12px;
  font-weight: 700;
}

.hint {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 999px;
  border: 1px solid var(--header-chip-border);
  color: var(--text-sub);
  background: var(--header-chip-bg);
  opacity: 0.85;
}

.time {
  color: var(--bubble-time);
}

.bubble {
  background: var(--bubble-user-bg);
  border: 1px solid var(--bubble-user-border);
  color: var(--bubble-user-text);
  border-radius: 14px;
  padding: 9px 10px;
}

.msg-row.self .bubble {
  background: var(--bubble-self-bg);
  border-color: var(--bubble-self-border);
  color: var(--bubble-self-text);
}

.msg-row.bot .bubble {
  background: var(--bubble-user-bg);
  border-color: var(--bubble-user-border);
  color: var(--bubble-user-text);
}

.msg-row.bot .bubble::before {
  content: '';
  display: block;
  width: 28px;
  height: 2px;
  margin-bottom: 8px;
  border-radius: 999px;
  background: var(--accent);
  opacity: 0.4;
}

.reply-box {
  border-left: 3px solid var(--reply-box-border);
  background: var(--reply-box-bg);
  border-radius: 8px;
  padding: 6px 8px;
  margin-bottom: 8px;
}

.reply-name {
  font-size: 12px;
  color: var(--reply-name);
}

.reply-preview {
  margin-top: 2px;
  font-size: 12px;
  color: var(--reply-preview);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.content {
  line-height: 1.45;
  white-space: pre-wrap;
}

.quote-btn {
  margin-top: 8px;
  border: 1px solid var(--quote-btn-border);
  background: var(--quote-btn-bg);
  color: var(--quote-btn-text);
  border-radius: 8px;
  font-size: 12px;
  padding: 4px 8px;
}

.msg-row.self .quote-btn {
  margin-left: auto;
  display: block;
}
</style>
