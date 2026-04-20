<script setup lang="ts">
import type { ChatMessage } from '@/types'

const props = defineProps<{
  modelValue: string
  loading: boolean
  replyingTo: ChatMessage | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  send: []
  cancelReply: []
}>()
</script>

<template>
  <div class="composer-wrap">
    <div v-if="props.replyingTo" class="replying-bar">
      <div class="replying-main">
        <div class="replying-title">正在回复 {{ props.replyingTo.nickname }}</div>
        <div class="replying-preview">{{ props.replyingTo.content }}</div>
      </div>
      <button class="cancel-btn" @click="emit('cancelReply')">取消</button>
    </div>

    <div class="composer">
      <input
        class="composer-input"
        :value="props.modelValue"
        @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
        @keydown.enter="emit('send')"
        placeholder="说点什么，Bot 正在听..."
      />
      <button class="send-btn" :disabled="props.loading" @click="emit('send')">
        {{ props.loading ? '...' : '发送' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.composer-wrap {
  position: fixed;
  left: 50%;
  transform: translateX(-50%);
  bottom: 0;
  width: min(760px, 100%);
  padding: 10px 12px calc(10px + env(safe-area-inset-bottom));
  border-top: 1px solid var(--line);
  background: var(--fixed-bottom-bg);
  backdrop-filter: blur(10px);
  z-index: 18;
}

.replying-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
  border: 1px solid var(--sheet-item-border);
  border-radius: 11px;
  padding: 7px 8px;
  background: var(--sheet-item-bg);
}

.replying-main {
  min-width: 0;
}

.replying-title {
  font-size: 12px;
  color: var(--text-main);
}

.replying-preview {
  margin-top: 2px;
  color: var(--text-sub);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cancel-btn {
  border: 0;
  border-radius: 8px;
  padding: 4px 8px;
  color: var(--btn-ghost-text);
  background: var(--btn-ghost-bg);
  font-size: 12px;
}

.composer {
  display: flex;
  gap: 10px;
}

.composer-input {
  width: 100%;
  border-radius: 12px;
  border: 1px solid var(--input-border);
  background: var(--input-bg);
  color: var(--input-text);
  padding: 10px 12px;
}

.send-btn {
  border: 0;
  border-radius: 12px;
  padding: 0 16px;
  color: var(--accent-contrast);
  font-weight: 700;
  background: linear-gradient(125deg, var(--btn-grad-1), var(--btn-grad-2));
}

.send-btn:disabled {
  opacity: 0.65;
}
</style>
